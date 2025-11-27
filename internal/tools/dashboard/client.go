// Package dashboard provides MCP tools for interacting with Grafana dashboards.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
)

const (
	// DefaultSearchLimit is the default limit for dashboard searches.
	DefaultSearchLimit = 50
)

// client provides methods for interacting with Grafana's dashboard API.
type client struct {
	httpClient *http.Client
	baseURL    string
}

// newClient creates a new dashboard client.
func newClient() (*client, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	return &client{
		httpClient: httpClient,
		baseURL:    grafanaURL,
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

// SearchResult represents a dashboard search result.
type SearchResult struct {
	ID          int      `json:"id"`
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	URI         string   `json:"uri"`
	URL         string   `json:"url"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	IsStarred   bool     `json:"isStarred"`
	FolderID    int      `json:"folderId,omitempty"`
	FolderUID   string   `json:"folderUid,omitempty"`
	FolderTitle string   `json:"folderTitle,omitempty"`
	FolderURL   string   `json:"folderUrl,omitempty"`
}

// searchDashboards searches for dashboards.
func (c *client) searchDashboards(ctx context.Context, query string, tag string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Add("type", "dash-db")

	if query != "" {
		params.Add("query", query)
	}
	if tag != "" {
		params.Add("tag", tag)
	}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/search", params)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(bodyBytes, &results); err != nil {
		return nil, fmt.Errorf("unmarshalling search results: %w", err)
	}

	return results, nil
}

// Response represents the response from getting a dashboard by UID.
type Response struct {
	Meta      Meta `json:"meta"`
	Dashboard any  `json:"dashboard"`
}

// Meta contains metadata about a dashboard.
type Meta struct {
	Type        string `json:"type"`
	CanSave     bool   `json:"canSave"`
	CanEdit     bool   `json:"canEdit"`
	CanAdmin    bool   `json:"canAdmin"`
	CanStar     bool   `json:"canStar"`
	CanDelete   bool   `json:"canDelete"`
	Slug        string `json:"slug"`
	URL         string `json:"url"`
	Expires     string `json:"expires"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	UpdatedBy   string `json:"updatedBy"`
	CreatedBy   string `json:"createdBy"`
	Version     int    `json:"version"`
	HasACL      bool   `json:"hasAcl"`
	IsFolder    bool   `json:"isFolder"`
	FolderID    int    `json:"folderId"`
	FolderUID   string `json:"folderUid"`
	FolderTitle string `json:"folderTitle"`
	FolderURL   string `json:"folderUrl"`
}

// getDashboardByUID gets a dashboard by its UID.
func (c *client) getDashboardByUID(ctx context.Context, uid string) (*Response, error) {
	path := fmt.Sprintf("/api/dashboards/uid/%s", url.PathEscape(uid))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling dashboard response: %w", err)
	}

	return &response, nil
}

// Summary provides a compact overview of a dashboard.
type Summary struct {
	UID         string            `json:"uid"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	FolderTitle string            `json:"folderTitle,omitempty"`
	URL         string            `json:"url,omitempty"`
	PanelCount  int               `json:"panelCount"`
	Panels      []PanelSummary    `json:"panels"`
	Variables   []VariableSummary `json:"variables,omitempty"`
	Created     string            `json:"created,omitempty"`
	Updated     string            `json:"updated,omitempty"`
}

// PanelSummary provides a compact overview of a panel.
type PanelSummary struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	QueryCount  int    `json:"queryCount"`
}

// VariableSummary provides a compact overview of a template variable.
type VariableSummary struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
}

// PanelQuery represents a query extracted from a dashboard panel.
type PanelQuery struct {
	PanelID        int            `json:"panelId"`
	PanelTitle     string         `json:"panelTitle"`
	DatasourceUID  string         `json:"datasourceUid,omitempty"`
	DatasourceType string         `json:"datasourceType,omitempty"`
	QueryExpr      string         `json:"queryExpr,omitempty"`
	RefID          string         `json:"refId,omitempty"`
	RawQuery       map[string]any `json:"rawQuery,omitempty"`
}
