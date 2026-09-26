package k8s

import (
	"strconv"

	"github.com/home-operations/gatus-sidecar/internal/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// matchesAnnotation accepts obj when auto-mode is on or when an explicit
// gatus annotation opts the resource in, unless the enabled annotation is
// explicitly falsy.
func matchesAnnotation(obj metav1.Object, auto bool, cfg *config.Config) bool {
	if isExplicitlyDisabled(obj.GetAnnotations(), cfg.EnabledAnnotation) {
		return false
	}
	return auto || hasGatusAnnotations(obj, cfg)
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
