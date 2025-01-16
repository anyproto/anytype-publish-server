package nameservice

import (
	"context"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/app/ocache"
	"github.com/anyproto/any-sync/nameservice/nameserviceclient"
	"github.com/anyproto/any-sync/nameservice/nameserviceproto"
)

const CName = "publish.nameservice"

func New() NameService {
	return new(nameService)
}

var log = logger.NewNamed(CName)

type NameService interface {
	ResolveName(ctx context.Context, name string) (result *nameserviceproto.NameAvailableResponse, err error)
	ResolveIdentity(ctx context.Context, identity string) (name string, err error)
	app.ComponentRunnable
}

type nameService struct {
	nnClient  nameserviceclient.AnyNsClientServiceBase
	nameCache ocache.OCache
}

func (n *nameService) Init(a *app.App) (err error) {
	n.nnClient = a.MustComponent(nameserviceclient.CName).(nameserviceclient.AnyNsClientServiceBase)
	n.nameCache = ocache.New(n.resolveName, ocache.WithLogger(log.Sugar()), ocache.WithGCPeriod(time.Hour), ocache.WithTTL(time.Hour))
	return nil
}

func (n *nameService) Name() (name string) {
	return CName
}

func (n *nameService) Run(ctx context.Context) (err error) {
	return
}

func (n *nameService) ResolveName(ctx context.Context, name string) (result *nameserviceproto.NameAvailableResponse, err error) {
	obj, err := n.nameCache.Get(ctx, name)
	if err != nil {
		return
	}
	return obj.(*nameObject).nsResp, nil
}

func (n *nameService) ResolveIdentity(ctx context.Context, identity string) (name string, err error) {
	resp, err := n.nnClient.GetNameByAnyId(ctx, &nameserviceproto.NameByAnyIdRequest{
		AnyAddress: identity,
	})
	if err != nil {
		return
	}
	if !resp.Found {
		return "", ocache.ErrNotExists
	}
	return resp.Name, nil
}

type nameObject struct {
	nsResp *nameserviceproto.NameAvailableResponse
}

func (n *nameObject) Close() (err error) {
	return nil
}

func (n *nameObject) TryClose(objectTTL time.Duration) (res bool, err error) {
	if objectTTL > time.Hour {
		return true, nil
	} else {
		return false, nil
	}
}

func (n *nameService) resolveName(ctx context.Context, name string) (object ocache.Object, err error) {
	resp, err := n.nnClient.IsNameAvailable(ctx, &nameserviceproto.NameAvailableRequest{
		FullName: name + ".any",
	})
	if err != nil {
		return nil, err
	}
	return &nameObject{nsResp: resp}, nil
}

func (n *nameService) Close(ctx context.Context) (err error) {
	return n.nameCache.Close()
}
