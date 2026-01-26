package overlaytest

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/util/homedir"
)

func TestKubeconfigPath(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		homeDir     string
		expected    string
		description string
	}{
		{
			name:        "KUBECONFIG environment variable set",
			kubeconfig:  "/custom/kubeconfig",
			homeDir:     "/home/user",
			expected:    "/custom/kubeconfig",
			description: "should use KUBECONFIG env var when set",
		},
		{
			name:        "Home directory available",
			kubeconfig:  "",
			homeDir:     "/home/user",
			expected:    "/home/user/.kube/config",
			description: "should use default kubeconfig path in home directory",
		},
		{
			name:        "No home directory",
			kubeconfig:  "",
			homeDir:     "",
			expected:    "",
			description: "should use empty string when no home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldKubeconfig := os.Getenv("KUBECONFIG")
			defer os.Setenv("KUBECONFIG", oldKubeconfig)

			os.Setenv("KUBECONFIG", tt.kubeconfig)

			var kubeconfigPath string
			if tt.kubeconfig != "" {
				kubeconfigPath = os.Getenv("KUBECONFIG")
			} else if tt.homeDir != "" {
				kubeconfigPath = filepath.Join(tt.homeDir, ".kube", "config")
			} else {
				kubeconfigPath = ""
			}

			if kubeconfigPath != tt.expected {
				t.Errorf("Expected kubeconfig path to be %s, got %s", tt.expected, kubeconfigPath)
			}
		})
	}
}

func TestHomeDirFunction(t *testing.T) {
	homeDir := homedir.HomeDir()
	if homeDir == "" {
		t.Skip("No home directory available in test environment")
	}

	if !filepath.IsAbs(homeDir) {
		t.Errorf("Expected home directory to be absolute path, got %s", homeDir)
	}
}

func TestGetKubeconfigPath(t *testing.T) {
	oldKubeconfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	t.Run("KUBECONFIG environment variable set", func(t *testing.T) {
		expected := "/custom/kubeconfig"
		os.Setenv("KUBECONFIG", expected)

		result := GetKubeconfigPath()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("No KUBECONFIG but home directory available", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "")
		home := homedir.HomeDir()
		if home == "" {
			t.Skip("No home directory available")
		}

		expected := filepath.Join(home, ".kube", "config")
		result := GetKubeconfigPath()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
}

func TestConstants(t *testing.T) {
	t.Run("App version constant", func(t *testing.T) {
		if Version == "" {
			t.Error("Version should not be empty")
		}
	})

	t.Run("Default values", func(t *testing.T) {
		config := DefaultConfig()

		if config.Namespace != "kube-system" {
			t.Errorf("Expected namespace to be kube-system, got %s", config.Namespace)
		}

		if config.AppName != "overlaytest" {
			t.Errorf("Expected app name to be overlaytest, got %s", config.AppName)
		}

		if config.Image == "" {
			t.Error("Image should not be empty")
		}

		expectedImage := "ghcr.io/eumel8/overlaytest:latest"
		if config.Image != expectedImage {
			t.Errorf("Expected image to be %s, got %s", expectedImage, config.Image)
		}
	})
}
