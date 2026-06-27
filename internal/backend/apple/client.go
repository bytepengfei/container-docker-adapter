package apple

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/bytepengfei/container-docker-adapter/internal/errors"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type Client struct {
	Binary string
	Runner CommandRunner
}

type Backend struct {
	client       *Client
	mu           sync.Mutex
	sessions     map[string]*processSession
	execConfigs  map[string]model.ExecConfig
	execSessions map[string]*processSession
}

func New(client *Client) *Backend {
	if client == nil {
		client = &Client{}
	}
	if client.Binary == "" {
		client.Binary = "/usr/local/bin/container"
	}
	if client.Runner == nil {
		client.Runner = ExecRunner{Binary: client.Binary}
	}
	return &Backend{
		client:       client,
		sessions:     make(map[string]*processSession),
		execConfigs:  make(map[string]model.ExecConfig),
		execSessions: make(map[string]*processSession),
	}
}

func (b *Backend) Ping(ctx context.Context) error {
	_, err := b.run(ctx, "system", "status")
	return err
}

func (b *Backend) Version(ctx context.Context) (model.Version, error) {
	result, err := b.run(ctx, "system", "version", "--format", "json")
	if err != nil {
		return model.Version{}, err
	}
	var versions []versionDTO
	if err := json.Unmarshal(result.Stdout, &versions); err != nil {
		return model.Version{}, fmt.Errorf("decode container system version: %w", err)
	}
	var version versionDTO
	for _, candidate := range versions {
		if candidate.AppName == "container" {
			version = candidate
			break
		}
	}
	return model.Version{
		Platform:      model.Platform{Name: "Apple Container"},
		Version:       version.Version,
		APIVersion:    "1.47",
		MinAPIVersion: "1.24",
		GitCommit:     version.Commit,
		GoVersion:     runtime.Version(),
		Os:            runtime.GOOS,
		Arch:          runtime.GOARCH,
	}, nil
}

func (b *Backend) Info(ctx context.Context) (model.Info, error) {
	containers, err := b.ListContainers(ctx, model.ListContainersOptions{All: true})
	if err != nil {
		return model.Info{}, err
	}
	images, err := b.ListImages(ctx, model.ListImagesOptions{})
	if err != nil {
		return model.Info{}, err
	}
	info := model.Info{
		ID:              "apple-container",
		Containers:      len(containers),
		Images:          len(images),
		Driver:          "apple-container",
		OperatingSystem: "macOS with Apple Container",
		OSType:          "linux",
		Architecture:    runtime.GOARCH,
		ServerVersion:   "1.0",
	}
	for _, container := range containers {
		switch container.State {
		case "running":
			info.ContainersRunning++
		case "paused":
			info.ContainersPaused++
		default:
			info.ContainersStopped++
		}
	}
	return info, nil
}

func (b *Backend) Capabilities(context.Context) (model.Capabilities, error) {
	return model.Capabilities{
		Exec:     true,
		Logs:     true,
		Attach:   true,
		Stats:    true,
		Volumes:  true,
		Networks: true,
		Build:    true,
	}, nil
}

func (b *Backend) ListContainers(ctx context.Context, opts model.ListContainersOptions) ([]model.Container, error) {
	args := []string{"list", "--format", "json"}
	if opts.All {
		args = append(args, "--all")
	}
	result, err := b.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var values []containerDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return nil, fmt.Errorf("decode container list: %w", err)
	}
	containers := make([]model.Container, 0, len(values))
	for _, value := range values {
		containers = append(containers, b.containerModel(value))
	}
	sort.Slice(containers, func(i, j int) bool { return containers[i].Created.After(containers[j].Created) })
	if opts.Limit > 0 && opts.Limit < len(containers) {
		containers = containers[:opts.Limit]
	}
	return containers, nil
}

