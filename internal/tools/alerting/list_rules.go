package alerting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type listRulesParams struct {
	Limit        int  `json:"limit,omitempty"`
	IncludeState bool `json:"includeState,omitempty"`
}

// alertStateKey creates a key for matching alerts across APIs.
func alertStateKey(title, ruleGroup string) string {
	return title + "|" + ruleGroup
}

func listRulesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params listRulesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	c, err := newClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating alerting client: %v", err)), nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultRulesLimit
	}

	// Always get rules from provisioning API (this has UIDs)
	rules, err := c.listRules(ctx, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build state map if state is requested
	stateMap := make(map[string]RuleSummary)
	if params.IncludeState {
		stateRules, stateErr := c.getRulesWithState(ctx)
		if stateErr != nil {
			// Log but don't fail - we still have the rules, just without state
			// State enrichment is best-effort
		} else {
			for _, sr := range stateRules {
				key := alertStateKey(sr.Title, sr.RuleGroup)
				stateMap[key] = sr
			}
		}
	}

	// Convert to summaries, enriching with state if available
	summaries := make([]RuleSummary, 0, len(rules))
	for _, r := range rules {
		summary := RuleSummary{
			UID:         r.UID,
			Title:       r.Title,
			FolderUID:   r.FolderUID,
			RuleGroup:   r.RuleGroup,
			For:         r.For,
			Labels:      r.Labels,
			Annotations: r.Annotations,
			IsPaused:    r.IsPaused,
		}

		if params.IncludeState {
			key := alertStateKey(r.Title, r.RuleGroup)
			if stateSummary, ok := stateMap[key]; ok {
				summary.State = stateSummary.State
				summary.Health = stateSummary.Health
			}
		}

		summaries = append(summaries, summary)
	}

	jsonData, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListRulesTool() mcp.Tool {
	return mcp.NewTool(
		"list_alert_rules",
		mcp.WithDescription("Lists Grafana alert rules with optional state information. "+
			"Returns rule UID, title, folder, group, labels, annotations, and pause status. "+
			"When includeState is true, also includes current firing state and health. "+
			"Use get_alert_rule_by_uid for full rule details including query definitions."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of rules to return (default: 100)"),
		),
		mcp.WithBoolean("includeState",
			mcp.Description("Include current firing state and health from Prometheus-style API (default: false)"),
		),
	)
}

// RegisterListRules registers the list_alert_rules tool.
func RegisterListRules(s *server.MCPServer) {
	s.AddTool(newListRulesTool(), listRulesHandler)
}
