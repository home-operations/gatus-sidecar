package config

import (
	"bytes"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load("gatus-sidecar", nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Output != DefaultOutputPath {
		t.Errorf("Output = %q, want %q", cfg.Output, DefaultOutputPath)
	}
	if cfg.DefaultInterval != DefaultInterval {
		t.Errorf("DefaultInterval = %v, want %v", cfg.DefaultInterval, DefaultInterval)
	}
	if cfg.TemplateAnnotation != DefaultTemplateAnnotation {
		t.Errorf("TemplateAnnotation = %q, want %q", cfg.TemplateAnnotation, DefaultTemplateAnnotation)
	}
	if cfg.AnyExplicitlyEnabled() {
		t.Errorf("AnyExplicitlyEnabled() should be false with default flags")
	}
}

func TestLoad_AllFlags(t *testing.T) {
	t.Parallel()
	args := []string{
		"--namespace=ns",
		"--gateway-name=gw1",
		"--gateway-name=gw2",
		"--ingress-class=nginx",
		"--ingress-class=traefik",
		"--enable-httproute=true",
		"--auto-ingress=true",
		"--output=/tmp/foo.yaml",
		"--default-interval=30s",
		"--annotation-config=k1",
		"--annotation-enabled=k2",
	}
	cfg, err := Load("test", args, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Namespace != "ns" ||
		!reflect.DeepEqual([]string(cfg.GatewayNames), []string{"gw1", "gw2"}) ||
		!reflect.DeepEqual([]string(cfg.IngressClasses), []string{"nginx", "traefik"}) {
		t.Errorf("filter flags incorrect: %+v", cfg)
	}
	if !cfg.Kinds[KindHTTPRoute].Enable || !cfg.Kinds[KindIngress].Auto {
		t.Errorf("enable flags incorrect: %+v", cfg)
	}
	if cfg.Output != "/tmp/foo.yaml" {
		t.Errorf("Output = %q", cfg.Output)
	}
	if cfg.DefaultInterval != 30*time.Second {
		t.Errorf("DefaultInterval = %v", cfg.DefaultInterval)
	}
	if cfg.TemplateAnnotation != "k1" || cfg.EnabledAnnotation != "k2" {
		t.Errorf("annotation flags incorrect: %+v", cfg)
	}
	if !cfg.AnyExplicitlyEnabled() {
		t.Errorf("AnyExplicitlyEnabled() should be true with --enable-httproute and --auto-ingress")
	}
}

func TestLoad_RejectsBadValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"empty output", []string{"--output="}},
		{"zero interval", []string{"--default-interval=0s"}},
		{"unknown flag", []string{"--nope"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load("test", tt.args, &bytes.Buffer{})
			if err == nil {
				t.Errorf("expected error for args %v", tt.args)
			}
		})
	}
}

func TestLoad_LogLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		arg     string
		want    slog.Level
		wantErr bool
	}{
		{"", slog.LevelInfo, false}, // default
		{"debug", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"Warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"trace", 0, true},
	}
	for _, tt := range cases {
		t.Run(tt.arg, func(t *testing.T) {
			t.Parallel()
			args := []string{}
			if tt.arg != "" {
				args = append(args, "--log-level="+tt.arg)
			}
			cfg, err := Load("test", args, &bytes.Buffer{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if err == nil && cfg.LogLevel != tt.want {
				t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, tt.want)
			}
		})
	}
}

func TestAnyExplicitlyEnabled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"none", Config{}, false},
		{"enable-ingress", Config{Kinds: map[string]*KindConfig{KindIngress: {Enable: true}}}, true},
		{"auto-service", Config{Kinds: map[string]*KindConfig{KindService: {Auto: true}}}, true},
		{"auto-ingressroute", Config{Kinds: map[string]*KindConfig{KindIngressRoute: {Auto: true}}}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.AnyExplicitlyEnabled(); got != tt.want {
				t.Errorf("AnyExplicitlyEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
