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
// without a live apiserver hit per reconcile.
type Fetcher interface {
	GetAnnotations(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) map[string]string
}

const defaultFetcherTTL = 30 * time.Second

// NewFetcher returns a Fetcher safe for concurrent use that caches
// annotation lookups (including not-found) for ~30s.
func NewFetcher(client dynamic.Interface) Fetcher {
	return &cachedFetcher{
		client: client,
		ttl:    defaultFetcherTTL,
		cache:  make(map[string]fetcherEntry),
	}
}

type fetcherEntry struct {
	annotations map[string]string
	expires     time.Time
}

type cachedFetcher struct {
	client dynamic.Interface
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]fetcherEntry
}

func (f *cachedFetcher) GetAnnotations(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) map[string]string {
	key := gvr.String() + "/" + namespace + "/" + name
	now := time.Now()

	f.mu.RLock()
	entry, ok := f.cache[key]
	f.mu.RUnlock()
	if ok && now.Before(entry.expires) {
		return entry.annotations
	}

	res := f.client.Resource(gvr)
	var iface dynamic.ResourceInterface = res
	if namespace != "" {
		iface = res.Namespace(namespace)
	}

	var ann map[string]string
	obj, err := iface.Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		ann = obj.GetAnnotations()
	case apierrors.IsNotFound(err):
		// Cache the absence so a missing parent doesn't probe per reconcile.
	default:
		slog.Debug("fetch parent annotations",
			"gvr", gvr.String(), "namespace", namespace, "name", name, "error", err)
	}

	f.mu.Lock()
	f.cache[key] = fetcherEntry{annotations: ann, expires: now.Add(f.ttl)}
	f.mu.Unlock()
	return ann
}
