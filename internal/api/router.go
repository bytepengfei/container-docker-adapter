package api

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
)

var versionPrefix = regexp.MustCompile(`^/v[0-9]+(?:\.[0-9]+)?(/.*)$`)

type router struct {
	system     *SystemController
	containers *ContainerController
	images     *ImageController
}

func NewRouter(backend backend.Backend) http.Handler {
	return &router{
		system:     NewSystemController(backend),
		containers: NewContainerController(backend),
		images:     NewImageController(backend),
	}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.URL.Path = normalizePath(req.URL.Path)

	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/_ping":
		r.system.Ping(w, req)
	case req.Method == http.MethodHead && req.URL.Path == "/_ping":
		r.system.Ping(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/version":
		r.system.Version(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/info":
		r.system.Info(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/containers/json":
		r.containers.List(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/containers/create":
		r.containers.Create(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/json"):
		r.containers.Inspect(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/start"):
		r.containers.Start(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/stop"):
		r.containers.Stop(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/containers/"):
		r.containers.Remove(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/logs"):
		r.containers.Logs(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/images/json":
		r.images.List(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/images/create":
		r.images.Create(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/images/"):
		r.images.Remove(w, req)
	default:
		writeNotImplemented(w, req.URL.Path)
	}
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if matches := versionPrefix.FindStringSubmatch(path); len(matches) == 2 {
		return matches[1]
	}
	return path
}

func pathID(path, prefix, suffix string) string {
	value := strings.TrimPrefix(path, prefix)
	value = strings.TrimSuffix(value, suffix)
	return strings.Trim(value, "/")
}
