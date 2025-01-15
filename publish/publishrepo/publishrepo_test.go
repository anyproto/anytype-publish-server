package publishrepo

import (
	"context"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/anyproto/anytype-publish-server/db"
	"github.com/anyproto/anytype-publish-server/domain"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

var ctx = context.Background()

func newTestObj() domain.Object {
	return domain.Object{
		Identity: "a1",
		SpaceId:  "s1",
		ObjectId: "o1",
		Uri:      "u1",
	}
}

func TestPublishRepo_ObjectCreate(t *testing.T) {
	t.Run("new publish", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		publish, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		assertObject(t, obj, publish.Object)
		require.NotEmpty(t, publish.Publish)
		assert.Equal(t, "v1", publish.Publish.Version)
		assert.NotEmpty(t, publish.Publish.Id)
		assert.NotEmpty(t, publish.Publish.UploadKey)
	})
	t.Run("update same object", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		_, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		var publish domain.ObjectWithPublish
		publish, err = fx.ObjectCreate(ctx, obj, "v2")
		require.NoError(t, err)
		require.NotEmpty(t, publish.Publish)
		assert.Equal(t, "v2", publish.Publish.Version)
	})
	t.Run("change uri", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		_, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		obj.Uri = "u2"
		var publish domain.ObjectWithPublish
		publish, err = fx.ObjectCreate(ctx, obj, "v2")
		require.NoError(t, err)
		require.NotEmpty(t, publish.Publish)
		assert.Equal(t, "v2", publish.Publish.Version)
	})
	t.Run("change uri to the taken one", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		_, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		obj.ObjectId = "o2"
		_, err = fx.ObjectCreate(ctx, obj, "v2")
		require.ErrorIs(t, err, publishapi.ErrUriNotUnique)
	})
}

func TestPublishRepo_ObjectPublishStatus(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		_, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		publish, err := fx.ObjectPublishStatus(ctx, obj)
		require.NoError(t, err)
		assertObject(t, obj, publish.Object)
		assert.Nil(t, publish.Publish)
	})
	t.Run("published", func(t *testing.T) {
		fx := newFixture(t)
		obj := newTestObj()
		publishObj, err := fx.ObjectCreate(ctx, obj, "v1")
		require.NoError(t, err)
		uploadKey := publishObj.Publish.UploadKey
		publish, err := fx.GetPublish(ctx, publishObj.Publish.Id)
		require.NoError(t, err)
		assert.Equal(t, publish.Publish.UploadKey, uploadKey)
		publish.Publish.Size = 123
		publish.Publish.Status = domain.PublishStatusPublished
		require.NoError(t, fx.FinalizePublish(ctx, publish))
		publishObj, err = fx.ObjectPublishStatus(ctx, obj)
		require.NoError(t, err)
		require.NotNil(t, publishObj.Publish)
		assert.Equal(t, domain.PublishStatusPublished, publishObj.Publish.Status)
		assert.Equal(t, int64(123), publishObj.Publish.Size)
	})
}

func TestPublishRepo_IterateReadyToDeleteIds(t *testing.T) {
	fx := newFixture(t)
	docs := []any{
		domain.Publish{Id: primitive.NewObjectID(), Status: domain.PublishStatusReadyToDelete},
		domain.Publish{Id: primitive.NewObjectID(), Status: domain.PublishStatusReadyToDelete},
	}
	_, _ = fx.PublishRepo.(*publishRepo).publishColl.InsertMany(ctx, docs)
	var res []string
	err := fx.IterateReadyToDeleteIds(ctx, func(id primitive.ObjectID) error {
		res = append(res, id.Hex())
		return nil
	})
	require.NoError(t, err)
	var exp []string
	for _, d := range docs {
		exp = append(exp, d.(domain.Publish).Id.Hex())
	}
	assert.Equal(t, exp, res)
}

func assertObject(t *testing.T, expected domain.Object, got domain.Object) {
	assert.Equal(t, expected.Identity+"/"+expected.Uri, got.Id)
	assert.Equal(t, expected.SpaceId, got.SpaceId)
	assert.Equal(t, expected.ObjectId, got.ObjectId)
	assert.Equal(t, expected.Uri, got.Uri)
}

func newFixture(t testing.TB) *fixture {
	fx := &fixture{
		PublishRepo: New(),
		a:           new(app.App),
	}
	fx.a.Register(&testConfig{
		Mongo: db.Mongo{
			Connect:  "mongodb://localhost:27017",
			Database: "publish_unittest",
		},
	}).
		Register(db.New()).
		Register(fx.PublishRepo)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		fx.finish(t)
	})
	return fx
}

type fixture struct {
	PublishRepo
	a *app.App
}

func (fx *fixture) finish(t testing.TB) {
	_ = fx.PublishRepo.(*publishRepo).publishColl.Drop(ctx)
	_ = fx.PublishRepo.(*publishRepo).objectsColl.Drop(ctx)
	require.NoError(t, fx.a.Close(ctx))
}

type testConfig struct {
	Mongo db.Mongo
}

func (t testConfig) Init(a *app.App) (err error) {
	return
}

func (t testConfig) Name() (name string) {
	return "config"
}

func (t testConfig) GetMongo() db.Mongo {
	return t.Mongo
}
