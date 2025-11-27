package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListTempoTagNamesParams defines the parameters for listing Tempo tag names.
type ListTempoTagNamesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	Scope         string `json:"scope,omitempty"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
}

func listTempoTagNamesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params ListTempoTagNamesParams
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

	tagNames, err := client.fetchTagNames(ctx, params.Scope, startUnix, endUnix)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(tagNames) == 0 {
		tagNames = []string{}
	}

	jsonData, err := json.MarshalIndent(tagNames, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListTempoTagNamesTool() mcp.Tool {
	return mcp.NewTool(
		"list_tempo_tag_names",
		mcp.WithDescription("Lists all available tag names (attributes) in a Tempo datasource. "+
			"Returns a list of tag name strings (e.g., [\"service.name\", \"http.method\", \"http.status_code\"]). "+
			"Optionally filter by scope (resource, span, intrinsic). "+
			"Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Tempo datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("scope",
			mcp.Description("Optional scope filter: 'resource', 'span', 'intrinsic', 'event', 'link', or 'instrumentation'"),
		),
		mcp.WithString("startRfc3339",
			mcp.Description("Start time in RFC3339 format (defaults to 1 hour ago)"),
		),
		mcp.WithString("endRfc3339",
			mcp.Description("End time in RFC3339 format (defaults to now)"),
		),
	)
}

// RegisterListTempoTagNames registers the list_tempo_tag_names tool.
func RegisterListTempoTagNames(s *server.MCPServer) {
	s.AddTool(newListTempoTagNamesTool(), listTempoTagNamesHandler)
}
