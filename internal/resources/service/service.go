package service

import (
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/resources"
)

// Definition creates a resource definition for Service resources
func Definition() *resources.ResourceDefinition {
	return &resources.ResourceDefinition{
		GVR: schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		TargetType:     reflect.TypeOf(corev1.Service{}),
		ConvertFunc:    resources.CreateConvertFunc(reflect.TypeOf(corev1.Service{})),
		AutoConfigFunc: func(cfg *config.Config) bool { return cfg.AutoService },
		URLExtractor:   urlExtractor,
		ConditionFunc:  conditionFunc,
	}
}

func urlExtractor(obj metav1.Object) string {
	service, ok := obj.(*corev1.Service)
	if !ok || len(service.Spec.Ports) == 0 {
		return ""
	}

	port := service.Spec.Ports[0].Port
	protocol := strings.ToLower(string(service.Spec.Ports[0].Protocol))

	return fmt.Sprintf("%s://%s.%s.svc:%d",
		protocol,
		service.Name,
		service.Namespace,
		port)
}

func conditionFunc(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	endpoint.Conditions = []string{"[CONNECTED] == true"}
}
