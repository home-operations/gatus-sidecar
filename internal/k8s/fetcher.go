package k8s

import (
	"context"
	"log/slog"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Fetcher resolves another object's annotations on demand. Each Resource
// implementation receives one to read its parent (Gateway, IngressClass, ...)
// without a live apiserver hit per reconcile. Safe for concurrent use.
type Fetcher struct {
	client dynamic.Interface

	mu    sync.Mutex
	cache map[string]fetcherEntry
}

type fetcherEntry struct {
	annotations map[string]string
	expires     time.Time
}

const fetcherTTL = 30 * time.Second

// NewFetcher returns a Fetcher that caches each lookup for ~30s.
func NewFetcher(client dynamic.Interface) *Fetcher {
	return &Fetcher{client: client, cache: make(map[string]fetcherEntry)}
}

// GetAnnotations returns the annotations of the named object, or nil when it
// doesn't exist or the sidecar isn't allowed to read it. An empty namespace
// addresses a cluster-scoped object. On any other error it keeps serving the
// last annotations it read, so a transient failure doesn't strip an
// endpoint's inherited template.
func (f *Fetcher) GetAnnotations(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) map[string]string {
	key := gvr.String() + "/" + namespace + "/" + name
	now := time.Now()

	f.mu.Lock()
	entry, ok := f.cache[key]
	f.mu.Unlock()
	if ok && now.Before(entry.expires) {
		return entry.annotations
	}

	obj, err := f.client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		slog.Debug("fetch parent annotations",
			"gvr", gvr.String(), "namespace", namespace, "name", name, "error", err)
	}
	switch {
	case err == nil:
		entry.annotations = obj.GetAnnotations()
	case apierrors.IsNotFound(err), apierrors.IsForbidden(err):
		// Forbidden is expected under a namespaced Role, which can't grant
		// cluster-scoped IngressClasses or Gateways in other namespaces.
		entry.annotations = nil
	}

	// Errors re-arm the TTL too, so an outage costs one GET per parent per TTL
	// rather than one per child reconcile.
	entry.expires = now.Add(fetcherTTL)
	f.mu.Lock()
	f.cache[key] = entry
	f.mu.Unlock()
	return entry.annotations
}
