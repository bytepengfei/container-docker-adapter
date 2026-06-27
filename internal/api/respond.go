package api

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	apperrors "github.com/pengfei/container-docker-adapter/internal/errors"
)

type errorMessage struct {
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var apiErr *apperrors.APIError
	if stderrors.As(err, &apiErr) {
		writeJSON(w, apiErr.StatusCode, errorMessage{Message: apiErr.Message})
		return
	}

	switch {
	case stderrors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorMessage{Message: "No such container"})
	case stderrors.Is(err, apperrors.ErrConflict):
		writeJSON(w, http.StatusConflict, errorMessage{Message: err.Error()})
	case stderrors.Is(err, apperrors.ErrBadRequest):
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: err.Error()})
	case stderrors.Is(err, apperrors.ErrNotImplemented):
		writeJSON(w, http.StatusNotImplemented, errorMessage{Message: "operation is not implemented by this backend"})
	default:
		writeJSON(w, http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}
}

func writeNotImplemented(w http.ResponseWriter, endpoint string) {
	writeJSON(w, http.StatusNotImplemented, errorMessage{
		Message: "Docker API endpoint is not implemented: " + endpoint,
	})
}
