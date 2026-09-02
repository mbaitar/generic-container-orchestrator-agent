package application

import (
	"context"
	"errors"

	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/control"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapStateError translates state controller errors into gRPC status errors,
// persistence failures should surface as internal errors instead of the
// conflict and not-found codes.
func mapStateError(err error) error {
	switch {
	case errors.Is(err, state.ErrApplicationExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, state.ErrApplicationNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

type Server struct {
	state *control.StateController

	applicationv1.UnimplementedApplicationServiceServer
}

func NewServer(state *control.StateController) *Server {
	return &Server{
		state: state,
	}
}

func (s *Server) GetApplication(ctx context.Context, req *applicationv1.GetApplicationRequest) (*applicationv1.GetApplicationResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name required")
	}

	state := s.state.GetCurrentState()
	app := state.GetApplication(req.Name)
	if app == nil {
		return nil, status.Errorf(codes.NotFound, "application '%s' not found", req.Name)
	}

	return &applicationv1.GetApplicationResponse{
		Application: app.ToApplicationV1(),
	}, nil
}

func (s *Server) CreateApplication(ctx context.Context, req *applicationv1.CreateApplicationRequest) (*applicationv1.CreateApplicationResponse, error) {
	app := resource.FromApplicationV1(req.Application)
	if app == nil {
		return nil, status.Error(codes.InvalidArgument, "requires application argument")
	}

	if err := app.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, err := s.state.CreateApplication(*app)
	if err != nil {
		return nil, mapStateError(err)
	}

	return &applicationv1.CreateApplicationResponse{}, nil
}

func (s *Server) UpdateApplication(ctx context.Context, req *applicationv1.UpdateApplicationRequest) (*applicationv1.UpdateApplicationResponse, error) {
	app := resource.FromApplicationV1(req.Application)
	if app == nil {
		return nil, status.Error(codes.InvalidArgument, "requires application argument")
	}

	if err := app.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, err := s.state.UpdateApplication(*app)
	if err != nil {
		return nil, mapStateError(err)
	}

	return &applicationv1.UpdateApplicationResponse{}, nil
}

func (s *Server) DeleteApplication(ctx context.Context, req *applicationv1.DeleteApplicationRequest) (*applicationv1.DeleteApplicationResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name required")
	}

	_, err := s.state.DeleteApplication(req.Name)
	if err != nil {
		return nil, mapStateError(err)
	}

	return &applicationv1.DeleteApplicationResponse{}, nil
}

func (s *Server) ListApplications(ctx context.Context, req *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
	state := s.state.GetCurrentState()
	apps := make([]*applicationv1.Application, 0)

	for _, app := range state.Applications {
		apps = append(apps, app.ToApplicationV1())
	}

	return &applicationv1.ListApplicationsResponse{
		Applications: apps,
	}, nil
}