func (b *Backend) InspectContainer(ctx context.Context, id string) (model.Container, error) {
	result, err := b.run(ctx, "inspect", id)
	if err != nil {
		return model.Container{}, err
	}
	var values []containerDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return model.Container{}, fmt.Errorf("decode container inspect: %w", err)
	}
	if len(values) == 0 {
		return model.Container{}, fmt.Errorf("%w: No such container: %s", apperrors.ErrNotFound, id)
	}
	return b.containerModel(values[0]), nil
}

func (b *Backend) CreateContainer(ctx context.Context, spec model.ContainerSpec) (model.ContainerCreateResult, error) {
	if spec.Image == "" {
		return model.ContainerCreateResult{}, fmt.Errorf("%w: image is required", apperrors.ErrBadRequest)
	}
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("docker-%x", time.Now().UnixNano())
	}
	args := []string{"create", "--name", name}
	for _, env := range spec.Env {
		args = append(args, "--env", env)
	}
	keys := make([]string, 0, len(spec.Labels))
	for key := range spec.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "--label", key+"="+spec.Labels[key])
	}
	if spec.WorkingDir != "" {
		args = append(args, "--workdir", spec.WorkingDir)
	}
	if spec.Tty {
		args = append(args, "--tty")
	}
	if spec.OpenStdin {
		args = append(args, "--interactive")
	}
	command := append([]string(nil), spec.Cmd...)
	if len(spec.Entrypoint) > 0 {
		args = append(args, "--entrypoint", spec.Entrypoint[0])
		command = append(spec.Entrypoint[1:], command...)
	}
	args = append(args, spec.Image)
	args = append(args, command...)
	result, err := b.run(ctx, args...)
	if err != nil {
		return model.ContainerCreateResult{}, err
	}
	id := strings.TrimSpace(string(result.Stdout))
	if id == "" {
		id = name
	}
	b.mu.Lock()
	b.sessions[id] = newProcessSession()
	b.mu.Unlock()
	return model.ContainerCreateResult{ID: id, Warnings: []string{}}, nil
}

func (b *Backend) StartContainer(ctx context.Context, id string) error {
	session := b.containerSession(id)
	if !session.markStarted() {
		return nil
	}
	return b.startContainerProcess(ctx, id, session)
}

func (b *Backend) startContainerProcess(ctx context.Context, id string, session *processSession) error {
	wait, err := b.client.Runner.Start(context.WithoutCancel(ctx), session, session, "start", "--attach", id)
	if err != nil {
		session.finish(processResult{exitCode: 1, err: b.translateError(err)})
		return b.translateError(err)
	}
	go func() {
		err := wait()
		exitCode := commandExitCode(err)
		session.finish(processResult{exitCode: exitCode, err: b.translateError(err)})
	}()
	return nil
}

func (b *Backend) WaitContainer(ctx context.Context, id string, condition string) (model.ContainerWaitResult, error) {
	session := b.containerSession(id)
	select {
	case <-ctx.Done():
		return model.ContainerWaitResult{}, ctx.Err()
	case <-session.done:
		result := session.wait()
		if condition == "removed" {
			if _, err := b.run(ctx, "delete", "--force", id); err != nil {
				return model.ContainerWaitResult{}, err
			}
		}
		waitResult := model.ContainerWaitResult{StatusCode: result.exitCode}
		if result.err != nil {
			waitResult.Error = &model.ContainerWaitError{Message: result.err.Error()}
		}
		return waitResult, nil
	}
}

func (b *Backend) StopContainer(ctx context.Context, id string, opts model.StopOptions) error {
	args := []string{"stop"}
	if opts.TimeoutSeconds != nil {
		args = append(args, "--time", strconv.Itoa(*opts.TimeoutSeconds))
	}
	args = append(args, id)
	_, err := b.run(ctx, args...)
	return err
}

