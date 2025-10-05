package resources

import (
	"context"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// ResourceHandler defines the interface for handling different Kubernetes resources
type ResourceHandler interface {
	// ShouldProcess returns true if this resource should be processed based on config filters
	ShouldProcess(obj metav1.Object, cfg *config.Config) bool
	// ExtractURL extracts the URL from the resource
	ExtractURL(obj metav1.Object) string
	// ApplyTemplate applies the resource-specific template to the endpoint
	ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint)
	// GetParentAnnotations retrieves annotations from the parent resource, if applicable
	GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string
}

// Handler implements ResourceHandler using ResourceDefinition
type Handler struct {
	definition    *ResourceDefinition
	dynamicClient dynamic.Interface
}

var _ ResourceHandler = (*Handler)(nil)

// NewHandler creates a new handler with the given resource definition
func NewHandler(definition *ResourceDefinition, dynamicClient dynamic.Interface) *Handler {
	return &Handler{
		definition:    definition,
		dynamicClient: dynamicClient,
	}
}

func (h *Handler) ShouldProcess(obj metav1.Object, cfg *config.Config) bool {
	// Check custom filter first if provided
	if h.definition.FilterFunc != nil && !h.definition.FilterFunc(obj, cfg) {
		return false
	}

	// Check auto-config setting
	if h.definition.AutoConfigFunc != nil {
		if h.definition.AutoConfigFunc(cfg) {
			return true
		}
	}

	// If auto is disabled, only process if it has required annotations
	return HasRequiredAnnotations(obj, cfg)
}

func (h *Handler) ExtractURL(obj metav1.Object) string {
	if h.definition.URLExtractor == nil {
		return ""
	}
	return h.definition.URLExtractor(obj)
}

func (h *Handler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	// Apply guarded template if needed
	if endpoint.Guarded && h.definition.GuardedFunc != nil {
		h.definition.GuardedFunc(obj, endpoint)
		return
	}

	// Apply normal conditions
	if h.definition.ConditionFunc != nil {
		h.definition.ConditionFunc(cfg, obj, endpoint)
	}
}

func (h *Handler) GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string {
	if h.definition.ParentExtractor == nil {
		return nil
	}
	return h.definition.ParentExtractor(ctx, obj, h.dynamicClient)
}
