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
	mux     *http.ServeMux
	server  *http.Server
	publish publish.Service
	config  gatewayconfig.Config
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
	g.mux.HandleFunc("/{name}/{uri...}", g.renderPageHandler)
	g.server = &http.Server{Addr: g.config.Addr, Handler: g.mux}
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
		StaticFilesPath:  "/static",
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

func (g *gateway) Close(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return g.server.Shutdown(ctx)
}
