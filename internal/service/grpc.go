package service

import (
	"net"
	"os"

	applicationv1 "github.com/mabaitar/gco/agent/gen/proto/application/v1"
	"github.com/mabaitar/gco/agent/internal/config"
	"github.com/mabaitar/gco/agent/internal/log"
	"github.com/mabaitar/gco/agent/internal/service/application"
	"github.com/mabaitar/gco/agent/pkg/control"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartGRPC registers the known services and starts listening using the configured address.
func StartGRPC(conf config.Grpc, controller *control.StateController) {
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

	// start listening for gRPC connections
	log.Infof("Started listening for gRPC connections on '%s'", conf.GetNetworkAddress())
	err = server.Serve(lis)
	if err != nil {
		log.Errorf("failed to serve gRPC: %v", err)
		os.Exit(1)
	}
}
