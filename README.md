# K8s Endpoint Monitor

A Kubernetes application that monitors HTTP endpoints and exports metrics to OpenTelemetry. The application automatically discovers endpoints to monitor based on Ingress resources in the Kubernetes cluster.

## Endpoint Discovery

The application automatically discovers HTTP endpoints to monitor by scanning Ingress resources in the Kubernetes cluster. By default, it scans all namespaces, but this can be configured using the namespace filtering mode and list. For each Ingress rule, it extracts:

- The host and path
- The protocol (http/https based on TLS configuration)
- The associated service name

Health check paths can be customized using the following Ingress annotations:
- `health.monitor/endpoint`: Sets a global health check path for all services in the Ingress
- `health.monitor/path.<service-name>`: Sets a specific health check path for the named service

## Configuration

By default, the application considers HTTP status codes in the 2xx range as successful. It can be configured to treat additional status codes (like 401 or 403) as successful as well.

The application can be configured using a configuration file (`config.yaml`) and/or environment variables.

### Configuration File

The default configuration file is `config.yaml` in the application's root directory. Here's an example:

```yaml
# Monitoring settings
monitoring:
  # Interval between endpoint checks in seconds
  interval: 30
  # HTTP status codes to consider as successful (in addition to 2xx)
  successStatusCodes: [401, 403]

# Metrics settings
metrics:
  # Interval for reporting metrics in seconds
  # This controls how often metrics are batched and sent to the OpenTelemetry collector,
  # which is separate from when metrics are collected (which happens after each endpoint check)
  interval: 10
  # URL of the OpenTelemetry collector
  otelCollectorURL: "signoz-otel-collector:4317"

# Discovery settings
discovery:
  # Namespace filtering mode: "allow" or "deny"
  namespaceMode: "allow"
  # List of namespaces to allow or deny based on the mode
  namespaces: ["default", "kube-system"]
```

### Environment Variables

The following environment variables can be used to override the configuration:

- `MONITOR_INTERVAL_SECONDS`: Interval between endpoint checks in seconds
- `METRICS_INTERVAL_SECONDS`: Interval for batching and sending metrics to the OpenTelemetry collector
- `OTEL_COLLECTOR_URL`: URL of the OpenTelemetry collector
- `SUCCESS_STATUS_CODES`: Comma-separated list of HTTP status codes to consider as successful (e.g., "401,403,404")
- `NAMESPACE_MODE`: Namespace filtering mode, either "allow" or "deny"
- `NAMESPACES`: Comma-separated list of namespaces to allow or deny based on the mode

Environment variables take precedence over the configuration file.

## Default Values

If no configuration is provided, the following default values are used:

- Monitoring interval: 30 seconds (how often endpoints are checked)
- Metrics interval: 10 seconds (how often metrics are batched and sent to the collector)
- OpenTelemetry collector URL: "signoz-otel-collector:4317"
- Success status codes: 401, 403, 404 (in addition to 2xx status codes)
- Namespace mode: "allow" (allow all namespaces)
- Namespaces: [] (empty list, which means all namespaces when mode is "allow")
