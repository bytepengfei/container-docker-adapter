package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type ContainerController struct {
	backend backend.Backend
}

func NewContainerController(backend backend.Backend) *ContainerController {
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
	opts := stopOptions(r)
	if err := c.backend.StopContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/stop"), opts); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Restart(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.RestartContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/restart"), stopOptions(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Kill(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.KillContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/kill"), r.URL.Query().Get("signal")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Pause(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.PauseContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/pause")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Unpause(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.UnpauseContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/unpause")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ContainerController) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.backend.ContainerStats(r.Context(), pathID(r.URL.Path, "/containers/", "/stats"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerContainerStats(stats))
}

func (c *ContainerController) Top(w http.ResponseWriter, r *http.Request) {
	top, err := c.backend.ContainerTop(r.Context(), pathID(r.URL.Path, "/containers/", "/top"), r.URL.Query().Get("ps_args"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, top)
}

func (c *ContainerController) Changes(w http.ResponseWriter, r *http.Request) {
	changes, err := c.backend.ContainerChanges(r.Context(), pathID(r.URL.Path, "/containers/", "/changes"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, changes)
}

func (c *ContainerController) GetArchive(w http.ResponseWriter, r *http.Request) {
	archive, err := c.backend.GetContainerArchive(r.Context(), pathID(r.URL.Path, "/containers/", "/archive"), model.ArchiveOptions{Path: r.URL.Query().Get("path")})
	if err != nil {
		writeError(w, err)
		return
	}
	defer archive.Close()
	w.Header().Set("Content-Type", "application/x-tar")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, archive)
}

func (c *ContainerController) PutArchive(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.PutContainerArchive(r.Context(), pathID(r.URL.Path, "/containers/", "/archive"), model.ArchiveOptions{Path: r.URL.Query().Get("path")}, r.Body); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *ContainerController) Attach(w http.ResponseWriter, r *http.Request) {
	id := pathID(r.URL.Path, "/containers/", "/attach")
	container, err := c.backend.InspectContainer(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	stream, err := c.backend.AttachContainer(r.Context(), id, logOptions(r))
	if err != nil {
		writeError(w, err)
		return
	}
	defer stream.Close()
	if err := writeDockerStream(w, r, stream, container.Tty, true); err != nil {
		return
	}
}

func (c *ContainerController) Resize(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.ResizeContainer(r.Context(), pathID(r.URL.Path, "/containers/", "/resize"), model.ResizeOptions{
		Height: parseInt(r.URL.Query().Get("h")),
		Width:  parseInt(r.URL.Query().Get("w")),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *ContainerController) Prune(w http.ResponseWriter, r *http.Request) {
	result, err := c.backend.PruneContainers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerPruneResult("container", result))
}

func (c *ContainerController) Logs(w http.ResponseWriter, r *http.Request) {
	id := pathID(r.URL.Path, "/containers/", "/logs")
	container, err := c.backend.InspectContainer(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	logs, err := c.backend.ContainerLogs(r.Context(), id, logOptions(r))
	if err != nil {
		writeError(w, err)
		return
	}
	defer logs.Close()

	if err := writeDockerStream(w, r, logs, container.Tty, false); err != nil {
		return
	}
}

func stopOptions(r *http.Request) model.StopOptions {
	var timeout *int
	if raw := r.URL.Query().Get("t"); raw != "" {
		value := parseInt(raw)
		timeout = &value
	}
	return model.StopOptions{TimeoutSeconds: timeout}
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

func logOptions(r *http.Request) model.LogOptions {
	return model.LogOptions{
		Follow:     parseBool(r.URL.Query().Get("follow")),
		Stdout:     parseBool(r.URL.Query().Get("stdout")),
		Stderr:     parseBool(r.URL.Query().Get("stderr")),
		Since:      r.URL.Query().Get("since"),
		Until:      r.URL.Query().Get("until"),
		Timestamps: parseBool(r.URL.Query().Get("timestamps")),
		Tail:       r.URL.Query().Get("tail"),
	}
}

func parseBool(value string) bool {
	result, _ := strconv.ParseBool(value)
	return result
}

func parseInt(value string) int {
	result, _ := strconv.Atoi(value)
	return result
}
