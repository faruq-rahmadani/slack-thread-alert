package domain

import (
	"fmt"
	"html"
	"strings"
	"time"
)

const SourceScalyr = "scalyr"

// maxScalyrLogLines keeps the log code block inside the Slack section limit,
// so truncation never cuts off the closing fence
const maxScalyrLogLines = 2500

type ScalyrAlertWebhookRequest struct {
	Status           string `json:"status"`
	AlertID          string `json:"alert_id"`
	AlertName        string `json:"alert_name"`
	AlertDescription string `json:"alert_description"`
	LinkToAlert      string `json:"link_to_alert"`
	AlertTrigger     string `json:"alert_trigger"`
	LastLogLines     string `json:"last_log_lines"`
	ChannelID        string `json:"channel_id"`
}

func (r *ScalyrAlertWebhookRequest) ToAlert() Alert {
	return Alert{
		Source:         SourceScalyr,
		ThreadKey:      r.GetThreadKey(),
		Status:         r.GetAlertStatus(),
		Title:          html.UnescapeString(r.AlertName),
		Summary:        html.UnescapeString(r.AlertTrigger),
		Message:        r.GetMessage(),
		MonitorLink:    html.UnescapeString(r.LinkToAlert),
		EventTime:      time.Now(), // Scalyr sends no event time
		SlackChannelID: strings.ToUpper(strings.TrimSpace(r.ChannelID)),
	}
}

func (r *ScalyrAlertWebhookRequest) GetThreadKey() string {
	if r.AlertID == "" {
		return ""
	}
	return SourceScalyr + ":" + r.AlertID
}

func (r *ScalyrAlertWebhookRequest) GetAlertStatus() AlertStatus {
	switch strings.ToLower(strings.TrimSpace(r.Status)) {
	case "triggered":
		return AlertStatusTriggered
	case "resolved":
		return AlertStatusRecovered
	case "":
		// Template without a status: #lastLogLines# is only filled for webhook-trigger
		if r.LastLogLines != "" {
			return AlertStatusTriggered
		}
		return AlertStatusRecovered
	default:
		return "Unknown"
	}
}

// GetMessage shows the description when it adds something to the name, followed by the last log lines
func (r *ScalyrAlertWebhookRequest) GetMessage() string {
	var parts []string

	if desc := html.UnescapeString(r.AlertDescription); desc != "" && desc != html.UnescapeString(r.AlertName) {
		parts = append(parts, desc)
	}

	if logs := r.GetLogLines(); logs != "" {
		parts = append(parts, fmt.Sprintf("*Last log lines*\n```\n%s\n```", logs))
	}

	return strings.Join(parts, "\n\n")
}

// GetLogLines turns #lastLogLines# into plain text, one event per line.
// <br> is replaced before unescaping, so an escaped "&lt;br&gt;" inside a log stays as is.
func (r *ScalyrAlertWebhookRequest) GetLogLines() string {
	logs := strings.ReplaceAll(r.LastLogLines, "<br>", "\n")
	logs = strings.TrimSpace(html.UnescapeString(logs))

	if runes := []rune(logs); len(runes) > maxScalyrLogLines {
		logs = string(runes[:maxScalyrLogLines]) + "…"
	}
	return logs
}
