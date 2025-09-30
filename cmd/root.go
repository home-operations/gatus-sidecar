package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/controller"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var err error
	switch cfg.Mode {
	case "httproute":
		err = controller.RunHTTPRoute(ctx, cfg)
	case "ingress":
		err = controller.RunIngress(ctx, cfg)
	default:
		slog.Error("invalid mode", "mode", cfg.Mode, "valid_modes", []string{"httproute", "ingress"})
		os.Exit(1)
	}

	if err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}
