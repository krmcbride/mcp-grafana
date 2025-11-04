# Architecture

## Overview

This MCP server provides **read-only** access to Grafana instances for observability and root cause analysis. It is designed following the same safety-first philosophy as the mcp-k8s project: no mutations, only safe read operations.

## Design Philosophy

### Read-Only Operations

Like mcp-k8s, this server is intentionally **read-only** and focuses on observability and debugging. It does NOT include tools that could:

- Create, update, or delete dashboards
- Modify alert rules or notification policies
- Create or update datasources
- Execute administrative actions
- Write annotations or create incidents

This makes it safe to use for debugging production issues without risk of accidental changes.

### Focus Areas

The tools are specifically chosen to support **root cause analysis** workflows:

1. **Log Analysis** - Query Loki logs to investigate application behavior
2. **Metrics Analysis** - Query Prometheus metrics to understand system performance
3. **Alert Context** - View firing alerts and their configurations
4. **Dashboard Discovery** - Find relevant dashboards for investigation
5. **Investigation Tracking** - Access Sift investigation results

## Architecture Components

### 1. Grafana Client Initialization

**Package:** `internal/client`

Responsible for:
- Creating authenticated Grafana API clients from environment variables
- Supporting both service account tokens and basic auth
- Managing TLS configuration
- Multi-org support via `GRAFANA_ORG_ID`

**Environment Variables:**
- `GRAFANA_URL` - Base URL of Grafana instance (required)
- `GRAFANA_SERVICE_ACCOUNT_TOKEN` or `GRAFANA_API_KEY` - Authentication token (required)
- `GRAFANA_ORG_ID` - Organization ID for multi-org setups (optional)

### 2. Tool Categories

#### Datasource Tools (`internal/tools/datasources.go`)

**Read-only operations:**
- `list_datasources` - List all datasources (with optional type filter)
- `get_datasource_by_uid` - Get detailed datasource information

**Purpose:** Discover available Prometheus, Loki, and Tempo datasources for querying.

#### Prometheus Tools (`internal/tools/prometheus.go`)

**Read-only operations:**
- `query_prometheus` - Execute PromQL queries (instant and range)
- `list_prometheus_metrics` - List available metric names
- `list_prometheus_label_names` - List label names for a metric
- `list_prometheus_label_values` - List label values for a label name
- `query_prometheus_metadata` - Get metadata for metrics

**Purpose:** Query metrics data for performance analysis and troubleshooting.

#### Loki Tools (`internal/tools/loki.go`)

**Read-only operations:**
- `query_loki` - Execute LogQL queries (logs and metrics queries)
- `list_loki_label_names` - List available label names
- `list_loki_label_values` - List values for a label name
- `get_loki_stats` - Get stream statistics (streams, chunks, entries, bytes)

**Purpose:** Query log data for application debugging and error investigation.

#### Dashboard Tools (`internal/tools/dashboards.go`)

**Read-only operations:**
- `search_dashboards` - Search for dashboards by query string
- `get_dashboard_summary` - Get dashboard overview (title, panels, variables) without full JSON
- `get_panel_queries` - Extract queries and datasource info from dashboard panels

**Purpose:** Discover relevant dashboards and understand what they're querying. Intentionally excludes `get_dashboard_by_uid` (full JSON) to avoid context window bloat.

#### Alerting Tools (`internal/tools/alerting.go`)

**Read-only operations:**
- `list_alert_rules` - List alert rules with state/health information
- `list_contact_points` - List notification contact points

**Purpose:** Understand which alerts are firing and their configuration.

#### Sift Tools (`internal/tools/sift.go`)

**Read-only operations:**
- `list_sift_investigations` - List Sift investigations with limit parameter
- `get_sift_investigation` - Get investigation details by UUID
- `get_sift_analysis` - Get specific analysis from an investigation

**Purpose:** Access automated investigation results for error patterns and anomalies.

#### Navigation Tools (`internal/tools/navigation.go`)

**Read-only operations:**
- `generate_deeplink` - Create accurate Grafana URLs for dashboards, panels, Explore, etc.

**Purpose:** Generate correct URLs for navigation instead of relying on LLM URL guessing.

### 3. Resources

**Package:** `internal/resources`

Planned resources for discovery:
- `grafana://datasources` - List of available datasources with UIDs
- `grafana://config` - Grafana instance configuration summary

**Purpose:** Provide MCP resources for discovering available datasources and configuration without running commands.

### 4. Prompts

**Package:** `internal/prompts`

Planned analysis prompts:
- `log_error_analysis` - Analyze Loki logs for error patterns
- `metric_spike_analysis` - Investigate metric anomalies using Prometheus
- `alert_investigation` - Investigate firing alerts with context

**Purpose:** Guide systematic investigation workflows using multiple tools.

## Implementation Patterns

### Tool Definition Pattern (from official MCP)

The official Grafana MCP uses a clean pattern with `MustTool` for compile-time tool creation:

```go
var ListDatasources = mcpgrafana.MustTool(
    "list_datasources",
    "List available Grafana datasources...",
    listDatasources,
    mcp.WithTitleAnnotation("List datasources"),
    mcp.WithIdempotentHintAnnotation(true),
    mcp.WithReadOnlyHintAnnotation(true),
)
```

**We will adapt this pattern** using the simpler `mcp.NewTool` from mark3labs/mcp-go that we're already using in mcp-k8s.

### Client Context Pattern

Following the official MCP pattern:
- Grafana client instance passed via context: `GrafanaClientFromContext(ctx)`
- Configuration (URL, auth, org) stored in context
- Allows for clean dependency injection and testing

### Error Handling

- Enhance errors with actionable guidance (e.g., "Use 'list_datasources' resource to find valid UIDs")
- Validate datasource UIDs before querying
- Provide clear error messages about authentication failures

### Response Formatting

- Return structured data (JSON) for Claude to analyze
- Provide summaries rather than full objects where possible (e.g., dashboard summaries vs full JSON)
- Include relevant metadata (timestamps, UIDs, states)

## Token Budget Considerations

Following official MCP best practices:

1. **Avoid full dashboard JSON** - Use summaries and JSONPath extraction instead
2. **Limit log line counts** - Default to 10-100 lines with explicit limits
3. **Paginate large result sets** - Support limit/offset for alerts, investigations
4. **Provide focused queries** - Encourage specific time ranges and label selectors

## Dependencies

### Core Dependencies

- `github.com/mark3labs/mcp-go` - MCP protocol implementation
- `github.com/grafana/grafana-openapi-client-go` - Official Grafana API client
- `github.com/prometheus/client_golang` - Prometheus API client
- `github.com/grafana/grafana-plugin-sdk-go` - For time range parsing utilities

### Testing Dependencies

- Standard Go testing
- Integration tests against local Grafana (docker-compose)

## Future Considerations

### Not in Initial Scope

- Tempo/tracing tools (defer until needed)
- Pyroscope/profiling tools (defer until needed)
- OnCall tools (not directly related to observability queries)
- Incident management (write operations, out of scope)
- Dashboard modification (write operations, out of scope)

### Potential Future Additions

- Grafana Cloud-specific features (if needed)
- Explore API direct access (alternative to datasource proxy)
- Query history and saved queries
- Annotation reading (for deployment markers, etc.)
