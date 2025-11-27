// Package tools provides MCP tools for interacting with Grafana resources.
package tools

import (
	"github.com/mark3labs/mcp-go/server"
)

func RegisterMCPTools(s *server.MCPServer) {
	// Register Loki query tools
	RegisterListLokiLabelNames(s)
	RegisterListLokiLabelValues(s)
	RegisterQueryLokiStats(s)
	RegisterQueryLokiLogs(s)

	// Register Prometheus query tools
	RegisterListPrometheusLabelNames(s)
	RegisterListPrometheusLabelValues(s)
	RegisterListPrometheusMetricNames(s)
	RegisterQueryPrometheus(s)

	// Register Alerting tools
	RegisterListAlertRules(s)
	RegisterGetAlertRuleByUID(s)
}
