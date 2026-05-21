// Package k8s contains the dynamic-informer controller and the Resource
// interface implemented by every monitored resource kind.
package k8s

import (
	"context"

	"github.com/home-operations/gatus-sidecar/internal/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Resource declares how to derive a Gatus endpoint from a Kubernetes object.
// Implementations are stateless value types; the [Controller] orchestrates
// them and owns mutation of the resulting Endpoint.
type Resource interface {
	GVR() schema.GroupVersionResource

	// Prefix is prepended to the endpoint name, e.g. "svc/" so an Ingress and
	// a Service sharing a metadata.name produce distinct endpoints.
	Prefix(cfg *config.Config) string

	Convert(u *unstructured.Unstructured) (metav1.Object, error)

	// Matches reports whether obj passes the per-kind filters (auto flags,
	// gateway/ingress class, annotation gate).
	Matches(obj metav1.Object, cfg *config.Config) bool

	// URL returns the URL gatus should probe, or "" if none can be derived.
	URL(obj metav1.Object) string

	DefaultConditions() []string

	// GuardHost returns the DNS-probe hostname when the endpoint is guarded,
	// or "" when the kind doesn't support guarding (Service).
	GuardHost(obj metav1.Object) string

	// ParentAnnotations returns the parent's annotations for template
	// inheritance (Gateway → HTTPRoute, IngressClass → Ingress) or nil.
	ParentAnnotations(ctx context.Context, obj metav1.Object, dc dynamic.Interface) map[string]string
}
