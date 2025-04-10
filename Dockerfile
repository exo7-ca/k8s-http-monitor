# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o k8s-http-monitor .

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/k8s-http-monitor .

# Copy the config file
COPY config.yaml .

# Create a non-root user
RUN adduser -D -u 1000 appuser

# Create directory for Kubernetes config (when running outside a cluster)
RUN mkdir -p /home/appuser/.kube && chown -R appuser:appuser /home/appuser

USER appuser

# Expose metrics port if needed
# EXPOSE 8080

# Environment variables (can be overridden at runtime)
ENV MONITOR_INTERVAL_SECONDS=30
ENV METRICS_INTERVAL_SECONDS=10

# Run the application
ENTRYPOINT ["./k8s-http-monitor"]

# Usage:
# Build: docker build -t k8s-endpoint-monitor .
#
# Run inside Kubernetes (uses in-cluster config):
# kubectl apply -f k8s-deployment.yaml
#
# Run outside Kubernetes (requires kubeconfig):
# docker run -v ~/.kube/config:/home/appuser/.kube/config k8s-endpoint-monitor
#
# Override configuration with environment variables:
# docker run -e OTEL_COLLECTOR_URL=my-collector:4317 -e MONITOR_INTERVAL_SECONDS=60 k8s-endpoint-monitor
