package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
)

const (
	DefaultPrometheusLimit = 100
)

// prometheusClient provides methods for interacting with Prometheus via Grafana's datasource proxy.
type prometheusClient struct {
	httpClient *http.Client
	baseURL    string // e.g., http://grafana/api/datasources/proxy/uid/{uid}
}

// newPrometheusClient creates a new Prometheus client for the given datasource UID.
func newPrometheusClient(datasourceUID string) (*prometheusClient, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", grafanaURL, datasourceUID)
	return &prometheusClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// makeRequest performs an HTTP request and returns the response body.
func (c *prometheusClient) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// prometheusResponse represents the standard Prometheus API response wrapper.
type prometheusResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error,omitempty"`
}

// parsePrometheusResponse parses a Prometheus API response and extracts the data.
func parsePrometheusResponse(bodyBytes []byte) (json.RawMessage, error) {
	var resp prometheusResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("prometheus API error: %s", resp.Error)
	}

	return resp.Data, nil
}

// fetchLabels fetches label names from Prometheus.
func (c *prometheusClient) fetchLabels(ctx context.Context, startRFC3339, endRFC3339 string) ([]string, error) {
	params := url.Values{}

	if startRFC3339 != "" {
		startTime, err := time.Parse(time.RFC3339, startRFC3339)
		if err != nil {
			return nil, fmt.Errorf("parsing start time: %w", err)
		}
		params.Add("start", fmt.Sprintf("%d", startTime.Unix()))
	}

	if endRFC3339 != "" {
		endTime, err := time.Parse(time.RFC3339, endRFC3339)
		if err != nil {
			return nil, fmt.Errorf("parsing end time: %w", err)
		}
		params.Add("end", fmt.Sprintf("%d", endTime.Unix()))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/v1/labels", params)
	if err != nil {
		return nil, err
	}

	data, err := parsePrometheusResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	var labels []string
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("unmarshalling labels: %w", err)
	}

	return labels, nil
}

// fetchLabelValues fetches values for a specific label from Prometheus.
func (c *prometheusClient) fetchLabelValues(ctx context.Context, labelName, startRFC3339, endRFC3339 string) ([]string, error) {
	params := url.Values{}

	if startRFC3339 != "" {
		startTime, err := time.Parse(time.RFC3339, startRFC3339)
		if err != nil {
			return nil, fmt.Errorf("parsing start time: %w", err)
		}
		params.Add("start", fmt.Sprintf("%d", startTime.Unix()))
	}

	if endRFC3339 != "" {
		endTime, err := time.Parse(time.RFC3339, endRFC3339)
		if err != nil {
			return nil, fmt.Errorf("parsing end time: %w", err)
		}
		params.Add("end", fmt.Sprintf("%d", endTime.Unix()))
	}

	path := fmt.Sprintf("/api/v1/label/%s/values", url.PathEscape(labelName))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, params)
	if err != nil {
		return nil, err
	}

	data, err := parsePrometheusResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshalling label values: %w", err)
	}

	return values, nil
}

// PrometheusQueryResult represents a query result from Prometheus.
type PrometheusQueryResult struct {
	ResultType string `json:"resultType"`
	Result     any    `json:"result"`
}

// query executes a PromQL query against Prometheus.
func (c *prometheusClient) query(ctx context.Context, expr string, timeRFC3339 string) (*PrometheusQueryResult, error) {
	params := url.Values{}
	params.Add("query", expr)

	if timeRFC3339 != "" {
		queryTime, err := time.Parse(time.RFC3339, timeRFC3339)
		if err != nil {
			return nil, fmt.Errorf("parsing query time: %w", err)
		}
		params.Add("time", fmt.Sprintf("%d", queryTime.Unix()))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/v1/query", params)
	if err != nil {
		return nil, err
	}

	data, err := parsePrometheusResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	var result PrometheusQueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling query result: %w", err)
	}

	return &result, nil
}

// queryRange executes a range PromQL query against Prometheus.
func (c *prometheusClient) queryRange(ctx context.Context, expr, startRFC3339, endRFC3339 string, stepSeconds int) (*PrometheusQueryResult, error) {
	params := url.Values{}
	params.Add("query", expr)

	startTime, err := time.Parse(time.RFC3339, startRFC3339)
	if err != nil {
		return nil, fmt.Errorf("parsing start time: %w", err)
	}
	params.Add("start", fmt.Sprintf("%d", startTime.Unix()))

	endTime, err := time.Parse(time.RFC3339, endRFC3339)
	if err != nil {
		return nil, fmt.Errorf("parsing end time: %w", err)
	}
	params.Add("end", fmt.Sprintf("%d", endTime.Unix()))

	params.Add("step", fmt.Sprintf("%d", stepSeconds))

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	data, err := parsePrometheusResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	var result PrometheusQueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling query range result: %w", err)
	}

	return &result, nil
}

// getDefaultPrometheusTimeRange returns default start/end times if not specified (last 1 hour).
func getDefaultPrometheusTimeRange(startRFC3339, endRFC3339 string) (string, string) {
	now := time.Now().UTC()
	if endRFC3339 == "" {
		endRFC3339 = now.Format(time.RFC3339)
	}
	if startRFC3339 == "" {
		startRFC3339 = now.Add(-1 * time.Hour).Format(time.RFC3339)
	}
	return startRFC3339, endRFC3339
}

// enforcePrometheusLimit ensures the limit doesn't exceed the maximum.
func enforcePrometheusLimit(requestedLimit, maxLimit int) int {
	if requestedLimit <= 0 {
		return DefaultPrometheusLimit
	}
	if maxLimit > 0 && requestedLimit > maxLimit {
		return maxLimit
	}
	return requestedLimit
}
