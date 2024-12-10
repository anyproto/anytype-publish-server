package publishclient

import (
	"context"

	"github.com/anyproto/any-sync/app"

	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

type Client interface {
	app.Component
	ResolveUri(ctx context.Context, uri string) (publish *publishapi.Publish, err error)
	GetPublishStatus(ctx context.Context, spaceId, objectId string) (publish *publishapi.Publish, err error)
	Publish(ctx context.Context, req *publishapi.PublishRequest) (uploadUrl string, err error)
	UnPublish(ctx context.Context, req *publishapi.UnPublishRequest) (err error)
	ListPublishes(ctx context.Context, spaceId string) (publishes []*publishapi.Publish, err error)
	UploadDir(ctx context.Context, dir string) (err error)
}
