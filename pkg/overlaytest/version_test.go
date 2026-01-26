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

func TestGetVersionPriority(t *testing.T) {
	// Save and restore original state
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

	t.Run("Priority: ENV var > ldflags > default", func(t *testing.T) {
		// Set ldflags version
		Version = "1.0.7"
		os.Unsetenv("APP_VERSION")

		// Should use ldflags version
		if got := GetVersion(); got != "1.0.7" {
			t.Errorf("Expected ldflags version 1.0.7, got %s", got)
		}

		// Set env var version - should override ldflags
		os.Setenv("APP_VERSION", "2.0.0")
		if got := GetVersion(); got != "2.0.0" {
			t.Errorf("Expected env var to override ldflags, got %s", got)
		}
	})

	t.Run("Empty env var falls back to ldflags", func(t *testing.T) {
		Version = "1.0.7"
		os.Setenv("APP_VERSION", "")

		// Empty env var should not be used
		result := GetVersion()
		if result == "" {
			t.Error("Expected non-empty version when env var is empty")
		}
	})
}

func TestVersionEdgeCases(t *testing.T) {
	oldEnv := os.Getenv("APP_VERSION")
	defer func() {
		if oldEnv != "" {
			os.Setenv("APP_VERSION", oldEnv)
		} else {
			os.Unsetenv("APP_VERSION")
		}
	}()

	t.Run("Version with special characters", func(t *testing.T) {
		testVersion := "v1.0.0-rc1+build.123"
		os.Setenv("APP_VERSION", testVersion)

		if got := GetVersion(); got != testVersion {
			t.Errorf("Expected version with special chars %s, got %s", testVersion, got)
		}
	})

	t.Run("Very long version string", func(t *testing.T) {
		longVersion := "1.0.0-alpha.beta.gamma.delta.epsilon.zeta.eta.theta.iota.kappa"
		os.Setenv("APP_VERSION", longVersion)

		if got := GetVersion(); got != longVersion {
			t.Errorf("Expected long version, got %s", got)
		}
	})

	t.Run("Version with whitespace", func(t *testing.T) {
		versionWithSpaces := " 1.0.0 "
		os.Setenv("APP_VERSION", versionWithSpaces)

		// Function doesn't trim, returns as-is
		if got := GetVersion(); got != versionWithSpaces {
			t.Errorf("Expected version with whitespace %q, got %q", versionWithSpaces, got)
		}
	})

	t.Run("Version with unicode", func(t *testing.T) {
		unicodeVersion := "1.0.0-β.release"
		os.Setenv("APP_VERSION", unicodeVersion)

		if got := GetVersion(); got != unicodeVersion {
			t.Errorf("Expected unicode version %s, got %s", unicodeVersion, got)
		}
	})

	t.Run("Numeric-only version", func(t *testing.T) {
		numericVersion := "100"
		os.Setenv("APP_VERSION", numericVersion)

		if got := GetVersion(); got != numericVersion {
			t.Errorf("Expected numeric version %s, got %s", numericVersion, got)
		}
	})
}

func TestVersionConsistency(t *testing.T) {
	oldEnv := os.Getenv("APP_VERSION")
	defer func() {
		if oldEnv != "" {
			os.Setenv("APP_VERSION", oldEnv)
		} else {
			os.Unsetenv("APP_VERSION")
		}
	}()

	t.Run("Multiple calls return same value", func(t *testing.T) {
		os.Setenv("APP_VERSION", "test-version")

		version1 := GetVersion()
		version2 := GetVersion()
		version3 := GetVersion()

		if version1 != version2 || version2 != version3 {
			t.Error("GetVersion should return consistent values across calls")
		}
	})

	t.Run("Changing env var affects result", func(t *testing.T) {
		os.Setenv("APP_VERSION", "1.0.0")
		version1 := GetVersion()

		os.Setenv("APP_VERSION", "2.0.0")
		version2 := GetVersion()

		if version1 == version2 {
			t.Error("Expected version to change when env var changes")
		}
	})
}

func TestVersionDefault(t *testing.T) {
	t.Run("Default version is set", func(t *testing.T) {
		if Version == "" {
			t.Error("Default Version constant should not be empty")
		}
	})

	t.Run("Default version format", func(t *testing.T) {
		// Version should follow semantic versioning (loosely)
		if len(Version) < 3 {
			t.Errorf("Version %s seems too short", Version)
		}
	})
}
