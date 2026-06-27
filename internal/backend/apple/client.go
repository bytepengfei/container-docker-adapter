package apple

import (
	"context"
	"io"

	apperrors "github.com/bytepengfei/container-docker-adapter/internal/errors"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type Client struct {
	BaseURL string
}

type Backend struct {
	client *Client
}

func New(client *Client) *Backend {
	return &Backend{client: client}
}

func (b *Backend) Ping(context.Context) error {
	return nil
}

func (b *Backend) Version(context.Context) (model.Version, error) {
	return model.Version{}, apperrors.ErrNotImplemented
}

func (b *Backend) Info(context.Context) (model.Info, error) {
	return model.Info{}, apperrors.ErrNotImplemented
}

func (b *Backend) Capabilities(context.Context) (model.Capabilities, error) {
	return model.Capabilities{}, apperrors.ErrNotImplemented
}

func (b *Backend) ListContainers(context.Context, model.ListContainersOptions) ([]model.Container, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) InspectContainer(context.Context, string) (model.Container, error) {
	return model.Container{}, apperrors.ErrNotImplemented
}

func (b *Backend) CreateContainer(context.Context, model.ContainerSpec) (model.ContainerCreateResult, error) {
	return model.ContainerCreateResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) StartContainer(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) StopContainer(context.Context, string, model.StopOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveContainer(context.Context, string, model.RemoveOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) PullImage(context.Context, string, model.RegistryAuth, io.Writer) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) ContainerLogs(context.Context, string, model.LogOptions) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}
