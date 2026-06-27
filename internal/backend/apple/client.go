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

func (b *Backend) RestartContainer(context.Context, string, model.StopOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) KillContainer(context.Context, string, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) PauseContainer(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) UnpauseContainer(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveContainer(context.Context, string, model.RemoveOptions) error {
	return apperrors.ErrNotImplemented
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

func (b *Backend) ListImages(context.Context, model.ListImagesOptions) ([]model.Image, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) InspectImage(context.Context, string) (model.Image, error) {
	return model.Image{}, apperrors.ErrNotImplemented
}

func (b *Backend) ImageHistory(context.Context, string) ([]model.ImageHistory, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) PullImage(context.Context, string, model.RegistryAuth, io.Writer) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) PushImage(context.Context, string, model.RegistryAuth, io.Writer) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) LoadImages(context.Context, io.Reader, io.Writer) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) GetImage(context.Context, string) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) RemoveImage(context.Context, string, model.RemoveImageOptions) ([]model.ImageDelete, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) PruneImages(context.Context) (model.PruneResult, error) {
	return model.PruneResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) ContainerLogs(context.Context, string, model.LogOptions) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) AttachContainer(context.Context, string, model.LogOptions) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) Events(context.Context) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) CreateExec(context.Context, string, model.ExecConfig) (model.ExecSession, error) {
	return model.ExecSession{}, apperrors.ErrNotImplemented
}

func (b *Backend) StartExec(context.Context, string, model.ExecConfig) (io.ReadCloser, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) InspectExec(context.Context, string) (model.ExecSession, error) {
	return model.ExecSession{}, apperrors.ErrNotImplemented
}

func (b *Backend) ResizeExec(context.Context, string, model.ResizeOptions) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) ListVolumes(context.Context) ([]model.Volume, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) CreateVolume(context.Context, model.VolumeSpec) (model.Volume, error) {
	return model.Volume{}, apperrors.ErrNotImplemented
}

func (b *Backend) InspectVolume(context.Context, string) (model.Volume, error) {
	return model.Volume{}, apperrors.ErrNotImplemented
}

func (b *Backend) RemoveVolume(context.Context, string, bool) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) PruneVolumes(context.Context) (model.PruneResult, error) {
	return model.PruneResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) ListNetworks(context.Context) ([]model.Network, error) {
	return nil, apperrors.ErrNotImplemented
}

func (b *Backend) CreateNetwork(context.Context, model.NetworkSpec) (model.Network, error) {
	return model.Network{}, apperrors.ErrNotImplemented
}

func (b *Backend) InspectNetwork(context.Context, string) (model.Network, error) {
	return model.Network{}, apperrors.ErrNotImplemented
}

func (b *Backend) ConnectNetwork(context.Context, string, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) DisconnectNetwork(context.Context, string, string, bool) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) RemoveNetwork(context.Context, string) error {
	return apperrors.ErrNotImplemented
}

func (b *Backend) PruneNetworks(context.Context) (model.PruneResult, error) {
	return model.PruneResult{}, apperrors.ErrNotImplemented
}

func (b *Backend) Authenticate(context.Context, model.RegistryAuth) (model.AuthResult, error) {
	return model.AuthResult{}, apperrors.ErrNotImplemented
}
