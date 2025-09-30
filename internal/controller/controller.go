package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gopkg.in/yaml.v3"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/generator"
	"github.com/home-operations/gatus-sidecar/internal/handler"
)

// Controller is a generic Kubernetes resource controller
type Controller struct {
	gvr     schema.GroupVersionResource
	handler handler.ResourceHandler
	convert func(*unstructured.Unstructured) (metav1.Object, error)
}

// NewIngressController creates a controller for Ingress resources
func NewIngressController(resourceHandler handler.ResourceHandler) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		handler: resourceHandler,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			ingress := &networkingv1.Ingress{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ingress); err != nil {
				return nil, fmt.Errorf("failed to convert to Ingress: %w", err)
			}
			return ingress, nil
		},
	}
}

// NewHTTPRouteController creates a controller for HTTPRoute resources
func NewHTTPRouteController(resourceHandler handler.ResourceHandler) *Controller {
	return &Controller{
		gvr:     gatewayv1.SchemeGroupVersion.WithResource("httproutes"),
		handler: resourceHandler,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			route := &gatewayv1.HTTPRoute{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, route); err != nil {
				return nil, fmt.Errorf("failed to convert to HTTPRoute: %w", err)
			}
			return route, nil
		},
	}
}

// Run starts the controller watch loop
func (c *Controller) Run(ctx context.Context, cfg *config.Config) error {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("get in-cluster config: %w", err)
	}

	dc, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("create dynamic client: %w", err)
	}

	for {
		if err := c.watchLoop(ctx, cfg, dc); err != nil {
			slog.Error("watch loop error", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *Controller) watchLoop(ctx context.Context, cfg *config.Config, dc dynamic.Interface) error {
	options := metav1.ListOptions{}

	w, err := dc.Resource(c.gvr).Namespace(cfg.Namespace).Watch(ctx, options)
	if err != nil {
		return fmt.Errorf("watch %s: %w", c.gvr.Resource, err)
	}
	defer w.Stop()

	ch := w.ResultChan()
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			// Convert unstructured object to typed object
			unstructuredObj, ok := evt.Object.(*unstructured.Unstructured)
			if !ok {
				slog.Error("unexpected object type", "type", fmt.Sprintf("%T", evt.Object))
				continue
			}

			obj, err := c.convert(unstructuredObj)
			if err != nil {
				slog.Error("failed to convert object", "error", err)
				continue
			}

			c.handleEvent(cfg, obj, evt.Type)
		}
	}
}

func (c *Controller) handleEvent(cfg *config.Config, obj metav1.Object, eventType watch.EventType) {
	// Skip if resource doesn't match our filter criteria
	if !c.handler.ShouldProcess(obj, cfg) {
		return
	}

	name := obj.GetName()
	filename := fmt.Sprintf("%s-%s.yaml", obj.GetName(), obj.GetNamespace())

	if eventType == watch.Deleted {
		if err := generator.Delete(cfg.OutputDir, filename); err != nil {
			slog.Error("failed to delete file for resource", c.handler.GetResourceName(), obj.GetName(), "error", err)
		} else {
			slog.Info("deleted file for resource", c.handler.GetResourceName(), obj.GetName())
		}
		return
	}

	// Get the URL from the resource
	url := c.handler.ExtractURL(obj)
	if url == "" {
		slog.Warn("resource has no hosts/hostnames", c.handler.GetResourceName(), obj.GetName())
		return
	}

	interval := cfg.DefaultInterval.String()
	dnsResolver := cfg.DefaultDNSResolver
	condition := cfg.DefaultCondition

	// Check for template annotation
	var templateData map[string]any
	annotations := obj.GetAnnotations()
	if annotations != nil {
		if templateStr, ok := annotations[cfg.TemplateAnnotation]; ok && templateStr != "" {
			if err := yaml.Unmarshal([]byte(templateStr), &templateData); err != nil {
				slog.Error("failed to unmarshal template for resource", c.handler.GetResourceName(), obj.GetName(), "error", err)
				return
			}
		}
	}

	data := map[string]any{
		"name":       name,
		"url":        url,
		"interval":   interval,
		"client":     map[string]any{"dns-resolver": dnsResolver},
		"conditions": []string{condition},
	}

	// Write with optional template data
	if err := generator.Write(data, cfg.OutputDir, filename, templateData); err != nil {
		slog.Error("write file for resource", c.handler.GetResourceName(), obj.GetName(), "error", err)
	} else {
		slog.Info("wrote file for resource", c.handler.GetResourceName(), obj.GetName())
	}
}
