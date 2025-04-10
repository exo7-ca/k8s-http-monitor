package monitoring

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/exo7-ca/k8s-http-monitor/pkg/discovery"
	"github.com/exo7-ca/k8s-http-monitor/pkg/metrics"
)

// Monitor checks the health of endpoints
type Monitor struct {
	discoveryClient    *discovery.Client
	metricsProvider    *metrics.Provider
	checkInterval      time.Duration
	timeout            time.Duration
	httpClient         *http.Client
	mu                 sync.Mutex // Protects the maps below
	endpointStatus     map[string]bool
	endpoints          map[string]discovery.Endpoint
	successStatusCodes []int
}

// Option is a functional option for configuring the monitor
type Option func(*Monitor)

// WithCheckInterval sets the interval between health checks
func WithCheckInterval(interval time.Duration) Option {
	return func(m *Monitor) {
		m.checkInterval = interval
	}
}

// WithTimeout sets the timeout for health checks
func WithTimeout(timeout time.Duration) Option {
	return func(m *Monitor) {
		m.timeout = timeout
	}
}

// WithSuccessStatusCodes sets the HTTP status codes that are considered successful
func WithSuccessStatusCodes(codes []int) Option {
	return func(m *Monitor) {
		m.successStatusCodes = codes
	}
}

func endpointKey(endpoint discovery.Endpoint) string {
	return fmt.Sprintf("%s/%s/%s%s",
		endpoint.Namespace,
		endpoint.IngressName,
		endpoint.URL,
		endpoint.Path)
}

// NewMonitor creates a new endpoint monitor
func NewMonitor(discoveryClient *discovery.Client, metricsProvider *metrics.Provider, options ...Option) *Monitor {
	m := &Monitor{
		discoveryClient:    discoveryClient,
		metricsProvider:    metricsProvider,
		checkInterval:      30 * time.Second,
		timeout:            10 * time.Second,
		endpointStatus:     make(map[string]bool),
		endpoints:          make(map[string]discovery.Endpoint),
		successStatusCodes: []int{401, 403, 404}, // Default success status codes
	}

	// Apply options
	for _, option := range options {
		option(m)
	}

	// Create HTTP client with timeout
	m.httpClient = &http.Client{
		Timeout: m.timeout,
	}

	return m
}

// Start begins monitoring endpoints
func (m *Monitor) Start(ctx context.Context) {
	// Register callback for the upGauge observable metric
	_, err := m.metricsProvider.GetMeter().RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			m.mu.Lock()
			defer m.mu.Unlock()

			for key, endpoint := range m.endpoints {
				isUp, exists := m.endpointStatus[key]
				if !exists {
					continue
				}

				// Create attributes for this endpoint
				attrs := []attribute.KeyValue{
					attribute.String("namespace", endpoint.Namespace),
					attribute.String("service", endpoint.ServiceName),
					attribute.String("ingress", endpoint.IngressName),
					attribute.String("url", endpoint.URL+endpoint.Path),
				}

				// Add labels as attributes
				for k, v := range endpoint.Labels {
					attrs = append(attrs, attribute.String(k, v))
				}

				// Set gauge value: 1 if up, 0 if down
				value := int64(0)
				if isUp {
					value = 1
				}

				o.ObserveInt64(m.metricsProvider.GetUpGauge(), value, metric.WithAttributes(attrs...))
			}

			return nil
		},
		m.metricsProvider.GetUpGauge(),
	)

	if err != nil {
		log.Printf("Error registering callback for upGauge: %v", err)
	}

	// Start periodic health checks
	go func() {
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()

		// Do an initial check
		m.checkEndpoints(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkEndpoints(ctx)
			}
		}
	}()
}

// checkEndpoints discovers and checks all endpoints
func (m *Monitor) checkEndpoints(ctx context.Context) {
	endpoints, err := m.discoveryClient.DiscoverIngressEndpoints(ctx)
	if err != nil {
		log.Printf("Error discovering endpoints: %v", err)
		return
	}

	for _, endpoint := range endpoints {
		go m.checkEndpoint(ctx, endpoint)
	}
}

func (m *Monitor) checkStatus(statusCode int) bool {
	success := statusCode >= 200 && statusCode < 300

	for _, code := range m.successStatusCodes {
		if statusCode == code {
			success = true
			break
		}
	}

	return success
}

// checkEndpoint checks a single endpoint
func (m *Monitor) checkEndpoint(ctx context.Context, endpoint discovery.Endpoint) {
	fullURL := endpoint.URL + endpoint.Path
	log.Printf("Checking endpoint: %s", fullURL)

	// Store endpoint information
	key := endpointKey(endpoint)
	m.mu.Lock()
	m.endpoints[key] = endpoint
	m.mu.Unlock()

	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v", fullURL, err)
		return
	}

	resp, err := m.httpClient.Do(req)
	endTime := time.Now()
	duration := float64(endTime.Sub(startTime).Milliseconds())

	// Create common attributes
	attrs := []attribute.KeyValue{
		attribute.String("namespace", endpoint.Namespace),
		attribute.String("service", endpoint.ServiceName),
		attribute.String("ingress", endpoint.IngressName),
		attribute.String("url", fullURL),
	}

	// Add labels as attributes
	for k, v := range endpoint.Labels {
		attrs = append(attrs, attribute.String(k, v))
	}

	if err != nil {
		// Handle errors
		log.Printf("Error checking %s: %v", fullURL, err)

		// Update status
		m.mu.Lock()
		m.endpointStatus[key] = false
		m.mu.Unlock()

		// Record metrics
		statusAttrs := append(attrs,
			attribute.String("status", ""),
			attribute.String("success", "false"),
		)

		m.metricsProvider.GetRequestCounter().Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		m.metricsProvider.GetResponseTimeHistogram().Record(ctx, duration, metric.WithAttributes(statusAttrs...))

	} else {
		// response
		defer resp.Body.Close()

		isUp := m.checkStatus(resp.StatusCode)

		// log
		if isUp {
			log.Printf("Endpoint %s is UP, status: %d, response time: %.2fms", fullURL, resp.StatusCode, duration)
		} else {
			log.Printf("Endpoint %s is DOWN, status: %d, response time: %.2fms", fullURL, resp.StatusCode, duration)
		}

		// Update status
		m.mu.Lock()
		m.endpointStatus[key] = isUp
		m.mu.Unlock()

		// Record metrics
		statusAttrs := append(attrs,
			attribute.String("status", strconv.Itoa(resp.StatusCode)),
			attribute.String("success", strconv.FormatBool(isUp)),
		)

		m.metricsProvider.GetRequestCounter().Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		m.metricsProvider.GetResponseTimeHistogram().Record(ctx, duration, metric.WithAttributes(statusAttrs...))
	}
}
