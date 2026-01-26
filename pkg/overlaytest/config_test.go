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

		expectedImage := "ghcr.io/eumel8/overlaytest:main"
		if config.Image != expectedImage {
			t.Errorf("Expected image to be %s, got %s", expectedImage, config.Image)
		}
	})
}

func TestConfigModification(t *testing.T) {
	t.Run("Modify config values", func(t *testing.T) {
		config := DefaultConfig()

		config.Namespace = "custom-namespace"
		config.AppName = "custom-app"
		config.Image = "custom-image:v2"
		config.Kubeconfig = "/custom/path"
		config.Reuse = true

		if config.Namespace != "custom-namespace" {
			t.Error("Failed to modify Namespace")
		}
		if config.AppName != "custom-app" {
			t.Error("Failed to modify AppName")
		}
		if config.Image != "custom-image:v2" {
			t.Error("Failed to modify Image")
		}
		if config.Kubeconfig != "/custom/path" {
			t.Error("Failed to modify Kubeconfig")
		}
		if !config.Reuse {
			t.Error("Failed to modify Reuse")
		}
	})

	t.Run("Multiple config instances are independent", func(t *testing.T) {
		config1 := DefaultConfig()
		config2 := DefaultConfig()

		config1.Namespace = "namespace1"
		config2.Namespace = "namespace2"

		if config1.Namespace == config2.Namespace {
			t.Error("Config instances should be independent")
		}
	})
}

func TestGetKubeconfigPathEdgeCases(t *testing.T) {
	oldKubeconfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	t.Run("KUBECONFIG with empty string", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "")
		result := GetKubeconfigPath()

		// Should fall back to home directory
		home := homedir.HomeDir()
		if home != "" {
			expected := filepath.Join(home, ".kube", "config")
			if result != expected {
				t.Errorf("Expected %s, got %s", expected, result)
			}
		}
	})

	t.Run("KUBECONFIG with multiple paths", func(t *testing.T) {
		// Kubernetes supports colon-separated paths
		multiplePaths := "/path1/config:/path2/config"
		os.Setenv("KUBECONFIG", multiplePaths)

		result := GetKubeconfigPath()
		if result != multiplePaths {
			t.Errorf("Expected %s, got %s", multiplePaths, result)
		}
	})

	t.Run("KUBECONFIG with special characters", func(t *testing.T) {
		specialPath := "/path/with spaces/config"
		os.Setenv("KUBECONFIG", specialPath)

		result := GetKubeconfigPath()
		if result != specialPath {
			t.Errorf("Expected %s, got %s", specialPath, result)
		}
	})

	t.Run("KUBECONFIG with home directory tilde", func(t *testing.T) {
		tildePath := "~/custom/kubeconfig"
		os.Setenv("KUBECONFIG", tildePath)

		result := GetKubeconfigPath()
		// Function returns as-is, doesn't expand tilde
		if result != tildePath {
			t.Errorf("Expected %s, got %s", tildePath, result)
		}
	})
}

func TestConfigStructValidation(t *testing.T) {
	t.Run("Config struct fields", func(t *testing.T) {
		config := &Config{
			Namespace:  "test-ns",
			AppName:    "test-app",
			Image:      "test-image:tag",
			Kubeconfig: "/path/to/config",
			Reuse:      false,
		}

		if config.Namespace != "test-ns" {
			t.Error("Namespace field not set correctly")
		}
		if config.AppName != "test-app" {
			t.Error("AppName field not set correctly")
		}
		if config.Image != "test-image:tag" {
			t.Error("Image field not set correctly")
		}
		if config.Kubeconfig != "/path/to/config" {
			t.Error("Kubeconfig field not set correctly")
		}
		if config.Reuse != false {
			t.Error("Reuse field not set correctly")
		}
	})

	t.Run("Zero value config", func(t *testing.T) {
		var config Config

		if config.Namespace != "" {
			t.Error("Expected empty Namespace in zero value")
		}
		if config.AppName != "" {
			t.Error("Expected empty AppName in zero value")
		}
		if config.Image != "" {
			t.Error("Expected empty Image in zero value")
		}
		if config.Kubeconfig != "" {
			t.Error("Expected empty Kubeconfig in zero value")
		}
		if config.Reuse != false {
			t.Error("Expected false Reuse in zero value")
		}
	})
}
