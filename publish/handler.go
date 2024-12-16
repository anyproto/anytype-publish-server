package publish

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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

type httpHandler struct {
	s *publishService
}

func (h httpHandler) init(m *http.ServeMux) {
	m.HandleFunc("/api/upload/:publishId/:uploadKey", h.Upload)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, errors.New("not found"))
	})
}

func (h httpHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}

	defer func() {
		_ = r.Body.Close()
	}()
	if url, err := h.s.UploadTar(r.Context(), r.PathValue("publishId"), r.PathValue("uploadKey"), r.Body); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var resp = struct {
			UploadUrl string `json:"uploadUrl"`
		}{
			UploadUrl: url,
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
	}
}

func writeErr(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	type errResp struct {
		Error string `json:"error"`
	}
	errData := errResp{Error: err.Error()}
	errDataBytes, _ := json.Marshal(errData)
	_, _ = w.Write(errDataBytes)
}
