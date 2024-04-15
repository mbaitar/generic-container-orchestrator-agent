package main

import (
	"os"

	"github.com/mbaitar/gco/agent/internal/config"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/provider"
	"github.com/mbaitar/gco/agent/internal/provider/docker"
	"github.com/mbaitar/gco/agent/internal/service"
	"github.com/mbaitar/gco/agent/pkg/control"
)

func createProvider(conf *config.Config) provider.Provider {
	if conf.Docker.Enabled {
		return docker.NewDockerProvider().WithConfig(conf.Docker)
	}

	log.Errorf("No provider has been enabled, please check your configuration")
	os.Exit(1)
	return nil // should not be reached
}

func createStateController(p provider.Provider) *control.StateController {
	ctrl, err := control.InitControl(p)
	if err != nil {
		log.Errorf("failed to initialize control: %v", err)
		os.Exit(1)
	} else {
		log.Info("Successfully initialized control loop using docker provider")
	}

	go ctrl.Start()

	state := control.NewStateController(ctrl)
	return state
}

func main() {
	// load config
	conf := config.DefaultConfig()
	conf.SetFlags()

	// setup application
	prov := createProvider(conf)
	controller := createStateController(prov)

	// start gRPC server
	go service.StartGRPC(conf.Grpc, controller)

	// start HTTP server
	service.StartHTTP(conf.Http, controller)
}
