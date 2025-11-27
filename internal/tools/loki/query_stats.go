package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Stats represents statistics from Loki's index/stats endpoint.
type Stats struct {
	Streams int `json:"streams"`
	Chunks  int `json:"chunks"`
	Entries int `json:"entries"`
	Bytes   int `json:"bytes"`
}

type queryStatsParams struct {
	DatasourceUID string `json:"datasourceUid"`
	LogQL         string `json:"logql"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
}

func (c *client) fetchStats(ctx context.Context, query, startRFC3339, endRFC3339 string) (*Stats, error) {
	params := url.Values{}
	params.Add("query", query)

	if err := addTimeRangeParams(params, startRFC3339, endRFC3339); err != nil {
		return nil, err
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/loki/api/v1/index/stats", params)
	if err != nil {
		return nil, err
	}

	var stats Stats
	if err := json.Unmarshal(bodyBytes, &stats); err != nil {
		return nil, fmt.Errorf("unmarshalling stats response: %w", err)
	}

	return &stats, nil
}

func queryStatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params queryStatsParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	c, err := newClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Loki client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)

	stats, err := c.fetchStats(ctx, params.LogQL, startTime, endTime)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newQueryStatsTool() mcp.Tool {
	return mcp.NewTool(
		"query_loki_stats",
		mcp.WithDescription("Retrieves statistics about log streams matching a LogQL selector within a Loki datasource and time range. Returns counts of streams, chunks, entries, and bytes. The logql parameter must be a simple label selector (e.g., '{app=\"nginx\"}') and does not support line filters or aggregations. Useful for checking query size before fetching logs. Defaults to the last hour."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Loki datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("logql",
			mcp.Description("LogQL label selector expression (e.g., '{app=\"nginx\"}')"),
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

// RegisterQueryStats registers the query_loki_stats tool with the MCP server.
func RegisterQueryStats(s *server.MCPServer) {
	s.AddTool(newQueryStatsTool(), queryStatsHandler)
}
