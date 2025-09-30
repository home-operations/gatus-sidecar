package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
)

// ResourceHandler defines the interface for handling different Kubernetes resources
type ResourceHandler interface {
	// ShouldProcess returns true if this resource should be processed based on config filters
	ShouldProcess(obj metav1.Object, cfg *config.Config) bool
	// ExtractURL extracts the URL from the resource
	ExtractURL(obj metav1.Object) string
	// GetResourceName returns the name to use for logging
	GetResourceName() string
}
