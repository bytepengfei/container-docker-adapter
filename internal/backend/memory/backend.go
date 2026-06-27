package memory

import (
	"archive/tar"
	"bytes"
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
	volumes    map[string]model.Volume
	networks   map[string]model.Network
	execs      map[string]model.ExecSession
}

func New() *Backend {
	now := time.Now().UTC()
	return &Backend{
		createdAt:  now,
		containers: map[string]model.Container{},
		volumes:    map[string]model.Volume{},
		networks: map[string]model.Network{
			"bridge": {
				ID:      "bridge",
				Name:    "bridge",
				Driver:  "bridge",
				Scope:   "local",
				Created: now,
				Labels:  map[string]string{},
				Options: map[string]string{},
			},
			"host": {
				ID:      "host",
				Name:    "host",
				Driver:  "host",
				Scope:   "local",
				Created: now,
				Labels:  map[string]string{},
				Options: map[string]string{},
			},
			"none": {
				ID:      "none",
				Name:    "none",
				Driver:  "null",
				Scope:   "local",
				Created: now,
				Labels:  map[string]string{},
				Options: map[string]string{},
			},
		},
		execs: map[string]model.ExecSession{},
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
		Exec:     true,
		Logs:     true,
		Attach:   true,
		Events:   true,
		Stats:    true,
		Volumes:  true,
		Networks: true,
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

func (b *Backend) RestartContainer(ctx context.Context, id string, opts model.StopOptions) error {
	if err := b.StopContainer(ctx, id, opts); err != nil {
		return err
	}
	return b.StartContainer(ctx, id)
}

func (b *Backend) KillContainer(_ context.Context, id string, signal string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	container, err := b.container(id)
	if err != nil {
		return err
	}
	if signal == "" {
		signal = "SIGKILL"
	}
	container.State = "exited"
	container.Status = "Exited (137) after " + signal
	b.containers[container.ID] = container
	return nil
}

func (b *Backend) PauseContainer(_ context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	container, err := b.container(id)
	if err != nil {
		return err
	}
	container.State = "paused"
	container.Status = "Paused"
	b.containers[container.ID] = container
	return nil
}

func (b *Backend) UnpauseContainer(_ context.Context, id string) error {
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

func (b *Backend) ContainerStats(_ context.Context, id string) (model.ContainerStats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	container, err := b.container(id)
	if err != nil {
		return model.ContainerStats{}, err
	}
	return model.ContainerStats{
		ID:          container.ID,
		Name:        firstName(container.Names),
		Read:        time.Now().UTC(),
		CPUUsage:    1000000,
		SystemUsage: 100000000,
		MemoryUsage: 32 * 1024 * 1024,
		MemoryLimit: 1024 * 1024 * 1024,
	}, nil
}

func (b *Backend) ContainerTop(_ context.Context, id string, _ string) (model.ContainerTop, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	container, err := b.container(id)
	if err != nil {
		return model.ContainerTop{}, err
	}
	return model.ContainerTop{
		Titles:    []string{"UID", "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD"},
		Processes: [][]string{{"root", "1", "0", "0", "00:00", "?", "00:00:00", defaultString(container.Command, "sh")}},
	}, nil
}

func (b *Backend) ContainerChanges(_ context.Context, id string) ([]model.ContainerChange, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, err := b.container(id); err != nil {
		return nil, err
	}
	return []model.ContainerChange{}, nil
}

func (b *Backend) GetContainerArchive(_ context.Context, id string, opts model.ArchiveOptions) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, err := b.container(id); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(tarBytes(defaultString(opts.Path, "/")))), nil
}

func (b *Backend) PutContainerArchive(_ context.Context, id string, _ model.ArchiveOptions, in io.Reader) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, err := b.container(id); err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, in)
	return nil
}

func (b *Backend) ResizeContainer(_ context.Context, id string, _ model.ResizeOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, err := b.container(id)
	return err
}

