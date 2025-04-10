package metrics

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Provider manages OpenTelemetry metrics
type Provider struct {
	meterProvider         *sdkmetric.MeterProvider
	meter                 metric.Meter
	upGauge               metric.Int64ObservableGauge
	requestCounter        metric.Int64Counter
	responseTimeHistogram metric.Float64Histogram
}

// NewProvider creates a new metrics provider
func NewProvider(ctx context.Context, otelCollectorURL string, metricsInterval time.Duration) (*Provider, error) {
	// Create OTLP exporter
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelCollectorURL),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Create resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("k8s-endpoint-monitor"),
	)

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(metricsInterval))),
	)
	otel.SetMeterProvider(meterProvider)

	// Create a meter
	meter := meterProvider.Meter("k8s-endpoint-monitor")

	// Create instruments
	upGauge, err := meter.Int64ObservableGauge(
		"http_endpoint_up",
		metric.WithDescription("Indicates if an endpoint is responding (1=up, 0=down)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	requestCounter, err := meter.Int64Counter(
		"http_endpoint_check_count",
		metric.WithDescription("Number of endpoint health checks performed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	responseTimeHistogram, err := meter.Float64Histogram(
		"http_endpoint_response_time",
		metric.WithDescription("Response time of endpoint health checks"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		meterProvider:         meterProvider,
		meter:                 meter,
		upGauge:               upGauge,
		requestCounter:        requestCounter,
		responseTimeHistogram: responseTimeHistogram,
	}, nil
}

// Shutdown stops the metric provider
func (p *Provider) Shutdown(ctx context.Context) {
	if err := p.meterProvider.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down meter provider: %v", err)
	}
}

// GetUpGauge returns the up gauge
func (p *Provider) GetUpGauge() metric.Int64ObservableGauge {
	return p.upGauge
}

// GetRequestCounter returns the request counter
func (p *Provider) GetRequestCounter() metric.Int64Counter {
	return p.requestCounter
}

// GetResponseTimeHistogram returns the response time histogram
func (p *Provider) GetResponseTimeHistogram() metric.Float64Histogram {
	return p.responseTimeHistogram
}

// GetMeter returns the meter
func (p *Provider) GetMeter() metric.Meter {
	return p.meter
}
