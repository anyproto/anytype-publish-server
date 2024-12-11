package publishclient

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/secureservice"
	"storj.io/drpc"

	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

func New() Client {
	return new(publishClient)
}

const CName = "publish.client"

type Client interface {
	app.Component
	ResolveUri(ctx context.Context, uri string) (publish *publishapi.Publish, err error)
	GetPublishStatus(ctx context.Context, spaceId, objectId string) (publish *publishapi.Publish, err error)
	Publish(ctx context.Context, req *publishapi.PublishRequest) (uploadUrl string, err error)
	UnPublish(ctx context.Context, req *publishapi.UnPublishRequest) (err error)
	ListPublishes(ctx context.Context, spaceId string) (publishes []*publishapi.Publish, err error)
	UploadDir(ctx context.Context, uploadUrl, dir string) (err error)
}

type publishClient struct {
	pool        pool.Pool
	peerService peerservice.PeerService
	peerIds     []string
}

func (p *publishClient) Init(a *app.App) (err error) {
	p.pool = a.MustComponent(pool.CName).(pool.Pool)
	p.peerService = a.MustComponent(peerservice.CName).(peerservice.PeerService)
	addrs := a.MustComponent("config").(configGetter).GetPublishServer().Addrs
	for _, addr := range addrs {
		p.peerService.SetPeerAddrs(addr.PeerId, addr.Addrs)
		p.peerIds = append(p.peerIds, addr.PeerId)
	}
	return
}

func (p *publishClient) Name() (name string) {
	return CName
}

func (p *publishClient) ResolveUri(ctx context.Context, uri string) (publish *publishapi.Publish, err error) {
	var resp *publishapi.ResolveUriResponse
	err = p.doClient(ctx, func(c publishapi.DRPCWebPublisherClient) (err error) {
		resp, err = c.ResolveUri(ctx, &publishapi.ResolveUriRequest{Uri: uri})
		return
	})
	if err != nil {
		return
	}
	return resp.Publish, nil
}

func (p *publishClient) GetPublishStatus(ctx context.Context, spaceId, objectId string) (publish *publishapi.Publish, err error) {
	var resp *publishapi.GetPublishStatusResponse
	err = p.doClient(ctx, func(c publishapi.DRPCWebPublisherClient) (err error) {
		resp, err = c.GetPublishStatus(ctx, &publishapi.GetPublishStatusRequest{SpaceId: spaceId, ObjectId: objectId})
		return
	})
	if err != nil {
		return
	}
	return resp.Publish, nil
}

func (p *publishClient) Publish(ctx context.Context, req *publishapi.PublishRequest) (uploadUrl string, err error) {
	var resp *publishapi.PublishResponse
	err = p.doClient(ctx, func(c publishapi.DRPCWebPublisherClient) (err error) {
		resp, err = c.Publish(ctx, req)
		return
	})
	if err != nil {
		return
	}
	return resp.UploadUrl, nil
}

func (p *publishClient) UnPublish(ctx context.Context, req *publishapi.UnPublishRequest) (err error) {
	return p.doClient(ctx, func(c publishapi.DRPCWebPublisherClient) (err error) {
		_, err = c.UnPublish(ctx, req)
		return
	})
}

func (p *publishClient) ListPublishes(ctx context.Context, spaceId string) (publishes []*publishapi.Publish, err error) {
	var resp *publishapi.ListPublishesResponse
	err = p.doClient(ctx, func(c publishapi.DRPCWebPublisherClient) (err error) {
		resp, err = c.ListPublishes(ctx, &publishapi.ListPublishesRequest{SpaceId: spaceId})
		return
	})
	if err != nil {
		return
	}
	return resp.Publishes, nil
}

func (p *publishClient) UploadDir(ctx context.Context, uploadUrl, dir string) (err error) {
	// Create a pipe for streaming the tar archive
	pr, pw := io.Pipe()

	// Start a goroutine for packing files into the tar archive
	go func() {
		tw := tar.NewWriter(pw)
		defer func() {
			_ = tw.Close()
			_ = pw.Close()
		}()

		// Walk through the directory and add files to the tar archive
		err = filepath.Walk(dir, func(file string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			// Check if the context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Create a header for the tar archive
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			// Set the correct relative file name
			relPath, err := filepath.Rel(dir, file)
			if err != nil {
				return err
			}
			header.Name = relPath

			// Write the header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// If it's a file, write its contents to the tar archive
			if !info.IsDir() {
				f, err := os.Open(file)
				if err != nil {
					return err
				}

				if _, err := io.Copy(tw, f); err != nil {
					_ = f.Close()
					return err
				}
				_ = f.Close()
			}

			return nil
		})

		// Close the writer with the resulting error
		_ = pw.CloseWithError(err)
	}()

	// Send the tar archive to the server as a POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadUrl, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload tar archive: %s", string(body))
	}

	return nil
}

func (p *publishClient) doClient(ctx context.Context, do func(c publishapi.DRPCWebPublisherClient) error) error {
	ctx = secureservice.CtxAllowAccountCheck(ctx)
	peer, err := p.pool.GetOneOf(ctx, p.peerIds)
	if err != nil {
		return err
	}
	return peer.DoDrpc(ctx, func(conn drpc.Conn) error {
		return do(publishapi.NewDRPCWebPublisherClient(conn))
	})
}
