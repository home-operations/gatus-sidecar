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
	AutoRoutes         bool
	AutoIngresses      bool
	AutoServices       bool
	AutoGroup          bool
	Output             string
	DefaultInterval    time.Duration
	DefaultDNSResolver string
	DefaultCondition   string
	TemplateAnnotation string
}

func Load() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Namespace, "namespace", "", "Namespace to watch (empty for all)")
	flag.StringVar(&cfg.GatewayName, "gateway-name", "", "Gateway name to filter HTTPRoutes (optional)")
	flag.StringVar(&cfg.IngressClass, "ingress-class", "", "Ingress class to filter Ingresses (optional)")
	flag.BoolVar(&cfg.AutoRoutes, "auto-routes", false, "Automatically create endpoints for HTTPRoutes")
	flag.BoolVar(&cfg.AutoIngresses, "auto-ingresses", false, "Automatically create endpoints for Ingresses")
	flag.BoolVar(&cfg.AutoServices, "auto-services", false, "Automatically create endpoints for Services")
	flag.BoolVar(&cfg.AutoGroup, "auto-group", false, "Automatically group endpoints by namespace (for Services) or gateway/ingress class (for HTTPRoutes/Ingresses)")
	flag.StringVar(&cfg.Output, "output", "/config/gatus-sidecar.yaml", "File to write generated YAML")
	flag.DurationVar(&cfg.DefaultInterval, "default-interval", time.Minute, "Default interval value for endpoints")
	flag.StringVar(&cfg.DefaultDNSResolver, "default-dns", "tcp://1.1.1.1:53", "Default DNS resolver for endpoints")
	flag.StringVar(&cfg.DefaultCondition, "default-condition", "[STATUS] == 200", "Default condition")
	flag.StringVar(&cfg.TemplateAnnotation, "annotation-config", "gatus.home-operations.com/endpoint", "Annotation key for YAML config override")
	flag.Parse()
	return cfg
}
