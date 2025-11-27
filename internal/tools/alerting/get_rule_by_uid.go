package alerting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type getRuleByUIDParams struct {
	UID string `json:"uid"`
}

func getRuleByUIDHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params getRuleByUIDParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.UID == "" {
		return mcp.NewToolResultError("uid is required"), nil
	}

	c, err := newClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating alerting client: %v", err)), nil
	}

	rule, err := c.getRuleByUID(ctx, params.UID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newGetRuleByUIDTool() mcp.Tool {
	return mcp.NewTool(
		"get_alert_rule_by_uid",
		mcp.WithDescription("Gets the full details of a Grafana alert rule by its UID. "+
			"Returns complete rule configuration including query definitions, conditions, "+
			"thresholds, and notification settings. "+
			"Use list_alert_rules first to find rule UIDs."),
		mcp.WithString("uid",
			mcp.Description("The UID of the alert rule"),
			mcp.Required(),
		),
	)
}

// RegisterGetRuleByUID registers the get_alert_rule_by_uid tool.
func RegisterGetRuleByUID(s *server.MCPServer) {
	s.AddTool(newGetRuleByUIDTool(), getRuleByUIDHandler)
}
