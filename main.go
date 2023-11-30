package main

import (
	"dsync.io/gco/agent/internal/config"
	"dsync.io/gco/agent/internal/log"
	"dsync.io/gco/agent/internal/provider"
	"dsync.io/gco/agent/internal/provider/docker"
	"dsync.io/gco/agent/internal/service"
	"dsync.io/gco/agent/pkg/control"
	"os"
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
