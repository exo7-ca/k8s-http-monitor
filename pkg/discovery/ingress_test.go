package discovery

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldProcessNamespace(t *testing.T) {
	tests := []struct {
		name           string
		namespaceMode  string
		namespaces     []string
		testNamespace  string
		expectedResult bool
	}{
		{
			name:           "allow mode with empty namespaces list",
			namespaceMode:  "allow",
			namespaces:     []string{},
			testNamespace:  "default",
			expectedResult: true,
		},
		{
			name:           "deny mode with empty namespaces list",
			namespaceMode:  "deny",
			namespaces:     []string{},
			testNamespace:  "default",
			expectedResult: false,
		},
		{
			name:           "allow mode with namespace in list",
			namespaceMode:  "allow",
			namespaces:     []string{"default", "kube-system"},
			testNamespace:  "default",
			expectedResult: true,
		},
		{
			name:           "allow mode with namespace not in list",
			namespaceMode:  "allow",
			namespaces:     []string{"kube-system"},
			testNamespace:  "default",
			expectedResult: false,
		},
		{
			name:           "deny mode with namespace in list",
			namespaceMode:  "deny",
			namespaces:     []string{"default", "kube-system"},
			testNamespace:  "default",
			expectedResult: false,
		},
		{
			name:           "deny mode with namespace not in list",
			namespaceMode:  "deny",
			namespaces:     []string{"kube-system"},
			testNamespace:  "default",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				namespaceMode: tt.namespaceMode,
				namespaces:    tt.namespaces,
			}

			result := client.shouldProcessNamespace(tt.testNamespace)
			if result != tt.expectedResult {
				t.Errorf("shouldProcessNamespace() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestSetNamespaceFilter(t *testing.T) {
	client := &Client{}

	// Test setting allow mode with namespaces
	client.SetNamespaceFilter("allow", []string{"default", "kube-system"})
	if client.namespaceMode != "allow" {
		t.Errorf("Expected namespace mode to be 'allow', got %s", client.namespaceMode)
	}
	if len(client.namespaces) != 2 || client.namespaces[0] != "default" || client.namespaces[1] != "kube-system" {
		t.Errorf("Expected namespaces to be ['default', 'kube-system'], got %v", client.namespaces)
	}

	// Test setting deny mode with different namespaces
	client.SetNamespaceFilter("deny", []string{"test"})
	if client.namespaceMode != "deny" {
		t.Errorf("Expected namespace mode to be 'deny', got %s", client.namespaceMode)
	}
	if len(client.namespaces) != 1 || client.namespaces[0] != "test" {
		t.Errorf("Expected namespaces to be ['test'], got %v", client.namespaces)
	}
}

func TestExtractEndpointsFromIngress(t *testing.T) {
	// Create a test ingress
	pathType := networkingv1.PathTypePrefix
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"health.monitor/endpoint":     "/health",
				"health.monitor/path.service": "/service-health",
			},
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{"secure.example.com"},
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "secure.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/secure",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "secure-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 443,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					// Rule without host should be skipped
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/no-host",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "no-host-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					// Rule without HTTP should be skipped
					Host: "no-http.example.com",
				},
			},
		},
	}

	// Extract endpoints
	endpoints := extractEndpointsFromIngress(ingress)

	// Check the number of endpoints
	if len(endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(endpoints))
	}

	// Check the endpoints
	// Sort the endpoints by URL to ensure consistent order
	if endpoints[0].URL == "https://secure.example.com" {
		endpoints[0], endpoints[1] = endpoints[1], endpoints[0]
	}

	// Check the first endpoint
	if endpoints[0].URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", endpoints[0].URL)
	}
	if endpoints[0].ServiceName != "service" {
		t.Errorf("Expected ServiceName 'service', got '%s'", endpoints[0].ServiceName)
	}
	if endpoints[0].Namespace != "default" {
		t.Errorf("Expected Namespace 'default', got '%s'", endpoints[0].Namespace)
	}
	if endpoints[0].IngressName != "test-ingress" {
		t.Errorf("Expected IngressName 'test-ingress', got '%s'", endpoints[0].IngressName)
	}
	// The path could be either the default or the health endpoint
	if endpoints[0].Path != "/" && endpoints[0].Path != "/service-health" {
		t.Errorf("Expected Path '/' or '/health', got '%s'", endpoints[0].Path)
	}

	// Check the second endpoint
	if endpoints[1].URL != "https://secure.example.com" {
		t.Errorf("Expected URL 'https://secure.example.com', got '%s'", endpoints[1].URL)
	}
	if endpoints[1].ServiceName != "secure-service" {
		t.Errorf("Expected ServiceName 'secure-service', got '%s'", endpoints[1].ServiceName)
	}
	// The path could be either the default or the service-specific health endpoint
	if endpoints[1].Path != "/secure" && endpoints[1].Path != "/health" {
		t.Errorf("Expected Path '/secure' or '/service-health', got '%s'", endpoints[1].Path)
	}
}
