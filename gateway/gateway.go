package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/app/ocache"
	"github.com/anyproto/any-sync/nameservice/nameserviceclient"
	"github.com/anyproto/any-sync/nameservice/nameserviceproto"
	"github.com/anyproto/anytype-publish-renderer/renderer"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-publish-server/gateway/gatewayconfig"
	"github.com/anyproto/anytype-publish-server/publish"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

func New() Gateway {
	return new(gateway)
}

const CName = "publish.gateway"

var log = logger.NewNamed(CName)

type Gateway interface {
	app.ComponentRunnable
}

type gateway struct {
	mux       *http.ServeMux
	server    *http.Server
	publish   publish.Service
	config    gatewayconfig.Config
	nnClient  nameserviceclient.AnyNsClientServiceBase
	nameCache ocache.OCache
}

func (g *gateway) Name() (name string) {
	return CName
}

func (g *gateway) Init(a *app.App) (err error) {
	g.publish = a.MustComponent(publish.CName).(publish.Service)
	g.config = a.MustComponent("config").(gatewayconfig.ConfigGetter).GetGateway()
	g.mux = http.NewServeMux()

	if g.config.ServeStatic {
		g.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	}
	g.mux.HandleFunc(`/name/{name}/{uri...}`, g.renderPageWithNameHandler)
	g.mux.HandleFunc("/{name}/{uri...}", g.renderPageHandler)
	g.server = &http.Server{Addr: g.config.Addr, Handler: g.mux}
	g.nnClient = a.MustComponent(nameserviceclient.CName).(nameserviceclient.AnyNsClientServiceBase)
	g.nameCache = ocache.New(g.resolveName, ocache.WithLogger(log.Sugar()), ocache.WithGCPeriod(time.Hour), ocache.WithTTL(time.Hour))
	return
}

func (g *gateway) Run(ctx context.Context) (err error) {
	var errCh = make(chan error)
	go func() {
		errCh <- g.server.ListenAndServe()
	}()
	select {
	case err = <-errCh:
		return err
	case <-time.After(200 * time.Millisecond):
		log.Info("gateway server started", zap.String("addr", g.config.Addr))
		return
	}
}

func (g *gateway) renderPageWithNameHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	identity, err := g.getIdentity(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r.SetPathValue("name", identity)
	g.renderPageHandler(w, r)
}

func (g *gateway) renderPageHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	uri := r.PathValue("uri")
	ctx := r.Context()
	pub, err := g.publish.ResolveUriWithName(ctx, name, uri)
	if err != nil {
		if errors.Is(err, publishapi.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if pub.ActivePublishId == nil {
		http.NotFound(w, r)
		return
	}

	publicFilesPath, err := url.JoinPath(g.config.PublishFilesURL, pub.ActivePublishId.Hex())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config := renderer.RenderConfig{
		StaticFilesPath:  g.config.StaticFilesURL,
		PublishFilesPath: publicFilesPath,
		PrismJsCdnUrl:    "https://cdn.jsdelivr.net/npm/prismjs@1.29.0",
		AnytypeCdnUrl:    "https://anytype-static.fra1.cdn.digitaloceanspaces.com",
		AnalyticsCode:    `<script>console.log("sending dummy analytics...")</script>`,
	}

	rend, err := renderer.NewRenderer(config)
	if err != nil {
		fmt.Printf("Error creating renderer: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err = rend.Render(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (g *gateway) getIdentity(ctx context.Context, name string) (identity string, err error) {
	obj, err := g.nameCache.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return obj.(*nameObject).nsResp.OwnerAnyAddress, nil
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

func (g *gateway) resolveName(ctx context.Context, name string) (object ocache.Object, err error) {
	resp, err := g.nnClient.IsNameAvailable(ctx, &nameserviceproto.NameAvailableRequest{
		FullName: name + ".any",
	})
	if err != nil {
		return nil, err
	}
	return &nameObject{nsResp: resp}, nil
}

func (g *gateway) Close(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return g.server.Shutdown(ctx)
}
