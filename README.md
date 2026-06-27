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

The adapter now has two backends:

- `apple` (default): invokes Apple Container CLI 1.0.x without shell interpolation and runs real containers.
- `memory`: deterministic simulation for unit tests and API development.

The real Apple backend is verified for `docker version`, `docker ps -a`, and the complete `docker run --rm hello-world` workflow. Other routes have different verification levels in the matrix below.

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

- `Route implemented`: The Docker API route exists, but the real Apple path may still return `501`.
- `Contract tested`: Request/response behavior is covered by automated API or fake-CLI tests.
- `CLI verified`: A real Docker CLI workflow passes against a non-Apple test backend.
- `Apple verified`: A real Docker CLI workflow passes against Apple Container 1.0.x.
- `Backend-dependent`: The Docker API route can be implemented only after the Apple Container backend exposes the required capability.
- `Not planned`: The feature is outside the compatibility-layer goal or does not map cleanly to Apple Container.

### System

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Route implemented | `GET /_ping` | Docker CLI daemon probe | Returns `OK`. |
| Route implemented | `HEAD /_ping` | Docker CLI daemon probe | Returns Docker compatibility headers. |
| Apple verified | `GET /version` | `docker version` | Advertises API `1.47`, minimum API `1.24`, with Apple backend version data. |
| Apple verified | `GET /info` | `docker info` | Aggregates real Apple container and image counts. |

### Containers

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Apple verified | `GET /containers/json` | `docker ps`, `docker container ls` | Parses Apple CLI 1.0 structured output. |
| Apple verified | `POST /containers/create` | `docker create`, part of `docker run` | Maps image, command, environment, labels, workdir, TTY, and stdin flags. |
| Route implemented | `GET /containers/{id}/json` | `docker inspect` | Basic inspect response shape exists. |
| Apple verified | `POST /containers/{id}/start` | `docker start`, part of `docker run` | Uses an Apple CLI process session and streams real output. |
| Apple verified | `POST /containers/{id}/wait` | `docker wait`, final step of attached `docker run` | Waits for the Apple init process and returns its exit status. |
| Route implemented | `POST /containers/{id}/stop` | `docker stop` | Memory backend changes state to `exited`. |
| Apple verified | `DELETE /containers/{id}` | `docker rm` | Verified as the final `--rm` step with no residual container. |
| Contract tested | `GET /containers/{id}/logs` | `docker logs` | Apple backend supports snapshot and follow output. |
| Route implemented | `POST /containers/{id}/restart` | `docker restart` | Memory backend simulates stop/start. |
| Contract tested | `POST /containers/{id}/kill` | `docker kill` | Maps to `container kill`. |
| Route implemented | `POST /containers/{id}/pause` | `docker pause` | Memory backend supports state transitions; Apple support is capability-dependent. |
| Route implemented | `POST /containers/{id}/unpause` | `docker unpause` | Memory backend supports state transitions; Apple support is capability-dependent. |
| Route implemented | `GET /containers/{id}/stats` | `docker stats` | Returns Docker-shaped metrics; live Apple metrics remain backend-dependent. |
| Route implemented | `GET /containers/{id}/top` | `docker top` | Returns Docker-shaped process data. |
| Route implemented | `GET /containers/{id}/changes` | `docker diff` | Route and translation exist; real change tracking is backend-dependent. |
| Route implemented | `GET /containers/{id}/archive` | `docker cp` from container | Streams a tar archive. |
| Route implemented | `PUT /containers/{id}/archive` | `docker cp` to container | Accepts a tar archive. |
| Apple verified | `POST /containers/{id}/attach` | `docker attach` | Supports upgrade, multiplex framing, and real Apple process output. |
| Route implemented | `POST /containers/{id}/resize` | `docker resize` | API shape exists; backend TTY support is required. |
| Route implemented | `POST /containers/prune` | `docker container prune` | Docker-compatible prune response fields are returned. |

### Exec and Streaming

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Contract tested | `POST /containers/{id}/exec` | `docker exec` | Creates an Apple CLI exec session. |
| Contract tested | `POST /exec/{id}/start` | `docker exec` | Maps environment, user, workdir and TTY to `container exec`. |
| Route implemented | `GET /exec/{id}/json` | `docker inspect` for exec | Returns Docker-shaped exec state. |
| Route implemented | `POST /exec/{id}/resize` | `docker exec -it` resize | API shape exists; backend TTY support is required. |
| Route implemented | `GET /events` | `docker events` | Streams Docker-shaped event JSON. |

