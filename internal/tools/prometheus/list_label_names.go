package prometheus

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
	Limit         int    `json:"limit,omitempty"`
}

func listLabelNamesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params listLabelNamesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	c, err := newClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Prometheus client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)
	labels, err := c.fetchLabels(ctx, startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Apply limit
	limit := enforceLimit(params.Limit, 0)
	if len(labels) > limit {
		labels = labels[:limit]
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
		"list_prometheus_label_names",
		mcp.WithDescription("Lists all available label names in a Prometheus datasource. "+
			"Returns a list of unique label strings (e.g., [\"__name__\", \"instance\", \"job\"]). "+
			"Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Prometheus datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of label names to return (default: 100)"),
		),
	)
}

// RegisterListLabelNames registers the list_prometheus_label_names tool.
func RegisterListLabelNames(s *server.MCPServer) {
	s.AddTool(newListLabelNamesTool(), listLabelNamesHandler)
}