func (b *Backend) RestartContainer(context.Context, string, model.StopOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) KillContainer(ctx context.Context, id string, signal string) error {
	args := []string{"kill"}
	if signal != "" {
		args = append(args, "--signal", signal)
	}
	args = append(args, id)
	_, err := b.run(ctx, args...)
	return err
}

func (b *Backend) PauseContainer(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) UnpauseContainer(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveContainer(ctx context.Context, id string, opts model.RemoveOptions) error {
	args := []string{"delete"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, id)
	_, err := b.run(ctx, args...)
	if err == nil {
		b.mu.Lock()
		delete(b.sessions, id)
		b.mu.Unlock()
	}
	return err
}

func (b *Backend) ContainerStats(context.Context, string) (model.ContainerStats, error) {
	return model.ContainerStats{}, apperrors.ErrNotImplemented
}

func (b *Backend) ContainerTop(context.Context, string, string) (model.ContainerTop, error) {
	return model.ContainerTop{}, apperrors.ErrNotImplemented
}

func (b *Backend) ContainerChanges(context.Context, string) ([]model.ContainerChange, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) GetContainerArchive(context.Context, string, model.ArchiveOptions) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) PutContainerArchive(context.Context, string, model.ArchiveOptions, io.Reader) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) ResizeContainer(context.Context, string, model.ResizeOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) PruneContainers(context.Context) (model.PruneResult, error) {
	return model.PruneResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) ListImages(ctx context.Context, _ model.ListImagesOptions) ([]model.Image, error) {
	result, err := b.run(ctx, "image", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	var values []imageDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return nil, fmt.Errorf("decode image list: %w", err)
	}
	images := make([]model.Image, 0, len(values))
	for _, value := range values {
		images = append(images, imageModel(value))
	}
	return images, nil
}

func (b *Backend) InspectImage(ctx context.Context, id string) (model.Image, error) {
	result, err := b.run(ctx, "image", "inspect", id)
	if err != nil {
		return model.Image{}, err
	}
	var values []imageDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return model.Image{}, fmt.Errorf("decode image inspect: %w", err)
	}
	if len(values) == 0 {
		return model.Image{}, fmt.Errorf("%w: No such image: %s", apperrors.ErrNotFound, id)
	}
	return imageModel(values[0]), nil
}

func (b *Backend) ImageHistory(context.Context, string) ([]model.ImageHistory, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) PullImage(ctx context.Context, ref string, _ model.RegistryAuth, out io.Writer) error {
	result, err := b.run(ctx, "image", "pull", "--progress", "plain", ref)
	writeDockerStatusOutput(out, result.Stdout)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "{\"status\":\"Downloaded newer image for %s\"}\n", ref)
	return nil
}

func (b *Backend) PushImage(ctx context.Context, ref string, _ model.RegistryAuth, out io.Writer) error {
	result, err := b.run(ctx, "image", "push", "--progress", "plain", ref)
	writeDockerStatusOutput(out, result.Stdout)
	return err
}

