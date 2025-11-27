package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	DefaultDashboardSearchLimit = 50
)

// SearchDashboardsParams defines the parameters for searching dashboards.
type SearchDashboardsParams struct {
	Query string `json:"query,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func searchDashboardsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params SearchDashboardsParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newDashboardClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating dashboard client: %v", err)), nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultDashboardSearchLimit
	}

	results, err := client.searchDashboards(ctx, params.Query, params.Tag, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(results) == 0 {
		results = []SearchResult{}
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newSearchDashboardsTool() mcp.Tool {
	return mcp.NewTool(
		"search_dashboards",
		mcp.WithDescription("Searches for Grafana dashboards by query string or tag. "+
			"Returns a list of matching dashboards with UID, title, tags, folder information, and URL. "+
			"Use the UID from results to call get_dashboard_summary or get_dashboard_panel_queries for more details."),
		mcp.WithString("query",
			mcp.Description("Search query string to match against dashboard titles"),
		),
		mcp.WithString("tag",
			mcp.Description("Filter dashboards by tag"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 50)"),
		),
	)
}

// RegisterSearchDashboards registers the search_dashboards tool.
func RegisterSearchDashboards(s *server.MCPServer) {
	s.AddTool(newSearchDashboardsTool(), searchDashboardsHandler)
}
