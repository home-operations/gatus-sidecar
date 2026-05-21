package resources

import (
	"context"
	"slices"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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

func (Ingress) Prefix(cfg *config.Config) string { return cfg.IngressPrefix }

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
	if cfg.AutoIngress {
		return true
	}
	return hasGatusAnnotations(obj, cfg)
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
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + path
	}
	if ingressUsesTLS(ing, host) {
		return "https://" + host + path
	}
	return "http://" + host + path
}

func (Ingress) DefaultConditions() []string { return httpDefaultConditions }

func (Ingress) GuardHost(obj metav1.Object) string {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return ""
	}
	return firstIngressHostname(ing)
}

func (Ingress) ParentAnnotations(ctx context.Context, obj metav1.Object, dc dynamic.Interface) map[string]string {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}
	className := ingressClassOf(ing)
	if className == "" {
		return nil
	}
	parent, err := dc.Resource(ingressClassGVR).Get(ctx, className, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return parent.GetAnnotations()
}

func firstIngressHostname(ing *networkingv1.Ingress) string {
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			return rule.Host
		}
	}
	return ""
}

// firstIngressHostAndPath returns the first non-empty hostname and the first
// probable path under it. Path is "" when the rule has no usable path.
func firstIngressHostAndPath(ing *networkingv1.Ingress) (host, path string) {
	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		host = rule.Host
		if rule.HTTP == nil {
			return host, ""
		}
		for _, p := range rule.HTTP.Paths {
			if isProbablePath(p.Path) {
				return host, p.Path
			}
		}
		return host, ""
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
