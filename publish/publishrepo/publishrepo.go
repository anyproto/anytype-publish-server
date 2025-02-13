package publishrepo

import (
	"context"
	"errors"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/anyproto/anytype-publish-server/db"
	"github.com/anyproto/anytype-publish-server/domain"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

const CName = "publish.repo"

func New() PublishRepo {
	return new(publishRepo)
}

type PublishRepo interface {
	ObjectCreate(ctx context.Context, object domain.Object, version string) (publish domain.ObjectWithPublish, err error)
	ObjectDelete(ctx context.Context, object domain.Object) (uri string, err error)
	ObjectPublishStatus(ctx context.Context, object domain.Object) (publish domain.ObjectWithPublish, err error)
	ResolveUri(ctx context.Context, identity, uri string) (publish domain.ObjectWithPublish, err error)
	ResolvePublishUri(ctx context.Context, identity, uri string) (publish domain.Object, err error)
	ListPublishes(ctx context.Context, identity string, spaceId string) ([]domain.ObjectWithPublish, error)
	GetPublish(ctx context.Context, id primitive.ObjectID) (publish domain.ObjectWithPublish, err error)
	FinalizePublish(ctx context.Context, publish domain.ObjectWithPublish) (err error)
	IterateReadyToDeleteIds(ctx context.Context, do func(id primitive.ObjectID) error) error
	DeletePublish(ctx context.Context, id primitive.ObjectID) (err error)
	DeleteOutdatedPublishes(ctx context.Context, before time.Time) (deletedCount int, err error)
	DeleteOutdatedObjects(ctx context.Context, before time.Time) (deletedCount int, err error)
	app.ComponentRunnable
}

var (
	publishIndexes = []mongo.IndexModel{
		{
			Keys: bson.D{
				{"status", 1},
			},
		},
	}
	objectIndexes = []mongo.IndexModel{
		{
			Keys: bson.D{
				{"identity", 1},
				{"spaceId", 1},
				{"objectId", 1},
			},
		},
	}
)

type publishRepo struct {
	db          db.Database
	publishColl *mongo.Collection
	objectsColl *mongo.Collection
}

func (p *publishRepo) Name() (name string) {
	return CName
}

func (p *publishRepo) Init(a *app.App) (err error) {
	p.db = a.MustComponent(db.CName).(db.Database)
	p.publishColl = p.db.Db().Collection("publish")
	p.objectsColl = p.db.Db().Collection("object")
	return
}

func (p *publishRepo) Run(ctx context.Context) (err error) {
	if err = ensureIndexes(ctx, p.objectsColl, objectIndexes...); err != nil {
		return
	}
	if err = ensureIndexes(ctx, p.publishColl, publishIndexes...); err != nil {
		return
	}
	return
}

func ensureIndexes(ctx context.Context, coll *mongo.Collection, indexes ...mongo.IndexModel) (err error) {
	existingIndexes, err := coll.Indexes().ListSpecifications(ctx)
	if err != nil {
		return
	}
	if len(existingIndexes) <= 1 {
		_, err = coll.Indexes().CreateMany(ctx, indexes)
	}
	return
}

func (p *publishRepo) ObjectCreate(ctx context.Context, object domain.Object, version string) (publish domain.ObjectWithPublish, err error) {
	objectId := object.Identity + "/" + object.Uri
	err = p.db.Tx(ctx, func(ctx mongo.SessionContext) (err error) {
		// check if we have the sharing for the space+object pair
		var existingObject *domain.Object
		query := bson.D{{"identity", object.Identity}, {"spaceId", object.SpaceId}, {"objectId", object.ObjectId}}
		if err = p.objectsColl.FindOne(ctx, query).Decode(&existingObject); err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				return
			}
		}
		if existingObject != nil {
			// change the uri
			if existingObject.Uri != object.Uri {
				if err = p.changeObjectUri(ctx, existingObject, object.Uri); err != nil {
					return
				}
			}
		} else {
			existingObject = &domain.Object{
				Id:        objectId,
				Identity:  object.Identity,
				SpaceId:   object.SpaceId,
				ObjectId:  object.ObjectId,
				Uri:       object.Uri,
				Timestamp: time.Now().Unix(),
			}
			if _, err = p.objectsColl.InsertOne(ctx, existingObject); err != nil {
				if mongo.IsDuplicateKeyError(err) {
					return publishapi.ErrUriNotUnique
				} else {
					return err
				}
			}
		}
		publish.Object = *existingObject
		if publish.Publish, err = p.createPublish(ctx, existingObject, version); err != nil {
			return
		}
		return
	})
	if err != nil {
		return
	}
	return
}