func (b *Backend) LoadImages(ctx context.Context, in io.Reader, out io.Writer) error {
	file, err := os.CreateTemp("", "container-docker-adapter-load-*.tar")
	if err != nil {
		return err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := io.Copy(file, in); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	result, err := b.run(ctx, "image", "load", "--input", path)
	writeDockerBuildOutput(out, result.Stdout)
	return err
}

func (b *Backend) GetImage(ctx context.Context, name string) (io.ReadCloser, error) {
	file, err := os.CreateTemp("", "container-docker-adapter-save-*.tar")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	if _, err := b.run(ctx, "image", "save", "--output", path, name); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	reader, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return &removeOnClose{ReadCloser: reader, path: path}, nil
}

func (b *Backend) RemoveImage(ctx context.Context, id string, opts model.RemoveImageOptions) ([]model.ImageDelete, error) {
	args := []string{"image", "delete"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, id)
	if _, err := b.run(ctx, args...); err != nil {
		return nil, err
	}
	return []model.ImageDelete{{Deleted: id}}, nil
}

func (b *Backend) PruneImages(context.Context) (model.PruneResult, error) {
	return model.PruneResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) ContainerLogs(ctx context.Context, id string, opts model.LogOptions) (io.ReadCloser, error) {
	if opts.Follow {
		session := b.existingContainerSession(id)
		if session != nil {
			return session.subscribe(), nil
		}
	}
	args := []string{"logs"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Tail != "" && opts.Tail != "all" {
		args = append(args, "-n", opts.Tail)
	}
	args = append(args, id)
	if opts.Follow {
		reader, writer := io.Pipe()
		wait, err := b.client.Runner.Start(ctx, writer, writer, args...)
		if err != nil {
			_ = writer.CloseWithError(b.translateError(err))
			return nil, b.translateError(err)
		}
		go func() {
			_ = writer.CloseWithError(b.translateError(wait()))
		}()
		return reader, nil
	}
	result, err := b.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(result.Stdout)), nil
}

func (b *Backend) AttachContainer(ctx context.Context, id string, _ model.LogOptions) (io.ReadCloser, error) {
	session := b.containerSession(id)
	stream := session.subscribe()
	if session.markStarted() {
		if err := b.startContainerProcess(ctx, id, session); err != nil {
			_ = stream.Close()
			return nil, err
		}
	}
	return stream, nil
}

func (b *Backend) Events(context.Context) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) CreateExec(ctx context.Context, containerID string, config model.ExecConfig) (model.ExecSession, error) {
	if _, err := b.InspectContainer(ctx, containerID); err != nil {
		return model.ExecSession{}, err
	}
	id := fmt.Sprintf("exec-%x", time.Now().UnixNano())
	config.ContainerID = containerID
	b.mu.Lock()
	b.execConfigs[id] = config
	b.execSessions[id] = newProcessSession()
	b.mu.Unlock()
	return model.ExecSession{ID: id, ContainerID: containerID, Config: config}, nil
}

func (b *Backend) StartExec(ctx context.Context, id string, _ model.ExecConfig) (io.ReadCloser, error) {
	b.mu.Lock()
	config, ok := b.execConfigs[id]
	session := b.execSessions[id]
	b.mu.Unlock()
	if !ok || session == nil {
		return nil, fmt.Errorf("%w: No such exec instance: %s", apperrors.ErrNotFound, id)
	}
	if !session.markStarted() {
		return nil, fmt.Errorf("%w: exec instance %s is already started", apperrors.ErrConflict, id)
	}
	args := []string{"exec"}
	for _, env := range config.Env {
		args = append(args, "--env", env)
	}
	if config.WorkingDir != "" {
		args = append(args, "--workdir", config.WorkingDir)
	}
	if config.User != "" {
		args = append(args, "--user", config.User)
	}
	if config.Tty {
		args = append(args, "--tty")
	}
	if config.AttachStdin {
		args = append(args, "--interactive")
	}
	args = append(args, config.ContainerID)
	args = append(args, config.Cmd...)
	wait, err := b.client.Runner.Start(ctx, session, session, args...)
	if err != nil {
		return nil, b.translateError(err)
	}
	go func() {
		err := wait()
		session.finish(processResult{exitCode: commandExitCode(err), err: b.translateError(err)})
	}()
	return session.subscribe(), nil
}

func (b *Backend) InspectExec(_ context.Context, id string) (model.ExecSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, ok := b.execConfigs[id]
	session := b.execSessions[id]
	if !ok || session == nil {
		return model.ExecSession{}, fmt.Errorf("%w: No such exec instance: %s", apperrors.ErrNotFound, id)
	}
	started, completed := session.state()
	return model.ExecSession{
		ID:          id,
		ContainerID: config.ContainerID,
		Running:     started && !completed,
		Config:      config,
	}, nil
}

