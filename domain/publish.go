package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type PublishStatus uint8

const (
	PublishStatusCreated PublishStatus = iota
	PublishStatusPublished
	PublishStatusReadyToDelete
)

type Publish struct {
	Id        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ObjectId  string             `json:"objectId" bson:"objectId"`
	Status    PublishStatus      `json:"status" bson:"status"`
	Version   string             `json:"version" bson:"version"`
	UploadKey string             `json:"uploadKey" bson:"uploadKey"`
	Size      int64              `json:"size" bson:"size"`
}
