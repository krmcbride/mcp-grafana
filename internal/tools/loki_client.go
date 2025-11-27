// Package tools provides shared Loki client functionality for querying logs via Grafana datasource proxy.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
)

const (
	// DefaultLokiLogLimit is the default number of log lines to return if not specified
	DefaultLokiLogLimit = 10

	// MaxLokiLogLimit is the maximum number of log lines that can be requested
	MaxLokiLogLimit = 100
)

// lokiClient wraps an HTTP client for making Loki API requests through Grafana datasource proxy.
type lokiClient struct {
	httpClient *http.Client
	baseURL    string
}

// newLokiClient creates a Loki client for the specified datasource UID.
// It uses the Grafana datasource proxy endpoint to route requests to Loki.
func newLokiClient(datasourceUID string) (*lokiClient, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	// Construct the Grafana datasource proxy base URL
	baseURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", grafanaURL, datasourceUID)

	return &lokiClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// buildURL constructs a full URL for a Loki API endpoint.
func (c *lokiClient) buildURL(path string) string {
	// Ensure proper slash handling between baseURL and path
	if !strings.HasSuffix(c.baseURL, "/") && !strings.HasPrefix(path, "/") {
		return c.baseURL + "/" + path
	} else if strings.HasSuffix(c.baseURL, "/") && strings.HasPrefix(path, "/") {
		return c.baseURL + strings.TrimPrefix(path, "/")
	}
	return c.baseURL + path
}

// makeRequest executes an HTTP request to the Loki API and returns the response body.
func (c *lokiClient) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	fullURL := c.buildURL(path)

	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki API returned status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body with 48MB limit to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 1024*1024*48)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("empty response from Loki API")
	}

	return bytes.TrimSpace(bodyBytes), nil
}

// labelResponse represents the JSON response from Loki label endpoints.
type labelResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data,omitempty"`
}

// fetchLabels is a helper to fetch label names or values from Loki.
func (c *lokiClient) fetchLabels(ctx context.Context, path, startRFC3339, endRFC3339 string) ([]string, error) {
	params := url.Values{}
	if startRFC3339 != "" {
		params.Add("start", startRFC3339)
	}
	if endRFC3339 != "" {
		params.Add("end", endRFC3339)
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", path, params)
	if err != nil {
		return nil, err
	}

	var response labelResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("loki API returned unexpected status: %s", response.Status)
	}

	if response.Data == nil {
		return []string{}, nil
	}

	return response.Data, nil
}

// getDefaultTimeRange returns default start and end times if not provided.
// Default range is the last 1 hour.
func getDefaultTimeRange(startRFC3339, endRFC3339 string) (string, string) {
	if startRFC3339 == "" {
		startRFC3339 = time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	}
	if endRFC3339 == "" {
		endRFC3339 = time.Now().Format(time.RFC3339)
	}
	return startRFC3339, endRFC3339
}

// addTimeRangeParams adds start and end time parameters to URL values.
// Converts RFC3339 timestamps to Unix nanoseconds as required by Loki.
func addTimeRangeParams(params url.Values, startRFC3339, endRFC3339 string) error {
	if startRFC3339 != "" {
		startTime, err := time.Parse(time.RFC3339, startRFC3339)
		if err != nil {
			return fmt.Errorf("parsing start time: %w", err)
		}
		params.Add("start", fmt.Sprintf("%d", startTime.UnixNano()))
	}

	if endRFC3339 != "" {
		endTime, err := time.Parse(time.RFC3339, endRFC3339)
		if err != nil {
			return fmt.Errorf("parsing end time: %w", err)
		}
		params.Add("end", fmt.Sprintf("%d", endTime.UnixNano()))
	}

	return nil
}

// enforceLogLimit ensures the log limit is within acceptable bounds.
func enforceLogLimit(requestedLimit int) int {
	if requestedLimit <= 0 {
		return DefaultLokiLogLimit
	}
	if requestedLimit > MaxLokiLogLimit {
		return MaxLokiLogLimit
	}
	return requestedLimit
}
