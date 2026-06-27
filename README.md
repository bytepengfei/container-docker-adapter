# Container Docker Adapter

`container-docker-adapter` is a Docker Engine API compatibility layer for Apple Container.

The goal is not to implement a full container engine. The adapter exposes a Docker-compatible HTTP API over a Unix socket, accepts requests from the Docker CLI and Docker-integrated tools, translates those requests into backend operations, and translates backend responses back into Docker Engine API shapes.

```text
docker CLI / IDE / tooling
          |
          | Docker Engine API
          v
  container-docker-adapter
          |
          | Backend interface
          v
  Apple Container API / container-apiserver
```

## Status

This repository currently contains the first working skeleton:

- Docker-compatible HTTP server over Unix socket
- Docker API version prefix normalization, for example `/v1.47/version`
- Basic system endpoints:
  - `GET /_ping`
  - `HEAD /_ping`
  - `GET /version`
  - `GET /info`
- Basic container endpoints:
  - `GET /containers/json`
  - `POST /containers/create`
  - `GET /containers/{id}/json`
  - `POST /containers/{id}/start`
  - `POST /containers/{id}/stop`
  - `DELETE /containers/{id}`
  - `GET /containers/{id}/logs`
- Basic image endpoints:
  - `GET /images/json`
  - `POST /images/create`
  - `DELETE /images/{id}`
- Internal Docker response translators
- Docker-style error responses for common failures
- In-memory backend for local API development and tests
- Apple backend package with client and translation entry points

The in-memory backend is only for development. It does not run real containers. Real Apple Container integration should be implemented under `internal/backend/apple`.

## Docker Engine API Compatibility Target

The target Docker Engine API contract is:

- Target API version: `v1.47`
- Minimum accepted API version: `v1.24`
- Transport: Unix domain socket
- Protocol: HTTP/1.1 with Docker Engine API-compatible paths, status codes, headers, JSON field names, and selected error messages

The adapter currently normalizes versioned Docker API paths, so these are routed to the same controller:

```text
/containers/json
/v1.47/containers/json
/v1.41/containers/json
```

The adapter advertises `ApiVersion: 1.47` and `MinAPIVersion: 1.24` from `GET /version`. The goal is practical Docker CLI and developer-tool compatibility, not full Docker Engine parity.

## Implementation Matrix

Status meanings:

- `Implemented`: The Docker API route exists and is covered by the current adapter shape. With the memory backend, behavior may be simulated.
- `Planned`: The endpoint is in scope but not implemented yet.
- `Backend-dependent`: The Docker API route can be implemented only after the Apple Container backend exposes the required capability.
- `Not planned`: The feature is outside the compatibility-layer goal or does not map cleanly to Apple Container.

### System

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Implemented | `GET /_ping` | Docker CLI daemon probe | Returns `OK`. |
| Implemented | `HEAD /_ping` | Docker CLI daemon probe | Returns Docker compatibility headers. |
| Implemented | `GET /version` | `docker version` | Advertises API `1.47`, minimum API `1.24`. |
| Implemented | `GET /info` | `docker info` | Returns adapter/backend info. |

### Containers

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Implemented | `GET /containers/json` | `docker ps`, `docker container ls` | Memory backend lists simulated containers. |
| Implemented | `POST /containers/create` | `docker create`, part of `docker run` | Memory backend creates simulated containers. |
| Implemented | `GET /containers/{id}/json` | `docker inspect` | Basic inspect response shape exists. |
| Implemented | `POST /containers/{id}/start` | `docker start`, part of `docker run` | Memory backend changes state to `running`. |
| Implemented | `POST /containers/{id}/stop` | `docker stop` | Memory backend changes state to `exited`. |
| Implemented | `DELETE /containers/{id}` | `docker rm` | Supports force/remove-volume flags at API shape level. |
| Implemented | `GET /containers/{id}/logs` | `docker logs` | Route exists; real logs are backend-dependent. |
| Planned | `POST /containers/{id}/restart` | `docker restart` | Can be built from stop/start if backend supports both. |
| Planned | `POST /containers/{id}/kill` | `docker kill` | Requires backend signal/terminate support. |
| Planned | `POST /containers/{id}/pause` | `docker pause` | Backend-dependent. |
| Planned | `POST /containers/{id}/unpause` | `docker unpause` | Backend-dependent. |
| Planned | `GET /containers/{id}/stats` | `docker stats` | Requires backend metrics stream. |
| Planned | `GET /containers/{id}/top` | `docker top` | Requires backend process listing. |
| Planned | `GET /containers/{id}/changes` | `docker diff` | Requires filesystem change tracking. |
| Planned | `GET /containers/{id}/archive` | `docker cp` from container | Requires backend filesystem archive support. |
| Planned | `PUT /containers/{id}/archive` | `docker cp` to container | Requires backend filesystem archive support. |
| Planned | `POST /containers/{id}/attach` | `docker attach` | Requires HTTP hijack/raw stream support. |
| Planned | `POST /containers/{id}/resize` | `docker resize` | Requires TTY resize support. |
| Planned | `POST /containers/prune` | `docker container prune` | Can be implemented after lifecycle behavior is real. |

