package prometheus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type queryParams struct {
	DatasourceUID string `json:"datasourceUid"`
	Expr          string `json:"expr"`
	QueryType     string `json:"queryType,omitempty"`    // "instant" or "range", defaults to "instant"
	TimeRFC3339   string `json:"timeRfc3339,omitempty"`  // For instant queries
	StartRFC3339  string `json:"startRfc3339,omitempty"` // For range queries
	EndRFC3339    string `json:"endRfc3339,omitempty"`   // For range queries
	StepSeconds   int    `json:"stepSeconds,omitempty"`  // For range queries
}

func queryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params queryParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.Expr == "" {
		return mcp.NewToolResultError("expr (PromQL expression) is required"), nil
	}

	c, err := newClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Prometheus client: %v", err)), nil
	}

	queryType := params.QueryType
	if queryType == "" {
		queryType = "instant"
	}

	var result *QueryResult

	switch queryType {
	case "instant":
		result, err = c.query(ctx, params.Expr, params.TimeRFC3339)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("executing instant query: %v", err)), nil
		}

	case "range":
		startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)

		stepSeconds := params.StepSeconds
		if stepSeconds <= 0 {
			stepSeconds = DefaultStepSeconds
		}

		result, err = c.queryRange(ctx, params.Expr, startTime, endTime, stepSeconds)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("executing range query: %v", err)), nil
		}

	default:
		return mcp.NewToolResultError(fmt.Sprintf("invalid queryType: %s (must be 'instant' or 'range')", queryType)), nil
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newQueryTool() mcp.Tool {
	return mcp.NewTool(
		"query_prometheus",
		mcp.WithDescription("Executes a PromQL query against a Prometheus datasource. "+
			"Supports both instant queries (at a single point in time) and range queries (over a time range). "+
			"For instant queries, optionally specify timeRfc3339. "+
			"For range queries, set queryType='range' and optionally specify startRfc3339, endRfc3339, and stepSeconds. "+
			"Returns the query result with resultType (vector, matrix, scalar, string) and result data."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Prometheus datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("expr",
			mcp.Description("PromQL expression to evaluate (e.g., 'up', 'rate(http_requests_total[5m])')"),
			mcp.Required(),
		),
		mcp.WithString("queryType",
			mcp.Description("Query type: 'instant' (default) for a single point in time, or 'range' for a time series"),
		),
		mcp.WithString("timeRfc3339",
			mcp.Description("Evaluation time for instant queries in RFC3339 format (defaults to now)"),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time for range queries in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time for range queries in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("stepSeconds",
			mcp.Description("Step interval for range queries in seconds (default: 60)"),
		),
	)
}

// RegisterQuery registers the query_prometheus tool.
func RegisterQuery(s *server.MCPServer) {
	s.AddTool(newQueryTool(), queryHandler)
}
