package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

func TestHandler_ShouldProcess(t *testing.T) {
	cfg := &config.Config{
		AutoService:        true,
		EnabledAnnotation:  "gatus.home-operations.com/enabled",
		TemplateAnnotation: "gatus.home-operations.com/endpoint",
	}

	tests := []struct {
		name       string
		definition *ResourceDefinition
		obj        metav1.Object
		cfg        *config.Config
		want       bool
	}{
		{
			name: "auto-config enabled processes all",
			definition: &ResourceDefinition{
				AutoConfigFunc: func(cfg *config.Config) bool { return cfg.AutoService },
			},
			obj:  &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			cfg:  cfg,
			want: true,
		},
		{
			name: "auto-config disabled requires annotations",
			definition: &ResourceDefinition{
				AutoConfigFunc: func(cfg *config.Config) bool { return false },
			},
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"gatus.home-operations.com/enabled": "true",
					},
				},
			},
			cfg:  cfg,
			want: true,
		},
		{
			name: "filter function can reject objects",
			definition: &ResourceDefinition{
				AutoConfigFunc: func(cfg *config.Config) bool { return true },
				FilterFunc: func(obj metav1.Object, cfg *config.Config) bool {
					return obj.GetName() == "allowed"
				},
			},
			obj:  &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "rejected"}},
			cfg:  cfg,
			want: false,
		},
		{
			name: "no annotations and auto disabled",
			definition: &ResourceDefinition{
				AutoConfigFunc: func(cfg *config.Config) bool { return false },
			},
			obj:  &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			cfg:  cfg,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.definition, nil)
			if got := h.ShouldProcess(tt.obj, tt.cfg); got != tt.want {
				t.Errorf("ShouldProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_ExtractURL(t *testing.T) {
	tests := []struct {
		name       string
		definition *ResourceDefinition
		obj        metav1.Object
		want       string
	}{
		{
			name: "returns empty string when no extractor",
			definition: &ResourceDefinition{
				URLExtractor: nil,
			},
			obj:  &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want: "",
		},
		{
			name: "extracts URL using custom extractor",
			definition: &ResourceDefinition{
				URLExtractor: func(obj metav1.Object) string {
					return "https://custom.url"
				},
			},
			obj:  &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want: "https://custom.url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.definition, nil)
			if got := h.ExtractURL(tt.obj); got != tt.want {
				t.Errorf("ExtractURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_ApplyTemplate(t *testing.T) {
	cfg := &config.Config{}

	tests := []struct {
		name       string
		definition *ResourceDefinition
		obj        metav1.Object
		endpoint   *endpoint.Endpoint
		wantCond   []string
	}{
		{
			name: "applies condition function for non-guarded endpoint",
			definition: &ResourceDefinition{
				ConditionFunc: func(cfg *config.Config, obj metav1.Object, e *endpoint.Endpoint) {
					e.Conditions = []string{"[STATUS] == 200"}
				},
			},
			obj:      &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			endpoint: &endpoint.Endpoint{Name: "test"},
			wantCond: []string{"[STATUS] == 200"},
		},
		{
			name: "applies guarded function for guarded endpoint",
			definition: &ResourceDefinition{
				ConditionFunc: func(cfg *config.Config, obj metav1.Object, e *endpoint.Endpoint) {
					e.Conditions = []string{"[STATUS] == 200"}
				},
				GuardedFunc: func(obj metav1.Object, e *endpoint.Endpoint) {
					e.Conditions = []string{"len([BODY]) == 0"}
				},
			},
			obj:      &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			endpoint: &endpoint.Endpoint{Name: "test", Guarded: true},
			wantCond: []string{"len([BODY]) == 0"},
		},
		{
			name:       "no functions defined",
			definition: &ResourceDefinition{},
			obj:        &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			endpoint:   &endpoint.Endpoint{Name: "test"},
			wantCond:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.definition, nil)
			h.ApplyTemplate(cfg, tt.obj, tt.endpoint)
			if !equalStringSlices(tt.endpoint.Conditions, tt.wantCond) {
				t.Errorf("Conditions = %v, want %v", tt.endpoint.Conditions, tt.wantCond)
			}
		})
	}
}

func TestHandler_GetParentAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)

	t.Run("returns nil when no parent extractor", func(t *testing.T) {
		h := NewHandler(&ResourceDefinition{}, nil)
		obj := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
		if got := h.GetParentAnnotations(context.Background(), obj); got != nil {
			t.Errorf("GetParentAnnotations() = %v, want nil", got)
		}
	})

	t.Run("calls parent extractor when defined", func(t *testing.T) {
		h := NewHandler(&ResourceDefinition{
			ParentExtractor: func(ctx context.Context, obj metav1.Object, client dynamic.Interface) map[string]string {
				return map[string]string{"parent": "annotation"}
			},
		}, client)
		obj := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
		got := h.GetParentAnnotations(context.Background(), obj)
		if got == nil || got["parent"] != "annotation" {
			t.Errorf("GetParentAnnotations() = %v, want {parent: annotation}", got)
		}
	})
}

func TestCreateConvertFunc(t *testing.T) {
	convertFunc := CreateConvertFunc(reflect.TypeOf(corev1.Service{}))

	t.Run("converts valid unstructured to service", func(t *testing.T) {
		u := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name":      "test-service",
					"namespace": "default",
				},
				"spec": map[string]any{
					"ports": []any{
						map[string]any{
							"port":     80,
							"protocol": "TCP",
						},
					},
				},
			},
		}

		obj, err := convertFunc(u)
		if err != nil {
			t.Fatalf("CreateConvertFunc() error = %v", err)
		}

		if obj.GetName() != "test-service" {
			t.Errorf("CreateConvertFunc() name = %v, want test-service", obj.GetName())
		}
		if obj.GetNamespace() != "default" {
			t.Errorf("CreateConvertFunc() namespace = %v, want default", obj.GetNamespace())
		}
	})
}

func TestHasRequiredAnnotations(t *testing.T) {
	cfg := &config.Config{
		EnabledAnnotation:  "gatus.home-operations.com/enabled",
		TemplateAnnotation: "gatus.home-operations.com/endpoint",
	}

	tests := []struct {
		name string
		obj  metav1.Object
		want bool
	}{
		{
			name: "has enabled annotation",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"gatus.home-operations.com/enabled": "true",
					},
				},
			},
			want: true,
		},
		{
			name: "has template annotation",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"gatus.home-operations.com/endpoint": "interval: 30s",
					},
				},
			},
			want: true,
		},
		{
			name: "has both annotations",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"gatus.home-operations.com/enabled":  "true",
						"gatus.home-operations.com/endpoint": "interval: 30s",
					},
				},
			},
			want: true,
		},
		{
			name: "no annotations",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			want: false,
		},
		{
			name: "nil annotations",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: nil,
				},
			},
			want: false,
		},
		{
			name: "unrelated annotations",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"other-annotation": "value",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasRequiredAnnotations(tt.obj, cfg); got != tt.want {
				t.Errorf("HasRequiredAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