func (b *Backend) ResizeExec(context.Context, string, model.ResizeOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) ListVolumes(ctx context.Context) ([]model.Volume, error) {
	result, err := b.run(ctx, "volume", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	var values []volumeDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return nil, fmt.Errorf("decode volume list: %w", err)
	}
	volumes := make([]model.Volume, 0, len(values))
	for _, value := range values {
		volumes = append(volumes, volumeModel(value))
	}
	return volumes, nil
}

func (b *Backend) CreateVolume(ctx context.Context, spec model.VolumeSpec) (model.Volume, error) {
	if spec.Driver != "" && spec.Driver != "local" {
		return model.Volume{}, fmt.Errorf("%w: Apple Container supports only the local volume driver", apperrors.ErrBadRequest)
	}
	args := []string{"volume", "create"}
	args = appendMapArgs(args, "--label", spec.Labels)
	args = appendMapArgs(args, "--opt", spec.DriverOpts)
	args = append(args, spec.Name)
	result, err := b.run(ctx, args...)
	if err != nil {
		return model.Volume{}, err
	}
	name := strings.TrimSpace(string(result.Stdout))
	if name == "" {
		name = spec.Name
	}
	return b.InspectVolume(ctx, name)
}

func (b *Backend) InspectVolume(ctx context.Context, name string) (model.Volume, error) {
	result, err := b.run(ctx, "volume", "inspect", name)
	if err != nil {
		return model.Volume{}, err
	}
	var values []volumeDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return model.Volume{}, fmt.Errorf("decode volume inspect: %w", err)
	}
	if len(values) == 0 {
		return model.Volume{}, fmt.Errorf("%w: No such volume: %s", apperrors.ErrNotFound, name)
	}
	return volumeModel(values[0]), nil
}

func (b *Backend) RemoveVolume(ctx context.Context, name string, _ bool) error {
	_, err := b.run(ctx, "volume", "delete", name)
	return err
}

func (b *Backend) PruneVolumes(ctx context.Context) (model.PruneResult, error) {
	result, err := b.run(ctx, "volume", "prune")
	if err != nil {
		return model.PruneResult{}, err
	}
	return model.PruneResult{Deleted: outputLines(result.Stdout)}, nil
}

func (b *Backend) ListNetworks(ctx context.Context) ([]model.Network, error) {
	result, err := b.run(ctx, "network", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	var values []networkDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return nil, fmt.Errorf("decode network list: %w", err)
	}
	networks := make([]model.Network, 0, len(values))
	for _, value := range values {
		networks = append(networks, networkModel(value))
	}
	return networks, nil
}

func (b *Backend) CreateNetwork(ctx context.Context, spec model.NetworkSpec) (model.Network, error) {
	args := []string{"network", "create"}
	if spec.Internal {
		args = append(args, "--internal")
	}
	args = appendMapArgs(args, "--label", spec.Labels)
	args = appendMapArgs(args, "--option", spec.Options)
	if spec.Driver != "" && spec.Driver != "bridge" {
		args = append(args, "--plugin", spec.Driver)
	}
	args = append(args, spec.Name)
	result, err := b.run(ctx, args...)
	if err != nil {
		return model.Network{}, err
	}
	name := strings.TrimSpace(string(result.Stdout))
	if name == "" {
		name = spec.Name
	}
	return b.InspectNetwork(ctx, name)
}

func (b *Backend) InspectNetwork(ctx context.Context, id string) (model.Network, error) {
	result, err := b.run(ctx, "network", "inspect", id)
	if err != nil {
		return model.Network{}, err
	}
	var values []networkDTO
	if err := json.Unmarshal(result.Stdout, &values); err != nil {
		return model.Network{}, fmt.Errorf("decode network inspect: %w", err)
	}
	if len(values) == 0 {
		return model.Network{}, fmt.Errorf("%w: No such network: %s", apperrors.ErrNotFound, id)
	}
	return networkModel(values[0]), nil
}

