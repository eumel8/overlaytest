package overlaytest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewKubernetesClient(t *testing.T) {
	t.Run("Invalid kubeconfig path", func(t *testing.T) {
		_, _, err := NewKubernetesClient("/nonexistent/path/to/kubeconfig")
		if err == nil {
			t.Error("Expected error for nonexistent kubeconfig path")
		}
	})

	t.Run("Empty kubeconfig path", func(t *testing.T) {
		_, _, err := NewKubernetesClient("")
		if err == nil {
			t.Error("Expected error for empty kubeconfig path")
		}
	})

	t.Run("Invalid kubeconfig content", func(t *testing.T) {
		// Create a temporary invalid kubeconfig file
		tmpDir := t.TempDir()
		invalidConfig := filepath.Join(tmpDir, "invalid-kubeconfig")

		err := os.WriteFile(invalidConfig, []byte("invalid yaml content {{{"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, _, err = NewKubernetesClient(invalidConfig)
		if err == nil {
			t.Error("Expected error for invalid kubeconfig content")
		}
	})

	t.Run("Valid but empty kubeconfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyConfig := filepath.Join(tmpDir, "empty-kubeconfig")

		// Create a valid YAML but empty kubeconfig
		err := os.WriteFile(emptyConfig, []byte("apiVersion: v1\nkind: Config\n"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, _, err = NewKubernetesClient(emptyConfig)
		if err == nil {
			t.Error("Expected error for empty kubeconfig (no clusters/contexts)")
		}
	})
}

func TestNewKubernetesClientErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigPath string
		expectError    bool
	}{
		{
			name:           "Directory instead of file",
			kubeconfigPath: t.TempDir(),
			expectError:    true,
		},
		{
			name:           "File with no permissions",
			kubeconfigPath: "/root/no-access-file",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NewKubernetesClient(tt.kubeconfigPath)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect error for %s, got: %v", tt.name, err)
			}
		})
	}
}
