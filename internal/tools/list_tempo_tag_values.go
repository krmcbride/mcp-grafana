package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListTempoTagValuesParams defines the parameters for listing Tempo tag values.
type ListTempoTagValuesParams struct {
	DatasourceUID string `json:"datasourceUid"`
	TagName       string `json:"tagName"`
	StartRFC3339  string `json:"startRfc3339,omitempty"`
	EndRFC3339    string `json:"endRfc3339,omitempty"`
}

func listTempoTagValuesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params ListTempoTagValuesParams
	if err := request.BindArguments(&params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.TagName == "" {
		return mcp.NewToolResultError("tagName is required"), nil
	}

	client, err := newTempoClient(params.DatasourceUID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating Tempo client: %v", err)), nil
	}

	startUnix, endUnix, err := getDefaultTempoTimeRange(params.StartRFC3339, params.EndRFC3339)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tagValues, err := client.fetchTagValues(ctx, params.TagName, startUnix, endUnix)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(tagValues) == 0 {
		tagValues = []string{}
	}

	jsonData, err := json.MarshalIndent(tagValues, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func newListTempoTagValuesTool() mcp.Tool {
	return mcp.NewTool(
		"list_tempo_tag_values",
		mcp.WithDescription("Retrieves all unique values for a specific tag name in a Tempo datasource. "+
			"Returns a list of string values (e.g., for tagName=\"service.name\", might return [\"api-gateway\", \"user-service\"]). "+
			"Defaults to the last hour if time range is not specified."),
		mcp.WithString("datasourceUid",
			mcp.Description("The UID of the Tempo datasource to query"),
			mcp.Required(),
		),
		mcp.WithString("tagName",
			mcp.Description("The tag name to get values for (e.g., \"service.name\", \"http.method\")"),
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

// RegisterListTempoTagValues registers the list_tempo_tag_values tool.
func RegisterListTempoTagValues(s *server.MCPServer) {
	s.AddTool(newListTempoTagValuesTool(), listTempoTagValuesHandler)
}
