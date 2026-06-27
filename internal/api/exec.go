package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type ExecController struct {
	backend backend.Backend
}

func NewExecController(backend backend.Backend) *ExecController {
	return &ExecController{backend: backend}
}

type dockerExecCreateRequest struct {
	AttachStdin  bool     `json:"AttachStdin"`
	AttachStdout bool     `json:"AttachStdout"`
	AttachStderr bool     `json:"AttachStderr"`
	Tty          bool     `json:"Tty"`
	Cmd          []string `json:"Cmd"`
	Env          []string `json:"Env"`
	WorkingDir   string   `json:"WorkingDir"`
	User         string   `json:"User"`
}

func (r dockerExecCreateRequest) Model(containerID string) model.ExecConfig {
	return model.ExecConfig{
		ContainerID:  containerID,
		AttachStdin:  r.AttachStdin,
		AttachStdout: r.AttachStdout,
		AttachStderr: r.AttachStderr,
		Tty:          r.Tty,
		Cmd:          r.Cmd,
		Env:          r.Env,
		WorkingDir:   r.WorkingDir,
		User:         r.User,
	}
}

func (c *ExecController) Create(w http.ResponseWriter, r *http.Request) {
	var req dockerExecCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
		return
	}

	session, err := c.backend.CreateExec(r.Context(), pathID(r.URL.Path, "/containers/", "/exec"), req.Model(""))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"Id": session.ID})
}

func (c *ExecController) Start(w http.ResponseWriter, r *http.Request) {
	var req dockerExecCreateRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	stream, err := c.backend.StartExec(r.Context(), pathID(r.URL.Path, "/exec/", "/start"), req.Model(""))
	if err != nil {
		writeError(w, err)
		return
	}
	defer stream.Close()
	w.Header().Set("Content-Type", "application/vnd.docker.raw-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, stream)
}

func (c *ExecController) Inspect(w http.ResponseWriter, r *http.Request) {
	session, err := c.backend.InspectExec(r.Context(), pathID(r.URL.Path, "/exec/", "/json"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerExecInspect(session))
}

func (c *ExecController) Resize(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.ResizeExec(r.Context(), pathID(r.URL.Path, "/exec/", "/resize"), model.ResizeOptions{
		Height: parseInt(r.URL.Query().Get("h")),
		Width:  parseInt(r.URL.Query().Get("w")),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