### Images and Registry

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Apple verified | `GET /images/json` | `docker images`, `docker image ls` | Parses Apple CLI 1.0 image metadata. |
| Contract tested | `POST /images/create` | `docker pull` | Maps to `container image pull --progress plain`. |
| Contract tested | `DELETE /images/{id}` | `docker rmi` | Maps to `container image delete`. |
| Contract tested | `GET /images/{id}/json` | `docker image inspect` | Returns Docker-shaped Apple image metadata. |
| Route implemented | `GET /images/{id}/history` | `docker history` | Route exists; complete history remains backend-dependent. |
| Contract tested | `POST /images/{name}/push` | `docker push` | Maps to `container image push`. |
| Contract tested | `POST /images/load` | `docker load` | Uses a private temporary archive with `container image load`. |
| Contract tested | `GET /images/{name}/get` | `docker save` | Streams and removes a temporary OCI archive. |
| Route implemented | `POST /images/prune` | `docker image prune` | Returns Docker-compatible prune fields. |

### Volumes

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Contract tested | `GET /volumes` | `docker volume ls` | Parses Apple structured volume data. |
| Contract tested | `POST /volumes/create` | `docker volume create` | Maps labels and driver options to Apple CLI. |
| Contract tested | `GET /volumes/{name}` | `docker volume inspect` | Maps Apple volume configuration. |
| Contract tested | `DELETE /volumes/{name}` | `docker volume rm` | Maps to `container volume delete`. |
| Contract tested | `POST /volumes/prune` | `docker volume prune` | Maps to `container volume prune`. |

### Networks

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Contract tested | `GET /networks` | `docker network ls` | Parses Apple structured network data. |
| Contract tested | `POST /networks/create` | `docker network create` | Maps labels, internal mode, options and plugin. |
| Contract tested | `GET /networks/{id}` | `docker network inspect` | Maps Apple network configuration. |
| Route implemented | `POST /networks/{id}/connect` | `docker network connect` | Apple networking mapping remains backend-dependent. |
| Route implemented | `POST /networks/{id}/disconnect` | `docker network disconnect` | Apple networking mapping remains backend-dependent. |
| Contract tested | `DELETE /networks/{id}` | `docker network rm` | Maps to `container network delete`. |
| Contract tested | `POST /networks/prune` | `docker network prune` | Maps to `container network prune`. |

### Build and Compose

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Contract tested | `POST /build` | `docker build` | Safely extracts the context and maps options to `container build`. |
| Contract tested | `POST /build/prune` | `docker builder prune` | Maps to `container builder prune`. |
| Backend-dependent | Compose-used container/network/volume APIs | `docker compose` | Compose support should emerge from enough container, network, volume, logs, exec, and inspect compatibility. |

### Auth

| Status | Docker API | Docker CLI | Notes |
| --- | --- | --- | --- |
| Contract tested | `POST /auth` | `docker login` | Sends the password through stdin to `container registry login`. |

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

Start the adapter with the default real Apple backend:

```sh
container system start
go run ./cmd/dockerd-compat \
  -backend apple \
  -container-bin /usr/local/bin/container \
  -socket /tmp/docker-compat.sock
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

Run a real Apple Container through the Docker CLI:

```sh
docker run --rm hello-world
```

To use the deterministic development backend instead:

```sh
go run ./cmd/dockerd-compat -backend memory -socket /tmp/docker-compat.sock
```

## Test

```sh
go test ./...
```

Real macOS E2E is opt-in and requires running Apple Container services:

```sh
APPLE_CONTAINER_E2E=1 ./scripts/e2e-apple.sh
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

The API layer exposes logs, attach, exec, and event streams with Docker content types and response shapes. Attach and interactive exec requests support HTTP connection hijacking, and non-TTY output is encoded with Docker's 8-byte multiplex frame. The memory backend provides deterministic test streams. Long-lived stream lifecycle and bidirectional stdin still require a duplex Apple backend transport.

## MVP Roadmap

The Apple CLI backend currently completes the core `docker run --rm` path. Items below are only complete when their matrix status reaches `Apple verified`; route-only memory simulations do not count as production completion.

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
