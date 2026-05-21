package k8s

import (
	"fmt"
	"log/slog"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RestConfig prefers in-cluster config and falls back to the standard
// kubeconfig loading rules.
func RestConfig() (*rest.Config, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
		slog.Info("loaded kubernetes config", "source", "in-cluster")
		return cfg, nil
	}

	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	slog.Info("loaded kubernetes config", "source", "kubeconfig")
	return cfg, nil
}
