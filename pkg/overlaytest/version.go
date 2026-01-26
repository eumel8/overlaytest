package overlaytest

import "os"

// Version is the application version
// Can be set at build time using ldflags:
//   -ldflags "-X github.com/eumel8/overlaytest/pkg/overlaytest.Version=x.y.z"
// Can also be overridden at runtime via APP_VERSION environment variable
var Version = "1.0.6" // Default version

// GetVersion returns the application version
// Priority: 1. APP_VERSION env var, 2. Build-time ldflags, 3. Default
func GetVersion() string {
	if envVersion := os.Getenv("APP_VERSION"); envVersion != "" {
		return envVersion
	}
	return Version
}
