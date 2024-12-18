package publish

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/periodicsync"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-publish-server/domain"
	"github.com/anyproto/anytype-publish-server/gateway/gatewayconfig"
	"github.com/anyproto/anytype-publish-server/publish/publishrepo"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
	"github.com/anyproto/anytype-publish-server/store"
)

const CName = "publish.service"

var log = logger.NewNamed(CName)

var defaultLimit = 10 << 20 // 10 Mb

func New() Service {
	return new(publishService)
}

type Service interface {
	ResolveUriWithName(ctx context.Context, name, uri string) (publish domain.ObjectWithPublish, err error)
	app.ComponentRunnable
}

type publishService struct {
	config        Config
	gatewayConfig gatewayconfig.Config
	store         store.Store
	repo          publishrepo.PublishRepo
	ticker        periodicsync.PeriodicSync
}

func (p *publishService) Init(a *app.App) (err error) {
	p.repo = a.MustComponent(publishrepo.CName).(publishrepo.PublishRepo)
	p.store = a.MustComponent(store.CName).(store.Store)
	p.config = a.MustComponent("config").(configGetter).GetPublish()
	p.gatewayConfig = a.MustComponent("config").(gatewayconfig.ConfigGetter).GetGateway()
	p.ticker = periodicsync.NewPeriodicSync(60, 0, p.Cleanup, log)
	return publishapi.DRPCRegisterWebPublisher(a.MustComponent(server.CName).(server.DRPCServer), &rpcHandler{s: p})
}

func (p *publishService) Run(ctx context.Context) (err error) {
	mux := http.NewServeMux()
	handler := httpHandler{s: p}
	handler.init(mux)
	var errChan = make(chan error)
	go func() {
		errChan <- http.ListenAndServe(p.config.HttpApiAddr, mux)
	}()
	select {
	case err = <-errChan:
		return err
	case <-time.After(200 * time.Millisecond):
		log.Info("http api server started", zap.String("addr", p.config.HttpApiAddr))
		return
	}
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

func (p *publishService) ResolveUriWithName(ctx context.Context, name, uri string) (publish domain.ObjectWithPublish, err error) {
	// TODO: do not request publish (only object)
	return p.repo.ResolveUri(ctx, name, uri)
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

func (p *publishService) UploadTar(ctx context.Context, publishId, uploadKey string, reader io.Reader) (resultUrl string, err error) {
	id, err := primitive.ObjectIDFromHex(publishId)
	if err != nil {
		return
	}
	publish, err := p.repo.GetPublish(ctx, id)
	if err != nil {
		return
	}
	if publish.UploadKey != uploadKey {
		return "", errors.New("invalid upload key")
	}
	if publish.Status != domain.PublishStatusCreated {
		return "", errors.New("publish is not in created state")
	}
	defer func() {
		if err != nil {
			_ = p.store.DeletePath(context.Background(), publishId)
		}
	}()
	var size int
	if size, err = p.uploadTar(ctx, publishId, reader, defaultLimit); err != nil {
		return
	}
	// TODO: validate here
	publish.Size = int64(size)
	publish.Status = domain.PublishStatusPublished
	publish.UploadKey = ""
	if err = p.repo.FinalizePublish(ctx, publish); err != nil {
		return
	}
	return url.JoinPath("https://", p.gatewayConfig.Domain, publish.ObjectId)
}

func (p *publishService) uploadTar(ctx context.Context, publishId string, reader io.Reader, limit int) (size int, err error) {
	tarReader := tar.NewReader(reader)
	var header *tar.Header
	for {
		if header, err = tarReader.Next(); errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return
		}
		if header.FileInfo().IsDir() {
			continue
		}
		fileName := strings.Join([]string{
			publishId,
			strings.TrimPrefix(header.Name, "/"),
		}, "/")
		file := store.File{
			Name:        fileName,
			ContentSize: int(header.Size),
			Reader:      tarReader,
		}
		if err = p.store.Put(ctx, file); err != nil {
			return
		}
		size += int(header.Size)
		if size > limit {
			return 0, errors.New("upload limit exceeded")
		}
	}
	return size, nil
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