### Exec and Streaming

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Planned | `POST /containers/{id}/exec` | `docker exec` | Requires backend exec capability. |
| Planned | `POST /exec/{id}/start` | `docker exec` | Requires raw stream/hijack handling. |
| Planned | `GET /exec/{id}/json` | `docker inspect` for exec | Requires exec session state. |
| Planned | `POST /exec/{id}/resize` | `docker exec -it` resize | Requires backend TTY resize support. |
| Planned | `GET /events` | `docker events` | Requires backend event stream and Docker event translation. |

### Images and Registry

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Implemented | `GET /images/json` | `docker images`, `docker image ls` | Memory backend lists simulated images. |
| Implemented | `POST /images/create` | `docker pull` | Memory backend simulates pull progress. |
| Implemented | `DELETE /images/{id}` | `docker rmi` | Memory backend removes simulated images. |
| Planned | `GET /images/{id}/json` | `docker image inspect` | Requires backend image metadata. |
| Planned | `GET /images/{id}/history` | `docker history` | Backend-dependent; may be unavailable. |
| Planned | `POST /images/{name}/push` | `docker push` | Depends on registry authentication and backend registry support. |
| Planned | `POST /images/load` | `docker load` | Requires image import support. |
| Planned | `GET /images/{name}/get` | `docker save` | Requires image export support. |
| Planned | `POST /images/prune` | `docker image prune` | Can be implemented after real image storage integration. |

### Volumes

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Planned | `GET /volumes` | `docker volume ls` | Requires mapping to Apple/backend storage model. |
| Planned | `POST /volumes/create` | `docker volume create` | Backend-dependent. |
| Planned | `GET /volumes/{name}` | `docker volume inspect` | Backend-dependent. |
| Planned | `DELETE /volumes/{name}` | `docker volume rm` | Backend-dependent. |
| Planned | `POST /volumes/prune` | `docker volume prune` | Backend-dependent. |

### Networks

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Planned | `GET /networks` | `docker network ls` | Requires mapping to Apple/backend networking model. |
| Planned | `POST /networks/create` | `docker network create` | Backend-dependent. |
| Planned | `GET /networks/{id}` | `docker network inspect` | Backend-dependent. |
| Planned | `POST /networks/{id}/connect` | `docker network connect` | Backend-dependent. |
| Planned | `POST /networks/{id}/disconnect` | `docker network disconnect` | Backend-dependent. |
| Planned | `DELETE /networks/{id}` | `docker network rm` | Backend-dependent. |
| Planned | `POST /networks/prune` | `docker network prune` | Backend-dependent. |

### Build and Compose

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Backend-dependent | `POST /build` | `docker build` | Prefer integrating BuildKit or an Apple-supported build path instead of implementing a builder here. |
| Backend-dependent | `POST /build/prune` | `docker builder prune` | Depends on build backend. |
| Backend-dependent | Compose-used container/network/volume APIs | `docker compose` | Compose support should emerge from enough container, network, volume, logs, exec, and inspect compatibility. |

### Auth

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Planned | `POST /auth` | `docker login` | Can be forwarded if Apple/backend owns registry credentials. Otherwise the adapter needs credential storage. |

### Not Planned or Not Implementable as an Adapter

| Status | Docker API area | Reason |
| --- | --- | --- |
| Not planned | Swarm APIs, for example `/swarm/*`, `/services/*`, `/nodes/*`, `/tasks/*` | Apple Container is not a Docker Swarm manager. Return `501 Not Implemented`. |
| Not planned | Plugin APIs, for example `/plugins/*` | Docker plugin management does not map to Apple Container. Return `501 Not Implemented`. |
| Not planned | Checkpoint APIs, for example `/containers/{id}/checkpoints/*` | Requires CRIU/checkpoint support that is outside the adapter contract. Return `501 Not Implemented` unless the backend gains native support. |
| Not planned | Docker daemon mutation APIs, for example daemon reload/update endpoints | This adapter is not a real Docker daemon. |
| Not planned | Low-level Docker Engine storage-driver behavior | Storage-driver semantics are Docker implementation details and should not leak through the adapter. |

## Project Layout

