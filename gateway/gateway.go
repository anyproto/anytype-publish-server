package gateway

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/app/ocache"
	"github.com/anyproto/anytype-publish-renderer/renderer"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-publish-server/gateway/gatewayconfig"
	"github.com/anyproto/anytype-publish-server/nameservice"
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
	mux         *http.ServeMux
	server      *http.Server
	publish     publish.Service
	config      gatewayconfig.Config
	nameService nameservice.NameService
	cache       ocache.OCache
}

func (g *gateway) Name() (name string) {
	return CName
}

func (g *gateway) Init(a *app.App) (err error) {
	g.publish = a.MustComponent(publish.CName).(publish.Service)
	g.nameService = a.MustComponent(nameservice.CName).(nameservice.NameService)
	g.config = a.MustComponent("config").(gatewayconfig.ConfigGetter).GetGateway()
	g.mux = http.NewServeMux()

	g.cache = ocache.New(g.loadCachedPage, ocache.WithTTL(time.Hour))

	if g.config.ServeStatic {
		g.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	}
	g.mux.HandleFunc(`/name/{name}/{uri...}`, g.renderPageWithNameHandler)
	g.mux.HandleFunc("/{identity}/{uri...}", g.renderPageHandler)
	g.server = &http.Server{Addr: g.config.Addr, Handler: g.mux}
	return
}

func (g *gateway) Run(ctx context.Context) (err error) {
	g.publish.SetInvalidateCacheCallback(g.invalidateCache)
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
	g.renderPage(r.Context(), w, identity, r.PathValue("uri"), true)
}

func (g *gateway) renderPageHandler(w http.ResponseWriter, r *http.Request) {
	g.renderPage(r.Context(), w, r.PathValue("identity"), r.PathValue("uri"), false)
}

func (g *gateway) renderPage(ctx context.Context, w http.ResponseWriter, identity, uri string, withName bool) {
	id := newCacheId(identity, uri, withName)
	obj, err := g.cache.Get(ctx, string(id))
	if err != nil {
		log.Error("page load error", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	cObject := obj.(*cacheObject)
	if cObject.isNotFound {
		http.NotFound(w, nil)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, bytes.NewReader(cObject.body))
	if err != nil {
		log.Error("page write error", zap.Error(err))
	}
}

func (g *gateway) getIdentity(ctx context.Context, name string) (identity string, err error) {
	obj, err := g.nameService.ResolveName(ctx, name)
	if err != nil {
		return "", err
	}
	return obj.OwnerAnyAddress, nil
}

func (g *gateway) loadCachedPage(ctx context.Context, id string) (ocache.Object, error) {
	cId := cacheId(id)
	identity := cId.Identity()
	uri := cId.Uri()
	pub, err := g.publish.ResolveUriWithIdentity(ctx, identity, uri)
	if err != nil {
		if errors.Is(err, publishapi.ErrNotFound) {
			return &cacheObject{isNotFound: true}, nil
		} else {
			return nil, err
		}
	}
	if pub.ActivePublishId == nil {
		return &cacheObject{isNotFound: true}, nil
	}

	publicFilesPath, err := url.JoinPath(g.config.PublishFilesURL, pub.ActivePublishId.Hex())
	if err != nil {
		return nil, err
	}

	var analyticsCode string
	if cId.WithName() {
		analyticsCode = g.config.AnalyticsCodeMembers
	} else {
		analyticsCode = g.config.AnalyticsCode
	}

	config := renderer.RenderConfig{
		StaticFilesPath:  g.config.StaticFilesURL,
		PublishFilesPath: publicFilesPath,
		PrismJsCdnUrl:    "https://cdn.jsdelivr.net/npm/prismjs@1.29.0",
		AnytypeCdnUrl:    "https://anytype-static.fra1.cdn.digitaloceanspaces.com",
		AnalyticsCode:    analyticsCode,
	}

	rend, err := renderer.NewRenderer(config)
	if err != nil {
		return nil, err
	}

	var buf = bytes.NewBuffer(make([]byte, 0, 5*1024))
	if err = rend.Render(buf); err != nil {
		return nil, err
	}
	return &cacheObject{body: buf.Bytes()}, nil
}

func (g *gateway) invalidateCache(identity, uri string) {
	_, _ = g.cache.Remove(context.Background(), string(newCacheId(identity, uri, false)))
	_, _ = g.cache.Remove(context.Background(), string(newCacheId(identity, uri, true)))
}

func (g *gateway) Close(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = g.cache.Close()
	return g.server.Shutdown(ctx)
}

var cacheIdSep = string([]byte{0})

func newCacheId(identity, uri string, withName bool) cacheId {
	var res strings.Builder
	res.WriteString(identity)
	res.WriteString(cacheIdSep)
	res.WriteString(uri)
	res.WriteString(cacheIdSep)
	if withName {
		res.WriteString("1")
	} else {
		res.WriteString("0")
	}
	return cacheId(res.String())
}

type cacheId string

func (c cacheId) Identity() string {
	return c.getElement(1)
}

func (c cacheId) Uri() string {
	return c.getElement(2)
}

func (c cacheId) WithName() bool {
	return c.getElement(3) == "1"
}

func (c cacheId) String() string {
	return strings.ReplaceAll(string(c), cacheIdSep, "/")
}

func (c cacheId) getElement(idx int) string {
	var prevIdx int
	for i := range idx {
		curIdx := strings.Index(string(c[prevIdx:]), cacheIdSep)
		if curIdx == -1 {
			if i+1 == idx {
				return string(c[prevIdx:])
			}
			return ""
		}
		if i+1 == idx {
			return string(c[prevIdx : curIdx+prevIdx])
		}
		prevIdx = prevIdx + curIdx + 1
	}
	return ""
}

type cacheObject struct {
	body       []byte
	isNotFound bool
}

func (c *cacheObject) Close() (err error) {
	return
}

func (c *cacheObject) TryClose(objectTTL time.Duration) (res bool, err error) {
	return true, nil
}
