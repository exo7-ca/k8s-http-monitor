package discovery

import (
	"context"
	"log"
	"path/filepath"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client handles Kubernetes API calls
type Client struct {
	clientset *kubernetes.Clientset
	namespaceMode string // "allow" or "deny"
	namespaces []string
}

// Endpoint represents a discovered endpoint
type Endpoint struct {
	Namespace   string
	ServiceName string
	IngressName string
	URL         string
	Path        string
	Labels      map[string]string
	Annotations map[string]string
}

// SetNamespaceFilter sets the namespace filtering mode and list
func (c *Client) SetNamespaceFilter(mode string, namespaces []string) {
	c.namespaceMode = mode
	c.namespaces = namespaces
}

// shouldProcessNamespace determines if a namespace should be processed based on the filtering mode
func (c *Client) shouldProcessNamespace(namespace string) bool {
	// If no namespaces are specified, follow the mode's default behavior
	if len(c.namespaces) == 0 {
		return c.namespaceMode == "allow" // Allow all if mode is "allow" and no namespaces specified
	}

	// Check if the namespace is in the list
	found := false
	for _, ns := range c.namespaces {
		if ns == namespace {
			found = true
			break
		}
	}

	// If mode is "allow", only process namespaces in the list
	// If mode is "deny", only process namespaces NOT in the list
	return (c.namespaceMode == "allow" && found) || (c.namespaceMode == "deny" && !found)
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		home := homedir.HomeDir()
		kubeconfigPath := filepath.Join(home, ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset: clientset,
		namespaceMode: "allow", // Default to allow all namespaces
		namespaces: []string{},
	}, nil
}

// DiscoverIngressEndpoints discovers all Ingress endpoints
func (c *Client) DiscoverIngressEndpoints(ctx context.Context) ([]Endpoint, error) {
	log.Println("Discovering Ingress endpoints")

	// List all ingresses across all namespaces
	ingresses, err := c.clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var endpoints []Endpoint

	// Process each ingress
	for _, ingress := range ingresses.Items {
		// Apply namespace filtering
		if !c.shouldProcessNamespace(ingress.Namespace) {
			continue
		}

		eps := extractEndpointsFromIngress(ingress)
		endpoints = append(endpoints, eps...)
	}

	log.Printf("Discovered %d endpoints from ingresses", len(endpoints))
	return endpoints, nil
}

func extractEndpointsFromIngress(ingress networkingv1.Ingress) []Endpoint {
	namespace := ingress.Namespace
	name := ingress.Name
	annotations := ingress.Annotations
	labels := ingress.Labels

	// Check if TLS is configured
	tls := len(ingress.Spec.TLS) > 0
	protocol := "http"
	if tls {
		protocol = "https"
	}

	var endpoints []Endpoint

	// Process each rule
	for _, rule := range ingress.Spec.Rules {
		if rule.Host == "" {
			continue
		}

		host := rule.Host

		// Process each path
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			pathType := string(*path.PathType)
			if pathType == "" {
				pathType = "Prefix"
			}

			routePath := path.Path
			if routePath == "" {
				routePath = "/"
			}

			// Extract service name
			if path.Backend.Service == nil {
				continue
			}
			serviceName := path.Backend.Service.Name

			// Build the URL
			url := protocol + "://" + host

			// Check for health endpoint annotation
			healthEndpoint := routePath
			if pathSpecificHealth, ok := annotations["health.monitor/path."+serviceName]; ok {
				healthEndpoint = pathSpecificHealth
			} else if generalHealth, ok := annotations["health.monitor/endpoint"]; ok {
				healthEndpoint = generalHealth
			}

			endpoints = append(endpoints, Endpoint{
				Namespace:   namespace,
				ServiceName: serviceName,
				IngressName: name,
				URL:         url,
				Path:        healthEndpoint,
				Labels:      labels,
				Annotations: annotations,
			})
		}
	}

	return endpoints
}
