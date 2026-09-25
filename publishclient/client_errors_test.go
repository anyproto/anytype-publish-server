package publishclient

import (
	"context"
	"errors"
	"testing"

	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/rpcerr"
	"storj.io/drpc"
	"storj.io/drpc/drpcerr"

	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

var otherServiceURIError = rpcerr.RegisterErr(errors.New("another service's error"), 1103)

type errorPool struct {
	pool.Pool
}

func (errorPool) GetOneOf(context.Context, []string) (peer.Peer, error) {
	return errorPeer{}, nil
}

type errorPeer struct {
	peer.Peer
}

func (errorPeer) DoDrpc(_ context.Context, do func(drpc.Conn) error) error {
	return do(errorConn{})
}

type errorConn struct {
	drpc.Conn
}

func (errorConn) Invoke(context.Context, string, drpc.Encoding, drpc.Message, drpc.Message) error {
	return drpcerr.WithCode(errors.New("uri already taken"), 1103)
}

func TestRPCsDecodePublisherErrors(t *testing.T) {
	p := &publishClient{pool: errorPool{}}
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"ResolveUri":       func() error { _, err := p.ResolveUri(ctx, "uri"); return err },
		"GetPublishStatus": func() error { _, err := p.GetPublishStatus(ctx, "space", "object"); return err },
		"Publish":          func() error { _, err := p.Publish(ctx, &publishapi.PublishRequest{}); return err },
		"UnPublish":        func() error { return p.UnPublish(ctx, &publishapi.UnPublishRequest{}) },
		"ListPublishes":    func() error { _, err := p.ListPublishes(ctx, "space"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if !errors.Is(err, publishapi.ErrUriNotUnique) {
				t.Fatalf("decoded %v, want publisher ErrUriNotUnique", err)
			}
			if errors.Is(err, otherServiceURIError) {
				t.Fatal("publisher response decoded as another service's error")
			}
		})
	}
}
