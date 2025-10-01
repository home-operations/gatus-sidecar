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
	AutoGroup          bool
	Output             string
	DefaultInterval    time.Duration
	DefaultDNSResolver string
	DefaultCondition   string
	TemplateAnnotation string
}

func Load() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Mode, "mode", "httproute", "Mode to run in: 'httproute' or 'ingress'")
	flag.StringVar(&cfg.Namespace, "namespace", "", "Namespace to watch (empty for all)")
	flag.StringVar(&cfg.GatewayName, "gateway-name", "", "Gateway name to filter HTTPRoutes (optional)")
	flag.StringVar(&cfg.IngressClass, "ingress-class", "", "Ingress class to filter Ingresses (optional)")
	flag.BoolVar(&cfg.AutoGroup, "auto-group", false, "Automatically group endpoints by gateway name or ingress class")
	flag.StringVar(&cfg.Output, "output", "/config/gatus-sidecar.yaml", "File to write generated YAML")
	flag.DurationVar(&cfg.DefaultInterval, "default-interval", time.Minute, "Default interval value for endpoints")
	flag.StringVar(&cfg.DefaultDNSResolver, "default-dns", "tcp://1.1.1.1:53", "Default DNS resolver for endpoints")
	flag.StringVar(&cfg.DefaultCondition, "default-condition", "[STATUS] == 200", "Default condition")
	flag.StringVar(&cfg.TemplateAnnotation, "annotation-config", "gatus.home-operations.com/endpoint", "Annotation key for YAML config override")
	flag.Parse()
	return cfg
}
