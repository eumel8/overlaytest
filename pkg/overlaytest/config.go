package overlaytest

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

// Config holds the application configuration
type Config struct {
	Namespace  string
	AppName    string
	Image      string
	Kubeconfig string
	Reuse      bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Namespace: "kube-system",
		AppName:   "overlaytest",
		// Default image: minimal Alpine-based image with bash and ping (~10MB compressed)
		// Previous image (deprecated): mtr.devops.telekom.de/mcsps/swiss-army-knife:latest
		Image: "ghcr.io/eumel8/overlaytest:main",
	}
}

// GetKubeconfigPath returns the kubeconfig path
// Priority: 1. KUBECONFIG env var, 2. ~/.kube/config, 3. empty string
func GetKubeconfigPath() string {
	if os.Getenv("KUBECONFIG") != "" {
		return os.Getenv("KUBECONFIG")
	} else if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	} else {
		return ""
	}
}
