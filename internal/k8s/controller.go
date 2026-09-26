package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/gatus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	fetcher  *Fetcher
	informer cache.SharedIndexInformer
	queue    workqueue.TypedRateLimitingInterface[string]
	log      *slog.Logger
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
		log:      slog.With("resource", r.GVR().Resource),
	}

	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, obj any) {
			c.enqueue(obj)
		},
		DeleteFunc: c.enqueue,
	})

	return c
}

// Resource returns the GVR resource name (e.g. "ingresses").
func (c *Controller) Resource() string {
	return c.resource.GVR().Resource
}

// Run blocks until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) error {
	c.log.Info("controller starting")
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("k8s: cache sync failed for %s", c.Resource())
	}
	c.log.Info("informer synced", "count", len(c.informer.GetIndexer().ListKeys()))

	// Drain the queue once before workers start so the file is flushed once,
	// not N times during initial sync.
	c.initialReconcile(ctx)
	if err := c.writer.Flush(); err != nil {
		c.log.Error("initial flush failed", "error", err)
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

// initialReconcile drains the queue without flushing. Failures are re-queued
// so the worker loop logs and retries them later.
func (c *Controller) initialReconcile(ctx context.Context) {
	for ctx.Err() == nil && c.queue.Len() > 0 {
		key, shutdown := c.queue.Get()
		if shutdown {
			return
		}
		if err := c.reconcile(ctx, key); err != nil {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
		c.queue.Done(key)
	}
}

func (c *Controller) enqueue(obj any) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.log.Error("derive cache key", "error", err)
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

	err := c.reconcile(ctx, key)
	if err == nil {
		err = c.writer.Flush()
	}
	if err != nil {
		retries := c.queue.NumRequeues(key)
		if retries < defaultMaxRetry {
			c.log.Warn("reconcile failed, requeueing",
				"key", key, "error", err, "retries", retries)
			c.queue.AddRateLimited(key)
			return true
		}
		c.log.Error("reconcile failed, giving up", "key", key, "error", err)
	}
	c.queue.Forget(key)
	return true
}

// reconcile inspects the informer cache for key and either Upserts or
// Deletes the corresponding endpoint in the writer. The caller flushes.
func (c *Controller) reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("k8s: split key %q: %w", key, err)
	}
	// Unique across kinds: the resource plural can't contain "/".
	endpointKey := c.Resource() + "/" + key

	raw, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("k8s: get %q: %w", key, err)
	}
	if !exists {
		c.removeEndpoint(endpointKey, namespace, name, "deleted")
		return nil
	}

	u, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("k8s: unexpected cache type %T", raw)
	}
	obj, err := c.resource.Convert(u)
	if err != nil {
		return err
	}

	if !c.resource.Matches(obj, c.cfg) || !matchesAnnotation(obj, c.cfg.AutoEnabled(c.resource.Kind()), c.cfg) {
		c.removeEndpoint(endpointKey, namespace, name, "not-matched")
		return nil
	}

	probeURL := c.resource.URL(obj, c.cfg)
	if probeURL == "" {
		// Per-resync per-resource; common for headless Services.
		c.log.Debug("resource has no derivable URL", "namespace", namespace, "name", name)
		c.removeEndpoint(endpointKey, namespace, name, "no-url")
		return nil
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
		Name:     c.cfg.Prefix(c.resource.Kind()) + name,
		URL:      probeURL,
		Interval: c.cfg.DefaultInterval.String(),
	}
	if gatus.IsGuarded(merged) {
		gatus.ApplyGuardedDNS(c.resource.GuardHost(obj), e)
	} else {
		e.Conditions = c.resource.DefaultConditions()
	}
	e.ApplyTemplate(merged)

	if c.writer.Upsert(endpointKey, e) {
		c.log.Info("updated endpoint", "namespace", namespace, "name", name, "url", e.URL)
	}
	return nil
}

func (c *Controller) buildTemplate(ctx context.Context, obj metav1.Object) (map[string]any, error) {
	parentAnnotations, err := c.resource.ParentAnnotations(ctx, obj, c.fetcher)
	if err != nil {
		return nil, err
	}
	parentTpl, err := gatus.ParseTemplate(parentAnnotations[c.cfg.TemplateAnnotation])
	if err != nil {
		return nil, fmt.Errorf("k8s: parent template: %w", err)
	}
	objTpl, err := gatus.ParseTemplate(obj.GetAnnotations()[c.cfg.TemplateAnnotation])
	if err != nil {
		return nil, fmt.Errorf("k8s: object template: %w", err)
	}
	return gatus.MergeTemplates(parentTpl, objTpl), nil
}

func (c *Controller) removeEndpoint(key, namespace, name, reason string) {
	if c.writer.Delete(key) {
		c.log.Info("removed endpoint", "namespace", namespace, "name", name, "reason", reason)
	}
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
