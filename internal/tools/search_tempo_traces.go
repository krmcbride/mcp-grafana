package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SearchTempoTracesParams defines the parameters for searching Tempo traces.
type SearchTempoTracesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	Query         string `json:"query,omitempty"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

func searchTempoTracesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params SearchTempoTracesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	client, err := newTempoClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Tempo client: %v", err)), nil
	}

	startUnix, endUnix, err := getDefaultTempoTimeRange(params.StartRFC3339, params.EndRFC3339)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	limit := enforceTempoTraceLimit(params.Limit)

	searchResult, err := client.searchTraces(ctx, params.Query, startUnix, endUnix, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.MarshalIndent(searchResult, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newSearchTempoTracesTool() mcp.Tool {
	return mcp.NewTool(
		"search_tempo_traces",
		mcp.WithDescription("Searches for traces in a Tempo datasource using TraceQL. "+
			"Returns a list of matching traces with trace ID, root service name, root trace name, start time, and duration. "+
			"TraceQL examples: '{service.name=\"api-gateway\"}', '{http.status_code>=400}', '{duration>1s}'. "+
			"If no query is provided, returns recent traces. "+
			"Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Tempo datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("query",
			mcp.Description("TraceQL query expression (e.g., '{service.name=\"api\"}', '{http.status_code>=400}'). If empty, returns recent traces."),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of traces to return (default: 20, max: 100)"),
		),
	)
}

// RegisterSearchTempoTraces registers the search_tempo_traces tool.
func RegisterSearchTempoTraces(s *server.MCPServer) {
	s.AddTool(newSearchTempoTracesTool(), searchTempoTracesHandler)
}
