package gateway

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
	"unsafe"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/anytype-publish-renderer/renderer"
	"github.com/golang/snappy"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-publish-server/gateway/gatewayconfig"
	"github.com/anyproto/anytype-publish-server/nameservice"
	"github.com/anyproto/anytype-publish-server/publish"
	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
	"github.com/anyproto/anytype-publish-server/redisprovider"
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
	mux           *http.ServeMux
	server        *http.Server
	publish       publish.Service
	config        gatewayconfig.Config
	nameService   nameservice.NameService
	renderVersion string
	redisClient   redis.UniversalClient
}

func (g *gateway) Name() (name string) {
	return CName
}

func (g *gateway) Init(a *app.App) (err error) {
	g.publish = a.MustComponent(publish.CName).(publish.Service)
	g.nameService = a.MustComponent(nameservice.CName).(nameservice.NameService)
	g.config = a.MustComponent("config").(gatewayconfig.ConfigGetter).GetGateway()
	g.mux = http.NewServeMux()

	g.redisClient = a.MustComponent(redisprovider.CName).(redisprovider.RedisProvider).Redis()

	g.renderVersion = renderVersion()
	if g.renderVersion == "" {
		return fmt.Errorf("render version not set")
	} else {
		log.Info("render version", zap.String("version", g.renderVersion))
	}

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
	g.handlePage(r.Context(), w, identity, r.PathValue("uri"), true)
}

func (g *gateway) renderPageHandler(w http.ResponseWriter, r *http.Request) {
	g.handlePage(r.Context(), w, r.PathValue("identity"), r.PathValue("uri"), false)
}

func (g *gateway) handlePage(ctx context.Context, w http.ResponseWriter, identity, uri string, withName bool) {
	id := newCacheId(identity, uri, withName)

	pageObj, cacheErr := g.cacheGet(ctx, id)
	if cacheErr != nil {
		if errors.Is(cacheErr, redis.Nil) {
			log.Debug("cache miss")
		} else {
			log.Warn("cache get error", zap.Error(cacheErr))
		}
	}

	var (
		isCacheMissed = pageObj == nil
		err           error
	)

	// we ignore cache if render version mismatch
	if !isCacheMissed && !pageObj.IsNotFound && pageObj.RenderVer != g.renderVersion {
		isCacheMissed = true
	}

	if isCacheMissed {
		if pageObj, err = g.renderPage(ctx, id); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	if pageObj.IsNotFound {
		http.NotFound(w, nil)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, strings.NewReader(pageObj.Body))
		if err != nil {
			log.Error("page write error", zap.Error(err))
		}
	}

	if isCacheMissed {
		if err = g.cacheSet(ctx, id, pageObj); err != nil {
			log.Error("cache set error", zap.Error(err))
		}
	}
}

func (g *gateway) cacheGet(ctx context.Context, key cacheId) (res *pageObject, err error) {
	var results = make([]*redis.StringCmd, 3)
	_, err = g.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		results[0] = pipe.GetEx(ctx, string(key)+":rver", time.Hour)
		results[1] = pipe.GetEx(ctx, string(key)+":notfound", time.Hour)
		results[2] = pipe.GetEx(ctx, string(key)+":body", time.Hour)
		return nil
	})

	if err != nil {
		return
	}

	for _, r := range results {
		if r.Err() != nil {
			err = r.Err()
			return
		}
	}

	var n int
	var decodedBody string
	dataBody := results[2].Val()
	bodyBytes := unsafe.Slice(unsafe.StringData(dataBody), len(dataBody))

	if n, err = snappy.DecodedLen(bodyBytes); err == nil {
		bodyBuf := make([]byte, n)
		var decoded []byte
		decoded, err = snappy.Decode(bodyBuf, bodyBytes)
		if err == nil {
			decodedBody = unsafe.String(unsafe.SliceData(decoded), len(decoded))
		} else {
			return
		}
	} else {
		return
	}

	obj := &pageObject{
		Body:       decodedBody,
		IsNotFound: results[1].Val() == "1",
		RenderVer:  results[0].Val(),
	}

	return obj, nil
}

func (g *gateway) cacheSet(ctx context.Context, key cacheId, data *pageObject) (err error) {
	results, err := g.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		isNotFound := "0"
		if data.IsNotFound {
			isNotFound = "1"
		}
		pipe.SetEx(ctx, string(key)+":rver", data.RenderVer, time.Hour)
		pipe.SetEx(ctx, string(key)+":notfound", isNotFound, time.Hour)

		bodyBytes := unsafe.Slice(unsafe.StringData(data.Body), len(data.Body))
		sBody := snappy.Encode(nil, bodyBytes)
		log.Debug("body size", zap.Int("before", len(data.Body)), zap.Int("after", len(sBody)))
		pipe.SetEx(ctx, string(key)+":body", sBody, time.Hour)
		return nil
	})

	if err != nil {
		return err
	}
	for _, r := range results {
		if r.Err() != nil {
			err = r.Err()
			return
		}
	}
	return nil
}

func (g *gateway) getIdentity(ctx context.Context, name string) (identity string, err error) {
	obj, err := g.nameService.ResolveName(ctx, name)
	if err != nil {
		return "", err
	}
	return obj.OwnerAnyAddress, nil
}

func (g *gateway) renderPage(ctx context.Context, id cacheId) (*pageObject, error) {
	cId := id
	identity := cId.Identity()
	uri := cId.Uri()
	pub, err := g.publish.ResolveUriWithIdentity(ctx, identity, uri)
	if err != nil {
		if errors.Is(err, publishapi.ErrNotFound) {
			return &pageObject{IsNotFound: true}, nil
		} else {
			return nil, err
		}
	}
	if pub.ActivePublishId == nil {
		return &pageObject{IsNotFound: true}, nil
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
	return &pageObject{
		Body:      buf.String(),
		RenderVer: g.renderVersion,
	}, nil
}

func (g *gateway) invalidateCache(identity, uri string) {
	withName := string(newCacheId(identity, uri, true))
	withoutName := string(newCacheId(identity, uri, false))
	err := g.redisClient.Del(
		context.Background(),
		withName+":rver",
		withName+":notfound",
		withName+":body",
		withoutName+":rver",
		withoutName+":notfound",
		withoutName+":body",
	).Err()
	if err != nil {
		log.Error("cache invalidate error", zap.Error(err))
	}
}

func (g *gateway) Close(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
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

func (c cacheId) RenderVersion() string {
	return c.getElement(4)
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

type pageObject struct {
	Body       string
	RenderVer  string
	IsNotFound bool
}

func renderVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, dep := range info.Deps {
		if dep.Path == "github.com/anyproto/anytype-publish-renderer" {
			return dep.Version
		}
	}
	return ""
}
