package backend

import (
	"context"
	"io"

	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type Backend interface {
	System
	Containers
	Images
	Streams
	Exec
	Volumes
	Networks
	Auth
}

type System interface {
	Info(ctx context.Context) (model.Info, error)
	Version(ctx context.Context) (model.Version, error)
	Ping(ctx context.Context) error
	Capabilities(ctx context.Context) (model.Capabilities, error)
}

type Containers interface {
	ListContainers(ctx context.Context, opts model.ListContainersOptions) ([]model.Container, error)
	InspectContainer(ctx context.Context, id string) (model.Container, error)
	CreateContainer(ctx context.Context, spec model.ContainerSpec) (model.ContainerCreateResult, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, opts model.StopOptions) error
	RestartContainer(ctx context.Context, id string, opts model.StopOptions) error
	KillContainer(ctx context.Context, id string, signal string) error
	PauseContainer(ctx context.Context, id string) error
	UnpauseContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string, opts model.RemoveOptions) error
	ContainerStats(ctx context.Context, id string) (model.ContainerStats, error)
	ContainerTop(ctx context.Context, id string, psArgs string) (model.ContainerTop, error)
	ContainerChanges(ctx context.Context, id string) ([]model.ContainerChange, error)
	GetContainerArchive(ctx context.Context, id string, opts model.ArchiveOptions) (io.ReadCloser, error)
	PutContainerArchive(ctx context.Context, id string, opts model.ArchiveOptions, in io.Reader) error
	ResizeContainer(ctx context.Context, id string, opts model.ResizeOptions) error
	PruneContainers(ctx context.Context) (model.PruneResult, error)
}

type Images interface {
	ListImages(ctx context.Context, opts model.ListImagesOptions) ([]model.Image, error)
	InspectImage(ctx context.Context, id string) (model.Image, error)
	ImageHistory(ctx context.Context, id string) ([]model.ImageHistory, error)
	PullImage(ctx context.Context, ref string, auth model.RegistryAuth, out io.Writer) error
	PushImage(ctx context.Context, ref string, auth model.RegistryAuth, out io.Writer) error
	LoadImages(ctx context.Context, in io.Reader, out io.Writer) error
	GetImage(ctx context.Context, name string) (io.ReadCloser, error)
	RemoveImage(ctx context.Context, id string, opts model.RemoveImageOptions) ([]model.ImageDelete, error)
	PruneImages(ctx context.Context) (model.PruneResult, error)
}

type Streams interface {
	ContainerLogs(ctx context.Context, id string, opts model.LogOptions) (io.ReadCloser, error)
	AttachContainer(ctx context.Context, id string, opts model.LogOptions) (io.ReadCloser, error)
	Events(ctx context.Context) (io.ReadCloser, error)
}

type Exec interface {
	CreateExec(ctx context.Context, containerID string, config model.ExecConfig) (model.ExecSession, error)
	StartExec(ctx context.Context, id string, config model.ExecConfig) (io.ReadCloser, error)
	InspectExec(ctx context.Context, id string) (model.ExecSession, error)
	ResizeExec(ctx context.Context, id string, opts model.ResizeOptions) error
}

type Volumes interface {
	ListVolumes(ctx context.Context) ([]model.Volume, error)
	CreateVolume(ctx context.Context, spec model.VolumeSpec) (model.Volume, error)
	InspectVolume(ctx context.Context, name string) (model.Volume, error)
	RemoveVolume(ctx context.Context, name string, force bool) error
	PruneVolumes(ctx context.Context) (model.PruneResult, error)
}

type Networks interface {
	ListNetworks(ctx context.Context) ([]model.Network, error)
	CreateNetwork(ctx context.Context, spec model.NetworkSpec) (model.Network, error)
	InspectNetwork(ctx context.Context, id string) (model.Network, error)
	ConnectNetwork(ctx context.Context, id string, container string) error
	DisconnectNetwork(ctx context.Context, id string, container string, force bool) error
	RemoveNetwork(ctx context.Context, id string) error
	PruneNetworks(ctx context.Context) (model.PruneResult, error)
}

type Auth interface {
	Authenticate(ctx context.Context, auth model.RegistryAuth) (model.AuthResult, error)
}
