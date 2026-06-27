package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/pengfei/container-docker-adapter/internal/model"
)

type ContainerController struct {
	backend interface {
		ListContainers(context.Context, model.ListContainersOptions) ([]model.Container, error)
		InspectContainer(context.Context, string) (model.Container, error)
		CreateContainer(context.Context, model.ContainerSpec) (model.ContainerCreateResult, error)
		StartContainer(context.Context, string) error
		StopContainer(context.Context, string, model.StopOptions) error
		RemoveContainer(context.Context, string, model.RemoveOptions) error
		ContainerLogs(context.Context, string, model.LogOptions) (io.ReadCloser, error)
	}
}

func NewContainerController(backend interface {
	ListContainers(context.Context, model.ListContainersOptions) ([]model.Container, error)
	InspectContainer(context.Context, string) (model.Container, error)
	CreateContainer(context.Context, model.ContainerSpec) (model.ContainerCreateResult, error)
	StartContainer(context.Context, string) error
	StopContainer(context.Context, string, model.StopOptions) error
	RemoveContainer(context.Context, string, model.RemoveOptions) error
	ContainerLogs(context.Context, string, model.LogOptions) (io.ReadCloser, error)
}) *ContainerController {
	return &ContainerController{backend: backend}
}

func (c *ContainerController) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	containers, err := c.backend.ListContainers(r.Context(), model.ListContainersOptions{
		All:    parseBool(query.Get("all")),
		Limit:  parseInt(query.Get("limit")),
		Size:   parseBool(query.Get("size")),
		Filter: query.Get("filters"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerContainers(containers))
}

func (c *ContainerController) Inspect(w http.ResponseWriter, r *http.Request) {
	container, err := c.backend.InspectContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/json"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerContainerInspect(container))
}

func (c *ContainerController) Create(w http.ResponseWriter, r *http.Request) {
	var req dockerContainerCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
		return
	}

	result, err := c.backend.CreateContainer(r.Context(), req.Model(r.URL.Query().Get("name")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (c *ContainerController) Start(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.StartContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/start")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Stop(w http.ResponseWriter, r *http.Request) {
	var timeout *int
	if raw := r.URL.Query().Get("t"); raw != "" {
		value := parseInt(raw)
		timeout = &value
	}

	if err := c.backend.StopContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/stop"), model.StopOptions{TimeoutSeconds: timeout}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Remove(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.RemoveContainer(r.Context(), pathID(r.URL.Path, "/containers/", ""), model.RemoveOptions{
		Force:         parseBool(r.URL.Query().Get("force")),
		RemoveVolumes: parseBool(r.URL.Query().Get("v")),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Logs(w http.ResponseWriter, r *http.Request) {
	logs, err := c.backend.ContainerLogs(r.Context(), pathID(r.URL.Path, "/containers/", "/logs"), model.LogOptions{
		Follow:     parseBool(r.URL.Query().Get("follow")),
		Stdout:     parseBool(r.URL.Query().Get("stdout")),
		Stderr:     parseBool(r.URL.Query().Get("stderr")),
		Since:      r.URL.Query().Get("since"),
		Until:      r.URL.Query().Get("until"),
		Timestamps: parseBool(r.URL.Query().Get("timestamps")),
		Tail:       r.URL.Query().Get("tail"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	defer logs.Close()

	w.Header().Set("Content-Type", "application/vnd.docker.raw-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, logs)
}

func parseBool(value string) bool {
	result, _ := strconv.ParseBool(value)
	return result
}

func parseInt(value string) int {
	result, _ := strconv.Atoi(value)
	return result
}
