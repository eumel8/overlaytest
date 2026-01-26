package overlaytest

import (
	"os"
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Save original
	oldVersion := Version
	oldEnv := os.Getenv("APP_VERSION")
	defer func() {
		Version = oldVersion
		if oldEnv != "" {
			os.Setenv("APP_VERSION", oldEnv)
		} else {
			os.Unsetenv("APP_VERSION")
		}
	}()

	// Test default version
	Version = "1.0.6"
	os.Unsetenv("APP_VERSION")
	if got := GetVersion(); got != "1.0.6" {
		t.Errorf("Expected version 1.0.6, got %s", got)
	}

	// Test environment variable override
	os.Setenv("APP_VERSION", "2.0.0")

	if got := GetVersion(); got != "2.0.0" {
		t.Errorf("Expected version 2.0.0 from env var, got %s", got)
	}

	// Test ldflags injection (simulated)
	os.Unsetenv("APP_VERSION")
	Version = "1.0.7"
	if got := GetVersion(); got != "1.0.7" {
		t.Errorf("Expected version 1.0.7 from ldflags, got %s", got)
	}
}

func TestAppVersion(t *testing.T) {
	expected := "1.0.6"
	if Version != expected {
		t.Errorf("Expected Version to be %s, got %s", expected, Version)
	}
}
