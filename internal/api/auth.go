package api

import (
	"io"
	"net/http"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type AuthController struct {
	backend backend.Backend
}

func NewAuthController(backend backend.Backend) *AuthController {
	return &AuthController{backend: backend}
}

func (c *AuthController) Auth(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
		return
	}
	result, err := c.backend.Authenticate(r.Context(), model.RegistryAuth{Raw: string(payload)})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
