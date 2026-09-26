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
	conditions     []string
	guardHost      string
	urlFn          func(metav1.Object) string
	parentAnnotsFn func(context.Context, metav1.Object, *Fetcher) map[string]string
}

func (fakeResource) GVR() schema.GroupVersionResource                            { return testGVR }
func (fakeResource) Kind() string                                                { return testKind }
func (f fakeResource) DefaultConditions() []string                               { return f.conditions }
func (f fakeResource) GuardHost(metav1.Object) string                            { return f.guardHost }
func (fakeResource) Convert(u *unstructured.Unstructured) (metav1.Object, error) { return u, nil }

func (fakeResource) Matches(metav1.Object, *config.Config) bool { return true }

func (f fakeResource) URL(obj metav1.Object, _ *config.Config) string {
	if f.urlFn != nil {
		return f.urlFn(obj)
	}
	return "https://example.com"
}

func (f fakeResource) ParentAnnotations(ctx context.Context, obj metav1.Object, fetcher *Fetcher) map[string]string {
	if f.parentAnnotsFn != nil {
		return f.parentAnnotsFn(ctx, obj, fetcher)
	}
	return nil
}

const testKind = "thing"

var testGVR = schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "things"}

// testConfig runs testKind in auto mode so objects need no opt-in annotation.
func testConfig() *config.Config {
	return &config.Config{
		Kinds:              map[string]*config.KindConfig{testKind: {Auto: true}},
		DefaultInterval:    30 * time.Second,
		TemplateAnnotation: "tpl",
		EnabledAnnotation:  "enabled",
	}
}

// makeUnstructured builds an *unstructured.Unstructured suitable for the fake
// dynamic client's tracker. All test resources live in "default/thing-a".
func makeUnstructured(annotations map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(testGVR.GroupVersion().WithKind("Thing"))
	u.SetNamespace("default")
	u.SetName("thing-a")
	if annotations != nil {
		u.SetAnnotations(annotations)
	}
	return u
}

// newFakeClient registers a list kind for testGVR so the dynamic informer can
// list it.
func newFakeClient() dynamic.Interface {
	return fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{testGVR: "ThingList"})
}