```text
.
|-- cmd/
|   `-- dockerd-compat/        # CLI entrypoint
|-- internal/
|   |-- api/                   # Docker Engine HTTP API router/controllers
|   |-- backend/               # Backend interface and implementations
|   |   |-- apple/             # Apple Container backend integration point
|   |   `-- memory/            # In-memory backend for tests and local probing
|   |-- errors/                # Error mapping primitives
|   `-- model/                 # Internal engine-neutral models
`-- go.mod
```

The important boundary is `internal/backend.Backend`. The Docker API layer should depend on this interface, not directly on Apple Container APIs.

## Requirements

- Go 1.25 or newer
- macOS for the intended Apple Container backend
- Docker CLI if you want to test the Unix socket with Docker commands

## Build

```sh
go build ./cmd/dockerd-compat
```

If your Go build cache is outside a writable location, use a local cache:

```sh
GOCACHE="$PWD/.gocache" go build ./cmd/dockerd-compat
```

## Run

Start the adapter on a user-owned Unix socket:

```sh
go run ./cmd/dockerd-compat -socket /tmp/docker-compat.sock
```

The default socket is:

```text
~/.docker-compat/docker.sock
```

You can also set it with:

```sh
DOCKER_COMPAT_SOCKET=/tmp/docker-compat.sock go run ./cmd/dockerd-compat
```

## Smoke Test with curl

```sh
curl --unix-socket /tmp/docker-compat.sock http://docker/_ping
curl --unix-socket /tmp/docker-compat.sock http://docker/v1.47/version
curl --unix-socket /tmp/docker-compat.sock http://docker/v1.47/info
curl --unix-socket /tmp/docker-compat.sock http://docker/v1.47/containers/json
curl --unix-socket /tmp/docker-compat.sock http://docker/v1.47/images/json
```

Expected `/_ping` response:

```text
OK
```

## Smoke Test with Docker CLI

Point the Docker CLI at the adapter socket:

```sh
export DOCKER_HOST=unix:///tmp/docker-compat.sock
docker version
docker info
docker ps
docker images
```

With the current memory backend, lifecycle commands are simulated:

```sh
docker create --name demo hello-world:latest echo hello
docker start demo
docker ps
docker inspect demo
docker logs demo
docker rm -f demo
```

These commands validate the Docker API compatibility layer, not real container execution.

## Test

```sh
go test ./...
```

Or with a repository-local build cache:

```sh
GOCACHE="$PWD/.gocache" go test ./...
```

## Architecture

### Docker API Server

`internal/api` owns the Docker-facing HTTP surface. It normalizes versioned paths such as `/v1.47/containers/json`, routes requests to controllers, and writes Docker-shaped JSON responses.

### Backend Interface

`internal/backend.Backend` defines the runtime-neutral operations:

```go
type Backend interface {
    System
    Containers
    Images
    Streams
}
```

This lets the API layer stay stable while backend implementations evolve.

### Translators

The adapter keeps internal models in `internal/model` and translates them into Docker response structures in `internal/api/translate.go`.

Apple-specific translations belong in `internal/backend/apple/translate.go`.

### Error Mapping

Backend errors should be converted into Docker-compatible HTTP status codes and error messages. Docker CLI behavior can depend on both status code and message text, so error strings should be treated as part of the compatibility surface.

### Streaming

The current code includes a basic logs endpoint shape. Real streaming support still needs backend integration for:

- `docker logs -f`
- `docker attach`
- `docker exec`
- `docker events`

## MVP Roadmap

### Phase 1: Read-only Docker CLI compatibility

- `docker version`
- `docker info`
- `docker ps`
- `docker images`

### Phase 2: Basic container lifecycle

- `docker create`
- `docker run`
- `docker start`
- `docker stop`
- `docker rm`

### Phase 3: Developer workflow

- `docker pull`
- `docker push`
- `docker logs`
- `docker exec`

### Phase 4: Inspection and observability

- `docker inspect`
- `docker cp`
- `docker stats`
- `docker events`

### Phase 5: Wider tooling compatibility

- Docker Compose compatibility
- BuildKit integration if needed
- Volumes
- Networks
- IDE integrations such as VS Code and JetBrains Docker tooling

## Capability Detection

Unsupported endpoints currently return `501 Not Implemented`. Backend capability detection should make this explicit, for example:

```go
Capabilities{
    Exec: true,
    Logs: true,
    Swarm: false,
}
```

Requests for unsupported features should fail fast with Docker-compatible status codes and messages.

## Development Notes

- Keep Docker API structs separate from backend structs.
- Do not expose Apple Container models directly through the Docker API layer.
- Add tests at the API boundary when adding or changing an endpoint.
- Prefer implementing backend behavior behind `internal/backend.Backend` instead of special-casing Apple behavior in controllers.
- Treat Docker CLI compatibility as a contract: path, method, status code, headers, JSON field names, and selected error messages all matter.