func (b *Backend) ConnectNetwork(context.Context, string, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) DisconnectNetwork(context.Context, string, string, bool) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveNetwork(ctx context.Context, id string) error {
	_, err := b.run(ctx, "network", "delete", id)
	return err
}

func (b *Backend) PruneNetworks(ctx context.Context) (model.PruneResult, error) {
	result, err := b.run(ctx, "network", "prune")
	if err != nil {
		return model.PruneResult{}, err
	}
	return model.PruneResult{Deleted: outputLines(result.Stdout)}, nil
}

func (b *Backend) Authenticate(ctx context.Context, auth model.RegistryAuth) (model.AuthResult, error) {
	var payload struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress"`
	}
	if err := json.Unmarshal([]byte(auth.Raw), &payload); err != nil {
		return model.AuthResult{}, fmt.Errorf("%w: invalid registry authentication payload", apperrors.ErrBadRequest)
	}
	if payload.ServerAddress == "" {
		payload.ServerAddress = "docker.io"
	}
	args := []string{"registry", "login", "--password-stdin"}
	if payload.Username != "" {
		args = append(args, "--username", payload.Username)
	}
	args = append(args, payload.ServerAddress)
	result, err := b.client.Runner.RunInput(ctx, strings.NewReader(payload.Password+"\n"), args...)
	if err != nil {
		return model.AuthResult{}, b.translateError(err)
	}
	status := strings.TrimSpace(string(result.Stdout))
	if status == "" {
		status = "Login Succeeded"
	}
	return model.AuthResult{Status: status}, nil
}

func (b *Backend) BuildImage(ctx context.Context, contextTar io.Reader, opts model.BuildOptions, out io.Writer) error {
	directory, err := os.MkdirTemp("", "container-docker-adapter-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := extractBuildContext(contextTar, directory); err != nil {
		return fmt.Errorf("%w: invalid build context: %v", apperrors.ErrBadRequest, err)
	}

	args := []string{"build", "--progress", "plain"}
	for _, tag := range opts.Tags {
		args = append(args, "--tag", tag)
	}
	if opts.Dockerfile != "" {
		args = append(args, "--file", opts.Dockerfile)
	}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}
	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}
	args = appendMapArgs(args, "--build-arg", opts.BuildArgs)
	args = appendMapArgs(args, "--label", opts.Labels)
	args = append(args, directory)
	result, err := b.run(ctx, args...)
	writeDockerBuildOutput(out, result.Stdout)
	return err
}

func (b *Backend) PruneBuildCache(ctx context.Context) (model.PruneResult, error) {
	result, err := b.run(ctx, "builder", "prune")
	if err != nil {
		return model.PruneResult{}, err
	}
	return model.PruneResult{Deleted: outputLines(result.Stdout)}, nil
}

func (b *Backend) run(ctx context.Context, args ...string) (CommandResult, error) {
	result, err := b.client.Runner.Run(ctx, args...)
	if err != nil {
		return result, b.translateError(err)
	}
	return result, nil
}

func (b *Backend) translateError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"),
		strings.Contains(message, "no such"),
		strings.Contains(message, "does not exist"):
		return fmt.Errorf("%w: %v", apperrors.ErrNotFound, err)
	case strings.Contains(message, "already exists"),
		strings.Contains(message, "already running"),
		strings.Contains(message, "invalidstate"),
		strings.Contains(message, "invalid state"):
		return fmt.Errorf("%w: %v", apperrors.ErrConflict, err)
	case strings.Contains(message, "invalid argument"),
		strings.Contains(message, "validation"):
		return fmt.Errorf("%w: %v", apperrors.ErrBadRequest, err)
	default:
		return err
	}
}

func (b *Backend) containerSession(id string) *processSession {
	b.mu.Lock()
	defer b.mu.Unlock()
	session := b.sessions[id]
	if session == nil {
		session = newProcessSession()
		b.sessions[id] = session
	}
	return session
}

