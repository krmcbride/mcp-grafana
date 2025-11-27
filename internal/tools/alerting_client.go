package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/krmcbride/mcp-grafana/internal/grafana"
)

const (
	DefaultAlertRulesLimit = 100
)

// alertingClient provides methods for interacting with Grafana's alerting API.
type alertingClient struct {
	httpClient *http.Client
	baseURL    string // e.g., http://grafana
}

// newAlertingClient creates a new alerting client.
func newAlertingClient() (*alertingClient, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	return &alertingClient{
		httpClient: httpClient,
		baseURL:    grafanaURL,
	}, nil
}

// makeRequest performs an HTTP request and returns the response body.
func (c *alertingClient) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// AlertRule represents an alert rule from the provisioning API.
type AlertRule struct {
	UID          string            `json:"uid"`
	Title        string            `json:"title"`
	FolderUID    string            `json:"folderUID"`
	RuleGroup    string            `json:"ruleGroup"`
	For          string            `json:"for"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Condition    string            `json:"condition"`
	NoDataState  string            `json:"noDataState"`
	ExecErrState string            `json:"execErrState"`
	Data         []AlertQueryData  `json:"data,omitempty"`
	Updated      string            `json:"updated,omitempty"`
	IsPaused     bool              `json:"isPaused"`
}

// AlertQueryData represents query data within an alert rule.
type AlertQueryData struct {
	RefID             string         `json:"refId"`
	QueryType         string         `json:"queryType,omitempty"`
	RelativeTimeRange map[string]int `json:"relativeTimeRange,omitempty"`
	DatasourceUID     string         `json:"datasourceUid,omitempty"`
	Model             any            `json:"model,omitempty"`
}

// AlertRuleSummary provides a compact summary of an alert rule.
type AlertRuleSummary struct {
	UID         string            `json:"uid"`
	Title       string            `json:"title"`
	State       string            `json:"state,omitempty"`
	Health      string            `json:"health,omitempty"`
	FolderUID   string            `json:"folderUID,omitempty"`
	RuleGroup   string            `json:"ruleGroup,omitempty"`
	For         string            `json:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	IsPaused    bool              `json:"isPaused"`
}

// RulerResponse represents the response from the Ruler API.
type RulerResponse map[string][]RulerRuleGroup

// RulerRuleGroup represents a rule group from the Ruler API.
type RulerRuleGroup struct {
	Name     string      `json:"name"`
	File     string      `json:"file"`
	Interval string      `json:"interval,omitempty"`
	Rules    []RulerRule `json:"rules"`
}

// RulerRule represents a rule from the Ruler API.
type RulerRule struct {
	GrafanaAlert *GrafanaAlertRule `json:"grafana_alert,omitempty"`
	Expr         string            `json:"expr,omitempty"`
	For          string            `json:"for,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// GrafanaAlertRule represents the Grafana-specific alert rule data.
type GrafanaAlertRule struct {
	UID          string `json:"uid"`
	Title        string `json:"title"`
	Condition    string `json:"condition"`
	NoDataState  string `json:"no_data_state"`
	ExecErrState string `json:"exec_err_state"`
	IsPaused     bool   `json:"is_paused"`
}

// PrometheusRulesResponse represents the response from the Prometheus-style rules API.
type PrometheusRulesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []PrometheusRuleGroup `json:"groups"`
	} `json:"data"`
}

// PrometheusRuleGroup represents a rule group.
type PrometheusRuleGroup struct {
	Name     string           `json:"name"`
	File     string           `json:"file"`
	Rules    []PrometheusRule `json:"rules"`
	Interval float64          `json:"interval"`
}

// PrometheusRule represents a Prometheus-style rule with state.
type PrometheusRule struct {
	Name           string            `json:"name"`
	Query          string            `json:"query"`
	Duration       float64           `json:"duration"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	State          string            `json:"state"`
	Health         string            `json:"health"`
	Type           string            `json:"type"`
	LastEvaluation string            `json:"lastEvaluation,omitempty"`
	EvaluationTime float64           `json:"evaluationTime,omitempty"`
}

// listAlertRules lists all alert rules from the provisioning API.
func (c *alertingClient) listAlertRules(ctx context.Context, limit int) ([]AlertRule, error) {
	params := url.Values{}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/v1/provisioning/alert-rules", params)
	if err != nil {
		return nil, err
	}

	var rules []AlertRule
	if err := json.Unmarshal(bodyBytes, &rules); err != nil {
		return nil, fmt.Errorf("unmarshalling alert rules: %w", err)
	}

	return rules, nil
}

// getAlertRuleByUID gets a specific alert rule by UID.
func (c *alertingClient) getAlertRuleByUID(ctx context.Context, uid string) (*AlertRule, error) {
	path := fmt.Sprintf("/api/v1/provisioning/alert-rules/%s", url.PathEscape(uid))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var rule AlertRule
	if err := json.Unmarshal(bodyBytes, &rule); err != nil {
		return nil, fmt.Errorf("unmarshalling alert rule: %w", err)
	}

	return &rule, nil
}

// getAlertRulesWithState gets alert rules with their current state from the Prometheus-style API.
func (c *alertingClient) getAlertRulesWithState(ctx context.Context) ([]AlertRuleSummary, error) {
	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/prometheus/grafana/api/v1/rules", nil)
	if err != nil {
		return nil, err
	}

	var resp PrometheusRulesResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling rules response: %w", err)
	}

	var summaries []AlertRuleSummary
	for _, group := range resp.Data.Groups {
		for _, rule := range group.Rules {
			if rule.Type != "alerting" {
				continue // Skip recording rules
			}
			summary := AlertRuleSummary{
				Title:       rule.Name,
				State:       rule.State,
				Health:      rule.Health,
				RuleGroup:   group.Name,
				Labels:      rule.Labels,
				Annotations: rule.Annotations,
			}
			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}