func (b *Backend) PruneContainers(context.Context) (model.PruneResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	deleted := []string{}
	for id, container := range b.containers {
		if container.State != "running" {
			deleted = append(deleted, id)
			delete(b.containers, id)
		}
	}
	return model.PruneResult{Deleted: deleted}, nil
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

func (b *Backend) InspectImage(_ context.Context, id string) (model.Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.image(id)
}

func (b *Backend) ImageHistory(_ context.Context, id string) ([]model.ImageHistory, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	image, err := b.image(id)
	if err != nil {
		return nil, err
	}
	return []model.ImageHistory{{
		ID:        image.ID,
		Created:   image.Created.Unix(),
		CreatedBy: "container-docker-adapter memory backend",
		Tags:      image.RepoTags,
		Size:      image.Size,
	}}, nil
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

func (b *Backend) PushImage(_ context.Context, ref string, _ model.RegistryAuth, out io.Writer) error {
	if ref == "" {
		return fmt.Errorf("%w: image name is required", apperrors.ErrBadRequest)
	}
	b.mu.RLock()
	_, err := b.image(ref)
	b.mu.RUnlock()
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	_ = encoder.Encode(map[string]string{"status": "Pushing to compatibility backend", "id": ref})
	_ = encoder.Encode(map[string]string{"status": "Pushed", "id": ref})
	return nil
}

func (b *Backend) LoadImages(_ context.Context, in io.Reader, out io.Writer) error {
	_, _ = io.Copy(io.Discard, in)
	ref := "loaded:latest"
	b.mu.Lock()
	b.images["sha256:loaded"] = model.Image{
		ID:          "sha256:loaded",
		RepoTags:    []string{ref},
		RepoDigests: []string{},
		Created:     time.Now().UTC(),
		Labels:      map[string]string{},
		Containers:  -1,
	}
	b.mu.Unlock()
	return json.NewEncoder(out).Encode(map[string]string{"stream": "Loaded image: " + ref + "\n"})
}

func (b *Backend) GetImage(_ context.Context, name string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, err := b.image(name); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(tarBytes("manifest.json"))), nil
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

func (b *Backend) PruneImages(context.Context) (model.PruneResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	deleted := []string{}
	for id, image := range b.images {
		if image.Containers <= 0 && !contains(image.RepoTags, "hello-world:latest") {
			deleted = append(deleted, id)
			delete(b.images, id)
		}
	}
	return model.PruneResult{Deleted: deleted}, nil
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

func (b *Backend) AttachContainer(ctx context.Context, id string, opts model.LogOptions) (io.ReadCloser, error) {
	return b.ContainerLogs(ctx, id, opts)
}

func (b *Backend) Events(context.Context) (io.ReadCloser, error) {
	event := model.Event{
		Type:     "daemon",
		Action:   "compatibility-adapter",
		Actor:    model.EventActor{ID: "memory", Attributes: map[string]string{"backend": "memory"}},
		Time:     time.Now().Unix(),
		TimeNano: time.Now().UnixNano(),
	}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(event)
	return io.NopCloser(&buf), nil
}

func (b *Backend) CreateExec(_ context.Context, containerID string, config model.ExecConfig) (model.ExecSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	container, err := b.container(containerID)
	if err != nil {
		return model.ExecSession{}, err
	}
	id := fmt.Sprintf("exec-%x", time.Now().UnixNano())
	config.ContainerID = container.ID
	session := model.ExecSession{
		ID:          id,
		ContainerID: container.ID,
		Config:      config,
	}
	b.execs[id] = session
	return session, nil
}

func (b *Backend) StartExec(_ context.Context, id string, _ model.ExecConfig) (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	session, ok := b.execs[id]
	if !ok {
		return nil, fmt.Errorf("%w: No such exec instance: %s", apperrors.ErrNotFound, id)
	}
	session.Running = false
	session.ExitCode = 0
	b.execs[id] = session
	return io.NopCloser(strings.NewReader("exec simulated by memory backend\n")), nil
}

func (b *Backend) InspectExec(_ context.Context, id string) (model.ExecSession, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	session, ok := b.execs[id]
	if !ok {
		return model.ExecSession{}, fmt.Errorf("%w: No such exec instance: %s", apperrors.ErrNotFound, id)
	}
	return session, nil
}

func (b *Backend) ResizeExec(_ context.Context, id string, _ model.ResizeOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, ok := b.execs[id]; !ok {
		return fmt.Errorf("%w: No such exec instance: %s", apperrors.ErrNotFound, id)
	}
	return nil
}

func (b *Backend) ListVolumes(context.Context) ([]model.Volume, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	volumes := make([]model.Volume, 0, len(b.volumes))
	for _, volume := range b.volumes {
		volumes = append(volumes, volume)
	}
	sort.Slice(volumes, func(i, j int) bool { return volumes[i].Name < volumes[j].Name })
	return volumes, nil
}

func (b *Backend) CreateVolume(_ context.Context, spec model.VolumeSpec) (model.Volume, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("volume-%x", time.Now().UnixNano())
	}
	volume := model.Volume{
		Name:       name,
		Driver:     defaultString(spec.Driver, "local"),
		Mountpoint: "/var/lib/docker-compat/volumes/" + name,
		Created:    time.Now().UTC(),
		Labels:     nonNilMap(spec.Labels),
		Options:    nonNilMap(spec.DriverOpts),
		Scope:      "local",
	}
	b.volumes[name] = volume
	return volume, nil
}

func (b *Backend) InspectVolume(_ context.Context, name string) (model.Volume, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	volume, ok := b.volumes[name]
	if !ok {
		return model.Volume{}, fmt.Errorf("%w: get %s: no such volume", apperrors.ErrNotFound, name)
	}
	return volume, nil
}

func (b *Backend) RemoveVolume(_ context.Context, name string, _ bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.volumes[name]; !ok {
		return fmt.Errorf("%w: get %s: no such volume", apperrors.ErrNotFound, name)
	}
	delete(b.volumes, name)
	return nil
}

func (b *Backend) PruneVolumes(context.Context) (model.PruneResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	deleted := []string{}
	for name := range b.volumes {
		deleted = append(deleted, name)
		delete(b.volumes, name)
	}
	return model.PruneResult{Deleted: deleted}, nil
}

func (b *Backend) ListNetworks(context.Context) ([]model.Network, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	networks := make([]model.Network, 0, len(b.networks))
	for _, network := range b.networks {
		networks = append(networks, network)
	}
	sort.Slice(networks, func(i, j int) bool { return networks[i].Name < networks[j].Name })
	return networks, nil
}

func (b *Backend) CreateNetwork(_ context.Context, spec model.NetworkSpec) (model.Network, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if spec.Name == "" {
		return model.Network{}, fmt.Errorf("%w: network name is required", apperrors.ErrBadRequest)
	}
	id := fmt.Sprintf("%x", time.Now().UnixNano())
	network := model.Network{
		ID:         id,
		Name:       spec.Name,
		Driver:     defaultString(spec.Driver, "bridge"),
		Scope:      "local",
		Internal:   spec.Internal,
		Attachable: spec.Attachable,
		Created:    time.Now().UTC(),
		Labels:     nonNilMap(spec.Labels),
		Options:    nonNilMap(spec.Options),
	}
	b.networks[id] = network
	return network, nil
}

func (b *Backend) InspectNetwork(_ context.Context, id string) (model.Network, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.network(id)
}

func (b *Backend) ConnectNetwork(_ context.Context, id string, containerID string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, err := b.network(id); err != nil {
		return err
	}
	if containerID != "" {
		_, err := b.container(containerID)
		return err
	}
	return nil
}

func (b *Backend) DisconnectNetwork(ctx context.Context, id string, containerID string, _ bool) error {
	return b.ConnectNetwork(ctx, id, containerID)
}

func (b *Backend) RemoveNetwork(_ context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	network, err := b.network(id)
	if err != nil {
		return err
	}
	if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
		return fmt.Errorf("%w: predefined network cannot be removed", apperrors.ErrConflict)
	}
	delete(b.networks, network.ID)
	return nil
}

func (b *Backend) PruneNetworks(context.Context) (model.PruneResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	deleted := []string{}
	for id, network := range b.networks {
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}
		deleted = append(deleted, id)
		delete(b.networks, id)
	}
	return model.PruneResult{Deleted: deleted}, nil
}

