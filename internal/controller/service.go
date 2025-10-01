package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

	// Check if the service has the required annotation
	if cfg.ServiceAnnotation == "" {
		return true // If no annotation is configured, process all services
	}

	annotations := service.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, hasAnnotation := annotations[cfg.ServiceAnnotation]
	return hasAnnotation
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

	// Example: http://service-name.namespace.svc:port
	port := service.Spec.Ports[0].Port
	protocol := service.Spec.Ports[0].Protocol
	url := fmt.Sprintf("%s://%s.%s.svc:%d", protocol, service.Name, service.Namespace, port)

	return url
}

func (h *ServiceHandler) GetResourceName() string {
	return "service"
}

func (h *ServiceHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) bool {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return false
	}

	endpoint.Client = nil // Services do not use custom client settings by default
	endpoint.Conditions = []string{"[CONNECTED] == true"}
	endpoint.AddExtraField("ui", map[string]any{
		"hide-url":      true,
		"hide-hostname": true,
	})

	if cfg.AutoGroup {
		endpoint.Group = service.Namespace
	}

	return true
}

// NewServiceController creates a controller for Service resources
func NewServiceController(resourceHandler handler.ResourceHandler, stateManager *manager.Manager) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		handler:      resourceHandler,
		stateManager: stateManager,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			service := &corev1.Service{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, service); err != nil {
				return nil, fmt.Errorf("failed to convert to Service: %w", err)
			}
			return service, nil
		},
	}
}

func RunService(ctx context.Context, cfg *config.Config) error {
	stateManager := manager.NewManager(cfg.Output)
	handler := &ServiceHandler{}
	ctrl := NewServiceController(handler, stateManager)
	return ctrl.Run(ctx, cfg)
}
