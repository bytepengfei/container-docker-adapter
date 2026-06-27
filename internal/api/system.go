package api

import (
	"net/http"

	"github.com/pengfei/container-docker-adapter/internal/backend"
)

type SystemController struct {
	backend backend.Backend
}

func NewSystemController(backend backend.Backend) *SystemController {
	return &SystemController{backend: backend}
}

func (c *SystemController) Ping(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.Ping(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("API-Version", "1.47")
	w.Header().Set("Docker-Experimental", "false")
	w.Header().Set("OSType", "linux")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write([]byte("OK"))
}

func (c *SystemController) Version(w http.ResponseWriter, r *http.Request) {
	version, err := c.backend.Version(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, version)
}

func (c *SystemController) Info(w http.ResponseWriter, r *http.Request) {
	info, err := c.backend.Info(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
