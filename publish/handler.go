package publish

import (
	"context"

	"github.com/anyproto/anytype-publish-server/domain"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

var _ publishapi.DRPCWebPublisherServer = (*rpcHandler)(nil)

type rpcHandler struct {
	s *publishService
}

func (r rpcHandler) ResolveUri(ctx context.Context, req *publishapi.ResolveUriRequest) (*publishapi.ResolveUriResponse, error) {
	obj, err := r.s.ResolveUri(ctx, req.Uri)
	if err != nil {
		return nil, err
	}
	return &publishapi.ResolveUriResponse{
		Publish: toPublish(obj),
	}, nil
}

func (r rpcHandler) GetPublishStatus(ctx context.Context, req *publishapi.GetPublishStatusRequest) (*publishapi.GetPublishStatusResponse, error) {
	obj, err := r.s.GetPublishStatus(ctx, req.SpaceId, req.ObjectId)
	if err != nil {
		return nil, err
	}
	return &publishapi.GetPublishStatusResponse{
		Publish: toPublish(obj),
	}, nil
}

func (r rpcHandler) Publish(ctx context.Context, req *publishapi.PublishRequest) (*publishapi.PublishResponse, error) {
	uploadUrl, err := r.s.Publish(ctx, domain.Object{SpaceId: req.SpaceId, ObjectId: req.ObjectId, Uri: req.Uri}, req.Version)
	if err != nil {
		return nil, err
	}
	return &publishapi.PublishResponse{
		UploadUrl: uploadUrl,
	}, nil
}

func (r rpcHandler) UnPublish(ctx context.Context, req *publishapi.UnPublishRequest) (resp *publishapi.Ok, err error) {
	if err = r.s.UnPublish(ctx, domain.Object{SpaceId: req.SpaceId, ObjectId: req.ObjectId}); err != nil {
		return
	}
	return &publishapi.Ok{}, nil
}

func (r rpcHandler) ListPublishes(ctx context.Context, req *publishapi.ListPublishesRequest) (*publishapi.ListPublishesResponse, error) {
	list, err := r.s.ListPublishes(ctx)
	if err != nil {
		return nil, err
	}
	resp := &publishapi.ListPublishesResponse{
		Publishes: make([]*publishapi.Publish, len(list)),
	}
	for i := range list {
		resp.Publishes[i] = toPublish(list[i])
	}
	return resp, nil
}

func toPublish(obj domain.ObjectWithPublish) *publishapi.Publish {
	publish := &publishapi.Publish{
		SpaceId:   obj.SpaceId,
		ObjectId:  obj.ObjectId,
		Uri:       obj.Uri,
		Timestamp: obj.Timestamp,
	}
	if obj.Publish != nil {
		if obj.Publish.Status == domain.PublishStatusPublished {
			publish.Status = publishapi.PublishStatus_PublishStatusPublished
			publish.Version = obj.Publish.Version
			publish.Size_ = obj.Publish.Size
		}
	}
	return publish
}
