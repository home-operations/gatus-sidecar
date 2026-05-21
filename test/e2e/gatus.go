//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

const (
	gatusImage         = "ghcr.io/twin/gatus:latest"
	gatusManifestPath  = "test/e2e/fixtures/gatus.yaml"
	gatusNamespace     = "gatus"
	gatusPodName       = "gatus"
	gatusConfigMap     = "gatus-config"
	gatusContainerPort = 8080
	gatusReadyTimeout  = 2 * time.Minute

	httpProbeTimeout = 2 * time.Second
)

// gatusProbe is a Gatus pod running in-cluster, reached over a port-forward.
type gatusProbe struct {
	t      *testing.T
	base   string
	cancel context.CancelFunc
	pf     *exec.Cmd
}

func (h *harness) deployGatus(cfgPath string) *gatusProbe {
	h.t.Helper()
	h.t.Log("deploying Gatus in cluster")

	h.run("docker", "pull", gatusImage)
	h.run("kind", "load", "docker-image", gatusImage, "--name", clusterName)

	// Reset namespace so reruns are deterministic.
	h.runQuiet("kubectl", "delete", "namespace", gatusNamespace, "--ignore-not-found", "--wait=true")

	manifest := filepath.Join(h.root, gatusManifestPath)
	h.kubectl("apply", "-f", manifest)
	h.t.Cleanup(func() {
		h.runQuiet("kubectl", "delete", "namespace", gatusNamespace, "--ignore-not-found", "--wait=false")
	})

	// The pod's configmap volume blocks until this exists; kubelet retries.
	h.kubectl("-n", gatusNamespace, "create", "configmap", gatusConfigMap,
		"--from-file=config.yaml="+cfgPath)
	h.kubectl("-n", gatusNamespace, "wait", "pod/"+gatusPodName,
		"--for=condition=Ready", "--timeout="+gatusReadyTimeout.String())

	port, err := freeTCPPort()
	if err != nil {
		h.t.Fatalf("free port: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	pf := exec.CommandContext(ctx, "kubectl", "-n", gatusNamespace, "port-forward",
		"pod/"+gatusPodName, fmt.Sprintf("%d:%d", port, gatusContainerPort))
	pf.Stdout = os.Stderr
	pf.Stderr = os.Stderr
	if err := pf.Start(); err != nil {
		cancel()
		h.t.Fatalf("start port-forward: %v", err)
	}

	g := &gatusProbe{
		t:      h.t,
		base:   fmt.Sprintf("http://127.0.0.1:%d", port),
		cancel: cancel,
		pf:     pf,
	}
	h.t.Cleanup(g.close)

	waitUntil(h.t, gatusReadyTimeout, func() error {
		return g.httpOK("/health")
	})
	return g
}

func (g *gatusProbe) close() {
	g.cancel()
	_ = g.pf.Wait()
}

// expectEndpoints polls /api/v1/endpoints/statuses until every name in want
// is present.
func (g *gatusProbe) expectEndpoints(want ...string) {
	g.t.Helper()
	waitUntil(g.t, defaultStepTimeout, func() error {
		names, err := g.endpointNames()
		if err != nil {
			return err
		}
		var missing []string
		for _, w := range want {
			if !slices.Contains(names, w) {
				missing = append(missing, w)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("missing endpoints %v (got %v)", missing, names)
		}
		return nil
	})
}

func (g *gatusProbe) httpOK(path string) error {
	resp, err := (&http.Client{Timeout: httpProbeTimeout}).Get(g.base + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
	return nil
}

func (g *gatusProbe) endpointNames() ([]string, error) {
	resp, err := (&http.Client{Timeout: httpProbeTimeout}).Get(g.base + "/api/v1/endpoints/statuses")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var statuses []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	names := make([]string, 0, len(statuses))
	for _, s := range statuses {
		names = append(names, s.Name)
	}
	return names, nil
}
