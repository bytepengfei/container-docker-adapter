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
	exec       *ExecController
	volumes    *VolumeController
	networks   *NetworkController
	auth       *AuthController
	build      *BuildController
}

func NewRouter(backend backend.Backend) http.Handler {
	return &router{
		system:     NewSystemController(backend),
		containers: NewContainerController(backend),
		images:     NewImageController(backend),
		exec:       NewExecController(backend),
		volumes:    NewVolumeController(backend),
		networks:   NewNetworkController(backend),
		auth:       NewAuthController(backend),
		build:      NewBuildController(backend),
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
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/wait"):
		r.containers.Wait(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/stop"):
		r.containers.Stop(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/restart"):
		r.containers.Restart(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/kill"):
		r.containers.Kill(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/pause"):
		r.containers.Pause(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/unpause"):
		r.containers.Unpause(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/stats"):
		r.containers.Stats(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/top"):
		r.containers.Top(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/changes"):
		r.containers.Changes(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/archive"):
		r.containers.GetArchive(w, req)
	case req.Method == http.MethodPut && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/archive"):
		r.containers.PutArchive(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/attach"):
		r.containers.Attach(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/resize"):
		r.containers.Resize(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/containers/prune":
		r.containers.Prune(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/containers/"):
		r.containers.Remove(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/logs"):
		r.containers.Logs(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/containers/") && strings.HasSuffix(req.URL.Path, "/exec"):
		r.exec.Create(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/exec/") && strings.HasSuffix(req.URL.Path, "/start"):
		r.exec.Start(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/exec/") && strings.HasSuffix(req.URL.Path, "/json"):
		r.exec.Inspect(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/exec/") && strings.HasSuffix(req.URL.Path, "/resize"):
		r.exec.Resize(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/events":
		r.system.Events(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/images/json":
		r.images.List(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/images/create":
		r.images.Create(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/images/") && strings.HasSuffix(req.URL.Path, "/json"):
		r.images.Inspect(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/images/") && strings.HasSuffix(req.URL.Path, "/history"):
		r.images.History(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/images/") && strings.HasSuffix(req.URL.Path, "/push"):
		r.images.Push(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/images/load":
		r.images.Load(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/images/") && strings.HasSuffix(req.URL.Path, "/get"):
		r.images.Get(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/images/prune":
		r.images.Prune(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/images/"):
		r.images.Remove(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/volumes":
		r.volumes.List(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/volumes/create":
		r.volumes.Create(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/volumes/"):
		r.volumes.Inspect(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/volumes/"):
		r.volumes.Remove(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/volumes/prune":
		r.volumes.Prune(w, req)
	case req.Method == http.MethodGet && req.URL.Path == "/networks":
		r.networks.List(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/networks/create":
		r.networks.Create(w, req)
	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/networks/"):
		r.networks.Inspect(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/networks/") && strings.HasSuffix(req.URL.Path, "/connect"):
		r.networks.Connect(w, req)
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/networks/") && strings.HasSuffix(req.URL.Path, "/disconnect"):
		r.networks.Disconnect(w, req)
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/networks/"):
		r.networks.Remove(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/networks/prune":
		r.networks.Prune(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/auth":
		r.auth.Auth(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/build":
		r.build.Build(w, req)
	case req.Method == http.MethodPost && req.URL.Path == "/build/prune":
		r.build.Prune(w, req)
	case isNotPlanned(req.URL.Path):
		writeNotImplemented(w, req.URL.Path)
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

func isNotPlanned(path string) bool {
	return strings.HasPrefix(path, "/swarm") ||
		strings.HasPrefix(path, "/services") ||
		strings.HasPrefix(path, "/nodes") ||
		strings.HasPrefix(path, "/tasks") ||
		strings.HasPrefix(path, "/plugins") ||
		strings.Contains(path, "/checkpoints")
}
