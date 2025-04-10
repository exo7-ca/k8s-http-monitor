package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/exo7-ca/k8s-http-monitor/pkg/config"
	"github.com/exo7-ca/k8s-http-monitor/pkg/discovery"
	"github.com/exo7-ca/k8s-http-monitor/pkg/metrics"
	"github.com/exo7-ca/k8s-http-monitor/pkg/monitoring"
)

func main() {
	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded: monitoring interval=%v, metrics interval=%v, otel collector URL=%s",
		cfg.MonitoringInterval, cfg.MetricsInterval, cfg.OtelCollectorURL)

	// Initialize the metrics provider
	metricsProvider, err := metrics.NewProvider(ctx, cfg.OtelCollectorURL, cfg.MetricsInterval)
	if err != nil {
		log.Fatalf("Failed to initialize metrics provider: %v", err)
	}
	defer metricsProvider.Shutdown(ctx)

	// Create a Kubernetes client
	discoveryClient, err := discovery.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Set namespace filtering
	discoveryClient.SetNamespaceFilter(cfg.NamespaceMode, cfg.Namespaces)

	// Create the monitor
	monitor := monitoring.NewMonitor(
		discoveryClient,
		metricsProvider,
		monitoring.WithCheckInterval(cfg.MonitoringInterval),
		monitoring.WithTimeout(10*time.Second),
		monitoring.WithSuccessStatusCodes(cfg.SuccessStatusCodes),
	)

	// Start the monitoring
	monitor.Start(ctx)

	// Wait for termination signal
	<-ctx.Done()
	log.Println("Shutting down gracefully...")
}
