package api

import (
	"encoding/json"
	"net/http"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type NetworkController struct {
	backend backend.Backend
}

func NewNetworkController(backend backend.Backend) *NetworkController {
	return &NetworkController{backend: backend}
}

func (c *NetworkController) List(w http.ResponseWriter, r *http.Request) {
	networks, err := c.backend.ListNetworks(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerNetworks(networks))
}

func (c *NetworkController) Create(w http.ResponseWriter, r *http.Request) {
	var spec model.NetworkSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
		return
	}
	network, err := c.backend.CreateNetwork(r.Context(), spec)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"Id": network.ID, "Warning": ""})
}

func (c *NetworkController) Inspect(w http.ResponseWriter, r *http.Request) {
	network, err := c.backend.InspectNetwork(r.Context(), pathID(r.URL.Path, "/networks/", ""))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerNetwork(network))
}

func (c *NetworkController) Connect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Container string `json:"Container"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := c.backend.ConnectNetwork(r.Context(), pathID(r.URL.Path, "/networks/", "/connect"), req.Container); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *NetworkController) Disconnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Container string `json:"Container"`
		Force     bool   `json:"Force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := c.backend.DisconnectNetwork(r.Context(), pathID(r.URL.Path, "/networks/", "/disconnect"), req.Container, req.Force); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *NetworkController) Remove(w http.ResponseWriter, r *http.Request) {
	if err := c.backend.RemoveNetwork(r.Context(), pathID(r.URL.Path, "/networks/", "")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *NetworkController) Prune(w http.ResponseWriter, r *http.Request) {
	result, err := c.backend.PruneNetworks(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DockerPruneResult("network", result))
}
