package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	MonitoringInterval time.Duration
	MetricsInterval    time.Duration
	OtelCollectorURL   string
	SuccessStatusCodes []int
	NamespaceMode      string // "allow" or "deny"
	Namespaces         []string
}

// ConfigFile represents the structure of the YAML config file
type ConfigFile struct {
	Monitoring struct {
		Interval           int   `yaml:"interval"`
		SuccessStatusCodes []int `yaml:"successStatusCodes"`
	} `yaml:"monitoring"`
	Metrics struct {
		Interval         int    `yaml:"interval"`
		OtelCollectorURL string `yaml:"otelCollectorURL"`
	} `yaml:"metrics"`
	Discovery struct {
		NamespaceMode string   `yaml:"namespaceMode"`
		Namespaces    []string `yaml:"namespaces"`
	} `yaml:"discovery"`
}

// Default configuration values
const (
	DefaultConfigFile         = "config.yaml"
	DefaultMonitoringInterval = 30 * time.Second
	DefaultMetricsInterval    = 10 * time.Second
	DefaultOtelCollectorURL   = "signoz-otel-collector:4317"
	DefaultNamespaceMode      = "allow" // "allow" means allow all namespaces by default
)

// Default success status codes (401, 403, 404 are considered successful by default)
var DefaultSuccessStatusCodes = []int{401, 403, 404}

// Environment variable names
const (
	EnvMonitoringInterval = "MONITOR_INTERVAL_SECONDS"
	EnvMetricsInterval    = "METRICS_INTERVAL_SECONDS"
	EnvOtelCollectorURL   = "OTEL_COLLECTOR_URL"
	EnvSuccessStatusCodes = "SUCCESS_STATUS_CODES"
	EnvNamespaceMode      = "NAMESPACE_MODE"
	EnvNamespaces         = "NAMESPACES"
)

// LoadConfig loads the configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// Set default configuration
	config := &Config{
		MonitoringInterval: DefaultMonitoringInterval,
		MetricsInterval:    DefaultMetricsInterval,
		OtelCollectorURL:   DefaultOtelCollectorURL,
		SuccessStatusCodes: DefaultSuccessStatusCodes,
		NamespaceMode:      DefaultNamespaceMode,
		Namespaces:         []string{},
	}

	// Try to read config file
	configFile := &ConfigFile{}
	configData, err := ioutil.ReadFile(DefaultConfigFile)
	if err == nil {
		if err := yaml.Unmarshal(configData, configFile); err != nil {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}

		// Apply config file values
		if configFile.Monitoring.Interval > 0 {
			config.MonitoringInterval = time.Duration(configFile.Monitoring.Interval) * time.Second
		}
		if len(configFile.Monitoring.SuccessStatusCodes) > 0 {
			config.SuccessStatusCodes = configFile.Monitoring.SuccessStatusCodes
		}
		if configFile.Metrics.Interval > 0 {
			config.MetricsInterval = time.Duration(configFile.Metrics.Interval) * time.Second
		}
		if configFile.Metrics.OtelCollectorURL != "" {
			config.OtelCollectorURL = configFile.Metrics.OtelCollectorURL
		}
		if configFile.Discovery.NamespaceMode != "" {
			config.NamespaceMode = configFile.Discovery.NamespaceMode
		}
		if len(configFile.Discovery.Namespaces) > 0 {
			config.Namespaces = configFile.Discovery.Namespaces
		}
	}

	// Override with environment variables if set
	if envInterval := os.Getenv(EnvMonitoringInterval); envInterval != "" {
		if seconds, err := strconv.Atoi(envInterval); err == nil && seconds > 0 {
			config.MonitoringInterval = time.Duration(seconds) * time.Second
		}
	}
	if envInterval := os.Getenv(EnvMetricsInterval); envInterval != "" {
		if seconds, err := strconv.Atoi(envInterval); err == nil && seconds > 0 {
			config.MetricsInterval = time.Duration(seconds) * time.Second
		}
	}
	if envURL := os.Getenv(EnvOtelCollectorURL); envURL != "" {
		config.OtelCollectorURL = envURL
	}

	// Parse success status codes from environment variable
	if envStatusCodes := os.Getenv(EnvSuccessStatusCodes); envStatusCodes != "" {
		// Split by comma
		statusCodeStrs := strings.Split(envStatusCodes, ",")
		var statusCodes []int
		for _, codeStr := range statusCodeStrs {
			if code, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
				statusCodes = append(statusCodes, code)
			}
		}
		if len(statusCodes) > 0 {
			config.SuccessStatusCodes = statusCodes
		}
	}

	// Parse namespace mode from environment variable
	if envMode := os.Getenv(EnvNamespaceMode); envMode != "" {
		mode := strings.ToLower(envMode)
		if mode == "allow" || mode == "deny" {
			config.NamespaceMode = mode
		}
	}

	// Parse namespaces from environment variable
	if envNamespaces := os.Getenv(EnvNamespaces); envNamespaces != "" {
		// Split by comma
		namespaces := strings.Split(envNamespaces, ",")
		for i, ns := range namespaces {
			namespaces[i] = strings.TrimSpace(ns)
		}
		config.Namespaces = namespaces
	}

	return config, nil
}
