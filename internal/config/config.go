// Package config loads the gatus-sidecar runtime configuration from CLI flags.
package config

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

const (
	DefaultOutputPath         = "/config/gatus-sidecar.yaml"
	DefaultInterval           = time.Minute
	DefaultTemplateAnnotation = "gatus.home-operations.com/endpoint"
	DefaultEnabledAnnotation  = "gatus.home-operations.com/enabled"
	DefaultLogLevel           = "info"
)

type Config struct {
	Namespace      string
	GatewayNames   StringSet
	IngressClasses StringSet

	EnableHTTPRoute    bool
	EnableIngress      bool
	EnableService      bool
	EnableIngressRoute bool

	AutoHTTPRoute    bool
	AutoIngress      bool
	AutoService      bool
	AutoIngressRoute bool

	Output          string
	DefaultInterval time.Duration
	ProbePaths      bool

	TemplateAnnotation string
	EnabledAnnotation  string

	IngressPrefix      string
	ServicePrefix      string
	HTTPRoutePrefix    string
	IngressRoutePrefix string

	LogLevel slog.Level
}

// Load parses args (without the program name) into a Config.
func Load(name string, args []string, errOut io.Writer) (*Config, error) {
	cfg := &Config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)

	fs.StringVar(&cfg.Namespace, "namespace", "", "Namespace to watch (empty for all namespaces)")
	fs.Var(&cfg.GatewayNames, "gateway-name", "Gateway name(s) to filter HTTPRoutes; may be repeated")
	fs.Var(&cfg.IngressClasses, "ingress-class", "Ingress class(es) to filter Ingresses; may be repeated")

	fs.BoolVar(&cfg.EnableHTTPRoute, "enable-httproute", false, "Enable HTTPRoute endpoint generation")
	fs.BoolVar(&cfg.EnableIngress, "enable-ingress", false, "Enable Ingress endpoint generation")
	fs.BoolVar(&cfg.EnableService, "enable-service", false, "Enable Service endpoint generation")
	fs.BoolVar(&cfg.EnableIngressRoute, "enable-ingressroute", false, "Enable Traefik IngressRoute endpoint generation")

	fs.BoolVar(&cfg.AutoHTTPRoute, "auto-httproute", false, "Automatically create endpoints for HTTPRoutes")
	fs.BoolVar(&cfg.AutoIngress, "auto-ingress", false, "Automatically create endpoints for Ingresses")
	fs.BoolVar(&cfg.AutoService, "auto-service", false, "Automatically create endpoints for Services")
	fs.BoolVar(&cfg.AutoIngressRoute, "auto-ingressroute", false, "Automatically create endpoints for Traefik IngressRoutes")

	fs.StringVar(&cfg.Output, "output", DefaultOutputPath, "File to write generated YAML")
	fs.DurationVar(&cfg.DefaultInterval, "default-interval", DefaultInterval, "Default interval value for endpoints")
	fs.BoolVar(&cfg.ProbePaths, "probe-paths", true, "Include paths from Ingress/HTTPRoute/IngressRoute match rules in probe URLs; set false to probe bare hostnames")
	fs.StringVar(&cfg.TemplateAnnotation, "annotation-config", DefaultTemplateAnnotation, "Annotation key for YAML config override")
	fs.StringVar(&cfg.EnabledAnnotation, "annotation-enabled", DefaultEnabledAnnotation, "Annotation key for enabling/disabling resource processing")

	fs.StringVar(&cfg.IngressPrefix, "prefix-ingress", "", "Prefix prepended to generated endpoint names for Ingress resources")
	fs.StringVar(&cfg.ServicePrefix, "prefix-service", "", "Prefix prepended to generated endpoint names for Service resources")
	fs.StringVar(&cfg.HTTPRoutePrefix, "prefix-httproute", "", "Prefix prepended to generated endpoint names for HTTPRoute resources")
	fs.StringVar(&cfg.IngressRoutePrefix, "prefix-ingressroute", "", "Prefix prepended to generated endpoint names for Traefik IngressRoute resources")

	logLevel := fs.String("log-level", DefaultLogLevel, "Log level: debug, info, warn, error")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.Output == "" {
		return nil, fmt.Errorf("--output must not be empty")
	}
	if cfg.DefaultInterval <= 0 {
		return nil, fmt.Errorf("--default-interval must be positive (got %s)", cfg.DefaultInterval)
	}
	lvl, err := parseLogLevel(*logLevel)
	if err != nil {
		return nil, err
	}
	cfg.LogLevel = lvl

	return cfg, nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("--log-level must be one of debug|info|warn|error (got %q)", s)
	}
}

// AnyExplicitlyEnabled reports whether any --enable-* or --auto-* flag is set.
func (c *Config) AnyExplicitlyEnabled() bool {
	return c.EnableHTTPRoute || c.EnableIngress || c.EnableService || c.EnableIngressRoute ||
		c.AutoHTTPRoute || c.AutoIngress || c.AutoService || c.AutoIngressRoute
}
