package domain

import (
	"fmt"
)

// ClusterLocalServiceURL returns a cluster local service url.
func ClusterLocalServiceURL(protocol, serviceName, namespace string, port int) string {
	url := ""

	// skip protocol if not provided
	if protocol != "" {
		url = fmt.Sprintf("%s://", protocol)
	}

	url += fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)

	// skip port if not provided
	if port != 0 {
		url += fmt.Sprintf(":%d", port)
	}

	return url
}
