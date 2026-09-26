// Package resources implements [k8s.Resource] for Ingress, Service, Gateway
// API HTTPRoute, and Traefik IngressRoute.
package resources

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	conditionHTTPOK    = "[STATUS] == 200"
	conditionConnected = "[CONNECTED] == true"
)

var (
	httpDefaultConditions = []string{conditionHTTPOK}
	tcpDefaultConditions  = []string{conditionConnected}
)

// formatURL composes scheme://host/path, honoring an embedded scheme on host
// (e.g. host = "http://example.com" yields host+path unchanged).
func formatURL(host, path string, useTLS bool) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + path
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return scheme + "://" + host + path
}

// matchesAnnotation accepts obj when auto-mode is on or when an explicit
// gatus annotation opts the resource in, unless the enabled annotation is
// explicitly falsy. Callers run any kind-specific filter (ingress class,
// gateway name) before this.
func matchesAnnotation(obj metav1.Object, auto bool, cfg *config.Config) bool {
	if isExplicitlyDisabled(obj.GetAnnotations(), cfg.EnabledAnnotation) {
		return false
	}
	return auto || hasGatusAnnotations(obj, cfg)
}

// registry maps each kind name to its Resource constructor. It is the single
// source of truth for which kinds exist and the order they're created in.
var registry = []struct {
	name string
	new  func() k8s.Resource
}{
	{config.KindIngress, func() k8s.Resource { return Ingress{} }},
	{config.KindHTTPRoute, func() k8s.Resource { return HTTPRoute{} }},
	{config.KindService, func() k8s.Resource { return Service{} }},
	{config.KindIngressRoute, func() k8s.Resource { return IngressRoute{} }},
}

// All returns the Resource implementations enabled by cfg. With no flag set,
// all kinds run in annotation-only mode.
func All(cfg *config.Config) []k8s.Resource {
	annotationOnly := !cfg.AnyExplicitlyEnabled()
	out := make([]k8s.Resource, 0, len(registry))
	for _, e := range registry {
		if annotationOnly || cfg.KindEnabled(e.name) {
			out = append(out, e.new())
		}
	}
	return out
}

func convertTo[T any](u *unstructured.Unstructured) (metav1.Object, error) {
	obj := new(T)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
		return nil, fmt.Errorf("convert to %s: %w", reflect.TypeOf(*obj).Name(), err)
	}
	o, ok := any(obj).(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("type %s does not implement metav1.Object", reflect.TypeOf(*obj).Name())
	}
	return o, nil
}

// hasGatusAnnotations reports whether obj opts in via either gatus annotation
// — the fallback for annotation-only mode.
func hasGatusAnnotations(obj metav1.Object, cfg *config.Config) bool {
	ann := obj.GetAnnotations()
	if _, ok := ann[cfg.EnabledAnnotation]; ok {
		return true
	}
	_, ok := ann[cfg.TemplateAnnotation]
	return ok
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