func (b *Backend) existingContainerSession(id string) *processSession {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sessions[id]
}

func (b *Backend) containerModel(value containerDTO) model.Container {
	state := value.Status.State
	status := dockerStatus(state, value.Configuration.CreationDate)
	if session := b.existingContainerSession(value.ID); session != nil {
		started, completed := session.state()
		if started && completed {
			state = "exited"
			status = "Exited"
		} else if started {
			state = "running"
			status = "Up"
		}
	}
	command := strings.TrimSpace(strings.Join(append([]string{value.Configuration.InitProcess.Executable}, value.Configuration.InitProcess.Arguments...), " "))
	return model.Container{
		ID:      value.ID,
		Names:   []string{value.ID},
		Image:   value.Configuration.Image.Reference,
		ImageID: value.Configuration.Image.Descriptor.Digest,
		Command: command,
		Created: value.Configuration.CreationDate,
		State:   state,
		Status:  status,
		Tty:     value.Configuration.InitProcess.Terminal,
		Labels:  value.Configuration.Labels,
	}
}

func imageModel(value imageDTO) model.Image {
	size := int64(0)
	for _, variant := range value.Variants {
		size += variant.Size
	}
	id := value.ID
	if !strings.HasPrefix(id, "sha256:") {
		id = "sha256:" + id
	}
	return model.Image{
		ID:          id,
		RepoTags:    []string{value.Configuration.Name},
		RepoDigests: []string{value.Configuration.Name + "@" + value.Configuration.Descriptor.Digest},
		Created:     value.Configuration.CreationDate,
		Size:        size,
		VirtualSize: size,
		Labels:      map[string]string{},
		Containers:  -1,
	}
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.Result.ExitCode >= 0 {
		return commandErr.Result.ExitCode
	}
	return 1
}

func appendMapArgs(args []string, flag string, values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, flag, key+"="+values[key])
	}
	return args
}

func outputLines(output []byte) []string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	return lines
}

func volumeModel(value volumeDTO) model.Volume {
	return model.Volume{
		Name:       value.Configuration.Name,
		Driver:     value.Configuration.Driver,
		Mountpoint: value.Configuration.Source,
		Created:    value.Configuration.CreationDate,
		Labels:     value.Configuration.Labels,
		Options:    value.Configuration.Options,
		Scope:      "local",
	}
}

func networkModel(value networkDTO) model.Network {
	driver := value.Configuration.Plugin
	if driver == "container-network-vmnet" {
		driver = "bridge"
	}
	return model.Network{
		ID:       value.ID,
		Name:     value.Configuration.Name,
		Driver:   driver,
		Scope:    "local",
		Internal: value.Configuration.Mode == "host",
		Created:  value.Configuration.CreationDate,
		Labels:   value.Configuration.Labels,
		Options:  value.Configuration.Options,
	}
}

type removeOnClose struct {
	io.ReadCloser
	path string
}

func extractBuildContext(in io.Reader, destination string) error {
	reader := tar.NewReader(in)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(header.Name)
		if clean == "." {
			continue
		}
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("path escapes build context: %s", header.Name)
		}
		target := filepath.Join(destination, clean)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, reader)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported archive entry: %s", header.Name)
		}
	}
}

func writeDockerBuildOutput(out io.Writer, output []byte) {
	encoder := json.NewEncoder(out)
	for _, line := range outputLines(output) {
		_ = encoder.Encode(map[string]string{"stream": line + "\n"})
	}
}

func writeDockerStatusOutput(out io.Writer, output []byte) {
	encoder := json.NewEncoder(out)
	for _, line := range outputLines(output) {
		_ = encoder.Encode(map[string]string{"status": line})
	}
}

func (r *removeOnClose) Close() error {
	err := r.ReadCloser.Close()
	removeErr := os.Remove(r.path)
	if err != nil {
		return err
	}
	return removeErr
}
