package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestFetcher_CachesAcrossCalls(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm := &unstructured.Unstructured{}
	cm.SetGroupVersionKind(gvr.GroupVersion().WithKind("ConfigMap"))
	cm.SetName("cfg")
	cm.SetNamespace("ns")
	cm.SetAnnotations(map[string]string{"k": "v"})
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), cm)

	var gets int
	client.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		return false, nil, nil
	})

	f := NewFetcher(client)
	for range 3 {
		ann := f.GetAnnotations(context.Background(), gvr, "ns", "cfg")
		if ann["k"] != "v" {
			t.Fatalf("annotations = %v, want {k:v}", ann)
		}
	}
	if gets != 1 {
		t.Errorf("apiserver Gets = %d, want 1 (cached)", gets)
	}
}

func TestFetcher_CachesNegativeLookups(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	client := fake.NewSimpleDynamicClient(runtime.NewScheme())

	var gets int
	client.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		return false, nil, nil
	})

	f := NewFetcher(client)
	for range 3 {
		if ann := f.GetAnnotations(context.Background(), gvr, "ns", "missing"); ann != nil {
			t.Fatalf("annotations = %v, want nil", ann)
		}
	}
	if gets != 1 {
		t.Errorf("apiserver Gets for missing object = %d, want 1 (negative cached)", gets)
	}
}

func TestFetcher_TreatsForbiddenAsNoAnnotations(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	client := fake.NewSimpleDynamicClient(runtime.NewScheme())

	var gets int
	client.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		return true, nil, apierrors.NewForbidden(gvr.GroupResource(), "cfg", errors.New("rbac"))
	})

	f := NewFetcher(client)
	for range 3 {
		if ann := f.GetAnnotations(context.Background(), gvr, "ns", "cfg"); ann != nil {
			t.Fatalf("annotations = %v, want nil", ann)
		}
	}
	if gets != 1 {
		t.Errorf("apiserver Gets = %d, want 1 (forbidden cached)", gets)
	}
}

func TestFetcher_ServesLastKnownOnTransientError(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm := &unstructured.Unstructured{}
	cm.SetGroupVersionKind(gvr.GroupVersion().WithKind("ConfigMap"))
	cm.SetName("cfg")
	cm.SetNamespace("ns")
	cm.SetAnnotations(map[string]string{"k": "v"})
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), cm)

	var gets int
	var fail bool
	client.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		if fail {
			return true, nil, errors.New("apiserver unavailable")
		}
		return false, nil, nil
	})

	f := NewFetcher(client)
	if ann := f.GetAnnotations(context.Background(), gvr, "ns", "cfg"); ann["k"] != "v" {
		t.Fatalf("annotations = %v, want {k:v}", ann)
	}

	fail = true
	expireAll(f)
	for range 3 {
		if ann := f.GetAnnotations(context.Background(), gvr, "ns", "cfg"); ann["k"] != "v" {
			t.Fatalf("annotations after transient error = %v, want last known {k:v}", ann)
		}
	}
	if gets != 2 {
		t.Errorf("apiserver Gets = %d, want 2 (error re-arms the TTL)", gets)
	}
}

func expireAll(f *Fetcher) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, e := range f.cache {
		e.expires = time.Time{}
		f.cache[k] = e
	}
}
