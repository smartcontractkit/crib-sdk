package domain

import (
	"testing"
)

func TestClusterLocalServiceURL(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		serviceName string
		namespace   string
		port        int
		expected    string
	}{
		{
			name:        "with protocol and port",
			protocol:    "http",
			serviceName: "my-service",
			namespace:   "default",
			port:        8080,
			expected:    "http://my-service.default.svc.cluster.local:8080",
		},
		{
			name:        "with protocol, no port",
			protocol:    "https",
			serviceName: "api-service",
			namespace:   "production",
			port:        0,
			expected:    "https://api-service.production.svc.cluster.local",
		},
		{
			name:        "no protocol, with port",
			protocol:    "",
			serviceName: "database",
			namespace:   "data",
			port:        5432,
			expected:    "database.data.svc.cluster.local:5432",
		},
		{
			name:        "no protocol, no port",
			protocol:    "",
			serviceName: "cache",
			namespace:   "infra",
			port:        0,
			expected:    "cache.infra.svc.cluster.local",
		},
		{
			name:        "grpc protocol with port",
			protocol:    "grpc",
			serviceName: "user-service",
			namespace:   "microservices",
			port:        9090,
			expected:    "grpc://user-service.microservices.svc.cluster.local:9090",
		},
		{
			name:        "tcp protocol with standard port",
			protocol:    "tcp",
			serviceName: "redis",
			namespace:   "cache",
			port:        6379,
			expected:    "tcp://redis.cache.svc.cluster.local:6379",
		},
		{
			name:        "service with hyphenated names",
			protocol:    "http",
			serviceName: "my-complex-service-name",
			namespace:   "my-namespace",
			port:        3000,
			expected:    "http://my-complex-service-name.my-namespace.svc.cluster.local:3000",
		},
		{
			name:        "minimal case",
			protocol:    "",
			serviceName: "svc",
			namespace:   "ns",
			port:        0,
			expected:    "svc.ns.svc.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClusterLocalServiceURL(tt.protocol, tt.serviceName, tt.namespace, tt.port)
			if result != tt.expected {
				t.Errorf("ClusterLocalServiceURL(%q, %q, %q, %d) = %q, want %q",
					tt.protocol, tt.serviceName, tt.namespace, tt.port, result, tt.expected)
			}
		})
	}
}

func TestClusterLocalServiceURL_EdgeCases(t *testing.T) {
	// Test with empty service name (edge case)
	result := ClusterLocalServiceURL("http", "", "default", 8080)
	expected := "http://.default.svc.cluster.local:8080"
	if result != expected {
		t.Errorf("ClusterLocalServiceURL with empty service name = %q, want %q", result, expected)
	}

	// Test with empty namespace (edge case)
	result = ClusterLocalServiceURL("http", "service", "", 8080)
	expected = "http://service..svc.cluster.local:8080"
	if result != expected {
		t.Errorf("ClusterLocalServiceURL with empty namespace = %q, want %q", result, expected)
	}

	// Test with negative port (should be treated as 0)
	result = ClusterLocalServiceURL("http", "service", "default", -1)
	expected = "http://service.default.svc.cluster.local:-1"
	if result != expected {
		t.Errorf("ClusterLocalServiceURL with negative port = %q, want %q", result, expected)
	}
}

// Benchmark the function to ensure it performs well
func BenchmarkClusterLocalServiceURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ClusterLocalServiceURL("http", "my-service", "default", 8080)
	}
}

func BenchmarkClusterLocalServiceURL_NoProtocol(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ClusterLocalServiceURL("", "my-service", "default", 0)
	}
}
