package api

import (
	"strings"
	"time"

	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type dockerContainerSummary struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	Command string            `json:"Command"`
	Created int64             `json:"Created"`
	Ports   []model.Port      `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Mounts  []model.Mount     `json:"Mounts"`
}

type dockerContainerInspect struct {
	ID      string        `json:"Id"`
	Name    string        `json:"Name"`
	Created string        `json:"Created"`
	Path    string        `json:"Path"`
	Args    []string      `json:"Args"`
	Image   string        `json:"Image"`
	State   dockerState   `json:"State"`
	Config  dockerConfig  `json:"Config"`
	Mounts  []model.Mount `json:"Mounts"`
}

type dockerState struct {
	Status     string `json:"Status"`
	Running    bool   `json:"Running"`
	Paused     bool   `json:"Paused"`
	Restarting bool   `json:"Restarting"`
	OOMKilled  bool   `json:"OOMKilled"`
	Dead       bool   `json:"Dead"`
	Pid        int    `json:"Pid"`
	ExitCode   int    `json:"ExitCode"`
	Error      string `json:"Error"`
	StartedAt  string `json:"StartedAt"`
	FinishedAt string `json:"FinishedAt"`
}

type dockerConfig struct {
	Image  string            `json:"Image"`
	Labels map[string]string `json:"Labels"`
}

type dockerImageSummary struct {
	ID          string            `json:"Id"`
	ParentID    string            `json:"ParentId"`
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests"`
	Created     int64             `json:"Created"`
	Size        int64             `json:"Size"`
	SharedSize  int64             `json:"SharedSize"`
	VirtualSize int64             `json:"VirtualSize"`
	Labels      map[string]string `json:"Labels"`
	Containers  int64             `json:"Containers"`
}

type dockerImageInspect struct {
	ID           string            `json:"Id"`
	RepoTags     []string          `json:"RepoTags"`
	RepoDigests  []string          `json:"RepoDigests"`
	Created      string            `json:"Created"`
	Size         int64             `json:"Size"`
	VirtualSize  int64             `json:"VirtualSize"`
	Labels       map[string]string `json:"Labels"`
	Architecture string            `json:"Architecture"`
	Os           string            `json:"Os"`
}

type dockerVolume struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	CreatedAt  string            `json:"CreatedAt"`
	Labels     map[string]string `json:"Labels"`
	Options    map[string]string `json:"Options"`
	Scope      string            `json:"Scope"`
}

type dockerNetwork struct {
	Name       string            `json:"Name"`
	ID         string            `json:"Id"`
	Created    string            `json:"Created"`
	Scope      string            `json:"Scope"`
	Driver     string            `json:"Driver"`
	EnableIPv6 bool              `json:"EnableIPv6"`
	Internal   bool              `json:"Internal"`
	Attachable bool              `json:"Attachable"`
	Ingress    bool              `json:"Ingress"`
	Labels     map[string]string `json:"Labels"`
	Options    map[string]string `json:"Options"`
	Containers map[string]any    `json:"Containers"`
}

type dockerContainerCreateRequest struct {
	Image      string            `json:"Image"`
	Cmd        []string          `json:"Cmd"`
	Entrypoint []string          `json:"Entrypoint"`
	Env        []string          `json:"Env"`
	Labels     map[string]string `json:"Labels"`
	WorkingDir string            `json:"WorkingDir"`
	Tty        bool              `json:"Tty"`
	OpenStdin  bool              `json:"OpenStdin"`
}

func (r dockerContainerCreateRequest) Model(name string) model.ContainerSpec {
	return model.ContainerSpec{
		Name:       name,
		Image:      r.Image,
		Cmd:        r.Cmd,
		Entrypoint: r.Entrypoint,
		Env:        r.Env,
		Labels:     r.Labels,
		WorkingDir: r.WorkingDir,
		Tty:        r.Tty,
		OpenStdin:  r.OpenStdin,
	}
}

func DockerContainers(containers []model.Container) []dockerContainerSummary {
	out := make([]dockerContainerSummary, 0, len(containers))
	for _, container := range containers {
		out = append(out, dockerContainerSummary{
			ID:      container.ID,
			Names:   dockerNames(container.Names),
			Image:   container.Image,
			ImageID: container.ImageID,
			Command: container.Command,
			Created: container.Created.Unix(),
			Ports:   container.Ports,
			Labels:  nonNilMap(container.Labels),
			State:   container.State,
			Status:  container.Status,
			Mounts:  container.Mounts,
		})
	}
	return out
}

func DockerContainerInspect(container model.Container) dockerContainerInspect {
	name := ""
	if len(container.Names) > 0 {
		name = dockerNames(container.Names)[0]
	}
	created := container.Created.UTC().Format(time.RFC3339Nano)
	return dockerContainerInspect{
		ID:      container.ID,
		Name:    name,
		Created: created,
		Image:   container.ImageID,
		State: dockerState{
			Status:     container.State,
			Running:    container.State == "running",
			Paused:     container.State == "paused",
			StartedAt:  created,
			FinishedAt: time.Time{}.UTC().Format(time.RFC3339Nano),
		},
		Config: dockerConfig{
			Image:  container.Image,
			Labels: nonNilMap(container.Labels),
		},
		Mounts: container.Mounts,
	}
}

