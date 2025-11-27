# mcp-grafana

A Model Context Protocol (MCP) server that provides **read-only** tools for safely interacting with Grafana instances. This server is designed to be safe for production use, deliberately excluding any tools that could modify resources or cause side effects.

## Design Philosophy

This MCP server is intentionally **read-only** and focuses on observability and root cause analysis. It does not include tools that could:

- Create, update, or delete dashboards
- Modify alert rules or notification policies
- Create or update datasources
- Execute administrative actions
- Write annotations or create incidents

This makes it safe to use for debugging production issues without risk of accidental changes.

## Tools

### Loki Tools (4 tools)

| Tool                     | Description                                                              |
| ------------------------ | ------------------------------------------------------------------------ |
| `list_loki_label_names`  | Lists all available label names in a Loki datasource                     |
| `list_loki_label_values` | Gets all unique values for a specific label name                         |
| `query_loki_stats`       | Checks query size before fetching logs (streams, chunks, entries, bytes) |
| `query_loki_logs`        | Executes LogQL queries and returns log entries                           |

### Prometheus Tools (4 tools)

| Tool                           | Description                                                |
| ------------------------------ | ---------------------------------------------------------- |
| `list_prometheus_label_names`  | Lists all available label names in a Prometheus datasource |
| `list_prometheus_label_values` | Gets all unique values for a specific label name           |
| `list_prometheus_metric_names` | Lists metric names with optional regex filtering           |
| `query_prometheus`             | Executes PromQL queries (instant and range)                |

### Tempo Tools (4 tools)

| Tool                    | Description                                                           |
| ----------------------- | --------------------------------------------------------------------- |
| `list_tempo_tag_names`  | Lists all available tag names (span attributes) in a Tempo datasource |
| `list_tempo_tag_values` | Gets all unique values for a specific tag name                        |
| `search_tempo_traces`   | Searches for traces using TraceQL                                     |
| `get_tempo_trace`       | Retrieves a complete trace by trace ID                                |

### Dashboard Tools (3 tools)

| Tool                          | Description                                                         |
| ----------------------------- | ------------------------------------------------------------------- |
| `search_dashboards`           | Searches for dashboards by query string or tag                      |
| `get_dashboard_summary`       | Gets a compact summary of a dashboard (panels, variables, metadata) |
| `get_dashboard_panel_queries` | Extracts all queries from a dashboard's panels                      |

### Alerting Tools (2 tools)

| Tool                    | Description                                                                   |
| ----------------------- | ----------------------------------------------------------------------------- |
| `list_alert_rules`      | Lists alert rules with optional state information (firing, pending, inactive) |
| `get_alert_rule_by_uid` | Gets detailed configuration of a specific alert rule                          |

## Resources

| Resource                | Description                                                        |
| ----------------------- | ------------------------------------------------------------------ |
| `grafana://datasources` | Lists available datasources with UIDs and types for easy discovery |

## Configuration

The server requires the following environment variables:

### Required

- `GRAFANA_URL` - Base URL of your Grafana instance (e.g., `http://localhost:3000`)
- `GRAFANA_API_KEY` - Service account token for authentication

### Creating a Service Account Token

1. In Grafana, go to **Administration → Service accounts**
2. Click **Add service account**
3. Set a display name (e.g., "MCP Server")
4. Assign the **Viewer** role (read-only access is sufficient for all tools)
5. Click **Add token** to generate an authentication token
6. Copy the token and set it as `GRAFANA_API_KEY`

For more information, see [Grafana Service Account documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/).

## Usage

### With Claude Code

```bash
# Configure in Claude Code settings (~/.claude/settings.json)
{
  "mcpServers": {
    "grafana": {
      "command": "/path/to/mcp-grafana/dist/server",
      "env": {
        "GRAFANA_URL": "http://localhost:3000",
        "GRAFANA_API_KEY": "your-service-account-token-here"
      }
    }
  }
}
```

## Related Projects

- [mcp-k8s](https://github.com/krmcbride/mcp-k8s) - Similar read-only MCP server for Kubernetes clusters
- [grafana/mcp-grafana](https://github.com/grafana/mcp-grafana) - Official Grafana MCP server
- [MCP Specification](https://modelcontextprotocol.io) - Model Context Protocol documentation
