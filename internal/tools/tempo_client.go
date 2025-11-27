package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
)

// uint64String unmarshals a JSON string into a uint64.
// Tempo's protobuf API serializes uint64 values as strings to avoid JavaScript precision issues.
type uint64String uint64

func (u *uint64String) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*u = 0
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing uint64 from string %q: %w", s, err)
	}
	*u = uint64String(v)
	return nil
}

const (
	DefaultTempoTraceLimit = 20
	MaxTempoTraceLimit     = 100
)

// tempoClient provides methods for interacting with Tempo via Grafana's datasource proxy.
type tempoClient struct {
	httpClient *http.Client
	baseURL    string // e.g., http://grafana/api/datasources/proxy/uid/{uid}
}

// newTempoClient creates a new Tempo client for the given datasource UID.
func newTempoClient(datasourceUID string) (*tempoClient, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", grafanaURL, datasourceUID)
	return &tempoClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// makeRequest performs an HTTP request and returns the response body.
func (c *tempoClient) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
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

// TempoTagsResponse represents the response from the tags endpoint.
type TempoTagsResponse struct {
	TagNames []string `json:"tagNames"`
	Scopes   []string `json:"scopes,omitempty"`
}

// fetchTagNames fetches tag names from Tempo.
func (c *tempoClient) fetchTagNames(ctx context.Context, scope, startUnix, endUnix string) ([]string, error) {
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

	var resp TempoTagsResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling tags response: %w", err)
	}

	return resp.TagNames, nil
}

// TempoTagValuesResponse represents the response from the tag values endpoint.
type TempoTagValuesResponse struct {
	TagValues []string `json:"tagValues"`
}

// fetchTagValues fetches values for a specific tag from Tempo.
func (c *tempoClient) fetchTagValues(ctx context.Context, tagName, startUnix, endUnix string) ([]string, error) {
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

	var resp TempoTagValuesResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling tag values response: %w", err)
	}

	return resp.TagValues, nil
}

// TempoSearchResponse represents the response from the search endpoint.
type TempoSearchResponse struct {
	Traces  []TempoTraceSearchResult `json:"traces"`
	Metrics *TempoSearchMetrics      `json:"metrics,omitempty"`
}

// TempoTraceSearchResult represents a single trace in search results.
type TempoTraceSearchResult struct {
	TraceID           string         `json:"traceID"`
	RootServiceName   string         `json:"rootServiceName"`
	RootTraceName     string         `json:"rootTraceName"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	DurationMs        int            `json:"durationMs"`
	SpanSets          []TempoSpanSet `json:"spanSets,omitempty"`
	ServiceStats      map[string]any `json:"serviceStats,omitempty"`
}

// TempoSpanSet represents a set of spans matching a query.
type TempoSpanSet struct {
	Spans   []TempoSpan `json:"spans"`
	Matched int         `json:"matched"`
}

// TempoSpan represents a span in a spanset.
type TempoSpan struct {
	SpanID            string           `json:"spanID"`
	StartTimeUnixNano string           `json:"startTimeUnixNano"`
	DurationNanos     string           `json:"durationNanos"`
	Attributes        []TempoAttribute `json:"attributes,omitempty"`
}

// TempoAttribute represents a span attribute.
type TempoAttribute struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// TempoSearchMetrics represents metrics from a search query.
// Field types match Tempo's protobuf definition.
// NOTE: uint64 fields are serialized as strings in JSON (protobuf convention for 64-bit ints),
// while uint32 fields come through as regular JSON numbers.
type TempoSearchMetrics struct {
	InspectedTraces int          `json:"inspectedTraces,omitempty"` // uint32 in proto → JSON number
	InspectedBytes  uint64String `json:"inspectedBytes,omitempty"`  // uint64 in proto → JSON string
}

// searchTraces searches for traces using TraceQL.
func (c *tempoClient) searchTraces(ctx context.Context, query, startUnix, endUnix string, limit int) (*TempoSearchResponse, error) {
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

	var resp TempoSearchResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling search response: %w", err)
	}

	return &resp, nil
}

// getTrace retrieves a trace by its ID.
func (c *tempoClient) getTrace(ctx context.Context, traceID string) (any, error) {
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

// getDefaultTempoTimeRange returns default start/end times if not specified (last 1 hour).
// Returns Unix epoch seconds as strings.
func getDefaultTempoTimeRange(startRFC3339, endRFC3339 string) (string, string, error) {
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

// enforceTempoTraceLimit ensures the limit is within bounds.
func enforceTempoTraceLimit(requestedLimit int) int {
	if requestedLimit <= 0 {
		return DefaultTempoTraceLimit
	}
	if requestedLimit > MaxTempoTraceLimit {
		return MaxTempoTraceLimit
	}
	return requestedLimit
}
