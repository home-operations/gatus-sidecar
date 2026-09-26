// Package k8s contains the dynamic-informer controller and the Resource
// interface implemented by every monitored resource kind.
package k8s

import (
	"context"

	"github.com/home-operations/gatus-sidecar/internal/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource declares how to derive a Gatus endpoint from a Kubernetes object.
// Implementations are stateless value types; the [Controller] orchestrates
// them and owns mutation of the resulting Endpoint.
type Resource interface {
	GVR() schema.GroupVersionResource

	// Kind is the config.Kind* identifier that selects this resource's
	// per-kind flags (auto mode, endpoint name prefix).
	Kind() string

	Convert(u *unstructured.Unstructured) (metav1.Object, error)

	// Matches reports whether obj passes the kind-specific filters (gateway
	// name, ingress class). The Controller applies the annotation gate.
	Matches(obj metav1.Object, cfg *config.Config) bool

	// URL returns the URL gatus should probe, or "" if none can be derived.
	URL(obj metav1.Object, cfg *config.Config) string

	DefaultConditions() []string

	// GuardHost returns the DNS-probe hostname when the endpoint is guarded,
	// or "" when the kind doesn't support guarding (Service).
	GuardHost(obj metav1.Object) string

	// ParentAnnotations returns the parent's annotations for template
	// inheritance (Gateway → HTTPRoute, IngressClass → Ingress) or nil.
	ParentAnnotations(ctx context.Context, obj metav1.Object, fetcher *Fetcher) (map[string]string, error)
}
