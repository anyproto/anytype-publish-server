package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Object struct {
	// {Identity/Uri}
	Id              string              `json:"id" bson:"_id,omitempty"`
	ActivePublishId *primitive.ObjectID `json:"activePublishId" bson:"activePublishId,omitempty"`
	Identity        string              `json:"identity" bson:"identity"`
	SpaceId         string              `json:"spaceId" bson:"spaceId"`
	ObjectId        string              `json:"objectId" bson:"objectId"`
	Uri             string              `json:"uri" bson:"uri"`
	Timestamp       int64               `json:"timestamp" bson:"timestamp"`
}

type ObjectWithPublish struct {
	Object
	Publish *Publish
}
