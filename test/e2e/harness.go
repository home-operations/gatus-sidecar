//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	clusterName       = "gatus-sidecar-e2e"
	fixtureNamespace  = "e2e"
	fixturesPath      = "test/e2e/fixtures/manifests.yaml"
	enabledAnnotation = "gatus.home-operations.com/enabled"

	gatewayAPIVersion    = "v1.2.0"
	gatewayAPIInstallURL = "https://github.com/kubernetes-sigs/gateway-api/releases/download/" +
		gatewayAPIVersion + "/standard-install.yaml"

	defaultProbeInterval = "15s"
	defaultStepTimeout   = 60 * time.Second
	pollInterval         = 200 * time.Millisecond
)

// gatewayAPICRDs lists every CRD the standard-install manifest creates so
// the cleanup deletes them all; only the first three are load-bearing for
// the test and get waited on.
var gatewayAPICRDs = []string{
	"httproutes.gateway.networking.k8s.io",
	"gateways.gateway.networking.k8s.io",
	"gatewayclasses.gateway.networking.k8s.io",
	"referencegrants.gateway.networking.k8s.io",
	"grpcroutes.gateway.networking.k8s.io",
}

type endpointDoc struct {
	Name       string         `yaml:"name"`
	URL        string         `yaml:"url"`
	Interval   string         `yaml:"interval"`
	Conditions []string       `yaml:"conditions,omitempty"`
	Extra      map[string]any `yaml:",inline"`
}

type harness struct {
	t       *testing.T
	root    string
	binPath string
	outPath string

	ctx     context.Context
	cancel  context.CancelFunc
	sidecar *exec.Cmd
}

// newHarness skips the test if KUBECONFIG isn't set (set by helm/kind-action
// in CI).
func newHarness(t *testing.T) *harness {
	t.Helper()
	if os.Getenv("KUBECONFIG") == "" {
		t.Skip("KUBECONFIG not set; skipping e2e")
	}
	ctx, cancel := context.WithCancel(context.Background())
	tmp := t.TempDir()
	h := &harness{
		t:       t,
		root:    goModRoot(t),
		binPath: filepath.Join(tmp, "gatus-sidecar"),
		outPath: filepath.Join(tmp, "endpoints.yaml"),
		ctx:     ctx,
		cancel:  cancel,
	}
	t.Cleanup(h.shutdown)
	return h
}

func (h *harness) shutdown() {
	h.cancel()
	if h.sidecar != nil {
		_ = h.sidecar.Wait()
	}
}

func (h *harness) installGatewayAPI() {
	h.t.Helper()
	h.t.Log("installing Gateway API CRDs")
	h.kubectl("apply", "-f", gatewayAPIInstallURL)

	waitArgs := []string{"wait", "--for=condition=Established", "--timeout=60s"}
	for _, c := range gatewayAPICRDs[:3] {
		waitArgs = append(waitArgs, "crd/"+c)
	}
	h.kubectl(waitArgs...)

	h.t.Cleanup(func() {
		h.runQuiet("kubectl", append([]string{"delete", "crd", "--ignore-not-found"},
			gatewayAPICRDs...)...)
	})
}

func (h *harness) applyFixtures() {
	h.t.Helper()
	h.t.Log("applying fixtures")
	fixtures := filepath.Join(h.root, fixturesPath)
	h.kubectl("apply", "-f", fixtures)
	h.t.Cleanup(func() {
		h.runQuiet("kubectl", "delete", "-f", fixtures, "--ignore-not-found")
	})
}

func (h *harness) buildSidecar() {
	h.t.Helper()
	h.t.Log("building gatus-sidecar")
	h.run("go", "build", "-o", h.binPath, "./cmd/gatus-sidecar")
}

func (h *harness) startSidecar(args ...string) {
	h.t.Helper()
	all := append([]string{
		"--output=" + h.outPath,
		"--default-interval=" + defaultProbeInterval,
	}, args...)
	cmd := exec.CommandContext(h.ctx, h.binPath, all...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		h.t.Fatalf("start sidecar: %v", err)
	}
	h.sidecar = cmd
}

// endpoints returns the YAML the sidecar has written so far, keyed by name.
// A missing file yields an empty map; a malformed file fails the test.
func (h *harness) endpoints() map[string]endpointDoc {
	h.t.Helper()
	data, err := os.ReadFile(h.outPath)
	if err != nil {
		return map[string]endpointDoc{}
	}
	var doc struct {
		Endpoints []endpointDoc `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		h.t.Fatalf("parse %s: %v\n---\n%s", h.outPath, err, data)
	}
	out := make(map[string]endpointDoc, len(doc.Endpoints))
	for _, e := range doc.Endpoints {
		out[e.Name] = e
	}
	return out
}

func (h *harness) waitForEndpointURL(name, url string) {
	h.t.Helper()
	waitUntil(h.t, defaultStepTimeout, func() error {
		ep, ok := h.endpoints()[name]
		switch {
		case !ok:
			return fmt.Errorf("endpoint %q not yet present", name)
		case ep.URL != url:
			return fmt.Errorf("endpoint %q url = %q, want %q", name, ep.URL, url)
		}
		return nil
	})
}

func (h *harness) waitForEndpointAbsent(name string) {
	h.t.Helper()
	waitUntil(h.t, defaultStepTimeout, func() error {
		if _, ok := h.endpoints()[name]; ok {
			return fmt.Errorf("endpoint %q still present", name)
		}
		return nil
	})
}

func (h *harness) kubectl(args ...string) {
	h.t.Helper()
	h.run("kubectl", args...)
}

// run streams output to stderr and fails the test on non-zero exit. The
// harness context controls cancellation.
func (h *harness) run(name string, args ...string) {
	h.t.Helper()
	cmd := exec.CommandContext(h.ctx, name, args...)
	cmd.Dir = h.root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		h.t.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
}

// runQuiet swallows both output and any error — for best-effort cleanup
// steps that must not fail the test.
func (*harness) runQuiet(name string, args ...string) {
	_ = exec.Command(name, args...).Run()
}

func waitUntil(t *testing.T, timeout time.Duration, check func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if last = check(); last == nil {
			return
		}
		time.Sleep(pollInterval)
	}
	if last == nil {
		last = errors.New("timed out")
	}
	t.Fatalf("condition not satisfied within %s: %v", timeout, last)
}

func goModRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		t.Fatalf("not in a Go module")
	}
	return filepath.Dir(gomod)
}

func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
