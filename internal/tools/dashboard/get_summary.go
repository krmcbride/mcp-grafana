package dashboard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type getSummaryParams struct {
	UID string `json:"uid"`
}

func getSummaryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params getSummaryParams
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

	summary := buildSummary(params.UID, dashResponse)

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// buildSummary builds a summary from a dashboard response.
func buildSummary(uid string, dashResponse *Response) *Summary {
	summary := &Summary{
		UID:         uid,
		FolderTitle: dashResponse.Meta.FolderTitle,
		URL:         dashResponse.Meta.URL,
		Created:     dashResponse.Meta.Created,
		Updated:     dashResponse.Meta.Updated,
	}

	// Extract dashboard data
	dashMap, ok := dashResponse.Dashboard.(map[string]any)
	if !ok {
		return summary
	}

	// Basic fields
	if title, ok := dashMap["title"].(string); ok {
		summary.Title = title
	}
	if desc, ok := dashMap["description"].(string); ok {
		summary.Description = desc
	}

	// Tags
	if tags, ok := dashMap["tags"].([]any); ok {
		for _, t := range tags {
			if tagStr, ok := t.(string); ok {
				summary.Tags = append(summary.Tags, tagStr)
			}
		}
	}

	// Panels
	if panels, ok := dashMap["panels"].([]any); ok {
		summary.PanelCount = len(panels)
		for _, p := range panels {
			if panelMap, ok := p.(map[string]any); ok {
				panelSummary := PanelSummary{}

				if id, ok := panelMap["id"].(float64); ok {
					panelSummary.ID = int(id)
				}
				if title, ok := panelMap["title"].(string); ok {
					panelSummary.Title = title
				}
				if pType, ok := panelMap["type"].(string); ok {
					panelSummary.Type = pType
				}
				if desc, ok := panelMap["description"].(string); ok {
					panelSummary.Description = desc
				}
				if targets, ok := panelMap["targets"].([]any); ok {
					panelSummary.QueryCount = len(targets)
				}

				summary.Panels = append(summary.Panels, panelSummary)
			}
		}
	}

	// Variables (from templating)
	if templating, ok := dashMap["templating"].(map[string]any); ok {
		if list, ok := templating["list"].([]any); ok {
			for _, v := range list {
				if varMap, ok := v.(map[string]any); ok {
					varSummary := VariableSummary{}

					if name, ok := varMap["name"].(string); ok {
						varSummary.Name = name
					}
					if vType, ok := varMap["type"].(string); ok {
						varSummary.Type = vType
					}
					if label, ok := varMap["label"].(string); ok {
						varSummary.Label = label
					}

					summary.Variables = append(summary.Variables, varSummary)
				}
			}
		}
	}

	return summary
}

func newGetSummaryTool() mcp.Tool {
	return mcp.NewTool(
		"get_dashboard_summary",
		mcp.WithDescription("Gets a compact summary of a Grafana dashboard including title, description, tags, "+
			"panel count and types, template variables, and metadata. "+
			"Use this instead of fetching the full dashboard JSON to minimize context window usage. "+
			"Use search_dashboards first to find the dashboard UID."),
		mcp.WithString("uid",
			mcp.Description("The UID of the dashboard"),
			mcp.Required(),
		),
	)
}

// RegisterGetSummary registers the get_dashboard_summary tool.
func RegisterGetSummary(s *server.MCPServer) {
	s.AddTool(newGetSummaryTool(), getSummaryHandler)
}
