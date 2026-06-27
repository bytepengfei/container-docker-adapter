package api

import (
	"encoding/json"
	"net/http"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type VolumeController struct {
	backend backend.Backend
}

func NewVolumeController(backend backend.Backend) *VolumeController {
	return &VolumeController{backend: backend}
}

func (c *VolumeController) List(w http.ResponseWriter, r *http.Request) {
	volumes, err := c.backend.ListVolumes(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"Volumes": DockerVolumes(volumes), "Warnings": []string{}})
}

func (c *VolumeController) Create(w http.ResponseWriter, r *http.Request) {
	var spec model.VolumeSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
		return
	}
	volume, err := c.backend.CreateVolume(r.Context(), spec)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, DockerVolume(volume))
}

func (c *VolumeController) Inspect(w http.ResponseWriter, r *http.Request) {
	volume, err := c.backend.InspectVolume(r.Context(), pathID(r.URL.Path, "/volumes/", ""))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerVolume(volume))
}

func (c *VolumeController) Remove(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.RemoveVolume(r.Context(), pathID(r.URL.Path, "/volumes/", ""), parseBool(r.URL.Query().Get("force"))); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *VolumeController) Prune(w http.ResponseWriter, r *http.Request) {
	result, err := c.backend.PruneVolumes(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerPruneResult("volume", result))
}
