package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pengfei/container-docker-adapter/internal/backend/memory"
)

func TestSystemEndpoints(t *testing.T) {
	handler := NewRouter(memory.New())

	ping := httptest.NewRecorder()
	handler.ServeHTTP(ping, httptest.NewRequest(http.MethodGet, "/_ping", nil))
	if ping.Code != http.StatusOK {
		t.Fatalf("ping status = %d, want %d", ping.Code, http.StatusOK)
	}
	if ping.Body.String() != "OK" {
		t.Fatalf("ping body = %q, want OK", ping.Body.String())
	}

	version := httptest.NewRecorder()
	handler.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/v1.47/version", nil))
	if version.Code != http.StatusOK {
		t.Fatalf("version status = %d, want %d", version.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(version.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ApiVersion"] != "1.47" {
		t.Fatalf("ApiVersion = %v, want 1.47", body["ApiVersion"])
	}
}

func TestContainerLifecycleRoutes(t *testing.T) {
	handler := NewRouter(memory.New())

	createBody := bytes.NewBufferString(`{"Image":"hello-world:latest","Cmd":["echo","hello"]}`)
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/v1.47/containers/create?name=demo", createBody))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}

	var created struct {
		ID       string   `json:"Id"`
		Warnings []string `json:"Warnings"`
	}
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("created ID is empty")
	}

	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/containers/"+created.ID+"/start", nil))
	if start.Code != http.StatusNoContent {
		t.Fatalf("start status = %d, want %d", start.Code, http.StatusNoContent)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/containers/json", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", list.Code, http.StatusOK)
	}
	var containers []map[string]any
	if err := json.NewDecoder(list.Body).Decode(&containers); err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("container count = %d, want 1", len(containers))
	}
	if containers[0]["State"] != "running" {
		t.Fatalf("container state = %v, want running", containers[0]["State"])
	}

	inspect := httptest.NewRecorder()
	handler.ServeHTTP(inspect, httptest.NewRequest(http.MethodGet, "/containers/demo/json", nil))
	if inspect.Code != http.StatusOK {
		t.Fatalf("inspect status = %d, want %d; body=%s", inspect.Code, http.StatusOK, inspect.Body.String())
	}

	removeConflict := httptest.NewRecorder()
	handler.ServeHTTP(removeConflict, httptest.NewRequest(http.MethodDelete, "/containers/demo", nil))
	if removeConflict.Code != http.StatusConflict {
		t.Fatalf("remove running status = %d, want %d", removeConflict.Code, http.StatusConflict)
	}

	remove := httptest.NewRecorder()
	handler.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/containers/demo?force=true", nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("force remove status = %d, want %d", remove.Code, http.StatusNoContent)
	}
}

func TestImagesRoutes(t *testing.T) {
	handler := NewRouter(memory.New())

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/images/json", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", list.Code, http.StatusOK)
	}

	pull := httptest.NewRecorder()
	handler.ServeHTTP(pull, httptest.NewRequest(http.MethodPost, "/images/create?fromImage=alpine&tag=latest", nil))
	if pull.Code != http.StatusOK {
		t.Fatalf("pull status = %d, want %d; body=%s", pull.Code, http.StatusOK, pull.Body.String())
	}
	if pull.Body.Len() == 0 {
		t.Fatal("pull response is empty")
	}
}
