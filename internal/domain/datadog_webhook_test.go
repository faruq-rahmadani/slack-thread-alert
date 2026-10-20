package domain

import (
	"testing"
	"time"
)

func TestDatadogAlertWebhookRequest_ToAlert(t *testing.T) {
	req := DatadogAlertWebhookRequest{
		ID:              "8862065024138091030",
		AlertID:         "123",
		AlertCycleKey:   "8862064975451760776",
		AlertTransition: "Re-Triggered",
		AlertTitle:      "[TEST] hehehe",
		AlertStatus:     "attendance over *** was > 100.0 in total during the last 5m",
		Message: "\n@webhook-Slack-Thread-Alert\n\n## Related links\n- Metrics Explorer\n\n\n" +
			"Metric value: 0.0\n\n" +
			"Metric Graph: https://app.datadoghq.com/monitors/0?link_source=monitor_notif · Monitor Status: https://app.datadoghq.com/monitors/0?link_source=monitor_notif",
		Link: "https://app.datadoghq.com/event/event?id=8862065024138091030",
		Tags: "channel_id:c0caneqbnn8,monitor",
		Date: "1792483103000",
	}

	got := req.ToAlert()

	if got.Status != AlertStatusTriggered {
		t.Errorf("Re-Triggered should map to Triggered, got %q", got.Status)
	}
	if got.SlackChannelID != "C0CANEQBNN8" {
		t.Errorf("channel id should be upper-cased, got %q", got.SlackChannelID)
	}
	if got.MonitorLink != "https://app.datadoghq.com/monitors/123" {
		t.Errorf("unexpected monitor link %q", got.MonitorLink)
	}
	if !got.EventTime.Equal(time.UnixMilli(1792483103000)) {
		t.Errorf("unexpected event time %s", got.EventTime)
	}

	wantMsg := "## Related links\n- Metrics Explorer\n\nMetric value: 0.0"
	if got.Message != wantMsg {
		t.Errorf("message not cleaned:\n got %q\nwant %q", got.Message, wantMsg)
	}
}

func TestDatadogAlertWebhookRequest_GetAlertStatus(t *testing.T) {
	tests := map[string]AlertStatus{
		"Triggered":    AlertStatusTriggered,
		"Re-Triggered": AlertStatusTriggered,
		"Renotify":     AlertStatusTriggered,
		"Warn":         AlertStatusWarn,
		"Re-Warn":      AlertStatusWarn,
		"No Data":      AlertStatusNoData,
		"Re-No Data":   AlertStatusNoData,
		"Recovered":    AlertStatusRecovered,
	}
	for transition, want := range tests {
		r := DatadogAlertWebhookRequest{AlertTransition: transition}
		if got := r.GetAlertStatus(); got != want {
			t.Errorf("%s: got %q, want %q", transition, got, want)
		}
	}
}
