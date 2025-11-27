// Package tempo provides MCP tools for querying traces via Grafana's Tempo datasource proxy.
package tempo

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
	// DefaultTraceLimit is the default number of traces to return.
	DefaultTraceLimit = 20

	// MaxTraceLimit is the maximum number of traces that can be requested.
	MaxTraceLimit = 100
)

// client provides methods for interacting with Tempo via Grafana's datasource proxy.
type client struct {
	httpClient *http.Client
	baseURL    string
}

// newClient creates a new Tempo client for the given datasource UID.
func newClient(datasourceUID string) (*client, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", grafanaURL, datasourceUID)
	return &client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// makeRequest performs an HTTP request and returns the response body.
func (c *client) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
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

// tagsResponse represents the response from the tags endpoint.
type tagsResponse struct {
	TagNames []string `json:"tagNames"`
	Scopes   []string `json:"scopes,omitempty"`
}

// fetchTagNames fetches tag names from Tempo.
func (c *client) fetchTagNames(ctx context.Context, scope, startUnix, endUnix string) ([]string, error) {
	params := url.Values{}

	if scope != "" {
		params.Add("scope", scope)
	}
	if startUnix != "" {
		params.Add("start", startUnix)
	}
	if endUnix != "" {
		params.Add("end", endUnix)
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/search/tags", params)
	if err != nil {
		return nil, err
	}

	var resp tagsResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling tags response: %w", err)
	}

	return resp.TagNames, nil
}

// tagValuesResponse represents the response from the tag values endpoint.
type tagValuesResponse struct {
	TagValues []string `json:"tagValues"`
}

// fetchTagValues fetches values for a specific tag from Tempo.
func (c *client) fetchTagValues(ctx context.Context, tagName, startUnix, endUnix string) ([]string, error) {
	params := url.Values{}

	if startUnix != "" {
		params.Add("start", startUnix)
	}
	if endUnix != "" {
		params.Add("end", endUnix)
	}

	path := fmt.Sprintf("/api/search/tag/%s/values", url.PathEscape(tagName))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, params)
	if err != nil {
		return nil, err
	}

	var resp tagValuesResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling tag values response: %w", err)
	}

	return resp.TagValues, nil
}

// SearchResponse represents the response from the search endpoint.
type SearchResponse struct {
	Traces  []TraceSearchResult `json:"traces"`
	Metrics *SearchMetrics      `json:"metrics,omitempty"`
}

// TraceSearchResult represents a single trace in search results.
type TraceSearchResult struct {
	TraceID           string         `json:"traceID"`
	RootServiceName   string         `json:"rootServiceName"`
	RootTraceName     string         `json:"rootTraceName"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	DurationMs        int            `json:"durationMs"`
	SpanSets          []SpanSet      `json:"spanSets,omitempty"`
	ServiceStats      map[string]any `json:"serviceStats,omitempty"`
}

// SpanSet represents a set of spans matching a query.
type SpanSet struct {
	Spans   []Span `json:"spans"`
	Matched int    `json:"matched"`
}

// Span represents a span in a spanset.
type Span struct {
	SpanID            string      `json:"spanID"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	DurationNanos     string      `json:"durationNanos"`
	Attributes        []Attribute `json:"attributes,omitempty"`
}

// Attribute represents a span attribute.
type Attribute struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// SearchMetrics represents metrics from a search query.
// Field types match Tempo's protobuf definition.
// NOTE: uint64 fields are serialized as strings in JSON (protobuf convention for 64-bit ints),
// while uint32 fields come through as regular JSON numbers.
type SearchMetrics struct {
	InspectedTraces int                  `json:"inspectedTraces,omitempty"` // uint32 in proto → JSON number
	InspectedBytes  grafana.Uint64String `json:"inspectedBytes,omitempty"`  // uint64 in proto → JSON string
}

// searchTraces searches for traces using TraceQL.
func (c *client) searchTraces(ctx context.Context, query, startUnix, endUnix string, limit int) (*SearchResponse, error) {
	params := url.Values{}

	if query != "" {
		params.Add("q", query)
	}
	if startUnix != "" {
		params.Add("start", startUnix)
	}
	if endUnix != "" {
		params.Add("end", endUnix)
	}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/search", params)
	if err != nil {
		return nil, err
	}

	var resp SearchResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling search response: %w", err)
	}

	return &resp, nil
}

// getTrace retrieves a trace by its ID.
func (c *client) getTrace(ctx context.Context, traceID string) (any, error) {
	path := fmt.Sprintf("/api/traces/%s", url.PathEscape(traceID))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var trace any
	if err := json.Unmarshal(bodyBytes, &trace); err != nil {
		return nil, fmt.Errorf("unmarshalling trace response: %w", err)
	}

	return trace, nil
}

// getDefaultTimeRange returns default start/end times if not specified (last 1 hour).
// Returns Unix epoch seconds as strings.
func getDefaultTimeRange(startRFC3339, endRFC3339 string) (string, string, error) {
	now := time.Now().UTC()
	var startUnix, endUnix string

	if endRFC3339 != "" {
		endTime, err := time.Parse(time.RFC3339, endRFC3339)
		if err != nil {
			return "", "", fmt.Errorf("parsing end time: %w", err)
		}
		endUnix = fmt.Sprintf("%d", endTime.Unix())
	} else {
		endUnix = fmt.Sprintf("%d", now.Unix())
	}

	if startRFC3339 != "" {
		startTime, err := time.Parse(time.RFC3339, startRFC3339)
		if err != nil {
			return "", "", fmt.Errorf("parsing start time: %w", err)
		}
		startUnix = fmt.Sprintf("%d", startTime.Unix())
	} else {
		startUnix = fmt.Sprintf("%d", now.Add(-1*time.Hour).Unix())
	}

	return startUnix, endUnix, nil
}

// enforceTraceLimit ensures the limit is within bounds.
func enforceTraceLimit(requestedLimit int) int {
	if requestedLimit <= 0 {
		return DefaultTraceLimit
	}
	if requestedLimit > MaxTraceLimit {
		return MaxTraceLimit
	}
	return requestedLimit
}
