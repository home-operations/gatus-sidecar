package config

import (
	"flag"
	"time"
)

type Config struct {
	Mode               string
	Namespace          string
	GatewayName        string
	IngressClass       string
	EnableHTTPRoute    bool
	EnableIngress      bool
	EnableService      bool
	EnableIngressRoute bool
	AutoHTTPRoute      bool
	AutoIngress        bool
	AutoService        bool
	AutoIngressRoute   bool
	Output             string
	DefaultInterval    time.Duration
	TemplateAnnotation string
	EnabledAnnotation  string
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Namespace, "namespace", "", "Namespace to watch (empty for all)")
	flag.StringVar(&cfg.GatewayName, "gateway-name", "", "Gateway name to filter HTTPRoutes (optional)")
	flag.StringVar(&cfg.IngressClass, "ingress-class", "", "Ingress class to filter Ingresses (optional)")
	flag.BoolVar(&cfg.EnableHTTPRoute, "enable-httproute", false, "Enable HTTPRoute endpoint generation")
	flag.BoolVar(&cfg.EnableIngress, "enable-ingress", false, "Enable Ingress endpoint generation")
	flag.BoolVar(&cfg.EnableService, "enable-service", false, "Enable Service endpoint generation")
	flag.BoolVar(&cfg.AutoHTTPRoute, "auto-httproute", false, "Automatically create endpoints for HTTPRoutes")
	flag.BoolVar(&cfg.AutoIngress, "auto-ingress", false, "Automatically create endpoints for Ingresses")
	flag.BoolVar(&cfg.AutoService, "auto-service", false, "Automatically create endpoints for Services")
	flag.BoolVar(&cfg.EnableIngressRoute, "enable-ingressroute", false, "Enable Traefik IngressRoute endpoint generation")
	flag.BoolVar(&cfg.AutoIngressRoute, "auto-ingressroute", false, "Automatically create endpoints for Traefik IngressRoutes")
	flag.StringVar(&cfg.Output, "output", "/config/gatus-sidecar.yaml", "File to write generated YAML")
	flag.DurationVar(&cfg.DefaultInterval, "default-interval", time.Minute, "Default interval value for endpoints")
	flag.StringVar(&cfg.TemplateAnnotation, "annotation-config", "gatus.home-operations.com/endpoint", "Annotation key for YAML config override")
	flag.StringVar(&cfg.EnabledAnnotation, "annotation-enabled", "gatus.home-operations.com/enabled", "Annotation key for enabling/disabling resource processing")

	flag.CommandLine.Init("", flag.ExitOnError)
	flag.Parse()

	return cfg
}
