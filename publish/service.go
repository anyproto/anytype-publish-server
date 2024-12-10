package publish

import (
	"context"
	"net/url"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/periodicsync"

	"github.com/anyproto/anytype-publish-server/domain"
	"github.com/anyproto/anytype-publish-server/publish/publishrepo"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

const CName = "publish.service"

var log = logger.NewNamed(CName)

type Service interface {
	app.ComponentRunnable
}

type publishService struct {
	config Config
	repo   publishrepo.PublishRepo
	ticker periodicsync.PeriodicSync
}

func (p *publishService) Init(a *app.App) (err error) {
	p.repo = a.MustComponent(publishrepo.CName).(publishrepo.PublishRepo)
	p.config = a.MustComponent("config").(configGetter).GetPublish()
	p.ticker = periodicsync.NewPeriodicSync(60, 0, p.Cleanup, log)
	return publishapi.DRPCRegisterWebPublisher(a.MustComponent(server.CName).(server.DRPCServer), &rpcHandler{s: p})
}

func (p *publishService) Run(ctx context.Context) (err error) {
	return
}

func (p *publishService) Name() (name string) {
	return CName
}

func (p *publishService) ResolveUri(ctx context.Context, uri string) (publish domain.ObjectWithPublish, err error) {
	identity, err := p.checkIdentity(ctx)
	if err != nil {
		return
	}
	return p.repo.ResolveUri(ctx, identity, uri)
}

func (p *publishService) GetPublishStatus(ctx context.Context, spaceId string, objectId string) (publish domain.ObjectWithPublish, err error) {
	identity, err := p.checkIdentity(ctx)
	if err != nil {
		return
	}
	obj := domain.Object{Identity: identity, SpaceId: spaceId, ObjectId: objectId}
	return p.repo.ObjectPublishStatus(ctx, obj)
}

func (p *publishService) Publish(ctx context.Context, object domain.Object, version string) (uploadUrl string, err error) {
	if object.Identity, err = p.checkIdentity(ctx); err != nil {
		return
	}
	publish, err := p.repo.ObjectCreate(ctx, object, version)
	if err != nil {
		return
	}
	return url.JoinPath(p.config.UploadUrlPrefix, publish.Publish.Id.Hex(), publish.Publish.UploadKey)
}

func (p *publishService) UnPublish(ctx context.Context, object domain.Object) (err error) {
	if object.Identity, err = p.checkIdentity(ctx); err != nil {
		return
	}
	return p.repo.ObjectDelete(ctx, object)
}

func (p *publishService) ListPublishes(ctx context.Context) (list []domain.ObjectWithPublish, err error) {
	identity, err := p.checkIdentity(ctx)
	if err != nil {
		return
	}
	return p.repo.ListPublishes(ctx, identity)
}

func (p *publishService) Cleanup(ctx context.Context) error {
	return nil
}

func (p *publishService) checkIdentity(ctx context.Context) (identity string, err error) {
	pubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return
	}
	return pubKey.Account(), nil
}

func (p *publishService) Close(ctx context.Context) (err error) {
	p.ticker.Close()
	return
}
