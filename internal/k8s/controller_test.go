package k8s

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/gatus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

// fakeResource is a minimal Resource implementation. Tests configure behavior
// by setting fields; unset fields fall back to inert defaults.
type fakeResource struct {
	gvr            schema.GroupVersionResource
	prefix         string
	conditions     []string
	guardHost      string
	urlFn          func(metav1.Object) string
	parentAnnotsFn func(context.Context, metav1.Object, dynamic.Interface) map[string]string
}

func (f fakeResource) GVR() schema.GroupVersionResource                          { return f.gvr }
func (f fakeResource) Prefix(*config.Config) string                              { return f.prefix }
func (f fakeResource) DefaultConditions() []string                               { return f.conditions }
func (f fakeResource) GuardHost(metav1.Object) string                            { return f.guardHost }
func (fakeResource) Convert(u *unstructured.Unstructured) (metav1.Object, error) { return u, nil }
func (fakeResource) Matches(metav1.Object, *config.Config) bool                  { return true }

func (f fakeResource) URL(obj metav1.Object) string {
	if f.urlFn != nil {
		return f.urlFn(obj)
	}
	return "https://example.com"
}

func (f fakeResource) ParentAnnotations(ctx context.Context, obj metav1.Object, dc dynamic.Interface) map[string]string {
	if f.parentAnnotsFn != nil {
		return f.parentAnnotsFn(ctx, obj, dc)
	}
	return nil
}

// makeUnstructured builds an *unstructured.Unstructured suitable for the fake
// dynamic client's tracker. All test resources live in "default/thing-a".
func makeUnstructured(gvr schema.GroupVersionResource, annotations map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    "Thing",
	})
	u.SetNamespace("default")
	u.SetName("thing-a")
	if annotations != nil {
		u.SetAnnotations(annotations)
	}
	return u
}

// newFakeClient registers a list kind for our GVR so the dynamic informer can
// list it.
func newFakeClient(gvr schema.GroupVersionResource) dynamic.Interface {
	scheme := runtime.NewScheme()
	gvk := schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: "Thing"}
	listGVK := schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: "ThingList"}
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(listGVK, &unstructured.UnstructuredList{})
	listKinds := map[schema.GroupVersionResource]string{gvr: "ThingList"}
	return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds)
}

func seed(t *testing.T, client dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) {
	t.Helper()
	if _, err := client.Resource(gvr).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestController_ReconcileAddsAndDeletesEndpoint(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}
	client := newFakeClient(gvr)
	seed(t, client, gvr, makeUnstructured(gvr, nil))

	cfg := &config.Config{
		DefaultInterval:    30 * time.Second,
		TemplateAnnotation: "tpl",
		EnabledAnnotation:  "enabled",
	}

	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))
	c := NewController(cfg, fakeResource{gvr: gvr, urlFn: func(metav1.Object) string { return "https://thing-a.example.com" }}, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()

	if !waitFor(t, func() bool { return writer.Len() == 1 }) {
		t.Fatalf("expected 1 endpoint, got %d", writer.Len())
	}

	if err := client.Resource(gvr).Namespace("default").Delete(ctx, "thing-a", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !waitFor(t, func() bool { return writer.Len() == 0 }) {
		t.Fatalf("expected 0 endpoints, got %d", writer.Len())
	}

	cancel()
	<-done
}

func TestController_DisabledAnnotationRemovesEndpoint(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}
	client := newFakeClient(gvr)
	seed(t, client, gvr, makeUnstructured(gvr, nil))

	cfg := &config.Config{
		DefaultInterval:    30 * time.Second,
		TemplateAnnotation: "tpl",
		EnabledAnnotation:  "enabled",
	}
	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))
	c := NewController(cfg, fakeResource{gvr: gvr}, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = c.Run(ctx) }()
	if !waitFor(t, func() bool { return writer.Len() == 1 }) {
		t.Fatalf("expected 1 endpoint, got %d", writer.Len())
	}

	live, err := client.Resource(gvr).Namespace("default").Get(ctx, "thing-a", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	live.SetAnnotations(map[string]string{"enabled": "false"})
	if _, err := client.Resource(gvr).Namespace("default").Update(ctx, live, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !waitFor(t, func() bool { return writer.Len() == 0 }) {
		t.Fatalf("expected endpoint to be removed, got %d", writer.Len())
	}
}

func TestController_MissingURLRemovesEndpoint(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}
	client := newFakeClient(gvr)
	seed(t, client, gvr, makeUnstructured(gvr, nil))

	cfg := &config.Config{DefaultInterval: 30 * time.Second, TemplateAnnotation: "tpl", EnabledAnnotation: "enabled"}
	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))

	c := NewController(cfg, fakeResource{
		gvr:   gvr,
		urlFn: func(metav1.Object) string { return "" },
	}, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = c.Run(ctx) }()

	// After sync the writer should still be empty - URL is empty so endpoint isn't added.
	time.Sleep(500 * time.Millisecond)
	if writer.Len() != 0 {
		t.Errorf("expected 0 endpoints when URL is empty, got %d", writer.Len())
	}
}

