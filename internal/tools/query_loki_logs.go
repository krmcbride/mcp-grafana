package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// logStream represents a stream of log entries from Loki.
type logStream struct {
	Stream map[string]string   `json:"stream"`
	Values [][]json.RawMessage `json:"values"` // [timestamp, value]
}

// queryRangeResponse represents the response from Loki's query_range API.
type queryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string      `json:"resultType"`
		Result     []logStream `json:"result"`
	} `json:"data"`
}

// LogEntry represents a single log entry with metadata.
type LogEntry struct {
	Timestamp string            `json:"timestamp"`
	Line      string            `json:"line,omitempty"`  // For log queries
	Value     *float64          `json:"value,omitempty"` // For metric queries
	Labels    map[string]string `json:"labels"`
}

// QueryLokiLogsParams defines parameters for querying Loki logs.
type QueryLokiLogsParams struct {
	DatasourceUID string `json:"datasourceUid"`
	LogQL         string `json:"logql"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Direction     string `json:"direction,omitempty"`
}

func (c *lokiClient) fetchLogs(ctx context.Context, query, startRFC3339, endRFC3339 string, limit int, direction string) ([]logStream, error) {
	params := url.Values{}
	params.Add("query", query)

	if err := addTimeRangeParams(params, startRFC3339, endRFC3339); err != nil {
		return nil, err
	}

	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}

	if direction != "" {
		params.Add("direction", direction)
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/loki/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	var response queryRangeResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling query response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("loki API returned unexpected status: %s", response.Status)
	}

	return response.Data.Result, nil
}

func queryLokiLogsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params QueryLokiLogsParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newLokiClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Loki client: %v", err)), nil
	}

	startTime, endTime := getDefaultTimeRange(params.StartRFC3339, params.EndRFC3339)
	limit := enforceLogLimit(params.Limit)

	direction := params.Direction
	if direction == "" {
		direction = "backward" // Newest first by default
	}

	streams, err := client.fetchLogs(ctx, params.LogQL, startTime, endTime, limit, direction)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(streams) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	// Convert streams to flat list of log entries
	var entries []LogEntry
	for _, stream := range streams {
		for _, value := range stream.Values {
			if len(value) < 2 {
				continue
			}

			entry := LogEntry{
				Timestamp: strings.Trim(string(value[0]), "\""),
				Labels:    stream.Stream,
			}

			// Handle metric queries (numeric values) vs log queries (strings)
			if stream.Stream["__type__"] == "metrics" {
				// Try parsing as numeric value
				var numStr string
				if err := json.Unmarshal(value[1], &numStr); err == nil {
					if v, err := strconv.ParseFloat(numStr, 64); err == nil {
						entry.Value = &v
					} else {
						continue // Skip invalid values
					}
				} else {
					var v float64
					if err := json.Unmarshal(value[1], &v); err == nil {
						entry.Value = &v
					} else {
						continue // Skip invalid values
					}
				}
			} else {
				// Parse as log line string
				var logLine string
				if err := json.Unmarshal(value[1], &logLine); err == nil {
					entry.Line = logLine
				} else {
					continue // Skip invalid lines
				}
			}

			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newQueryLokiLogsTool() mcp.Tool {
	return mcp.NewTool(
		"query_loki_logs",
		mcp.WithDescription("Executes a LogQL query against a Loki datasource to retrieve log entries. Supports full LogQL syntax including label matchers, filters, and pipeline operations (e.g., '{app=\"nginx\"} |= \"error\"'). Returns a list of log entries with timestamp, labels, and log line. Defaults to last hour, 10 entries, newest first. Consider using query_loki_stats first to check query size."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Loki datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("logql",
			mcp.Description("LogQL query expression (e.g., '{app=\"nginx\"} |= \"error\"')"),
			mcp.Required(),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of log lines to return (default: 10, max: 100)"),
		),
		mcp.WithString("direction",
			mcp.Description("Query direction: 'forward' (oldest first) or 'backward' (newest first, default)"),
		),
	)
}

// RegisterQueryLokiLogs registers the query_loki_logs tool with the MCP server.
func RegisterQueryLokiLogs(s *server.MCPServer) {
	s.AddTool(newQueryLokiLogsTool(), queryLokiLogsHandler)
}
