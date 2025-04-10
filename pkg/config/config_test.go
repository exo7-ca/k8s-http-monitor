package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear any environment variables that might affect the test
	os.Unsetenv(EnvMonitoringInterval)
	os.Unsetenv(EnvMetricsInterval)
	os.Unsetenv(EnvOtelCollectorURL)
	os.Unsetenv(EnvSuccessStatusCodes)
	os.Unsetenv(EnvNamespaceMode)
	os.Unsetenv(EnvNamespaces)

	// Create a temporary file with a non-existent path
	tempFile := "nonexistent_config.yaml"


	// Set to our non-existent file
	os.Setenv("CONFIG_FILE", tempFile)

	// Restore after test
	defer func() {
		os.Unsetenv("CONFIG_FILE")
	}()

	// Load the config, which should use defaults
	cfg, err := LoadConfig()

	// Check that there was no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the defaults were used
	if cfg.MonitoringInterval != DefaultMonitoringInterval {
		t.Errorf("Expected monitoring interval %v, got %v", DefaultMonitoringInterval, cfg.MonitoringInterval)
	}

	if cfg.MetricsInterval != DefaultMetricsInterval {
		t.Errorf("Expected metrics interval %v, got %v", DefaultMetricsInterval, cfg.MetricsInterval)
	}

	if cfg.OtelCollectorURL != DefaultOtelCollectorURL {
		t.Errorf("Expected OTEL collector URL %s, got %s", DefaultOtelCollectorURL, cfg.OtelCollectorURL)
	}

	if len(cfg.SuccessStatusCodes) != len(DefaultSuccessStatusCodes) {
		t.Errorf("Expected %d success status codes, got %d", len(DefaultSuccessStatusCodes), len(cfg.SuccessStatusCodes))
	}

	if cfg.NamespaceMode != DefaultNamespaceMode {
		t.Errorf("Expected namespace mode %s, got %s", DefaultNamespaceMode, cfg.NamespaceMode)
	}

	if len(cfg.Namespaces) != 0 {
		t.Errorf("Expected empty namespaces, got %v", cfg.Namespaces)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Create a non-existent file path
	tempFile := "nonexistent_config.yaml"

	// Set to our non-existent file
	os.Setenv("CONFIG_FILE", tempFile)

	// Set environment variables
	os.Setenv(EnvMonitoringInterval, "60")
	os.Setenv(EnvMetricsInterval, "20")
	os.Setenv(EnvOtelCollectorURL, "test-collector:4317")
	os.Setenv(EnvSuccessStatusCodes, "401, 403, 404, 500")
	os.Setenv(EnvNamespaceMode, "deny")
	os.Setenv(EnvNamespaces, "default, kube-system")

	// Clean up after the test
	defer func() {
		os.Unsetenv("CONFIG_FILE")
		os.Unsetenv(EnvMonitoringInterval)
		os.Unsetenv(EnvMetricsInterval)
		os.Unsetenv(EnvOtelCollectorURL)
		os.Unsetenv(EnvSuccessStatusCodes)
		os.Unsetenv(EnvNamespaceMode)
		os.Unsetenv(EnvNamespaces)
	}()

	// Load the config
	cfg, err := LoadConfig()

	// Check that there was no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the environment variables were used
	if cfg.MonitoringInterval != 60*time.Second {
		t.Errorf("Expected monitoring interval %v, got %v", 60*time.Second, cfg.MonitoringInterval)
	}

	if cfg.MetricsInterval != 20*time.Second {
		t.Errorf("Expected metrics interval %v, got %v", 20*time.Second, cfg.MetricsInterval)
	}

	if cfg.OtelCollectorURL != "test-collector:4317" {
		t.Errorf("Expected OTEL collector URL %s, got %s", "test-collector:4317", cfg.OtelCollectorURL)
	}

	expectedCodes := []int{401, 403, 404, 500}
	if len(cfg.SuccessStatusCodes) != len(expectedCodes) {
		t.Errorf("Expected %d success status codes, got %d", len(expectedCodes), len(cfg.SuccessStatusCodes))
	} else {
		for i, code := range expectedCodes {
			if i < len(cfg.SuccessStatusCodes) && cfg.SuccessStatusCodes[i] != code {
				t.Errorf("Expected status code %d at index %d, got %d", code, i, cfg.SuccessStatusCodes[i])
			}
		}
	}

	if cfg.NamespaceMode != "deny" {
		t.Errorf("Expected namespace mode %s, got %s", "deny", cfg.NamespaceMode)
	}

	expectedNamespaces := []string{"default", "kube-system"}
	if len(cfg.Namespaces) != len(expectedNamespaces) {
		t.Errorf("Expected %d namespaces, got %d", len(expectedNamespaces), len(cfg.Namespaces))
	} else {
		for i, ns := range expectedNamespaces {
			if i < len(cfg.Namespaces) && cfg.Namespaces[i] != ns {
				t.Errorf("Expected namespace %s at index %d, got %s", ns, i, cfg.Namespaces[i])
			}
		}
	}
}

