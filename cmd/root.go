package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/controller"
	"github.com/home-operations/gatus-sidecar/internal/resources/httproute"
	"github.com/home-operations/gatus-sidecar/internal/resources/ingress"
	"github.com/home-operations/gatus-sidecar/internal/resources/service"
	"github.com/home-operations/gatus-sidecar/internal/state"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a single shared state manager
	stateManager := state.NewManager(cfg.Output)

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("get in-cluster config", "error", err)
		os.Exit(1)
	}

	dc, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		slog.Error("create dynamic client", "error", err)
		os.Exit(1)
	}

	// Register default resource types
	controllers := []*controller.Controller{
		controller.New(ingress.Definition(), stateManager, dc),
		controller.New(httproute.Definition(), stateManager, dc),
		controller.New(service.Definition(), stateManager, dc),
	}

	// Run all controllers concurrently
	if err := runControllers(ctx, cfg, controllers); err != nil {
		slog.Error("Controller execution failed", "error", err)
		os.Exit(1)
	}

	slog.Info("All controllers have finished successfully")
}

func runControllers(ctx context.Context, cfg *config.Config, controllers []*controller.Controller) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(controllers))

	for _, c := range controllers {
		wg.Go(func() {
			slog.Info("Starting controller", "controller", c.GetResource())

			if err := c.Run(ctx, cfg); err != nil {
				slog.Error("Controller error", "controller", c.GetResource(), "error", err)
				errChan <- err
			}
		})
	}

	// Wait for either all controllers to finish or an error to occur
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Return the first error encountered
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
