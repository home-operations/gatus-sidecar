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
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	restCfg, err := getKubeConfig()
	if err != nil {
		slog.Error("get kubernetes config", "error", err)
		os.Exit(1)
	}

	dc, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		slog.Error("create dynamic client", "error", err)
		os.Exit(1)
	}

	// Create a single shared state manager
	stateManager := state.NewManager(cfg.Output)

	// Initialize controllers slice
	controllers := []*controller.Controller{}

	// Determine if default controllers should be enabled
	defaultControllers := !cfg.EnableHTTPRoute && !cfg.EnableIngress && !cfg.EnableService

	// Conditionally register controllers based on config
	if cfg.EnableHTTPRoute || cfg.AutoHTTPRoute || defaultControllers {
		controllers = append(controllers, controller.New(httproute.Definition(), stateManager, dc))
	}
	if cfg.EnableIngress || cfg.AutoIngress || defaultControllers {
		controllers = append(controllers, controller.New(ingress.Definition(), stateManager, dc))
	}
	if cfg.EnableService || cfg.AutoService || defaultControllers {
		controllers = append(controllers, controller.New(service.Definition(), stateManager, dc))
	}

	// If no controllers are enabled, log a warning and exit
	if len(controllers) == 0 {
		slog.Warn("No controllers enabled. Exiting.")
		return
	}

	// Run all controllers concurrently
	if err := runControllers(ctx, cfg, controllers); err != nil {
		slog.Error("Controller execution failed", "error", err)
		os.Exit(1)
	}

	slog.Info("All controllers have finished successfully")
}

func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	if cfg, err := rest.InClusterConfig(); err == nil {
		slog.Info("using in-cluster kubernetes config")
		return cfg, nil
	}

	// Fall back to kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	slog.Info("using kubeconfig")
	return cfg, nil
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
	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
