//go:build e2e

// Package e2e drives an end-to-end test against a real Kubernetes cluster
// (typically Kind): it builds the gatus-sidecar binary, applies the fixture
// manifests, runs the sidecar against the cluster, asserts the generated YAML
// across create/update/delete transitions, and finally stands up an upstream
// Gatus pod in the cluster to verify the same endpoints round-trip through
// Gatus's API.
//
// Run with: `go test -tags e2e -timeout 10m ./test/e2e/...`. Honors
// KUBECONFIG (set automatically by helm/kind-action in CI).
package e2e

import "testing"

func TestE2E(t *testing.T) {
	h := newHarness(t)
	h.installGatewayAPI()
	h.applyFixtures()
	h.buildSidecar()
	h.startSidecar(
		"--auto-ingress",
		"--auto-service",
		"--auto-httproute",
		"--prefix-ingress=ing/",
		"--prefix-service=svc/",
	)

	t.Run("emits expected endpoints", func(t *testing.T) {
		// Ingress path /api/v1 is captured in the URL (#33).
		h.waitForEndpointURL("ing/e2e-ingress", "https://e2e-ingress.example.com/api/v1")
		h.waitForEndpointURL("svc/e2e-service", "tcp://e2e-service.e2e.svc:8080")
		// Service e2e-ingress collides with Ingress e2e-ingress by name;
		// prefixes keep them distinct in Gatus (#59).
		h.waitForEndpointURL("svc/e2e-ingress", "tcp://e2e-ingress.e2e.svc:9090")
		h.waitForEndpointURL("e2e-route", "https://e2e-route.example.com")

		eps := h.endpoints()
		if _, ok := eps["svc/e2e-service-disabled"]; ok {
			t.Error("disabled service should not produce an endpoint")
		}
		if got := eps["ing/e2e-ingress"].Interval; got != "45s" {
			t.Errorf("ing/e2e-ingress interval = %q, want 45s (inherited from IngressClass)", got)
		}
		if got, ok := eps["e2e-route"].Extra["alerts"]; !ok || got == nil {
			t.Errorf("e2e-route should inherit alerts from parent Gateway, got %v", eps["e2e-route"].Extra)
		}
		if conds := eps["svc/e2e-service"].Conditions; len(conds) != 2 {
			t.Errorf("svc/e2e-service conditions = %v, want 2 entries", conds)
		}
	})

	t.Run("deletes endpoint when resource is removed", func(t *testing.T) {
		h.kubectl("-n", fixtureNamespace, "delete", "service", "e2e-service")
		h.waitForEndpointAbsent("svc/e2e-service")
	})

	t.Run("removes endpoint when enabled annotation flips false", func(t *testing.T) {
		h.kubectl("-n", fixtureNamespace, "annotate", "--overwrite",
			"ingress", "e2e-ingress", enabledAnnotation+"=false")
		h.waitForEndpointAbsent("ing/e2e-ingress")
	})

	t.Run("Gatus loads and lists the generated endpoints", func(t *testing.T) {
		var want []string
		for name := range h.endpoints() {
			want = append(want, name)
		}
		if len(want) == 0 {
			t.Fatal("no endpoints to verify against Gatus")
		}
		h.deployGatus(h.outPath).expectEndpoints(want...)
	})
}
