package controller

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"gopkg.in/yaml.v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/resources"
	"github.com/home-operations/gatus-sidecar/internal/state"
)

type Controller struct {
	gvr           schema.GroupVersionResource
	options       metav1.ListOptions
	handler       resources.ResourceHandler
	convert       func(*unstructured.Unstructured) (metav1.Object, error)
	stateManager  *state.Manager
	dynamicClient dynamic.Interface
}

// New creates a controller using a ResourceDefinition
func New(definition *resources.ResourceDefinition, stateManager *state.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr:           definition.GVR,
		options:       metav1.ListOptions{},
		handler:       resources.NewHandler(definition, dynamicClient),
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert:       definition.ConvertFunc,
	}
}

func (c *Controller) GetResource() string {
	return c.gvr.Resource
}

func (c *Controller) Run(ctx context.Context, cfg *config.Config) error {
	// Initial listing to populate state
	if err := c.initialList(ctx, cfg); err != nil {
		return fmt.Errorf("initial list failed for %s: %w", c.gvr.Resource, err)
	}

	// Watch loop
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

func (c *Controller) initialList(ctx context.Context, cfg *config.Config) error {
	list, err := c.dynamicClient.Resource(c.gvr).Namespace(cfg.Namespace).List(ctx, c.options)
	if err != nil {
		return fmt.Errorf("list %s: %w", c.gvr.Resource, err)
	}

	for i, item := range list.Items {
		obj, err := c.convert(&item)
		if err != nil {
			slog.Error("failed to convert resource", "resource", c.gvr.Resource, "error", err)
			continue
		}

		// Skip write for all but last to reduce I/O
		isNotLast := i != len(list.Items)-1
		c.handleEvent(ctx, cfg, obj, watch.Added, isNotLast)
	}

	return nil
}

func (c *Controller) watchLoop(ctx context.Context, cfg *config.Config) error {
	w, err := c.dynamicClient.Resource(c.gvr).Namespace(cfg.Namespace).Watch(ctx, c.options)
	if err != nil {
		return fmt.Errorf("watch %s: %w", c.gvr.Resource, err)
	}
	defer w.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-w.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			c.processEvent(ctx, cfg, evt)
		}
	}
}

func (c *Controller) processEvent(ctx context.Context, cfg *config.Config, evt watch.Event) {
	obj, err := c.convertEvent(evt)
	if err != nil {
		slog.Error("failed to process event", "error", err)
		return
	}

	c.handleEvent(ctx, cfg, obj, evt.Type, false)
}

func (c *Controller) convertEvent(evt watch.Event) (metav1.Object, error) {
	unstructuredObj, ok := evt.Object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", evt.Object)
	}

	return c.convert(unstructuredObj)
}

func (c *Controller) handleEvent(ctx context.Context, cfg *config.Config, obj metav1.Object, eventType watch.EventType, skipWrite bool) {
	name := obj.GetName()
	namespace := obj.GetNamespace()
	annotations := obj.GetAnnotations()
	resource := c.gvr.Resource
	key := fmt.Sprintf("%s.%s.%s", name, namespace, resource)

	// Early returns for deletion or non-processable resources
	if !c.handler.ShouldProcess(obj, cfg) || eventType == watch.Deleted {
		c.removeFromState(key, resource, name, namespace)
		return
	}

	// Validate URL availability
	url := c.handler.ExtractURL(obj)
	if url == "" {
		slog.Warn("resource has no url", "resource", resource, "name", name, "namespace", namespace)
		return
	}

	// Check if endpoint is explicitly disabled
	if c.isEndpointDisabled(annotations, cfg) {
		c.removeFromState(key, resource, name, namespace)
		return
	}

	// Process template data
	templateData, err := c.buildTemplateData(ctx, obj, cfg)
	if err != nil {
		slog.Error("failed to build template data", "resource", resource, "name", name, "namespace", namespace, "error", err)
		return
	}

	// Create and configure endpoint
	endpoint := &endpoint.Endpoint{
		Name:     name,
		URL:      url,
		Interval: cfg.DefaultInterval.String(),
		Guarded:  c.isGuardedEndpoint(templateData),
	}

	c.handler.ApplyTemplate(cfg, obj, endpoint)
	if templateData != nil {
		endpoint.ApplyTemplate(templateData)
	}

	// Update state
	if changed := c.stateManager.AddOrUpdate(key, endpoint, skipWrite); changed {
		slog.Info("updated endpoint in state", "resource", resource, "name", name, "namespace", namespace, "skipWrite", skipWrite)
	}
}

func (c *Controller) isEndpointDisabled(annotations map[string]string, cfg *config.Config) bool {
	enabledValue, ok := annotations[cfg.EnabledAnnotation]
	return ok && enabledValue != "true" && enabledValue != "1"
}

func (c *Controller) removeFromState(key, resource, name, namespace string) {
	if removed := c.stateManager.Remove(key); removed {
		slog.Info("removed endpoint from state", "resource", resource, "name", name, "namespace", namespace)
	}
}

func (c *Controller) isGuardedEndpoint(templateData map[string]any) bool {
	if templateData == nil {
		return false
	}
	_, exists := templateData["guarded"]
	return exists
}

func (c *Controller) buildTemplateData(ctx context.Context, obj metav1.Object, cfg *config.Config) (map[string]any, error) {
	annotations := obj.GetAnnotations()

	parentAnnotations := c.handler.GetParentAnnotations(ctx, obj)
	if parentAnnotations == nil {
		parentAnnotations = make(map[string]string)
	}

	parentTemplateData, err := c.parseTemplateData(parentAnnotations, cfg.TemplateAnnotation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent template: %w", err)
	}

	objectTemplateData, err := c.parseTemplateData(annotations, cfg.TemplateAnnotation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse object template: %w", err)
	}

	return c.deepMergeTemplates(parentTemplateData, objectTemplateData), nil
}

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

func (c *Controller) deepMergeTemplates(parent, child map[string]any) map[string]any {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	result := make(map[string]any)
	maps.Copy(result, parent)

	for key, childValue := range child {
		if parentValue, exists := result[key]; exists {
			if parentMap, ok := parentValue.(map[string]any); ok {
				if childMap, ok := childValue.(map[string]any); ok {
					result[key] = c.deepMergeTemplates(parentMap, childMap)
					continue
				}
			}
		}
		result[key] = childValue
	}

	return result
}
