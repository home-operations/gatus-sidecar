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

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/handler"
	"github.com/home-operations/gatus-sidecar/internal/manager"
)

// Controller is a generic Kubernetes resource controller
type Controller struct {
	gvr           schema.GroupVersionResource
	options       metav1.ListOptions
	handler       handler.ResourceHandler
	convert       func(*unstructured.Unstructured) (metav1.Object, error)
	stateManager  *manager.Manager
	dynamicClient dynamic.Interface
}

// Run starts the controller watch loop
func (c *Controller) Run(ctx context.Context, cfg *config.Config) error {
	for {
		if err := c.watchLoop(ctx, cfg); err != nil {
			slog.Error("watch loop error", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *Controller) GetResource() string {
	return c.gvr.Resource
}

func (c *Controller) watchLoop(ctx context.Context, cfg *config.Config) error {
	w, err := c.dynamicClient.Resource(c.gvr).Namespace(cfg.Namespace).Watch(ctx, c.options)
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

			c.handleEvent(ctx, cfg, obj, evt.Type)
		}
	}
}

func (c *Controller) handleEvent(ctx context.Context, cfg *config.Config, obj metav1.Object, eventType watch.EventType) {
	name := obj.GetName()
	namespace := obj.GetNamespace()
	annotations := obj.GetAnnotations()
	resource := c.gvr.Resource

	key := fmt.Sprintf("%s:%s:%s", name, namespace, resource)

	// If the resource should not be processed or has been deleted, remove it from state
	if !c.handler.ShouldProcess(obj, cfg) || eventType == watch.Deleted {
		removed := c.stateManager.Remove(key)
		if removed {
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

	// Check for enabled annotation and template annotation
	if enabledValue, ok := annotations[cfg.EnabledAnnotation]; ok && enabledValue != "true" && enabledValue != "1" {
		removed := c.stateManager.Remove(key)
		if removed {
			slog.Info("removed endpoint from state", "resource", resource, "name", name, "namespace", namespace)
		}
		return
	}

	// Get parent annotations (e.g. Gateways can provide annotations for HTTPRoutes), then merge in object annotations.
	parentAnnotations := c.handler.GetParentAnnotations(ctx, obj)
	if parentAnnotations == nil {
		parentAnnotations = make(map[string]string)
	}

	var templateData map[string]any

	// Parse parent template data first
	parentTemplateData, err := c.parseTemplateData(parentAnnotations, cfg.TemplateAnnotation)
	if err != nil {
		slog.Error("failed to unmarshal parent template for resource", "resource", resource, "name", name, "namespace", namespace, "error", err)
		return
	}

	// Parse object template data
	objectTemplateData, err := c.parseTemplateData(obj.GetAnnotations(), cfg.TemplateAnnotation)
	if err != nil {
		slog.Error("failed to unmarshal object template for resource", "resource", resource, "name", name, "namespace", namespace, "error", err)
		return
	}

	// Deep merge parent and object template data
	templateData = c.deepMergeTemplates(parentTemplateData, objectTemplateData)

	// Internal home-ops opinionated "guarded" endpoint feature
	guarded := templateData != nil && templateData["guarded"] != nil

	// Create endpoint state with defaults
	endpoint := &endpoint.Endpoint{
		Name:     name,
		URL:      url,
		Interval: cfg.DefaultInterval.String(),
		Guarded:  guarded,
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

// parseTemplateData extracts and parses template data from annotations
func (c *Controller) parseTemplateData(annotations map[string]string, annotationKey string) (map[string]any, error) {
	templateStr, ok := annotations[annotationKey]
	if !ok || templateStr == "" {
		return nil, nil
	}

	var templateData map[string]any
	if err := yaml.Unmarshal([]byte(templateStr), &templateData); err != nil {
		return nil, err
	}
	return templateData, nil
}

// deepMergeTemplates recursively merges two template data maps, with the second map taking precedence
func (c *Controller) deepMergeTemplates(parent, child map[string]any) map[string]any {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	result := make(map[string]any)

	// Copy all parent values first
	for key, value := range parent {
		result[key] = value
	}

	// Merge child values, recursively merging maps
	for key, childValue := range child {
		if parentValue, exists := result[key]; exists {
			// If both values are maps, recursively merge them
			if parentMap, parentIsMap := parentValue.(map[string]any); parentIsMap {
				if childMap, childIsMap := childValue.(map[string]any); childIsMap {
					result[key] = c.deepMergeTemplates(parentMap, childMap)
					continue
				}
			}
		}
		// Otherwise, child value overwrites parent value
		result[key] = childValue
	}

	return result
}
