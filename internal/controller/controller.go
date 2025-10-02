package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gopkg.in/yaml.v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/handler"
	"github.com/home-operations/gatus-sidecar/internal/manager"
)

// Controller is a generic Kubernetes resource controller
type Controller struct {
	gvr          schema.GroupVersionResource
	options      metav1.ListOptions
	handler      handler.ResourceHandler
	convert      func(*unstructured.Unstructured) (metav1.Object, error)
	stateManager *manager.Manager
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
	w, err := dc.Resource(c.gvr).Namespace(cfg.Namespace).Watch(ctx, c.options)
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
	namespace := obj.GetNamespace()
	resource := c.gvr.Resource
	key := fmt.Sprintf("%s:%s:%s", name, namespace, resource)

	if eventType == watch.Deleted {
		changed := c.stateManager.Remove(key)
		if changed {
			slog.Info("removed endpoint from state", "resource", resource, "name", name, "namespace", namespace)
		}
		return
	}

	// Get the URL from the resource
	url := c.handler.ExtractURL(obj)
	if url == "" {
		slog.Warn("resource has no url", "resource", resource, "name", name, "namespace", namespace)
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
				slog.Error("failed to unmarshal template for resource", "resource", resource, "name", name, "namespace", namespace, "error", err)
				return
			}
		}
	}

	// Create endpoint state with defaults
	endpoint := &endpoint.Endpoint{
		Name:       name,
		URL:        url,
		Interval:   interval,
		Client:     map[string]any{"dns-resolver": dnsResolver},
		Conditions: []string{condition},
	}

	// Apply resource-specific template if available
	c.handler.ApplyTemplate(cfg, obj, endpoint)

	// Apply template overrides if present
	if templateData != nil {
		endpoint.ApplyTemplate(templateData)
	}

	// Update state
	changed := c.stateManager.AddOrUpdate(key, endpoint)
	if changed {
		slog.Info("updated endpoint in state", "resource", resource, "name", name, "namespace", namespace)
	}
}
