package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/exo7-ca/k8s-http-monitor/pkg/config"
	"github.com/exo7-ca/k8s-http-monitor/pkg/discovery"
	"github.com/exo7-ca/k8s-http-monitor/pkg/metrics"
	"github.com/exo7-ca/k8s-http-monitor/pkg/monitoring"
)

func startHealthServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create a simple health check handler
	http.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		// For liveness, just return 200 if the server is running
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// For readiness, check if all components are initialized
		// You might want to check if the Kubernetes client and metrics are working
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	server := &http.Server{
		Addr: ":8080",
	}

	// Start the server in a goroutine
	go func() {
		log.Println("Starting health check server on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Health check server failed: %v", err)
		}
	}()

	// Wait for context cancellation to shutdown
	<-ctx.Done()
	log.Println("Shutting down health check server")
	server.Shutdown(context.Background())
}

func main() {
	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)

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

	go startHealthServer(ctx, &wg)

	// Start the monitoring
	monitor.Start(ctx)

	// Wait for termination signal
	wg.Wait()
	log.Println("Application shutdown complete")
}
