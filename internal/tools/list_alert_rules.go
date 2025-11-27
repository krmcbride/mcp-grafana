package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListAlertRulesParams defines the parameters for listing alert rules.
type ListAlertRulesParams struct {
	IncludeState bool `json:"includeState,omitempty"`
	Limit        int  `json:"limit,omitempty"`
}

func listAlertRulesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params ListAlertRulesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newAlertingClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating alerting client: %v", err)), nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultAlertRulesLimit
	}

	// If state is requested, use the Prometheus-style API
	if params.IncludeState {
		summaries, err := client.getAlertRulesWithState(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Apply limit
		if len(summaries) > limit {
			summaries = summaries[:limit]
		}

		if len(summaries) == 0 {
			summaries = []AlertRuleSummary{}
		}

		jsonData, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}

	// Otherwise use the provisioning API
	rules, err := client.listAlertRules(ctx, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Convert to summaries
	summaries := make([]AlertRuleSummary, 0, len(rules))
	for _, r := range rules {
		summaries = append(summaries, AlertRuleSummary{
			UID:         r.UID,
			Title:       r.Title,
			FolderUID:   r.FolderUID,
			RuleGroup:   r.RuleGroup,
			For:         r.For,
			Labels:      r.Labels,
			Annotations: r.Annotations,
			IsPaused:    r.IsPaused,
		})
	}

	if len(summaries) == 0 {
		summaries = []AlertRuleSummary{}
	}

	jsonData, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListAlertRulesTool() mcp.Tool {
	return mcp.NewTool(
		"list_alert_rules",
		mcp.WithDescription("Lists Grafana alert rules. "+
			"Returns a summary of each alert including UID, title, rule group, labels, and annotations. "+
			"Set includeState=true to also include current state (firing, pending, inactive) and health information. "+
			"Note: When includeState=true, the response comes from a different API and may have slightly different fields."),
		mcp.WithBoolean("includeState",
			mcp.Description("If true, include current alert state (firing, pending, inactive) and health. Default: false"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of alert rules to return (default: 100)"),
		),
	)
}

// RegisterListAlertRules registers the list_alert_rules tool.
func RegisterListAlertRules(s *server.MCPServer) {
	s.AddTool(newListAlertRulesTool(), listAlertRulesHandler)
}
