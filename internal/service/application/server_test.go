package application

import (
	"context"
	"testing"

	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// the validation paths return before the state controller is touched, so a
// nil controller is sufficient for these tests.

func TestServer_CreateApplication_requiresApplication(t *testing.T) {
	server := NewServer(nil)

	_, err := server.CreateApplication(context.Background(), &applicationv1.CreateApplicationRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a missing application")
}

func TestServer_CreateApplication_rejectsInvalidApplication(t *testing.T) {
	server := NewServer(nil)

	req := &applicationv1.CreateApplicationRequest{
		Application: &applicationv1.Application{
			Name:          "app",
			Image:         &applicationv1.Image{Name: "nginx", Tag: "latest"},
			RestartPolicy: "forever",
		},
	}

	_, err := server.CreateApplication(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected an invalid restart policy")
}

func TestServer_CreateApplication_rejectsReservedLabel(t *testing.T) {
	server := NewServer(nil)

	req := &applicationv1.CreateApplicationRequest{
		Application: &applicationv1.Application{
			Name:   "app",
			Image:  &applicationv1.Image{Name: "nginx", Tag: "latest"},
			Labels: map[string]string{"gco.io/name": "hijack"},
		},
	}

	_, err := server.CreateApplication(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a reserved label")
}

func TestServer_UpdateApplication_rejectsInvalidApplication(t *testing.T) {
	server := NewServer(nil)

	req := &applicationv1.UpdateApplicationRequest{
		Application: &applicationv1.Application{
			Name:      "app",
			Image:     &applicationv1.Image{Name: "nginx", Tag: "latest"},
			Resources: &applicationv1.Resources{Memory: "4m"},
		},
	}

	_, err := server.UpdateApplication(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a memory limit below the minimum")
}

func TestServer_UpdateApplication_requiresApplication(t *testing.T) {
	server := NewServer(nil)

	_, err := server.UpdateApplication(context.Background(), &applicationv1.UpdateApplicationRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a missing application")
}

func TestServer_GetApplication_requiresName(t *testing.T) {
	server := NewServer(nil)

	_, err := server.GetApplication(context.Background(), &applicationv1.GetApplicationRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a missing name")
}

func TestServer_DeleteApplication_requiresName(t *testing.T) {
	server := NewServer(nil)

	_, err := server.DeleteApplication(context.Background(), &applicationv1.DeleteApplicationRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "should have rejected a missing name")
}
