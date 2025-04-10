package monitoring

import (
	"testing"
	"time"

	"github.com/exo7-ca/k8s-http-monitor/pkg/discovery"
)

// TestWithCheckInterval tests the WithCheckInterval option
func TestWithCheckInterval(t *testing.T) {
	interval := 60 * time.Second
	option := WithCheckInterval(interval)

	// Create a monitor with default values
	m := &Monitor{
		checkInterval: 30 * time.Second,
	}

	// Apply the option
	option(m)

	// Check that the interval was set correctly
	if m.checkInterval != interval {
		t.Errorf("WithCheckInterval(%v) did not set checkInterval correctly, got %v", interval, m.checkInterval)
	}
}

// TestWithTimeout tests the WithTimeout option
func TestWithTimeout(t *testing.T) {
	timeout := 5 * time.Second
	option := WithTimeout(timeout)

	// Create a monitor with default values
	m := &Monitor{
		timeout: 10 * time.Second,
	}

	// Apply the option
	option(m)

	// Check that the timeout was set correctly
	if m.timeout != timeout {
		t.Errorf("WithTimeout(%v) did not set timeout correctly, got %v", timeout, m.timeout)
	}
}

// TestWithSuccessStatusCodes tests the WithSuccessStatusCodes option
func TestWithSuccessStatusCodes(t *testing.T) {
	codes := []int{200, 201}
	option := WithSuccessStatusCodes(codes)

	// Create a monitor with default values
	m := &Monitor{
		successStatusCodes: []int{401, 403, 404},
	}

	// Apply the option
	option(m)

	// Check that the codes were set correctly
	if len(m.successStatusCodes) != len(codes) {
		t.Errorf("WithSuccessStatusCodes(%v) did not set successStatusCodes correctly, got %v", codes, m.successStatusCodes)
	}
	for i, code := range codes {
		if m.successStatusCodes[i] != code {
			t.Errorf("WithSuccessStatusCodes(%v) did not set successStatusCodes correctly, got %v", codes, m.successStatusCodes)
		}
	}
}

// TestCheckStatus tests the checkStatus function
func TestCheckStatus(t *testing.T) {
	// Create monitor with custom success status codes
	monitor := &Monitor{
		successStatusCodes: []int{401, 403, 404},
	}

	// Test cases
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{200, true},  // 2xx is always success
		{299, true},  // 2xx is always success
		{300, false}, // 3xx is not success by default
		{401, true},  // Custom success code
		{403, true},  // Custom success code
		{404, true},  // Custom success code
		{500, false}, // Not a success code
	}

	for _, tc := range testCases {
		result := monitor.checkStatus(tc.statusCode)
		if result != tc.expected {
			t.Errorf("checkStatus(%d) = %v, expected %v", tc.statusCode, result, tc.expected)
		}
	}
}

// TestEndpointKey tests the endpointKey function
func TestEndpointKey(t *testing.T) {
	// Create a test endpoint
	endpoint := discovery.Endpoint{
		Namespace:   "default",
		ServiceName: "service",
		IngressName: "ingress",
		URL:         "http://example.com",
		Path:        "/health",
	}

	// Get the key
	key := endpointKey(endpoint)
	expected := "default/ingress/http://example.com/health"

	// Check the key
	if key != expected {
		t.Errorf("endpointKey() = %v, expected %v", key, expected)
	}
}

// TestCheckEndpointHTTP tests the checkEndpoint function with HTTP responses
func TestCheckEndpointHTTP(t *testing.T) {
	// Skip this test as it requires a real metrics provider
	t.Skip("Skipping test that requires a real metrics provider")
}
