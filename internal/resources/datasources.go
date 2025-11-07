package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Datasource represents a Grafana datasource with key identification fields.
type Datasource struct {
	UID       string `json:"uid"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsDefault bool   `json:"isDefault"`
	URL       string `json:"url,omitempty"`
}

func RegisterDatasourcesMCPResource(s *server.MCPServer) {
	s.AddResource(newDatasourcesMCPResource(), datasourcesHandler)
}

// Resource schema
func newDatasourcesMCPResource() mcp.Resource {
	return mcp.NewResource("grafana://datasources", "grafana_datasources",
		mcp.WithResourceDescription("Available Grafana datasources - lists datasource UIDs, names, and types "+
			"for easy discovery when using tools like query_loki_logs or other datasource-specific operations. "+
			"Use this resource to find valid datasourceUid values instead of having to specify them manually."),
		mcp.WithMIMEType("application/json"),
	)
}

// Resource handler
func datasourcesHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get authenticated HTTP client
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, fmt.Errorf("creating Grafana client: %w", err)
	}

	// Build request to list datasources
	req, err := http.NewRequestWithContext(ctx, "GET", grafanaURL+"/api/datasources", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching datasources: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var rawDatasources []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rawDatasources); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract relevant fields
	datasources := make([]Datasource, 0, len(rawDatasources))
	for _, ds := range rawDatasources {
		datasource := Datasource{}

		if uid, ok := ds["uid"].(string); ok {
			datasource.UID = uid
		}
		if name, ok := ds["name"].(string); ok {
			datasource.Name = name
		}
		if dsType, ok := ds["type"].(string); ok {
			datasource.Type = dsType
		}
		if isDefault, ok := ds["isDefault"].(bool); ok {
			datasource.IsDefault = isDefault
		}
		if url, ok := ds["url"].(string); ok {
			datasource.URL = url
		}

		datasources = append(datasources, datasource)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(datasources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling datasources: %w", err)
	}

	// Return as MCP resource contents
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "grafana://datasources",
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}
