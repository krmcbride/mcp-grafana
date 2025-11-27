package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListPrometheusMetricNamesParams defines the parameters for listing Prometheus metric names.
type ListPrometheusMetricNamesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	Regex         string `json:"regex,omitempty"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

func listPrometheusMetricNamesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params ListPrometheusMetricNamesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newPrometheusClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Prometheus client: %v", err)), nil
	}

	startTime, endTime := getDefaultPrometheusTimeRange(params.StartRFC3339, params.EndRFC3339)

	// Fetch all metric names using __name__ label
	metricNames, err := client.fetchLabelValues(ctx, "__name__", startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Filter by regex if provided
	if params.Regex != "" {
		re, err := regexp.Compile(params.Regex)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid regex: %v", err)), nil
		}

		filtered := make([]string, 0)
		for _, name := range metricNames {
			if re.MatchString(name) {
				filtered = append(filtered, name)
			}
		}
		metricNames = filtered
	}

	// Apply limit
	limit := enforcePrometheusLimit(params.Limit, 0)
	if len(metricNames) > limit {
		metricNames = metricNames[:limit]
	}

	if len(metricNames) == 0 {
		metricNames = []string{}
	}

	jsonData, err := json.MarshalIndent(metricNames, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListPrometheusMetricNamesTool() mcp.Tool {
	return mcp.NewTool(
		"list_prometheus_metric_names",
		mcp.WithDescription("Lists metric names in a Prometheus datasource. "+
			"Returns a list of metric names (e.g., [\"up\", \"node_cpu_seconds_total\"]). "+
			"Supports filtering by regex pattern. "+
			"Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Prometheus datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("regex",
			mcp.Description("Optional regex pattern to filter metric names (e.g., \"node_.*\" for node exporter metrics)"),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of metric names to return (default: 100)"),
		),
	)
}

// RegisterListPrometheusMetricNames registers the list_prometheus_metric_names tool.
func RegisterListPrometheusMetricNames(s *server.MCPServer) {
	s.AddTool(newListPrometheusMetricNamesTool(), listPrometheusMetricNamesHandler)
}
