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

const (
	serviceAPIGroup      = ""
	serviceAPIVersion    = "v1"
	servicesResource     = "services"
	serviceClusterDomain = "svc"
	serviceCondition     = "[CONNECTED] == true"
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

	// If AutoService is enabled, process all services
	if cfg.AutoService {
		return true
	}

	// If AutoService is disabled, only process if it has required annotations
	return hasRequiredAnnotations(service, cfg)
}

func (h *ServiceHandler) ExtractURL(obj metav1.Object) string {
	if service, ok := obj.(*corev1.Service); ok {
		if len(service.Spec.Ports) == 0 {
			return ""
		}
		return h.buildServiceURL(service)
	}

	return ""
}

func (h *ServiceHandler) buildServiceURL(service *corev1.Service) string {
	port := service.Spec.Ports[0].Port
	protocol := strings.ToLower(string(service.Spec.Ports[0].Protocol))

	return fmt.Sprintf("%s://%s.%s.%s:%d",
		protocol,
		service.Name,
		service.Namespace,
		serviceClusterDomain,
		port)
}

func (h *ServiceHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	endpoint.Conditions = []string{serviceCondition}
}

func (h *ServiceHandler) GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string {
	return nil
}

// NewServiceController creates a controller for Service resources
func NewServiceController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    serviceAPIGroup,
			Version:  serviceAPIVersion,
			Resource: servicesResource,
		},
		options:       metav1.ListOptions{},
		handler:       &ServiceHandler{},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert:       convertUnstructuredToService,
	}
}

func convertUnstructuredToService(u *unstructured.Unstructured) (metav1.Object, error) {
	service := &corev1.Service{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, service); err != nil {
		return nil, fmt.Errorf("failed to convert to Service: %w", err)
	}
	return service, nil
}
