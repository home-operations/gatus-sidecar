// gatus-sidecar generates Gatus monitoring configuration from Kubernetes
// resources (Ingress, Service, HTTPRoute, Traefik IngressRoute).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/gatus"
	"github.com/home-operations/gatus-sidecar/internal/k8s"
	"github.com/home-operations/gatus-sidecar/internal/resources"

	"k8s.io/client-go/dynamic"
)

var (
	Version = "local"
	Gitsha  = "?"
)

func main() {
	if err := run(os.Args[0], os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(name string, args []string) error {
	cfg, err := config.Load(name, args, os.Stderr)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel})))
	slog.Info("starting gatus-sidecar", "version", Version, "gitsha", Gitsha)

	enabled := resources.All(cfg)
	if len(enabled) == 0 {
		slog.Warn("no resource controllers enabled; exiting")
		return nil
	}

	restCfg, err := k8s.RestConfig()
	if err != nil {
		return err
	}
	dc, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	writer := gatus.NewWriter(cfg.Output)

	var wg sync.WaitGroup
	for _, r := range enabled {
		c := k8s.NewController(cfg, r, writer, dc)
		wg.Go(func() {
			if err := c.Run(ctx); err != nil {
				slog.Error("controller stopped", "resource", c.Resource(), "error", err)
				cancel()
			}
		})
	}
	wg.Wait()
	slog.Info("shutdown complete")
	return nil
}
