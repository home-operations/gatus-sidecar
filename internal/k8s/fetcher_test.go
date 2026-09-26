package k8s

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestFetcher_CachesAcrossCalls(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind("ConfigMap"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind("ConfigMapList"), &unstructured.UnstructuredList{})

	cm := &unstructured.Unstructured{}
	cm.SetGroupVersionKind(gvr.GroupVersion().WithKind("ConfigMap"))
	cm.SetName("cfg")
	cm.SetNamespace("ns")
	cm.SetAnnotations(map[string]string{"k": "v"})
	client := fake.NewSimpleDynamicClient(scheme, cm)

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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind("ConfigMap"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind("ConfigMapList"), &unstructured.UnstructuredList{})
	client := fake.NewSimpleDynamicClient(scheme)

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
