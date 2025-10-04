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
	AutoHTTPRoute      bool
	AutoIngress        bool
	AutoService        bool
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
	flag.BoolVar(&cfg.AutoHTTPRoute, "auto-httproute", false, "Automatically create endpoints for HTTPRoutes")
	flag.BoolVar(&cfg.AutoIngress, "auto-ingress", false, "Automatically create endpoints for Ingresses")
	flag.BoolVar(&cfg.AutoService, "auto-service", false, "Automatically create endpoints for Services")
	flag.StringVar(&cfg.Output, "output", "/config/gatus-sidecar.yaml", "File to write generated YAML")
	flag.DurationVar(&cfg.DefaultInterval, "default-interval", time.Minute, "Default interval value for endpoints")
	flag.StringVar(&cfg.TemplateAnnotation, "annotation-config", "gatus.home-operations.com/endpoint", "Annotation key for YAML config override")
	flag.StringVar(&cfg.EnabledAnnotation, "annotation-enabled", "gatus.home-operations.com/enabled", "Annotation key for enabling/disabling resource processing")
	flag.Parse()
	return cfg
}
