package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetTempoTraceParams defines the parameters for getting a Tempo trace by ID.
type GetTempoTraceParams struct {
	DatasourceUID string `json:"datasourceUid"`
	TraceID       string `json:"traceId"`
}

func getTempoTraceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params GetTempoTraceParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.TraceID == "" {
		return mcp.NewToolResultError("traceId is required"), nil
	}

	client, err := newTempoClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Tempo client: %v", err)), nil
	}

	trace, err := client.getTrace(ctx, params.TraceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newGetTempoTraceTool() mcp.Tool {
	return mcp.NewTool(
		"get_tempo_trace",
		mcp.WithDescription("Retrieves a complete trace by its trace ID from a Tempo datasource. "+
			"Returns the full trace data including all spans, their attributes, and timing information. "+
			"Use search_tempo_traces first to find trace IDs of interest."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Tempo datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("traceId",
			mcp.Description("The trace ID to retrieve (32-character hex string)"),
			mcp.Required(),
		),
	)
}

// RegisterGetTempoTrace registers the get_tempo_trace tool.
func RegisterGetTempoTrace(s *server.MCPServer) {
	s.AddTool(newGetTempoTraceTool(), getTempoTraceHandler)
}
