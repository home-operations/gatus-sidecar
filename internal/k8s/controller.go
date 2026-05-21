package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/gatus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	defaultResync   = 10 * time.Minute
	defaultWorkers  = 2
	defaultMaxRetry = 5
)

// Controller watches a single Resource type and reconciles changes into the
// shared gatus.Writer.
type Controller struct {
	cfg      *config.Config
	resource Resource
	writer   *gatus.Writer
	fetcher  Fetcher
	informer cache.SharedIndexInformer
	queue    workqueue.TypedRateLimitingInterface[string]
}

func NewController(cfg *config.Config, r Resource, w *gatus.Writer, client dynamic.Interface) *Controller {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client, defaultResync, cfg.Namespace, nil,
	)
	informer := factory.ForResource(r.GVR()).Informer()
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: r.GVR().Resource},
	)

	c := &Controller{
		cfg:      cfg,
		resource: r,
		writer:   w,
		fetcher:  NewFetcher(client),
		informer: informer,
		queue:    queue,
	}

	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, obj any) {
			c.enqueue(obj)
		},
		DeleteFunc: func(obj any) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			c.enqueue(obj)
		},
	})

	return c
}

// Resource returns the GVR resource name (e.g. "ingresses").
func (c *Controller) Resource() string {
	return c.resource.GVR().Resource
}

// Run blocks until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) error {
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("cache sync failed for %s", c.Resource())
	}
	slog.Info("informer synced", "resource", c.Resource(), "count", len(c.informer.GetIndexer().ListKeys()))

	// Drain the queue once before workers start so the file is flushed once,
	// not N times during initial sync.
	c.initialReconcile(ctx)
	if err := c.writer.Flush(); err != nil {
		slog.Error("initial flush failed", "resource", c.Resource(), "error", err)
	}

	var wg sync.WaitGroup
	for range defaultWorkers {
		wg.Go(func() { c.runWorker(ctx) })
	}

	<-ctx.Done()
	c.queue.ShutDown()
	wg.Wait()
	return nil
}

// initialReconcile drains the queue with flush suppressed. Failures are
// re-queued so the worker loop logs and retries them later.
func (c *Controller) initialReconcile(ctx context.Context) {
	for ctx.Err() == nil && c.queue.Len() > 0 {
		key, shutdown := c.queue.Get()
		if shutdown {
			return
		}
		if err := c.reconcile(ctx, key, false); err != nil {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
		c.queue.Done(key)
	}
}

func (c *Controller) enqueue(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		slog.Error("derive cache key", "resource", c.Resource(), "error", err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNext(ctx) {
	}
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	if err := c.reconcile(ctx, key, true); err != nil {
		if c.queue.NumRequeues(key) < defaultMaxRetry {
			slog.Warn("reconcile failed, requeueing",
				"resource", c.Resource(), "key", key, "error", err,
				"retries", c.queue.NumRequeues(key))
			c.queue.AddRateLimited(key)
			return true
		}
		slog.Error("reconcile failed, giving up",
			"resource", c.Resource(), "key", key, "error", err)
	}
	c.queue.Forget(key)
	return true
}

// reconcile inspects the informer cache for key and either Upserts or
// Deletes the corresponding endpoint. flush controls whether the writer
// rewrites the output file after this call.
func (c *Controller) reconcile(ctx context.Context, key string, flush bool) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("split key %q: %w", key, err)
	}
	endpointKey := makeEndpointKey(name, namespace, c.resource.GVR())

	raw, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("get %q: %w", key, err)
	}
	if !exists {
		return c.removeEndpoint(endpointKey, namespace, name, "deleted", flush)
	}

	u, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected cache type %T", raw)
	}
	obj, err := c.resource.Convert(u)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	if !c.resource.Matches(obj, c.cfg) || isExplicitlyDisabled(obj.GetAnnotations(), c.cfg.EnabledAnnotation) {
		return c.removeEndpoint(endpointKey, namespace, name, "not-matched", flush)
	}

	probeURL := c.resource.URL(obj)
	if probeURL == "" {
		// Per-resync per-resource; common for headless Services.
		slog.Debug("resource has no derivable URL",
			"resource", c.Resource(), "namespace", namespace, "name", name)
		return c.removeEndpoint(endpointKey, namespace, name, "no-url", flush)
	}

	merged, err := c.buildTemplate(ctx, obj)
	if err != nil {
		return err
	}

	// "path:" beats --probe-paths; "url:" beats both (applied via ApplyTemplate).
	if override, ok := gatus.PathOverride(merged); ok {
		probeURL = setURLPath(probeURL, override)
	} else if !c.cfg.ProbePaths {
		probeURL = setURLPath(probeURL, "")
	}

	e := &gatus.Endpoint{
		Name:     c.resource.Prefix(c.cfg) + name,
		URL:      probeURL,
		Interval: c.cfg.DefaultInterval.String(),
	}
	if gatus.IsGuarded(merged) {
		if host := c.resource.GuardHost(obj); host != "" {
			gatus.ApplyGuardedDNS(host, e)
		}
	} else {
		e.Conditions = c.resource.DefaultConditions()
	}
	e.ApplyTemplate(merged)

	changed, err := c.writer.Upsert(endpointKey, e, flush)
	if err != nil {
		return fmt.Errorf("write after upsert: %w", err)
	}
	if changed {
		slog.Info("updated endpoint",
			"resource", c.Resource(), "namespace", namespace, "name", name, "url", e.URL)
	}
	return nil
}

func (c *Controller) buildTemplate(ctx context.Context, obj metav1.Object) (map[string]any, error) {
	parentAnnotations := c.resource.ParentAnnotations(ctx, obj, c.fetcher)
	parentTpl, err := gatus.ParseTemplate(parentAnnotations[c.cfg.TemplateAnnotation])
	if err != nil {
		return nil, fmt.Errorf("parent template: %w", err)
	}
	objTpl, err := gatus.ParseTemplate(obj.GetAnnotations()[c.cfg.TemplateAnnotation])
	if err != nil {
		return nil, fmt.Errorf("object template: %w", err)
	}
	return gatus.MergeTemplates(parentTpl, objTpl), nil
}

func (c *Controller) removeEndpoint(key, namespace, name, reason string, flush bool) error {
	removed, err := c.writer.Delete(key, flush)
	if err != nil {
		return fmt.Errorf("write after delete: %w", err)
	}
	if removed {
		slog.Info("removed endpoint",
			"resource", c.Resource(), "namespace", namespace, "name", name, "reason", reason)
	}
	return nil
}

// makeEndpointKey returns a writer key unique across resource kinds. The
// "/" separator can't appear in any of the three components (names and
// namespaces follow DNS rules; resource is a plural identifier).
func makeEndpointKey(name, namespace string, gvr schema.GroupVersionResource) string {
	return gvr.Resource + "/" + namespace + "/" + name
}

// setURLPath replaces rawURL's path with path (empty clears it). rawURL
// is returned unchanged when it doesn't parse as an absolute URL.
func setURLPath(rawURL, path string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" {
		return rawURL
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = path
	return u.String()
}

// isExplicitlyDisabled returns true only when the annotation is present *and*
// falsy. Absence is not "disabled". Unparseable values (e.g. empty, "yes")
// are treated as disabled so a typo can't silently widen monitoring.
func isExplicitlyDisabled(annotations map[string]string, key string) bool {
	v, ok := annotations[key]
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(v)
	return err != nil || !enabled
}
