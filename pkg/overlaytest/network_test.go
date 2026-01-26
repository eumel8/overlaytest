package overlaytest

import (
	"testing"
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
