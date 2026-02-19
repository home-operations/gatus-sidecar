package resources

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
)

// ResourceDefinition defines how to handle a specific Kubernetes resource type
type ResourceDefinition struct {
	// Basic resource identification
	GVR         schema.GroupVersionResource
	TargetType  reflect.Type
	ConvertFunc func(*unstructured.Unstructured) (metav1.Object, error)

	// Configuration matching
	AutoConfigFunc AutoConfigFunc // Function to check if auto-processing is enabled
	FilterFunc     FilterFunc     // Optional filter function for additional filtering

	// URL extraction
	URLExtractor URLExtractor

	// Template and condition logic
	ConditionFunc ConditionFunc
	GuardedFunc   GuardedFunc

	// Parent relationship (for Gateway API resources)
	ParentExtractor ParentExtractor
}

// FilterFunc allows custom filtering logic beyond auto-config
type FilterFunc func(obj metav1.Object, cfg *config.Config) bool

// AutoConfigFunc checks if auto-processing is enabled for this resource type
type AutoConfigFunc func(cfg *config.Config) bool

// URLExtractor extracts URLs from different resource types
type URLExtractor func(obj metav1.Object) string

// ConditionFunc applies appropriate conditions based on resource type
type ConditionFunc func(cfg *config.Config, obj metav1.Object, e *endpoint.Endpoint)

// GuardedFunc determines if an endpoint should be guarded and applies guarded template
type GuardedFunc func(obj metav1.Object, e *endpoint.Endpoint)

// ParentExtractor gets parent annotations for hierarchical resources
type ParentExtractor func(ctx context.Context, obj metav1.Object, client dynamic.Interface) map[string]string

// CreateConvertFunc creates a conversion function for a specific target type
func CreateConvertFunc(targetType reflect.Type) func(*unstructured.Unstructured) (metav1.Object, error) {
	return func(u *unstructured.Unstructured) (metav1.Object, error) {
		obj := reflect.New(targetType).Interface()
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
			return nil, fmt.Errorf("failed to convert to %s: %w", targetType.Name(), err)
		}
		return obj.(metav1.Object), nil
	}
}

// HasRequiredAnnotations checks if an object has the required annotations for processing
func HasRequiredAnnotations(obj metav1.Object, cfg *config.Config) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, hasEnabledAnnotation := annotations[cfg.EnabledAnnotation]
	_, hasTemplateAnnotation := annotations[cfg.TemplateAnnotation]

	return hasEnabledAnnotation || hasTemplateAnnotation
}
