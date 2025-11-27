package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetAlertRuleByUIDParams defines the parameters for getting an alert rule by UID.
type GetAlertRuleByUIDParams struct {
	UID string `json:"uid"`
}

func getAlertRuleByUIDHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params GetAlertRuleByUIDParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.UID == "" {
		return mcp.NewToolResultError("uid is required"), nil
	}

	client, err := newAlertingClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating alerting client: %v", err)), nil
	}

	rule, err := client.getAlertRuleByUID(ctx, params.UID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newGetAlertRuleByUIDTool() mcp.Tool {
	return mcp.NewTool(
		"get_alert_rule_by_uid",
		mcp.WithDescription("Retrieves the full configuration of a specific Grafana alert rule by its UID. "+
			"Returns detailed information including title, condition, query data, folder, rule group, "+
			"no-data and error state settings, labels, annotations, and whether the rule is paused. "+
			"Use list_alert_rules first to find the UID."),
		mcp.WithString("uid",
			mcp.Description("The UID of the alert rule to retrieve"),
			mcp.Required(),
		),
	)
}

// RegisterGetAlertRuleByUID registers the get_alert_rule_by_uid tool.
func RegisterGetAlertRuleByUID(s *server.MCPServer) {
	s.AddTool(newGetAlertRuleByUIDTool(), getAlertRuleByUIDHandler)
}
