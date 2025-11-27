package prometheus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type listLabelValuesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	LabelName     string `json:"labelName"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

func listLabelValuesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params listLabelValuesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.LabelName == "" {
		return mcp.NewToolResultError("labelName is required"), nil
	}

	c, err := newClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Prometheus client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)
	values, err := c.fetchLabelValues(ctx, params.LabelName, startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Apply limit
	limit := enforceLimit(params.Limit, 0)
	if len(values) > limit {
		values = values[:limit]
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

func newListLabelValuesTool() mcp.Tool {
	return mcp.NewTool(
		"list_prometheus_label_values",
		mcp.WithDescription("Retrieves all unique values for a specific label name in a Prometheus datasource. "+
			"Returns a list of string values (e.g., for labelName=\"job\", might return [\"prometheus\", \"node-exporter\"]). "+
			"Use __name__ as the label name to get all metric names. Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Prometheus datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("labelName",
			mcp.Description("The label name to get values for (e.g., \"job\", \"instance\", or \"__name__\" for metric names)"),
			mcp.Required(),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of values to return (default: 100)"),
		),
	)
}

// RegisterListLabelValues registers the list_prometheus_label_values tool.
func RegisterListLabelValues(s *server.MCPServer) {
	s.AddTool(newListLabelValuesTool(), listLabelValuesHandler)
}
