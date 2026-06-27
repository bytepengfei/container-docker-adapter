package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type ImageController struct {
	backend interface {
		ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error)
		InspectImage(context.Context, string) (model.Image, error)
		ImageHistory(context.Context, string) ([]model.ImageHistory, error)
		PullImage(context.Context, string, model.RegistryAuth, io.Writer) error
		PushImage(context.Context, string, model.RegistryAuth, io.Writer) error
		LoadImages(context.Context, io.Reader, io.Writer) error
		GetImage(context.Context, string) (io.ReadCloser, error)
		RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error)
		PruneImages(context.Context) (model.PruneResult, error)
	}
}

func NewImageController(backend interface {
	ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error)
	InspectImage(context.Context, string) (model.Image, error)
	ImageHistory(context.Context, string) ([]model.ImageHistory, error)
	PullImage(context.Context, string, model.RegistryAuth, io.Writer) error
	PushImage(context.Context, string, model.RegistryAuth, io.Writer) error
	LoadImages(context.Context, io.Reader, io.Writer) error
	GetImage(context.Context, string) (io.ReadCloser, error)
	RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error)
	PruneImages(context.Context) (model.PruneResult, error)
}) *ImageController {
	return &ImageController{backend: backend}
}

func (c *ImageController) List(w http.ResponseWriter, r *http.Request) {
	images, err := c.backend.ListImages(r.Context(), model.ListImagesOptions{
		All:     parseBool(r.URL.Query().Get("all")),
		Filter:  r.URL.Query().Get("filter"),
		Filters: r.URL.Query().Get("filters"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerImages(images))
}

func (c *ImageController) Create(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("fromImage")
	if tag := r.URL.Query().Get("tag"); ref != "" && tag != "" && !strings.Contains(ref, ":") {
		ref += ":" + tag
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := c.backend.PullImage(r.Context(), ref, model.RegistryAuth{Raw: r.Header.Get("X-Registry-Auth")}, w); err != nil {
		writeStreamError(w, err)
		return
	}
}

func (c *ImageController) Inspect(w http.ResponseWriter, r *http.Request) {
	image, err := c.backend.InspectImage(r.Context(), pathID(r.URL.Path, "/images/", "/json"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerImageInspect(image))
}

func (c *ImageController) History(w http.ResponseWriter, r *http.Request) {
	history, err := c.backend.ImageHistory(r.Context(), pathID(r.URL.Path, "/images/", "/history"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (c *ImageController) Push(w http.ResponseWriter, r *http.Request) {
	ref := pathID(r.URL.Path, "/images/", "/push")
	if tag := r.URL.Query().Get("tag"); tag != "" && !strings.Contains(ref, ":") {
		ref += ":" + tag
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := c.backend.PushImage(r.Context(), ref, model.RegistryAuth{Raw: r.Header.Get("X-Registry-Auth")}, w); err != nil {
		writeStreamError(w, err)
	}
}

func (c *ImageController) Load(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := c.backend.LoadImages(r.Context(), r.Body, w); err != nil {
		writeStreamError(w, err)
	}
}

func writeStreamError(w io.Writer, err error) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errorDetail": map[string]string{"message": err.Error()},
		"error":       err.Error(),
	})
}

func (c *ImageController) Get(w http.ResponseWriter, r *http.Request) {
	archive, err := c.backend.GetImage(r.Context(), pathID(r.URL.Path, "/images/", "/get"))
	if err != nil {
		writeError(w, err)
		return
	}
	defer archive.Close()
	w.Header().Set("Content-Type", "application/x-tar")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, archive)
}

func (c *ImageController) Remove(w http.ResponseWriter, r *http.Request) {
	deleted, err := c.backend.RemoveImage(r.Context(), pathID(r.URL.Path, "/images/", ""), model.RemoveImageOptions{
		Force:   parseBool(r.URL.Query().Get("force")),
		NoPrune: parseBool(r.URL.Query().Get("noprune")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deleted)
}

func (c *ImageController) Prune(w http.ResponseWriter, r *http.Request) {
	result, err := c.backend.PruneImages(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerPruneResult("image", result))
}
