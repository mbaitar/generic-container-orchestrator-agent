package service

import (
	"context"
	"net"
	"os"

	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/mbaitar/gco/agent/internal/config"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/service/application"
	"github.com/mbaitar/gco/agent/pkg/control"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartGRPC registers the known services and starts listening using the configured address.
// The server is gracefully stopped when the given context is cancelled.
func StartGRPC(ctx context.Context, conf config.Grpc, controller *control.StateController) {
	if !conf.Enabled {
		log.Debug("gRPC server has not been enabled")
		return
	}

	// bind network address
	lis, err := net.Listen("tcp", conf.GetNetworkAddress())
	if err != nil {
		log.Errorf("failed to listen: %v", err)
		os.Exit(1)
	}

	// create gRPC server
	server := grpc.NewServer()
	applicationv1.RegisterApplicationServiceServer(server, application.NewServer(controller))

	if conf.EnableReflection {
		log.Debug("gRPC reflection mode has been enabled")
		reflection.Register(server)
	}

	// stop the server gracefully when the context is cancelled
	go func() {
		<-ctx.Done()
		log.Debug("Gracefully stopping gRPC server")
		server.GracefulStop()
	}()

	// start listening for gRPC connections
	log.Infof("Started listening for gRPC connections on '%s'", conf.GetNetworkAddress())
	err = server.Serve(lis)
	if err != nil {
		log.Errorf("failed to serve gRPC: %v", err)
		os.Exit(1)
	}

	log.Info("gRPC server has been stopped")
}