func (p *publishRepo) changeObjectUri(ctx context.Context, object *domain.Object, uri string) (err error) {
	if _, err = p.objectsColl.DeleteOne(ctx, bson.D{{"_id", object.Id}}); err != nil {
		return
	}
	object.Id = object.Identity + "/" + uri
	object.Uri = uri
	if _, err = p.objectsColl.InsertOne(ctx, object); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return publishapi.ErrUriNotUnique
		}
		return
	}
	return
}

func (p *publishRepo) createPublish(ctx context.Context, object *domain.Object, version string) (publish *domain.Publish, err error) {
	publish = &domain.Publish{
		Id:        primitive.NewObjectID(),
		ObjectId:  object.Id,
		Status:    domain.PublishStatusCreated,
		Version:   version,
		UploadKey: uuid.New().String(),
	}
	if _, err = p.publishColl.InsertOne(ctx, publish); err != nil {
		return
	}
	return
}

func (p *publishRepo) ObjectPublishStatus(ctx context.Context, object domain.Object) (publish domain.ObjectWithPublish, err error) {
	return p.getPublishByQuery(ctx, bson.D{
		{"identity", object.Identity},
		{"spaceId", object.SpaceId},
		{"objectId", object.ObjectId},
	}, true)
}

func (p *publishRepo) ResolveUri(ctx context.Context, identity, uri string) (publish domain.ObjectWithPublish, err error) {
	return p.getPublishByQuery(ctx, bson.D{{"_id", identity + "/" + uri}}, true)
}

func (p *publishRepo) ResolvePublishUri(ctx context.Context, identity, uri string) (publish domain.Object, err error) {
	objectWithPublish, err := p.getPublishByQuery(ctx, bson.D{{"_id", identity + "/" + uri}}, false)
	if err != nil {
		return
	}
	return objectWithPublish.Object, nil
}

func (p *publishRepo) getPublishByQuery(ctx context.Context, query any, withPublish bool) (publish domain.ObjectWithPublish, err error) {
	if err = p.objectsColl.FindOne(ctx, query).Decode(&publish.Object); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ObjectWithPublish{}, publishapi.ErrNotFound
		} else {
			return
		}
	}
	if withPublish && publish.ActivePublishId != nil {
		if err = p.publishColl.FindOne(ctx, bson.M{"_id": *publish.ActivePublishId}).Decode(&publish.Publish); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				publish.ActivePublishId = nil
			} else {
				return
			}
		}
	}
	return
}

func (p *publishRepo) ListPublishes(ctx context.Context, identity string, spaceId string) ([]domain.ObjectWithPublish, error) {
	filter := bson.D{{"identity", identity}}
	if spaceId != "" {
		filter = append(filter, bson.E{Key: "spaceId", Value: spaceId})
	}
	cur, err := p.objectsColl.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	var publishes []domain.ObjectWithPublish
	for cur.Next(ctx) {
		var publish domain.ObjectWithPublish
		if err = cur.Decode(&publish.Object); err != nil {
			return nil, err
		}
		if publish.ActivePublishId != nil {
			_ = p.publishColl.FindOne(ctx, bson.D{{"_id", *publish.ActivePublishId}}).Decode(&publish.Publish)
		}
		publishes = append(publishes, publish)
	}
	return publishes, nil
}

func (p *publishRepo) ObjectDelete(ctx context.Context, object domain.Object) (uri string, err error) {
	err = p.db.Tx(ctx, func(ctx mongo.SessionContext) (err error) {
		var query = bson.D{{"identity", object.Identity}, {"spaceId", object.SpaceId}, {"objectId", object.ObjectId}}
		var existingObject domain.Object
		if err = p.objectsColl.FindOne(ctx, query).Decode(&existingObject); err != nil {
			return
		}
		uri = existingObject.Uri
		if _, err = p.objectsColl.DeleteOne(ctx, bson.D{{"_id", existingObject.Id}}); err != nil {
			return
		}
		if existingObject.ActivePublishId != nil {
			return p.markPublishToDelete(ctx, *existingObject.ActivePublishId)
		}
		return
	})
	return
}