func (b *Backend) Authenticate(context.Context, model.RegistryAuth) (model.AuthResult, error) {
	return model.AuthResult{Status: "Login Succeeded"}, nil
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

func (b *Backend) image(id string) (model.Image, error) {
	if image, ok := b.images[id]; ok {
		return image, nil
	}
	for _, image := range b.images {
		if strings.HasPrefix(image.ID, id) || contains(image.RepoTags, id) || contains(image.RepoDigests, id) {
			return image, nil
		}
	}
	return model.Image{}, fmt.Errorf("%w: No such image: %s", apperrors.ErrNotFound, id)
}

func (b *Backend) network(id string) (model.Network, error) {
	if network, ok := b.networks[id]; ok {
		return network, nil
	}
	for _, network := range b.networks {
		if network.Name == id || strings.HasPrefix(network.ID, id) {
			return network, nil
		}
	}
	return model.Network{}, fmt.Errorf("%w: network %s not found", apperrors.ErrNotFound, id)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func firstName(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimPrefix(values[0], "/")
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nonNilMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func tarBytes(name string) []byte {
	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	content := []byte("generated by container-docker-adapter memory backend\n")
	_ = writer.WriteHeader(&tar.Header{
		Name: strings.TrimPrefix(defaultString(name, "compat.txt"), "/"),
		Mode: 0600,
		Size: int64(len(content)),
	})
	_, _ = writer.Write(content)
	_ = writer.Close()
	return buf.Bytes()
}
