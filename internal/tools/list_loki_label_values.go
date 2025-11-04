package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListLokiLabelValuesParams defines parameters for listing Loki label values.
type ListLokiLabelValuesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	LabelName     string `json:"labelName"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
}

func listLokiLabelValuesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params ListLokiLabelValuesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newLokiClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Loki client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)

	path := fmt.Sprintf("/loki/api/v1/label/%s/values", params.LabelName)
	values, err := client.fetchLabels(ctx, path, startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(values) == 0 {
		values = []string{}
	}

	jsonData, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListLokiLabelValuesTool() mcp.Tool {
	return mcp.NewTool(
		"list_loki_label_values",
		mcp.WithDescription("Retrieves all unique values for a specific label name within a Loki datasource and time range. Returns a list of string values (e.g., for labelName=\"env\", might return [\"prod\", \"staging\", \"dev\"]). Useful for discovering filter options. Defaults to the last hour if time range is omitted."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Loki datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("labelName",
			mcp.Description("The name of the label to retrieve values for (e.g., 'app', 'env', 'pod')"),
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

// RegisterListLokiLabelValues registers the list_loki_label_values tool with the MCP server.
func RegisterListLokiLabelValues(s *server.MCPServer) {
	s.AddTool(newListLokiLabelValuesTool(), listLokiLabelValuesHandler)
}