func seed(t *testing.T, client dynamic.Interface, obj *unstructured.Unstructured) {
	t.Helper()
	if _, err := client.Resource(testGVR).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// newStartedWriter returns a started Writer and its output path.
func newStartedWriter(t *testing.T) (*gatus.Writer, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.yaml")
	w := gatus.NewWriter(path)
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return w, path
}

// waitForOutput waits until the output file contains want and returns it.
// Upsert and Flush are separate steps, so the writer's in-memory state can
// run ahead of the file.
func waitForOutput(t *testing.T, path, want string) string {
	t.Helper()
	var out string
	if !waitFor(t, func() bool {
		data, _ := os.ReadFile(path)
		out = string(data)
		return strings.Contains(out, want)
	}) {
		t.Fatalf("output never contained %q:\n%s", want, out)
	}
	return out
}

func TestController_ReconcileAddsAndDeletesEndpoint(t *testing.T) {
	client := newFakeClient()
	seed(t, client, makeUnstructured(nil))

	cfg := testConfig()

	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))
	c := NewController(cfg, fakeResource{urlFn: func(metav1.Object) string { return "https://thing-a.example.com" }}, writer, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()

	select {
	case <-c.synced:
	case <-time.After(waitTimeout):
		t.Fatal("controller never reported synced")
	}
	if writer.Len() != 1 {
		t.Fatalf("expected 1 endpoint after initial sync, got %d", writer.Len())
	}

	if err := client.Resource(testGVR).Namespace("default").Delete(ctx, "thing-a", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !waitFor(t, func() bool { return writer.Len() == 0 }) {
		t.Fatalf("expected 0 endpoints, got %d", writer.Len())
	}

	cancel()
	<-done
}

func TestController_DisabledAnnotationRemovesEndpoint(t *testing.T) {
	client := newFakeClient()
	seed(t, client, makeUnstructured(nil))

	cfg := testConfig()
	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))
	c := NewController(cfg, fakeResource{}, writer, client)

	ctx := t.Context()
	go func() { _ = c.Run(ctx) }()
	if !waitFor(t, func() bool { return writer.Len() == 1 }) {
		t.Fatalf("expected 1 endpoint, got %d", writer.Len())
	}

	live, err := client.Resource(testGVR).Namespace("default").Get(ctx, "thing-a", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	live.SetAnnotations(map[string]string{"enabled": "false"})
	if _, err := client.Resource(testGVR).Namespace("default").Update(ctx, live, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !waitFor(t, func() bool { return writer.Len() == 0 }) {
		t.Fatalf("expected endpoint to be removed, got %d", writer.Len())
	}
}

func TestController_MissingURLRemovesEndpoint(t *testing.T) {
	cfg := testConfig()
	writer := gatus.NewWriter(filepath.Join(t.TempDir(), "out.yaml"))

	c := NewController(cfg, fakeResource{
		urlFn: func(metav1.Object) string { return "" },
	}, writer, newFakeClient())

	// Drive reconcile directly off the indexer so the assertion is
	// deterministic — an empty URL must never produce an endpoint.
	if err := c.informer.GetIndexer().Add(makeUnstructured(nil)); err != nil {
		t.Fatalf("seed indexer: %v", err)
	}
	if err := c.reconcile(context.Background(), "default/thing-a"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if writer.Len() != 0 {
		t.Errorf("expected 0 endpoints when URL is empty, got %d", writer.Len())
	}
}

func TestController_GuardedWithoutHostKeepsDefaultConditions(t *testing.T) {
	client := newFakeClient()
	seed(t, client, makeUnstructured(map[string]string{"tpl": "guarded: true\n"}))
	writer, outPath := newStartedWriter(t)

	c := NewController(testConfig(), fakeResource{conditions: []string{"[CONNECTED] == true"}}, writer, client)
	go func() { _ = c.Run(t.Context()) }()

	out := waitForOutput(t, outPath, "name: thing-a")
	if !strings.Contains(out, "[CONNECTED] == true") {
		t.Errorf("guarded endpoint with no guard host should keep the default conditions:\n%s", out)
	}
}

func TestWaitSynced(t *testing.T) {
	synced := &Controller{resource: fakeResource{}, synced: make(chan struct{})}
	close(synced.synced)
	stuck := &Controller{resource: fakeResource{}, synced: make(chan struct{})}

	if pending := WaitSynced(t.Context(), []*Controller{synced}, time.Minute); pending != nil {
		t.Errorf("pending = %v, want none", pending)
	}
	if pending := WaitSynced(t.Context(), []*Controller{synced, stuck}, 10*time.Millisecond); len(pending) != 1 {
		t.Errorf("pending = %v, want only the unsynced controller", pending)
	}
}

func TestSetURLPath(t *testing.T) {
	cases := []struct {
		name    string
		rawURL  string
		newPath string
		want    string
	}{
		{"replace path", "https://x.example.com/api", "/healthz", "https://x.example.com/healthz"},
		{"strip path", "https://x.example.com/api", "", "https://x.example.com"},
		{"add path when none", "https://x.example.com", "/alive", "https://x.example.com/alive"},
		{"non-rooted gets leading slash", "https://x.example.com/api", "alive", "https://x.example.com/alive"},
		{"preserves port", "https://x.example.com:8443/api", "/healthz", "https://x.example.com:8443/healthz"},
		{"preserves query", "https://x.example.com/api?q=1", "/healthz", "https://x.example.com/healthz?q=1"},
		{"unparseable returns as-is", "not a url", "/healthz", "not a url"},
		{"scheme-less returns as-is", "x.example.com/api", "/healthz", "x.example.com/api"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := setURLPath(tt.rawURL, tt.newPath); got != tt.want {
				t.Errorf("setURLPath(%q, %q) = %q, want %q", tt.rawURL, tt.newPath, got, tt.want)
			}
		})
	}
}

func TestController_AppliesPrefixToEndpointName(t *testing.T) {
	client := newFakeClient()
	seed(t, client, makeUnstructured(nil))

	cfg := testConfig()
	cfg.Kinds[testKind].Prefix = "svc-"
	writer, outPath := newStartedWriter(t)

	c := NewController(cfg, fakeResource{
		urlFn: func(metav1.Object) string { return "https://x" },
	}, writer, client)

	ctx := t.Context()
	go func() { _ = c.Run(ctx) }()

	waitForOutput(t, outPath, "name: svc-thing-a")
}

func TestController_TemplateInheritanceAndGuarded(t *testing.T) {
	client := newFakeClient()

	// Object's own template overrides the parent's interval and turns on guarded.
	obj := makeUnstructured(map[string]string{
		"tpl": "interval: 10s\nguarded: true\n",
	})
	seed(t, client, obj)

	cfg := testConfig()
	writer, outPath := newStartedWriter(t)

	r := fakeResource{
		conditions: []string{"[STATUS] == 200"},
		guardHost:  "guarded.example.com",
		urlFn:      func(metav1.Object) string { return "https://thing-a.example.com" },
		parentAnnotsFn: func(context.Context, metav1.Object, *Fetcher) map[string]string {
			// Parent supplies group; child supplies interval and guarded.
			return map[string]string{"tpl": "group: parent-group\ninterval: 60s\n"}
		},
	}
	c := NewController(cfg, r, writer, client)

	ctx := t.Context()
	go func() { _ = c.Run(ctx) }()

	out := waitForOutput(t, outPath, "name: thing-a")
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

func TestController_PathOverrideAndProbePathsFlag(t *testing.T) {
	cases := []struct {
		name       string
		annotation string
		probePaths bool
		wantURL    string
	}{
		{"default keeps auto path", "", true, "https://thing-a.example.com/api"},
		{"probe-paths=false strips path", "", false, "https://thing-a.example.com"},
		{"annotation path overrides auto", "path: /healthz\n", true, "https://thing-a.example.com/healthz"},
		{"empty annotation path forces bare", `path: ""` + "\n", true, "https://thing-a.example.com"},
		{"annotation wins over probe-paths=false", "path: /healthz\n", false, "https://thing-a.example.com/healthz"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeClient()
			ann := map[string]string{}
			if tt.annotation != "" {
				ann["tpl"] = tt.annotation
			}
			seed(t, client, makeUnstructured(ann))

			cfg := testConfig()
			cfg.ProbePaths = tt.probePaths
			writer, outPath := newStartedWriter(t)

			r := fakeResource{
				urlFn: func(metav1.Object) string { return "https://thing-a.example.com/api" },
			}
			c := NewController(cfg, r, writer, client)

			ctx := t.Context()
			go func() { _ = c.Run(ctx) }()

			waitForOutput(t, outPath, "url: "+tt.wantURL+"\n")
		})
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
