package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// Captured from Scalyr with the template from the original alert (no status, no channel), trimmed
const (
	scalyrTriggerPayload = `{"last_log_lines":"<br>Wed Oct 21, 2026 3:58:55 AM - ffi-scalyr-production-fj5k5 - {&quot;path&quot;:&quot;/api/public/v2/attendance&quot;,&quot;response_code&quot;:500}<br>Wed Oct 21, 2026 3:59:03 AM - ffi-scalyr-production-g9kqn - {&quot;path&quot;:&quot;/api/public/v2/attendance&quot;,&quot;response_code&quot;:500}","alert_id":"scalyr-alert-CujbyZrDBLmSDh45","alert_name":"[ops-tool] Lion HTTP 500 per 1 min","alert_description":"[ops-tool] Lion HTTP 500 per 1 min","link_to_alert":"https://app.scalyr.com/alert?id=CujbyZrDBLmSDh45&amp;teamToken=xxx","alert_trigger":"count:1 minutes(project = &quot;lion&quot; environment=&#39;production&#39; response_code &gt;= 500) &gt; 0","parsed_logs":[{"response_code":500}]}`
	scalyrResolvePayload = `{"alert_id":"scalyr-alert-CujbyZrDBLmSDh45","alert_name":"[ops-tool] Lion HTTP 500 per 1 min","alert_description":"[ops-tool] Lion HTTP 500 per 1 min","link_to_alert":"https://app.scalyr.com/alert?id=CujbyZrDBLmSDh45&amp;teamToken=xxx","alert_trigger":"count:1 minutes(project = &quot;lion&quot; environment=&#39;production&#39; response_code &gt;= 500) &gt; 0"}`
)

func parseScalyr(t *testing.T, payload string) ScalyrAlertWebhookRequest {
	t.Helper()
	var req ScalyrAlertWebhookRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return req
}

func TestScalyrAlertWebhookRequest_ToAlert(t *testing.T) {
	req := parseScalyr(t, scalyrTriggerPayload)
	req.ChannelID = "c0caneqbnn8"

	got := req.ToAlert()

	if got.Status != AlertStatusTriggered {
		t.Errorf("trigger with log lines should be Triggered, got %q", got.Status)
	}
	if got.ThreadKey != "scalyr:scalyr-alert-CujbyZrDBLmSDh45" {
		t.Errorf("unexpected thread key %q", got.ThreadKey)
	}
	if got.SlackChannelID != "C0CANEQBNN8" {
		t.Errorf("channel id should be upper-cased, got %q", got.SlackChannelID)
	}
	if got.Title != "[ops-tool] Lion HTTP 500 per 1 min" {
		t.Errorf("unexpected title %q", got.Title)
	}
	if want := `count:1 minutes(project = "lion" environment='production' response_code >= 500) > 0`; got.Summary != want {
		t.Errorf("trigger not unescaped:\n got %q\nwant %q", got.Summary, want)
	}
	if want := "https://app.scalyr.com/alert?id=CujbyZrDBLmSDh45&teamToken=xxx"; got.MonitorLink != want {
		t.Errorf("link not unescaped: got %q", got.MonitorLink)
	}
	if got.EventTime.IsZero() {
		t.Error("event time should default to now")
	}

	wantMsg := "*Last log lines*\n```\n" +
		`Wed Oct 21, 2026 3:58:55 AM - ffi-scalyr-production-fj5k5 - {"path":"/api/public/v2/attendance","response_code":500}` + "\n" +
		`Wed Oct 21, 2026 3:59:03 AM - ffi-scalyr-production-g9kqn - {"path":"/api/public/v2/attendance","response_code":500}` +
		"\n```"
	if got.Message != wantMsg {
		t.Errorf("unexpected message:\n got %q\nwant %q", got.Message, wantMsg)
	}
}

func TestScalyrAlertWebhookRequest_SameThreadKeyForTriggerAndResolve(t *testing.T) {
	trigger := parseScalyr(t, scalyrTriggerPayload)
	resolve := parseScalyr(t, scalyrResolvePayload)

	if trigger.GetThreadKey() != resolve.GetThreadKey() {
		t.Errorf("trigger %q and resolve %q should share the thread", trigger.GetThreadKey(), resolve.GetThreadKey())
	}
	if got := resolve.ToAlert(); got.Status != AlertStatusRecovered || got.Message != "" {
		t.Errorf("resolve should be Recovered without message, got %q %q", got.Status, got.Message)
	}
}

func TestScalyrAlertWebhookRequest_GetAlertStatus(t *testing.T) {
	tests := []struct {
		status, logs string
		want         AlertStatus
	}{
		{"triggered", "", AlertStatusTriggered},
		{"Triggered", "", AlertStatusTriggered},
		{"resolved", "", AlertStatusRecovered},
		// explicit status wins over the log lines fallback
		{"resolved", "some log", AlertStatusRecovered},
		{"", "some log", AlertStatusTriggered},
		{"", "", AlertStatusRecovered},
		{"muted", "", "Unknown"},
	}
	for _, tt := range tests {
		r := ScalyrAlertWebhookRequest{Status: tt.status, LastLogLines: tt.logs}
		if got := r.GetAlertStatus(); got != tt.want {
			t.Errorf("status=%q logs=%q: got %q, want %q", tt.status, tt.logs, got, tt.want)
		}
	}
}

func TestScalyrAlertWebhookRequest_EmptyIDIsInvalid(t *testing.T) {
	r := ScalyrAlertWebhookRequest{ChannelID: "C1"}
	alert := r.ToAlert()
	if err := alert.Validate(); err == nil {
		t.Error("alert without #id# should be invalid")
	}
}

func TestScalyrAlertWebhookRequest_GetMessage(t *testing.T) {
	r := ScalyrAlertWebhookRequest{AlertName: "HTTP 500", AlertDescription: "HTTP 500\nCheck the lion pods"}
	if got := r.GetMessage(); got != "HTTP 500\nCheck the lion pods" {
		t.Errorf("description differing from the name should be shown, got %q", got)
	}

	r = ScalyrAlertWebhookRequest{LastLogLines: strings.Repeat("x", maxScalyrLogLines+10)}
	if got := r.GetMessage(); !strings.HasSuffix(got, "…\n```") {
		t.Errorf("long logs should be truncated inside the code block, got suffix %q", got[len(got)-10:])
	}
}
