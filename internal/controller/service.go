package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/handler"
	"github.com/home-operations/gatus-sidecar/internal/manager"
)

// ServiceHandler handles Service resources
type ServiceHandler struct{}

// Ensure ServiceHandler implements the ResourceHandler interface
var _ handler.ResourceHandler = (*ServiceHandler)(nil)

func (h *ServiceHandler) ShouldProcess(obj metav1.Object, cfg *config.Config) bool {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return false
	}

	// If AutoServices is disabled, only process if it has the annotation
	if !cfg.AutoServices {
		annotations := service.GetAnnotations()
		if annotations == nil {
			return false
		}

		_, hasEnabledAnnotation := annotations[cfg.EnabledAnnotation]
		_, hasGuardedAnnotation := annotations[cfg.GuardedAnnotation]
		_, hasTemplateAnnotation := annotations[cfg.TemplateAnnotation]

		return hasEnabledAnnotation || hasGuardedAnnotation || hasTemplateAnnotation
	}

	return true
}

func (h *ServiceHandler) ExtractURL(obj metav1.Object) string {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return ""
	}

	// Construct the URL using the first port defined in the Service
	if len(service.Spec.Ports) == 0 {
		return ""
	}

	// Example: tcp://service-name.namespace.svc:1234
	port := service.Spec.Ports[0].Port
	protocol := strings.ToLower(string(service.Spec.Ports[0].Protocol))
	url := fmt.Sprintf("%s://%s.%s.svc:%d", protocol, service.Name, service.Namespace, port)

	return url
}

func (h *ServiceHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	endpoint.Client = nil // Use default client configuration
	endpoint.Conditions = []string{"[CONNECTED] == true"}

	if cfg.AutoGroup {
		endpoint.Group = obj.GetNamespace()
	}
}

func (h *ServiceHandler) GetParentAnnotations(context.Context, metav1.Object) map[string]string {
	return map[string]string{}
}

// NewServiceController creates a controller for Service resources
func NewServiceController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		options:       metav1.ListOptions{},
		handler:       &ServiceHandler{},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			service := &corev1.Service{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, service); err != nil {
				return nil, fmt.Errorf("failed to convert to Service: %w", err)
			}
			return service, nil
		},
	}
}
