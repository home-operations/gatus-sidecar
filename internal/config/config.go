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

// Kind identifiers — the canonical set of watchable resource kinds. The values
// double as the suffix of the per-kind flags (e.g. KindIngress → --enable-ingress).
const (
	KindIngress      = "ingress"
	KindHTTPRoute    = "httproute"
	KindService      = "service"
	KindIngressRoute = "ingressroute"
)

// kindMeta drives per-kind flag registration and help text.
var kindMeta = []struct {
	name    string
	display string
	plural  string
}{
	{KindIngress, "Ingress", "Ingresses"},
	{KindHTTPRoute, "HTTPRoute", "HTTPRoutes"},
	{KindService, "Service", "Services"},
	{KindIngressRoute, "Traefik IngressRoute", "Traefik IngressRoutes"},
}

// KindConfig holds the per-kind flag values.
type KindConfig struct {
	Enable bool
	Auto   bool
	Prefix string
}

type Config struct {
	Namespace      string
	GatewayNames   StringSet
	IngressClasses StringSet

	Kinds map[string]*KindConfig

	Output          string
	DefaultInterval time.Duration
	ProbePaths      bool

	TemplateAnnotation string
	EnabledAnnotation  string

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

	cfg.Kinds = make(map[string]*KindConfig, len(kindMeta))
	for _, k := range kindMeta {
		kc := &KindConfig{}
		cfg.Kinds[k.name] = kc
		fs.BoolVar(&kc.Enable, "enable-"+k.name, false, fmt.Sprintf("Enable %s endpoint generation", k.display))
		fs.BoolVar(&kc.Auto, "auto-"+k.name, false, fmt.Sprintf("Automatically create endpoints for %s", k.plural))
		fs.StringVar(&kc.Prefix, "prefix-"+k.name, "", fmt.Sprintf("Prefix prepended to generated endpoint names for %s resources", k.display))
	}

	fs.StringVar(&cfg.Output, "output", DefaultOutputPath, "File to write generated YAML")
	fs.DurationVar(&cfg.DefaultInterval, "default-interval", DefaultInterval, "Default interval value for endpoints")
	fs.BoolVar(&cfg.ProbePaths, "probe-paths", true, "Include paths from Ingress/HTTPRoute/IngressRoute match rules in probe URLs; set false to probe bare hostnames")
	fs.StringVar(&cfg.TemplateAnnotation, "annotation-config", DefaultTemplateAnnotation, "Annotation key for YAML config override")
	fs.StringVar(&cfg.EnabledAnnotation, "annotation-enabled", DefaultEnabledAnnotation, "Annotation key for enabling/disabling resource processing")

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
	for _, k := range c.Kinds {
		if k.Enable || k.Auto {
			return true
		}
	}
	return false
}

// KindEnabled reports whether the named kind is enabled by either its
// --enable-<kind> or --auto-<kind> flag.
func (c *Config) KindEnabled(name string) bool {
	k := c.Kinds[name]
	return k != nil && (k.Enable || k.Auto)
}

// AutoEnabled reports whether the named kind is in auto-discovery mode.
func (c *Config) AutoEnabled(name string) bool {
	k := c.Kinds[name]
	return k != nil && k.Auto
}

// Prefix returns the endpoint-name prefix configured for the named kind.
func (c *Config) Prefix(name string) string {
	if k := c.Kinds[name]; k != nil {
		return k.Prefix
	}
	return ""
}
