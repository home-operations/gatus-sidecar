package k8s

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilcache "k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/dynamic"
)

// Fetcher resolves another object's annotations on demand. Each Resource
// implementation receives one to read its parent (Gateway, IngressClass, ...)
// without a live apiserver hit per reconcile. Safe for concurrent use.
type Fetcher struct {
	client dynamic.Interface
	cache  *utilcache.Expiring
}

const fetcherTTL = 30 * time.Second

// NewFetcher returns a Fetcher that caches annotation lookups (including
// not-found) for ~30s.
func NewFetcher(client dynamic.Interface) *Fetcher {
	return &Fetcher{client: client, cache: utilcache.NewExpiring()}
}

// GetAnnotations returns the annotations of the named object, or nil when
// it doesn't exist. An empty namespace addresses a cluster-scoped object.
// Other errors are returned uncached so the caller can retry rather than
// act on a transient failure as if the object had no annotations.
func (f *Fetcher) GetAnnotations(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (map[string]string, error) {
	key := gvr.String() + "/" + namespace + "/" + name
	if v, ok := f.cache.Get(key); ok {
		return v.(map[string]string), nil
	}

	var ann map[string]string
	obj, err := f.client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		ann = obj.GetAnnotations()
	case apierrors.IsNotFound(err):
		// Cache the absence so a missing parent doesn't probe per reconcile.
	default:
		return nil, fmt.Errorf("k8s: get %s %s/%s: %w", gvr.Resource, namespace, name, err)
	}

	f.cache.Set(key, ann, fetcherTTL)
	return ann, nil
}
