package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	apperrors "github.com/bytepengfei/container-docker-adapter/internal/errors"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type Backend struct {
	mu         sync.RWMutex
	createdAt  time.Time
	containers map[string]model.Container
	images     map[string]model.Image
}

func New() *Backend {
	now := time.Now().UTC()
	return &Backend{
		createdAt:  now,
		containers: map[string]model.Container{},
		images: map[string]model.Image{
			"sha256:example": {
				ID:          "sha256:example",
				RepoTags:    []string{"hello-world:latest"},
				RepoDigests: []string{},
				Created:     now,
				Size:        12345,
				VirtualSize: 12345,
				Labels:      map[string]string{},
				Containers:  -1,
			},
		},
	}
}

func (b *Backend) Ping(context.Context) error {
	return nil
}

func (b *Backend) Version(context.Context) (model.Version, error) {
	return model.Version{
		Platform:      model.Platform{Name: "Docker Compatibility Adapter"},
		Version:       "0.1.0",
		APIVersion:    "1.47",
		MinAPIVersion: "1.24",
		GitCommit:     "dev",
		GoVersion:     runtime.Version(),
		Os:            "linux",
		Arch:          runtime.GOARCH,
		KernelVersion: runtime.GOOS,
		BuildTime:     b.createdAt.Format(time.RFC3339),
		Components: []model.Component{
			{Name: "adapter", Version: "0.1.0"},
		},
	}, nil
}

func (b *Backend) Info(context.Context) (model.Info, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	running := 0
	stopped := 0
	paused := 0
	for _, container := range b.containers {
		switch container.State {
		case "running":
			running++
		case "paused":
			paused++
		default:
			stopped++
		}
	}

	return model.Info{
		ID:                "docker-compat-memory",
		Containers:        len(b.containers),
		ContainersRunning: running,
		ContainersPaused:  paused,
		ContainersStopped: stopped,
		Images:            len(b.images),
		Driver:            "apple-container-adapter",
		OperatingSystem:   "Apple Container compatibility backend",
		OSType:            "linux",
		Architecture:      runtime.GOARCH,
		NCPU:              runtime.NumCPU(),
		ServerVersion:     "0.1.0",
		DockerRootDir:     "",
		Warnings:          []string{"memory backend is for adapter development only"},
	}, nil
}

func (b *Backend) Capabilities(context.Context) (model.Capabilities, error) {
	return model.Capabilities{
		Logs: true,
	}, nil
}

func (b *Backend) ListContainers(_ context.Context, opts model.ListContainersOptions) ([]model.Container, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	containers := make([]model.Container, 0, len(b.containers))
	for _, container := range b.containers {
		if !opts.All && container.State != "running" {
			continue
		}
		containers = append(containers, container)
	}
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Created.After(containers[j].Created)
	})
	if opts.Limit > 0 && opts.Limit < len(containers) {
		containers = containers[:opts.Limit]
	}
	return containers, nil
}

func (b *Backend) InspectContainer(_ context.Context, id string) (model.Container, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.container(id)
}

func (b *Backend) CreateContainer(_ context.Context, spec model.ContainerSpec) (model.ContainerCreateResult, error) {
	if spec.Image == "" {
		return model.ContainerCreateResult{}, fmt.Errorf("%w: image is required", apperrors.ErrBadRequest)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("%x", time.Now().UnixNano())
	name := spec.Name
	if name == "" {
		name = "compat-" + id[:8]
	}
	container := model.Container{
		ID:      id,
		Names:   []string{name},
		Image:   spec.Image,
		ImageID: b.imageIDForRef(spec.Image),
		Command: strings.Join(spec.Cmd, " "),
		Created: time.Now().UTC(),
		State:   "created",
		Status:  "Created",
		Labels:  spec.Labels,
	}
	if container.Labels == nil {
		container.Labels = map[string]string{}
	}
	b.containers[id] = container

	return model.ContainerCreateResult{ID: id, Warnings: []string{}}, nil
}

func (b *Backend) StartContainer(_ context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	container, err := b.container(id)
	if err != nil {
		return err
	}
	container.State = "running"
	container.Status = "Up less than a second"
	b.containers[container.ID] = container
	return nil
}

func (b *Backend) StopContainer(_ context.Context, id string, _ model.StopOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	container, err := b.container(id)
	if err != nil {
		return err
	}
	container.State = "exited"
	container.Status = "Exited (0) less than a second ago"
	b.containers[container.ID] = container
	return nil
}

func (b *Backend) RemoveContainer(_ context.Context, id string, opts model.RemoveOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	container, err := b.container(id)
	if err != nil {
		return err
	}
	if container.State == "running" && !opts.Force {
		return fmt.Errorf("%w: You cannot remove a running container. Stop the container before attempting removal or force remove", apperrors.ErrConflict)
	}
	delete(b.containers, container.ID)
	return nil
}

func (b *Backend) ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	images := make([]model.Image, 0, len(b.images))
	for _, image := range b.images {
		images = append(images, image)
	}
	sort.Slice(images, func(i, j int) bool {
		return images[i].Created.After(images[j].Created)
	})
	return images, nil
}

func (b *Backend) PullImage(_ context.Context, ref string, _ model.RegistryAuth, out io.Writer) error {
	if ref == "" {
		return fmt.Errorf("%w: fromImage is required", apperrors.ErrBadRequest)
	}

	b.mu.Lock()
	id := "sha256:" + strings.NewReplacer("/", "-", ":", "-").Replace(ref)
	b.images[id] = model.Image{
		ID:          id,
		RepoTags:    []string{ref},
		RepoDigests: []string{},
		Created:     time.Now().UTC(),
		Size:        0,
		VirtualSize: 0,
		Labels:      map[string]string{},
		Containers:  -1,
	}
	b.mu.Unlock()

	encoder := json.NewEncoder(out)
	_ = encoder.Encode(map[string]string{"status": "Pulling from compatibility backend", "id": ref})
	_ = encoder.Encode(map[string]string{"status": "Download complete", "id": ref})
	_ = encoder.Encode(map[string]string{"status": "Pull complete", "id": ref})
	return nil
}

func (b *Backend) RemoveImage(_ context.Context, id string, _ model.RemoveImageOptions) ([]model.ImageDelete, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	image, ok := b.images[id]
	if !ok {
		for imageID, candidate := range b.images {
			if contains(candidate.RepoTags, id) {
				id = imageID
				image = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("%w: No such image: %s", apperrors.ErrNotFound, id)
	}
	delete(b.images, id)

	deleted := []model.ImageDelete{{Deleted: image.ID}}
	for _, tag := range image.RepoTags {
		deleted = append([]model.ImageDelete{{Untagged: tag}}, deleted...)
	}
	return deleted, nil
}

func (b *Backend) ContainerLogs(_ context.Context, id string, _ model.LogOptions) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	container, err := b.container(id)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader("logs are not connected to a real Apple Container backend yet: " + container.ID + "\n")), nil
}

func (b *Backend) container(id string) (model.Container, error) {
	if container, ok := b.containers[id]; ok {
		return container, nil
	}
	for _, container := range b.containers {
		if strings.HasPrefix(container.ID, id) || contains(container.Names, id) || contains(container.Names, strings.TrimPrefix(id, "/")) {
			return container, nil
		}
	}
	return model.Container{}, fmt.Errorf("%w: No such container: %s", apperrors.ErrNotFound, id)
}

func (b *Backend) imageIDForRef(ref string) string {
	for _, image := range b.images {
		if contains(image.RepoTags, ref) {
			return image.ID
		}
	}
	return ""
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
