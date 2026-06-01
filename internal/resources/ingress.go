package resources

import (
	"context"
	"slices"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const legacyIngressClassAnnotation = "kubernetes.io/ingress.class"

var (
	ingressGVR = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}
	ingressClassGVR = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingressclasses",
	}
)

type Ingress struct{}

func (Ingress) GVR() schema.GroupVersionResource { return ingressGVR }

func (Ingress) Prefix(cfg *config.Config) string { return cfg.Prefix(config.KindIngress) }

func (Ingress) Convert(u *unstructured.Unstructured) (metav1.Object, error) {
	return convertTo[networkingv1.Ingress](u)
}

func (Ingress) Matches(obj metav1.Object, cfg *config.Config) bool {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return false
	}
	if len(cfg.IngressClasses) > 0 && !cfg.IngressClasses.Contains(ingressClassOf(ing)) {
		return false
	}
	return matchesAnnotation(obj, cfg.AutoEnabled(config.KindIngress), cfg)
}

func (Ingress) URL(obj metav1.Object) string {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return ""
	}
	host, path := firstIngressHostAndPath(ing)
	if host == "" {
		return ""
	}
	return formatURL(host, path, ingressUsesTLS(ing, host))
}

func (Ingress) DefaultConditions() []string { return httpDefaultConditions }

func (Ingress) GuardHost(obj metav1.Object) string {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return ""
	}
	host, _ := firstIngressHostAndPath(ing)
	return host
}

func (Ingress) ParentAnnotations(ctx context.Context, obj metav1.Object, fetcher k8s.Fetcher) map[string]string {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}
	className := ingressClassOf(ing)
	if className == "" {
		return nil
	}
	return fetcher.GetAnnotations(ctx, ingressClassGVR, "", className)
}

// firstIngressHostAndPath returns the first non-empty hostname and the first
// probable path under it. Path is "" when the rule has no usable path.
func firstIngressHostAndPath(ing *networkingv1.Ingress) (string, string) {
	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		if rule.HTTP != nil {
			for _, p := range rule.HTTP.Paths {
				if isProbablePath(p.Path) {
					return rule.Host, p.Path
				}
			}
		}
		return rule.Host, ""
	}
	return "", ""
}

// isProbablePath rejects empty, root, and non-rooted values
// (ImplementationSpecific paths from some controllers can be regex-like).
func isProbablePath(p string) bool {
	return p != "" && p != "/" && strings.HasPrefix(p, "/")
}

func ingressUsesTLS(ing *networkingv1.Ingress, host string) bool {
	for _, tls := range ing.Spec.TLS {
		if slices.Contains(tls.Hosts, host) {
			return true
		}
	}
	return false
}

func ingressClassOf(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	return ing.Annotations[legacyIngressClassAnnotation]
}
