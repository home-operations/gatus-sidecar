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
	"github.com/home-operations/gatus-sidecar/internal/manager"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a single shared state manager
	stateManager := manager.NewManager(cfg.Output)

	// Create all controllers
	controllers := []*controller.Controller{
		controller.NewHTTPRouteController(&controller.HTTPRouteHandler{}, stateManager),
		controller.NewIngressController(&controller.IngressHandler{}, stateManager),
		controller.NewServiceController(&controller.ServiceHandler{}, stateManager),
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
		wg.Add(1)
		go func(ctrl *controller.Controller) {
			defer wg.Done()
			slog.Info("Starting controller")

			if err := ctrl.Run(ctx, cfg); err != nil {
				slog.Error("Controller error", "error", err)
				errChan <- err
			}
		}(c)
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
