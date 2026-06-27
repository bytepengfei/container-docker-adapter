package api

import (
	"strings"
	"time"

	"github.com/pengfei/container-docker-adapter/internal/model"
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
