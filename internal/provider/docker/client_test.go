package docker

import (
	"bytes"
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
)

type TestClient struct {
	client.APIClient

	// ContainerStart
	containerStartArgs   [][]any
	containerStartReturn error

	// ContainerCreate
	containerCreateArgs      [][]any
	containerCreateReturnErr error
	containerCreateReturnId  string

	// ContainerRemove
	containerRemoveArgs   [][]any
	containerRemoveReturn error

	// ContainerList
	containerListArgs             [][]any
	containerListReturnContainers []container.Summary
	containerListReturnErr        error

	// ContainerInspect
	containerInspectArgs   [][]any
	containerInspectReturn []container.InspectResponse
	containerInspectErr    error

	// ImagePull
	imagePullArgs      [][]any
	imagePullReturnErr error
}

func NewTestClient() *TestClient {
	return &TestClient{
		containerStartArgs:   make([][]any, 0),
		containerStartReturn: nil,

		containerCreateReturnErr: nil,
		containerCreateReturnId:  "",
		containerCreateArgs:      make([][]any, 0),

		containerRemoveArgs:   make([][]any, 0),
		containerRemoveReturn: nil,

		containerListArgs:             make([][]any, 0),
		containerListReturnContainers: make([]container.Summary, 0),
		containerListReturnErr:        nil,

		imagePullArgs:      make([][]any, 0),
		imagePullReturnErr: nil,
	}
}

func (t *TestClient) ContainerStart(ctx context.Context, id string, opts container.StartOptions) error {
	args := make([]any, 3)
	args[0] = ctx
	args[1] = id
	args[2] = opts

	t.containerStartArgs = append(t.containerStartArgs, args)
	return t.containerStartReturn
}

func (t *TestClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.CreateResponse, error) {
	args := make([]any, 6)
	args[0] = ctx
	args[1] = config
	args[2] = hostConfig
	args[3] = networkingConfig
	args[4] = platform
	args[5] = containerName

	t.containerCreateArgs = append(t.containerCreateArgs, args)
	return container.CreateResponse{ID: t.containerCreateReturnId}, t.containerCreateReturnErr
}

func (t *TestClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	args := make([]any, 2)
	args[0] = ctx
	args[1] = options

	t.containerListArgs = append(t.containerListArgs, args)
	return t.containerListReturnContainers, t.containerListReturnErr
}

func (t *TestClient) ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error {
	args := make([]any, 3)
	args[0] = ctx
	args[1] = container
	args[2] = options

	t.containerRemoveArgs = append(t.containerRemoveArgs, args)
	return t.containerRemoveReturn
}

func (t *TestClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	args := make([]any, 3)
	args[0] = ctx
	args[1] = refStr
	args[2] = options
	t.imagePullArgs = append(t.imagePullArgs, args)

	reader := bytes.NewBufferString("image pulled")
	return io.NopCloser(reader), t.imagePullReturnErr
}

func (t *TestClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	args := make([]any, 2)
	args[0] = ctx
	args[1] = containerID
	t.containerInspectArgs = append(t.containerInspectArgs, args)

	next := container.InspectResponse{ContainerJSONBase: &container.ContainerJSONBase{}}
	if len(t.containerInspectReturn) > 0 {
		next = t.containerInspectReturn[0]
		t.containerInspectReturn = t.containerInspectReturn[1:]
	}

	return next, t.containerInspectErr
}

func (t *TestClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	summary := make([]image.Summary, 0)
	return summary, nil
}
