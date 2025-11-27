package loki

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type listLabelNamesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
}

func listLabelNamesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params listLabelNamesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	c, err := newClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Loki client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)

	labels, err := c.fetchLabels(ctx, "/loki/api/v1/labels", startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(labels) == 0 {
		labels = []string{}
	}

	jsonData, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListLabelNamesTool() mcp.Tool {
	return mcp.NewTool(
		"list_loki_label_names",
		mcp.WithDescription("Lists all available label names (keys) found in logs within a Loki datasource and time range. Returns a list of unique label strings (e.g., [\"app\", \"env\", \"pod\"]). Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Loki datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
	)
}

// RegisterListLabelNames registers the list_loki_label_names tool with the MCP server.
func RegisterListLabelNames(s *server.MCPServer) {
	s.AddTool(newListLabelNamesTool(), listLabelNamesHandler)
}
