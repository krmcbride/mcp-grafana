package dashboard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type getPanelQueriesParams struct {
	UID string `json:"uid"`
}

func getPanelQueriesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params getPanelQueriesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.UID == "" {
		return mcp.NewToolResultError("uid is required"), nil
	}

	c, err := newClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating dashboard client: %v", err)), nil
	}

	dashResponse, err := c.getDashboardByUID(ctx, params.UID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queries := extractPanelQueries(dashResponse)

	if len(queries) == 0 {
		queries = []PanelQuery{}
	}

	jsonData, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// extractPanelQueries extracts all queries from a dashboard's panels.
func extractPanelQueries(dashResponse *Response) []PanelQuery {
	var queries []PanelQuery

	dashMap, ok := dashResponse.Dashboard.(map[string]any)
	if !ok {
		return queries
	}

	panels, ok := dashMap["panels"].([]any)
	if !ok {
		return queries
	}

	for _, p := range panels {
		panelMap, ok := p.(map[string]any)
		if !ok {
			continue
		}

		panelID := 0
		if id, ok := panelMap["id"].(float64); ok {
			panelID = int(id)
		}

		panelTitle := ""
		if title, ok := panelMap["title"].(string); ok {
			panelTitle = title
		}

		// Get panel-level datasource
		panelDsUID := ""
		panelDsType := ""
		if ds, ok := panelMap["datasource"].(map[string]any); ok {
			if uid, ok := ds["uid"].(string); ok {
				panelDsUID = uid
			}
			if dsType, ok := ds["type"].(string); ok {
				panelDsType = dsType
			}
		}

		// Extract queries from targets
		targets, ok := panelMap["targets"].([]any)
		if !ok {
			continue
		}

		for _, t := range targets {
			targetMap, ok := t.(map[string]any)
			if !ok {
				continue
			}

			query := PanelQuery{
				PanelID:        panelID,
				PanelTitle:     panelTitle,
				DatasourceUID:  panelDsUID,
				DatasourceType: panelDsType,
			}

			// Check for target-level datasource override
			if ds, ok := targetMap["datasource"].(map[string]any); ok {
				if uid, ok := ds["uid"].(string); ok {
					query.DatasourceUID = uid
				}
				if dsType, ok := ds["type"].(string); ok {
					query.DatasourceType = dsType
				}
			}

			// Extract refId
			if refID, ok := targetMap["refId"].(string); ok {
				query.RefID = refID
			}

			// Extract query expression based on datasource type
			// Prometheus/Loki use "expr", some use "query", etc.
			if expr, ok := targetMap["expr"].(string); ok && expr != "" {
				query.QueryExpr = expr
			} else if queryStr, ok := targetMap["query"].(string); ok && queryStr != "" {
				query.QueryExpr = queryStr
			}

			// Store the raw query for complex queries
			query.RawQuery = targetMap

			queries = append(queries, query)
		}
	}

	return queries
}

func newGetPanelQueriesTool() mcp.Tool {
	return mcp.NewTool(
		"get_dashboard_panel_queries",
		mcp.WithDescription("Extracts all queries from a Grafana dashboard's panels. "+
			"Returns the panel ID, title, datasource information, and query expressions for each panel target. "+
			"Useful for understanding what a dashboard is monitoring and for running those queries directly. "+
			"Note: If datasourceUid is a template variable (e.g., '$datasource'), "+
			"you'll need to resolve it using the grafana://datasources resource."),
		mcp.WithString("uid",
			mcp.Description("The UID of the dashboard"),
			mcp.Required(),
		),
	)
}

// RegisterGetPanelQueries registers the get_dashboard_panel_queries tool.
func RegisterGetPanelQueries(s *server.MCPServer) {
	s.AddTool(newGetPanelQueriesTool(), getPanelQueriesHandler)
}
