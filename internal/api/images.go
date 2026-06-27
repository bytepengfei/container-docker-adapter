package api

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type ImageController struct {
	backend interface {
		ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error)
		PullImage(context.Context, string, model.RegistryAuth, io.Writer) error
		RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error)
	}
}

func NewImageController(backend interface {
	ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error)
	PullImage(context.Context, string, model.RegistryAuth, io.Writer) error
	RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error)
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
		return
	}
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
