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

// alertStateKey creates a key for matching alerts across APIs.
// Uses title + ruleGroup since the Prometheus API doesn't return UIDs.
func alertStateKey(title, ruleGroup string) string {
	return title + "|" + ruleGroup
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

	// Always get rules from provisioning API (this has UIDs)
	rules, err := client.listAlertRules(ctx, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build state map if state is requested
	stateMap := make(map[string]AlertRuleSummary)
	if params.IncludeState {
		stateRules, stateErr := client.getAlertRulesWithState(ctx)
		if stateErr != nil {
			// Log but continue - we can still return rules without state
			// State is nice-to-have, UIDs are essential
		} else {
			for _, sr := range stateRules {
				key := alertStateKey(sr.Title, sr.RuleGroup)
				stateMap[key] = sr
			}
		}
	}

	// Convert to summaries, enriching with state if available
	summaries := make([]AlertRuleSummary, 0, len(rules))
	for _, r := range rules {
		summary := AlertRuleSummary{
			UID:         r.UID,
			Title:       r.Title,
			FolderUID:   r.FolderUID,
			RuleGroup:   r.RuleGroup,
			For:         r.For,
			Labels:      r.Labels,
			Annotations: r.Annotations,
			IsPaused:    r.IsPaused,
		}

		// Enrich with state if available
		if params.IncludeState {
			key := alertStateKey(r.Title, r.RuleGroup)
			if stateSummary, ok := stateMap[key]; ok {
				summary.State = stateSummary.State
				summary.Health = stateSummary.Health
			}
		}

		summaries = append(summaries, summary)
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