func TestLoadConfigInvalidEnv(t *testing.T) {
	// Create a non-existent file path
	tempFile := "nonexistent_config.yaml"

	// Set to our non-existent file
	os.Setenv("CONFIG_FILE", tempFile)

	// Set invalid environment variables
	os.Setenv(EnvMonitoringInterval, "invalid")
	os.Setenv(EnvMetricsInterval, "invalid")
	os.Setenv(EnvSuccessStatusCodes, "invalid, codes")
	os.Setenv(EnvNamespaceMode, "invalid")

	// Clean up after the test
	defer func() {
		os.Unsetenv("CONFIG_FILE")
		os.Unsetenv(EnvMonitoringInterval)
		os.Unsetenv(EnvMetricsInterval)
		os.Unsetenv(EnvSuccessStatusCodes)
		os.Unsetenv(EnvNamespaceMode)
	}()

	// Load the config
	cfg, err := LoadConfig()

	// Check that there was no error (should fall back to defaults)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the defaults were used
	if cfg.MonitoringInterval != DefaultMonitoringInterval {
		t.Errorf("Expected monitoring interval %v, got %v", DefaultMonitoringInterval, cfg.MonitoringInterval)
	}

	if cfg.MetricsInterval != DefaultMetricsInterval {
		t.Errorf("Expected metrics interval %v, got %v", DefaultMetricsInterval, cfg.MetricsInterval)
	}

	if len(cfg.SuccessStatusCodes) != len(DefaultSuccessStatusCodes) {
		t.Errorf("Expected %d success status codes, got %d", len(DefaultSuccessStatusCodes), len(cfg.SuccessStatusCodes))
	}

	if cfg.NamespaceMode != DefaultNamespaceMode {
		t.Errorf("Expected namespace mode %s, got %s", DefaultNamespaceMode, cfg.NamespaceMode)
	}
}

