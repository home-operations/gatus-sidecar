package k8s

import (
	"context"
	"errors"
	"testing"

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
		ann, err := f.GetAnnotations(context.Background(), gvr, "ns", "cfg")
		if err != nil {
			t.Fatalf("GetAnnotations: %v", err)
		}
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
		ann, err := f.GetAnnotations(context.Background(), gvr, "ns", "missing")
		if err != nil {
			t.Fatalf("GetAnnotations: %v", err)
		}
		if ann != nil {
			t.Fatalf("annotations = %v, want nil", ann)
		}
	}
	if gets != 1 {
		t.Errorf("apiserver Gets for missing object = %d, want 1 (negative cached)", gets)
	}
}

func TestFetcher_DoesNotCacheTransientErrors(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	client := fake.NewSimpleDynamicClient(runtime.NewScheme())

	var gets int
	client.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		return true, nil, errors.New("apiserver unavailable")
	})

	f := NewFetcher(client)
	for range 3 {
		if _, err := f.GetAnnotations(context.Background(), gvr, "ns", "cfg"); err == nil {
			t.Fatal("expected an error")
		}
	}
	if gets != 3 {
		t.Errorf("apiserver Gets = %d, want 3 (errors not cached)", gets)
	}
}
