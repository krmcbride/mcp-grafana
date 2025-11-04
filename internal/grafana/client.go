// Package grafana provides client utilities for interacting with Grafana API.
package grafana

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// GetHTTPClientForGrafana creates an authenticated HTTP client for Grafana API calls.
// It reads configuration from environment variables:
//   - GRAFANA_URL: Base URL of the Grafana instance (e.g., http://localhost:3000)
//   - GRAFANA_API_KEY: Service account token or API key for authentication
//
// Returns:
//   - An *http.Client configured with Bearer token authentication
//   - The base Grafana URL (with trailing slash removed)
//   - An error if required environment variables are missing
//
// The returned client is configured with:
//   - 30 second timeout
//   - Bearer token authentication via custom transport
//
// Example usage:
//
//	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
//	if err != nil {
//	    return fmt.Errorf("failed to create Grafana client: %w", err)
//	}
//	lokiURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s/loki/api/v1/query_range", grafanaURL, datasourceUID)
func GetHTTPClientForGrafana() (*http.Client, string, error) {
	grafanaURL := strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")
	if grafanaURL == "" {
		return nil, "", enhanceConfigError(
			fmt.Errorf("GRAFANA_URL environment variable not set"),
		)
	}

	apiKey := os.Getenv("GRAFANA_API_KEY")
	if apiKey == "" {
		return nil, "", enhanceConfigError(
			fmt.Errorf("GRAFANA_API_KEY environment variable not set"),
		)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &bearerAuthTransport{
			apiKey:    apiKey,
			transport: http.DefaultTransport,
		},
	}

	return client, grafanaURL, nil
}

// bearerAuthTransport is an http.RoundTripper that injects Bearer token authentication.
// It wraps an underlying transport and adds the Authorization header to all requests.
type bearerAuthTransport struct {
	apiKey    string
	transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper by adding Bearer token authentication to requests.
func (t *bearerAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.transport.RoundTrip(req)
}

// enhanceConfigError wraps configuration errors with helpful guidance for users.
func enhanceConfigError(err error) error {
	return fmt.Errorf("%w\n\nPlease ensure the following environment variables are set:\n  GRAFANA_URL       - Base URL of your Grafana instance (e.g., http://localhost:3000)\n  GRAFANA_API_KEY   - Service account token for authentication\n\nTo create a service account token:\n  1. In Grafana, go to Administration → Service accounts\n  2. Click 'Add service account'\n  3. Set a display name and assign the 'Viewer' role\n  4. Click 'Add token' and copy the generated token", err)
}