func TestIsExplicitlyDisabled(t *testing.T) {
	cases := []struct {
		name string
		ann  map[string]string
		want bool
	}{
		{"absent", nil, false},
		{"true", map[string]string{"enabled": "true"}, false},
		{"one", map[string]string{"enabled": "1"}, false},
		{"false", map[string]string{"enabled": "false"}, true},
		{"empty", map[string]string{"enabled": ""}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExplicitlyDisabled(tt.ann, "enabled"); got != tt.want {
				t.Errorf("isExplicitlyDisabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeEndpointKey(t *testing.T) {
	got := makeEndpointKey("a", "ns", schema.GroupVersionResource{Resource: "ingresses"})
	want := "a.ns.ingresses"
	if got != want {
		t.Errorf("makeEndpointKey() = %q, want %q", got, want)
	}
}

func TestController_AppliesPrefixToEndpointName(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}
	client := newFakeClient(gvr)
	seed(t, client, gvr, makeUnstructured(gvr, nil))

	cfg := &config.Config{
		DefaultInterval:    30 * time.Second,
		TemplateAnnotation: "tpl",
		EnabledAnnotation:  "enabled",
	}
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	writer := gatus.NewWriter(outPath)

	c := NewController(cfg, fakeResource{
		gvr:    gvr,
		prefix: "svc-",
		urlFn:  func(metav1.Object) string { return "https://x" },
	}, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = c.Run(ctx) }()

	if !waitFor(t, func() bool { return writer.Len() == 1 }) {
		t.Fatalf("expected 1 endpoint")
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "name: svc-thing-a") {
		t.Errorf("output should contain prefixed name; got:\n%s", data)
	}
}

func TestController_TemplateInheritanceAndGuarded(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}
	client := newFakeClient(gvr)

	// Object's own template overrides the parent's interval and turns on guarded.
	obj := makeUnstructured(gvr, map[string]string{
		"tpl": "interval: 10s\nguarded: true\n",
	})
	seed(t, client, gvr, obj)

	cfg := &config.Config{
		DefaultInterval:    30 * time.Second,
		TemplateAnnotation: "tpl",
		EnabledAnnotation:  "enabled",
	}
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	writer := gatus.NewWriter(outPath)

	r := fakeResource{
		gvr:        gvr,
		conditions: []string{"[STATUS] == 200"},
		guardHost:  "guarded.example.com",
		urlFn:      func(metav1.Object) string { return "https://thing-a.example.com" },
		parentAnnotsFn: func(context.Context, metav1.Object, dynamic.Interface) map[string]string {
			// Parent supplies group; child supplies interval and guarded.
			return map[string]string{"tpl": "group: parent-group\ninterval: 60s\n"}
		},
	}
	c := NewController(cfg, r, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = c.Run(ctx) }()

	if !waitFor(t, func() bool { return writer.Len() == 1 }) {
		t.Fatalf("expected 1 endpoint, got %d", writer.Len())
	}

	// Now read the file and verify the merged endpoint.
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	out := string(data)
	for _, want := range []string{
		"group: parent-group",
		"interval: 10s",
		"url: 1.1.1.1",
		"query-name: guarded.example.com",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

const waitTimeout = 5 * time.Second

func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}
