package api

import (
	"encoding/json"
	"net/http"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type BuildController struct {
	backend backend.Backend
}

func NewBuildController(backend backend.Backend) *BuildController {
	return &BuildController{backend: backend}
}

func (c *BuildController) Build(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	opts := model.BuildOptions{
		Tags:       query["t"],
		Dockerfile: query.Get("dockerfile"),
		NoCache:    parseBool(query.Get("nocache")),
		Pull:       parseBool(query.Get("pull")),
		Target:     query.Get("target"),
		Platform:   query.Get("platform"),
		BuildArgs:  parseStringMap(query.Get("buildargs")),
		Labels:     parseStringMap(query.Get("labels")),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = c.backend.BuildImage(r.Context(), r.Body, opts, w)
}

func (c *BuildController) Prune(w http.ResponseWriter, r *http.Request) {
	result, err := c.backend.PruneBuildCache(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"CachesDeleted":  nonNilSlice(result.Deleted),
		"SpaceReclaimed": result.SpaceReclaimed,
	})
}

func parseStringMap(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return map[string]string{}
	}
	return values
}
