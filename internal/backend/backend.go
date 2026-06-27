package backend

import (
	"context"
	"io"

	"github.com/pengfei/container-docker-adapter/internal/model"
)

type Backend interface {
	System
	Containers
	Images
	Streams
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
	RemoveContainer(ctx context.Context, id string, opts model.RemoveOptions) error
}

type Images interface {
	ListImages(ctx context.Context, opts model.ListImagesOptions) ([]model.Image, error)
	PullImage(ctx context.Context, ref string, auth model.RegistryAuth, out io.Writer) error
	RemoveImage(ctx context.Context, id string, opts model.RemoveImageOptions) ([]model.ImageDelete, error)
}

type Streams interface {
	ContainerLogs(ctx context.Context, id string, opts model.LogOptions) (io.ReadCloser, error)
}
