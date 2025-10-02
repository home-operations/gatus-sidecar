package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
)

// ResourceHandler defines the interface for handling different Kubernetes resources
type ResourceHandler interface {
	// ShouldProcess returns true if this resource should be processed based on config filters
	ShouldProcess(obj metav1.Object, cfg *config.Config) bool
	// ExtractURL extracts the URL from the resource
	ExtractURL(obj metav1.Object) string
	// ApplyTemplate applies the resource-specific template to the endpoint
	ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) bool
}