func TestLoadConfigWithFile(t *testing.T) {
	// Save the original config file if it exists
	originalExists := false
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		originalExists = true
		// Rename the original file
		if err := os.Rename(DefaultConfigFile, DefaultConfigFile+".bak"); err != nil {
			t.Fatalf("Failed to backup original config file: %v", err)
		}
		defer func() {
			// Restore the original file
			if err := os.Rename(DefaultConfigFile+".bak", DefaultConfigFile); err != nil {
				t.Fatalf("Failed to restore original config file: %v", err)
			}
		}()
	}

	// Create a temporary config file with the default name
	content := `monitoring:
  interval: 45
  successStatusCodes: [200, 201, 401, 403, 404]
metrics:
  interval: 15
  otelCollectorURL: "file-collector:4317"
discovery:
  namespaceMode: "deny"
  namespaces: ["test1", "test2"]
`
	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}
	defer func() {
		// Remove the temporary file if no original existed
		if !originalExists {
			os.Remove(DefaultConfigFile)
		}
	}()

	// Clear any environment variables that might affect the test
	os.Unsetenv(EnvMonitoringInterval)
	os.Unsetenv(EnvMetricsInterval)
	os.Unsetenv(EnvOtelCollectorURL)
	os.Unsetenv(EnvSuccessStatusCodes)
	os.Unsetenv(EnvNamespaceMode)
	os.Unsetenv(EnvNamespaces)

	// Load the config
	cfg, err := LoadConfig()

	// Check that there was no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the file values were used
	if cfg.MonitoringInterval != 45*time.Second {
		t.Errorf("Expected monitoring interval %v, got %v", 45*time.Second, cfg.MonitoringInterval)
	}

	if cfg.MetricsInterval != 15*time.Second {
		t.Errorf("Expected metrics interval %v, got %v", 15*time.Second, cfg.MetricsInterval)
	}

	if cfg.OtelCollectorURL != "file-collector:4317" {
		t.Errorf("Expected OTEL collector URL %s, got %s", "file-collector:4317", cfg.OtelCollectorURL)
	}

	expectedCodes := []int{200, 201, 401, 403, 404}
	if len(cfg.SuccessStatusCodes) != len(expectedCodes) {
		t.Errorf("Expected %d success status codes, got %d", len(expectedCodes), len(cfg.SuccessStatusCodes))
	} else {
		for i, code := range expectedCodes {
			if i < len(cfg.SuccessStatusCodes) && cfg.SuccessStatusCodes[i] != code {
				t.Errorf("Expected status code %d at index %d, got %d", code, i, cfg.SuccessStatusCodes[i])
			}
		}
	}

	if cfg.NamespaceMode != "deny" {
		t.Errorf("Expected namespace mode %s, got %s", "deny", cfg.NamespaceMode)
	}

	expectedNamespaces := []string{"test1", "test2"}
	if len(cfg.Namespaces) != len(expectedNamespaces) {
		t.Errorf("Expected %d namespaces, got %d", len(expectedNamespaces), len(cfg.Namespaces))
	} else {
		for i, ns := range expectedNamespaces {
			if i < len(cfg.Namespaces) && cfg.Namespaces[i] != ns {
				t.Errorf("Expected namespace %s at index %d, got %s", ns, i, cfg.Namespaces[i])
			}
		}
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	// Save the original config file if it exists
	originalExists := false
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		originalExists = true
		// Rename the original file
		if err := os.Rename(DefaultConfigFile, DefaultConfigFile+".bak"); err != nil {
			t.Fatalf("Failed to backup original config file: %v", err)
		}
		defer func() {
			// Restore the original file
			if err := os.Rename(DefaultConfigFile+".bak", DefaultConfigFile); err != nil {
				t.Fatalf("Failed to restore original config file: %v", err)
			}
		}()
	}

	// Create a temporary config file with the default name
	content := `monitoring:
  interval: 45
  successStatusCodes: [200, 201]
metrics:
  interval: 15
  otelCollectorURL: "file-collector:4317"
discovery:
  namespaceMode: "deny"
  namespaces: ["test1", "test2"]
`
	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}
	defer func() {
		// Remove the temporary file if no original existed
		if !originalExists {
			os.Remove(DefaultConfigFile)
		}
	}()

	// Set environment variables that should override the file
	os.Setenv(EnvMonitoringInterval, "60")
	os.Setenv(EnvMetricsInterval, "20")
	os.Setenv(EnvOtelCollectorURL, "env-collector:4317")
	os.Setenv(EnvSuccessStatusCodes, "401, 403, 404")
	os.Setenv(EnvNamespaceMode, "allow")
	os.Setenv(EnvNamespaces, "default, kube-system")

	// Clean up after the test
	defer func() {
		os.Unsetenv("CONFIG_FILE")
		os.Unsetenv(EnvMonitoringInterval)
		os.Unsetenv(EnvMetricsInterval)
		os.Unsetenv(EnvOtelCollectorURL)
		os.Unsetenv(EnvSuccessStatusCodes)
		os.Unsetenv(EnvNamespaceMode)
		os.Unsetenv(EnvNamespaces)
	}()

	// Load the config
	cfg, err := LoadConfig()

	// Check that there was no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the environment variables overrode the file values
	if cfg.MonitoringInterval != 60*time.Second {
		t.Errorf("Expected monitoring interval %v, got %v", 60*time.Second, cfg.MonitoringInterval)
	}

	if cfg.MetricsInterval != 20*time.Second {
		t.Errorf("Expected metrics interval %v, got %v", 20*time.Second, cfg.MetricsInterval)
	}

	if cfg.OtelCollectorURL != "env-collector:4317" {
		t.Errorf("Expected OTEL collector URL %s, got %s", "env-collector:4317", cfg.OtelCollectorURL)
	}

	expectedCodes := []int{401, 403, 404}
	if len(cfg.SuccessStatusCodes) != len(expectedCodes) {
		t.Errorf("Expected %d success status codes, got %d", len(expectedCodes), len(cfg.SuccessStatusCodes))
	} else {
		for i, code := range expectedCodes {
			if i < len(cfg.SuccessStatusCodes) && cfg.SuccessStatusCodes[i] != code {
				t.Errorf("Expected status code %d at index %d, got %d", code, i, cfg.SuccessStatusCodes[i])
			}
		}
	}

	if cfg.NamespaceMode != "allow" {
		t.Errorf("Expected namespace mode %s, got %s", "allow", cfg.NamespaceMode)
	}

	expectedNamespaces := []string{"default", "kube-system"}
	if len(cfg.Namespaces) != len(expectedNamespaces) {
		t.Errorf("Expected %d namespaces, got %d", len(expectedNamespaces), len(cfg.Namespaces))
	} else {
		for i, ns := range expectedNamespaces {
			if i < len(cfg.Namespaces) && cfg.Namespaces[i] != ns {
				t.Errorf("Expected namespace %s at index %d, got %s", ns, i, cfg.Namespaces[i])
			}
		}
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	// Save the original config file if it exists
	originalExists := false
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		originalExists = true
		// Rename the original file
		if err := os.Rename(DefaultConfigFile, DefaultConfigFile+".bak"); err != nil {
			t.Fatalf("Failed to backup original config file: %v", err)
		}
		defer func() {
			// Restore the original file
			if err := os.Rename(DefaultConfigFile+".bak", DefaultConfigFile); err != nil {
				t.Fatalf("Failed to restore original config file: %v", err)
			}
		}()
	}

	// Create a temporary config file with invalid YAML
	content := `this is not valid yaml`

	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}
	defer func() {
		// Remove the temporary file if no original existed
		if !originalExists {
			os.Remove(DefaultConfigFile)
		}
	}()

	// Clean up after the test
	defer func() {
		os.Unsetenv("CONFIG_FILE")
	}()

	// Load the config
	_, err := LoadConfig()

	// Check that there was an error
	if err == nil {
		t.Fatalf("Expected error for invalid YAML, got nil")
	}
}
