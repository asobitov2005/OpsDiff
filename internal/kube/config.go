package kube

import (
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func ResolveKubeconfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}

	if configured := os.Getenv("KUBECONFIG"); configured != "" {
		parts := strings.Split(configured, string(os.PathListSeparator))
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".kube/config"
	}

	return filepath.Join(home, ".kube", "config")
}

func LoadRESTConfig(explicit string) (*rest.Config, *clientcmdapi.Config, string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicit != "" {
		rules.ExplicitPath = explicit
	}

	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})

	restConfig, err := loader.ClientConfig()
	if err != nil {
		return nil, nil, ResolveKubeconfigPath(explicit), err
	}

	rawConfig, err := loader.RawConfig()
	if err != nil {
		return nil, nil, ResolveKubeconfigPath(explicit), err
	}

	return restConfig, &rawConfig, ResolveKubeconfigPath(explicit), nil
}

func CurrentClusterName(config *clientcmdapi.Config) string {
	if config == nil {
		return ""
	}

	contextConfig, ok := config.Contexts[config.CurrentContext]
	if !ok || contextConfig == nil {
		return ""
	}

	return contextConfig.Cluster
}
