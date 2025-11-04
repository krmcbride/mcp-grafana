# Enhanced MCP Server Development

## MCP Tool Development Patterns

**Tool Structure Template:**

```go
func NewToolName(serverName string) mcp.Tool {
    return mcp.NewTool(
        serverName+"_tool_name",
        "Description of what this tool does",
        map[string]mcp.ToolInputSchema{
            "required_param": {
                Type: "string",
                Description: "Description of the required parameter with MCP resource guidance",
            },
        },
        handleToolName,
    )
}
```

**Error Handling Best Practices:**

- Always enhance context-related errors with MCP resource guidance
- Use structured error responses with actionable suggestions
- Log errors to stderr only (never stdout due to stdio transport)
- Provide context about which MCP resources can help resolve issues

**Interactive Testing Workflow:**

1. Use `make mcp-shell` for interactive tool testing
2. Test edge cases like invalid contexts, missing resources
3. Verify error messages guide users to appropriate MCP resources
4. Validate tool descriptions are clear and actionable

**Tool Registration Pattern:**

- Register tools in `internal/tools/register.go`
- Update both CLAUDE.md and README.md documentation
- Add integration tests for new tools

## MCP Resource Development

**Resource Naming Convention:**

- Use descriptive URI schemes
- Provide clear resource descriptions in registration
- Enable resource discovery through intuitive naming

**Resource Implementation Guidelines:**

- Return structured data suitable for Claude analysis
- Include metadata that helps with decision making
- Design resources to reduce need for external commands
- Document resource schema and expected usage patterns

## MCP Prompt Development

**Prompt Structure Requirements:**

- Required parameters must be clearly specified
- Optional parameters should have sensible defaults
- Provide clear guidance on what analysis to perform
- Structure prompts to guide systematic investigation

**Analysis Prompt Pattern:**

```go
// Guides assistant through systematic analysis
// Required: context, optional: namespace
// Provides step-by-step investigation approach
```
