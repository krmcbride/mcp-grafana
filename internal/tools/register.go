// Package tools provides MCP tools for interacting with Grafana resources.
package tools

import (
	"github.com/krmcbride/mcp-grafana/internal/tools/alerting"
	"github.com/krmcbride/mcp-grafana/internal/tools/dashboard"
	"github.com/krmcbride/mcp-grafana/internal/tools/loki"
	"github.com/krmcbride/mcp-grafana/internal/tools/prometheus"
	"github.com/krmcbride/mcp-grafana/internal/tools/tempo"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterMCPTools(s *server.MCPServer) {
	// Register Loki query tools
	loki.RegisterListLabelNames(s)
	loki.RegisterListLabelValues(s)
	loki.RegisterQueryStats(s)
	loki.RegisterQueryLogs(s)

	// Register Prometheus query tools
	prometheus.RegisterListLabelNames(s)
	prometheus.RegisterListLabelValues(s)
	prometheus.RegisterListMetricNames(s)
	prometheus.RegisterQuery(s)

	// Register Tempo tracing tools
	tempo.RegisterListTagNames(s)
	tempo.RegisterListTagValues(s)
	tempo.RegisterSearchTraces(s)
	tempo.RegisterGetTrace(s)

	// Register Dashboard tools
	dashboard.RegisterSearch(s)
	dashboard.RegisterGetSummary(s)
	dashboard.RegisterGetPanelQueries(s)

	// Register Alerting tools
	alerting.RegisterListRules(s)
	alerting.RegisterGetRuleByUID(s)
}