func (p *publishRepo) markPublishToDelete(ctx context.Context, id primitive.ObjectID) (err error) {
	if _, err = p.publishColl.UpdateOne(
		ctx,
		bson.D{{"_id", id}},
		bson.D{{"$set", bson.D{{"status", domain.PublishStatusReadyToDelete}}}},
	); err != nil {
		return
	}
	return
}

func (p *publishRepo) GetPublish(ctx context.Context, id primitive.ObjectID) (publish domain.ObjectWithPublish, err error) {
	var pub domain.Publish
	if err = p.publishColl.FindOne(ctx, bson.D{{"_id", id}}).Decode(&pub); err != nil {
		return
	}
	var obj domain.Object
	if err = p.objectsColl.FindOne(ctx, bson.D{{"_id", pub.ObjectId}}).Decode(&obj); err != nil {
		return
	}
	return domain.ObjectWithPublish{
		Object:  obj,
		Publish: &pub,
	}, nil
}

func (p *publishRepo) FinalizePublish(ctx context.Context, publish domain.ObjectWithPublish) (err error) {
	return p.db.Tx(ctx, func(ctx mongo.SessionContext) (err error) {
		var obj = publish.Object
		// mark previous publish to delete
		if obj.ActivePublishId != nil {
			if err = p.markPublishToDelete(ctx, *obj.ActivePublishId); err != nil {
				return err
			}
		}
		// update publish
		if _, err = p.publishColl.UpdateOne(
			ctx,
			bson.D{{"_id", publish.Publish.Id}},
			bson.D{{"$set", bson.D{
				{"status", publish.Publish.Status},
				{"size", publish.Publish.Size},
			}}},
		); err != nil {
			return
		}
		// update object
		if _, err = p.objectsColl.UpdateOne(
			ctx,
			bson.D{{"_id", obj.Id}},
			bson.D{{"$set", bson.D{
				{"activePublishId", publish.Publish.Id},
				{"updatedTimestamp", time.Now().Unix()},
			}}},
		); err != nil {
			return
		}
		return
	})
}

func (p *publishRepo) IterateReadyToDeleteIds(ctx context.Context, do func(id primitive.ObjectID) error) error {
	opts := options.Find().SetProjection(bson.D{{"_id", 1}})
	cur, err := p.publishColl.Find(ctx, bson.D{{"status", domain.PublishStatusReadyToDelete}}, opts)
	if err != nil {
		return err
	}
	defer func() {
		_ = cur.Close(context.Background())
	}()
	var doc = struct {
		Id primitive.ObjectID `bson:"_id"`
	}{}
	for cur.Next(ctx) {
		if err = cur.Decode(&doc); err != nil {
			return err
		}
		if err = do(doc.Id); err != nil {
			return err
		}
	}
	return nil
}

func (p *publishRepo) DeletePublish(ctx context.Context, id primitive.ObjectID) (err error) {
	_, err = p.publishColl.DeleteOne(ctx, bson.D{{"_id", id}})
	return
}

func (p *publishRepo) DeleteOutdatedPublishes(ctx context.Context, before time.Time) (deleted int, err error) {
	query := bson.D{
		{"status", domain.PublishStatusCreated},
		{"_id", bson.D{
			{"$lt", primitive.NewObjectIDFromTimestamp(before)},
		}},
	}
	res, err := p.publishColl.DeleteMany(ctx, query)
	if err != nil {
		return
	}
	return int(res.DeletedCount), nil
}

func (p *publishRepo) DeleteOutdatedObjects(ctx context.Context, before time.Time) (deleted int, err error) {
	query := bson.D{
		{"activePublishId", bson.D{
			{"$exists", false},
		}},
		{"timestamp", bson.D{
			{"$lt", before.Unix()},
		}},
	}
	res, err := p.publishColl.DeleteMany(ctx, query)
	if err != nil {
		return
	}
	return int(res.DeletedCount), nil
}

func (p *publishRepo) Close(ctx context.Context) (err error) {
	return
}
