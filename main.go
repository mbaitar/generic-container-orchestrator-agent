package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

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

func createStateController(ctx context.Context, p provider.Provider) *control.StateController {
	ctrl, err := control.InitControl(ctx, p)
	if err != nil {
		log.Errorf("failed to initialize control: %v", err)
		os.Exit(1)
	} else {
		log.Info("Successfully initialized control loop using docker provider")
	}

	go ctrl.Start(ctx)

	state := control.NewStateController(ctrl)
	return state
}

func main() {
	// load config
	conf := config.DefaultConfig()
	conf.SetFlags()

	// cancel the root context on SIGINT/SIGTERM for a graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// setup application
	prov := createProvider(conf)
	controller := createStateController(ctx, prov)

	// start gRPC and HTTP servers
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		service.StartGRPC(ctx, conf.Grpc, controller)
	}()

	go func() {
		defer wg.Done()
		service.StartHTTP(ctx, conf.Http, controller)
	}()

	// wait for a shutdown signal and let the servers drain
	<-ctx.Done()
	log.Info("Shutdown signal received, stopping servers")
	wg.Wait()
	log.Info("Agent has been stopped")
}