func DockerImages(images []model.Image) []dockerImageSummary {
	out := make([]dockerImageSummary, 0, len(images))
	for _, image := range images {
		out = append(out, dockerImageSummary{
			ID:          image.ID,
			RepoTags:    nonNilSlice(image.RepoTags),
			RepoDigests: nonNilSlice(image.RepoDigests),
			Created:     image.Created.Unix(),
			Size:        image.Size,
			SharedSize:  image.SharedSize,
			VirtualSize: image.VirtualSize,
			Labels:      nonNilMap(image.Labels),
			Containers:  image.Containers,
		})
	}
	return out
}

func DockerImageInspect(image model.Image) dockerImageInspect {
	return dockerImageInspect{
		ID:           image.ID,
		RepoTags:     nonNilSlice(image.RepoTags),
		RepoDigests:  nonNilSlice(image.RepoDigests),
		Created:      image.Created.UTC().Format(time.RFC3339Nano),
		Size:         image.Size,
		VirtualSize:  image.VirtualSize,
		Labels:       nonNilMap(image.Labels),
		Architecture: "arm64",
		Os:           "linux",
	}
}

func DockerContainerStats(stats model.ContainerStats) map[string]any {
	return map[string]any{
		"read": stats.Read.UTC().Format(time.RFC3339Nano),
		"id":   stats.ID,
		"name": stats.Name,
		"cpu_stats": map[string]any{
			"cpu_usage":        map[string]any{"total_usage": stats.CPUUsage},
			"system_cpu_usage": stats.SystemUsage,
		},
		"precpu_stats": map[string]any{},
		"memory_stats": map[string]any{
			"usage": stats.MemoryUsage,
			"limit": stats.MemoryLimit,
		},
		"networks": map[string]any{},
	}
}

func DockerPruneResult(kind string, result model.PruneResult) map[string]any {
	field := map[string]string{
		"container": "ContainersDeleted",
		"image":     "ImagesDeleted",
		"volume":    "VolumesDeleted",
		"network":   "NetworksDeleted",
	}[kind]
	return map[string]any{
		field:            nonNilSlice(result.Deleted),
		"SpaceReclaimed": result.SpaceReclaimed,
	}
}

func DockerExecInspect(session model.ExecSession) map[string]any {
	return map[string]any{
		"ID":       session.ID,
		"Running":  session.Running,
		"ExitCode": session.ExitCode,
		"ProcessConfig": map[string]any{
			"entrypoint": firstOrEmpty(session.Config.Cmd),
			"arguments":  restOrEmpty(session.Config.Cmd),
			"tty":        session.Config.Tty,
			"user":       session.Config.User,
		},
		"OpenStdin":   session.Config.AttachStdin,
		"OpenStderr":  session.Config.AttachStderr,
		"OpenStdout":  session.Config.AttachStdout,
		"CanRemove":   true,
		"ContainerID": session.ContainerID,
	}
}

func DockerVolumes(volumes []model.Volume) []dockerVolume {
	out := make([]dockerVolume, 0, len(volumes))
	for _, volume := range volumes {
		out = append(out, DockerVolume(volume))
	}
	return out
}

func DockerVolume(volume model.Volume) dockerVolume {
	return dockerVolume{
		Name:       volume.Name,
		Driver:     defaultString(volume.Driver, "local"),
		Mountpoint: volume.Mountpoint,
		CreatedAt:  volume.Created.UTC().Format(time.RFC3339Nano),
		Labels:     nonNilMap(volume.Labels),
		Options:    nonNilMap(volume.Options),
		Scope:      defaultString(volume.Scope, "local"),
	}
}

func DockerNetworks(networks []model.Network) []dockerNetwork {
	out := make([]dockerNetwork, 0, len(networks))
	for _, network := range networks {
		out = append(out, DockerNetwork(network))
	}
	return out
}

func DockerNetwork(network model.Network) dockerNetwork {
	return dockerNetwork{
		Name:       network.Name,
		ID:         network.ID,
		Created:    network.Created.UTC().Format(time.RFC3339Nano),
		Scope:      defaultString(network.Scope, "local"),
		Driver:     defaultString(network.Driver, "bridge"),
		Internal:   network.Internal,
		Attachable: network.Attachable,
		Ingress:    network.Ingress,
		Labels:     nonNilMap(network.Labels),
		Options:    nonNilMap(network.Options),
		Containers: map[string]any{},
	}
}

func dockerNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "/") {
			out = append(out, name)
			continue
		}
		out = append(out, "/"+name)
	}
	return out
}

func nonNilMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func nonNilSlice(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func restOrEmpty(values []string) []string {
	if len(values) <= 1 {
		return []string{}
	}
	return values[1:]
}
