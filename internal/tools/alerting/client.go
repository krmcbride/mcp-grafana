// Package alerting provides MCP tools for interacting with Grafana's alerting API.
package alerting

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
	// DefaultRulesLimit is the default limit for listing alert rules.
	DefaultRulesLimit = 100
)

// client provides methods for interacting with Grafana's alerting API.
type client struct {
	httpClient *http.Client
	baseURL    string
}

// newClient creates a new alerting client.
func newClient() (*client, error) {
	httpClient, grafanaURL, err := grafana.GetHTTPClientForGrafana()
	if err != nil {
		return nil, err
	}

	return &client{
		httpClient: httpClient,
		baseURL:    grafanaURL,
	}, nil
}

// makeRequest performs an HTTP request and returns the response body.
func (c *client) makeRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
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
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// Rule represents an alert rule from the provisioning API.
type Rule struct {
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
	Data         []QueryData       `json:"data,omitempty"`
	Updated      string            `json:"updated,omitempty"`
	IsPaused     bool              `json:"isPaused"`
}

// QueryData represents query data within an alert rule.
type QueryData struct {
	RefID             string         `json:"refId"`
	QueryType         string         `json:"queryType,omitempty"`
	RelativeTimeRange map[string]int `json:"relativeTimeRange,omitempty"`
	DatasourceUID     string         `json:"datasourceUid,omitempty"`
	Model             any            `json:"model,omitempty"`
}

// RuleSummary provides a compact summary of an alert rule.
type RuleSummary struct {
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

// prometheusRulesResponse represents the response from the Prometheus-style rules API.
type prometheusRulesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []prometheusRuleGroup `json:"groups"`
	} `json:"data"`
}

// prometheusRuleGroup represents a rule group.
type prometheusRuleGroup struct {
	Name     string           `json:"name"`
	File     string           `json:"file"`
	Rules    []prometheusRule `json:"rules"`
	Interval float64          `json:"interval"`
}

// prometheusRule represents a Prometheus-style rule with state.
type prometheusRule struct {
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

// listRules lists all alert rules from the provisioning API.
func (c *client) listRules(ctx context.Context, limit int) ([]Rule, error) {
	params := url.Values{}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}

	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/v1/provisioning/alert-rules", params)
	if err != nil {
		return nil, err
	}

	var rules []Rule
	if err := json.Unmarshal(bodyBytes, &rules); err != nil {
		return nil, fmt.Errorf("unmarshalling alert rules: %w", err)
	}

	return rules, nil
}

// getRuleByUID gets a specific alert rule by UID.
func (c *client) getRuleByUID(ctx context.Context, uid string) (*Rule, error) {
	path := fmt.Sprintf("/api/v1/provisioning/alert-rules/%s", url.PathEscape(uid))
	bodyBytes, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var rule Rule
	if err := json.Unmarshal(bodyBytes, &rule); err != nil {
		return nil, fmt.Errorf("unmarshalling alert rule: %w", err)
	}

	return &rule, nil
}

// getRulesWithState gets alert rules with their current state from the Prometheus-style API.
func (c *client) getRulesWithState(ctx context.Context) ([]RuleSummary, error) {
	bodyBytes, err := c.makeRequest(ctx, "GET", "/api/prometheus/grafana/api/v1/rules", nil)
	if err != nil {
		return nil, err
	}

	var resp prometheusRulesResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling rules response: %w", err)
	}

	var summaries []RuleSummary
	for _, group := range resp.Data.Groups {
		for _, rule := range group.Rules {
			if rule.Type != "alerting" {
				continue // Skip recording rules
			}
			summary := RuleSummary{
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
