package domain

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const channelIDDatadogTag = "channel_id"
const SourceDatadog = "datadog"

// DatadogAlertWebhookRequest is the payload configured in the Datadog webhook integration.
// Datadog sends every template variable as a string.
type DatadogAlertWebhookRequest struct {
	ID              string `json:"id"`
	AlertID         string `json:"alert_id"`
	AlertCycleKey   string `json:"alert_cycle_key"`
	AggregKey       string `json:"aggreg_key"`
	AlertTransition string `json:"alert_transition"`
	AlertType       string `json:"alert_type"`
	AlertPriority   string `json:"alert_priority"`
	AlertTitle      string `json:"alert_title"`
	AlertStatus     string `json:"alert_status"`
	AlertQuery      string `json:"alert_query"`
	AlertScope      string `json:"alert_scope"`
	EventType       string `json:"event_type"`
	Message         string `json:"message"`
	Link            string `json:"link"`
	Snapshot        string `json:"snapshot"`
	Hostname        string `json:"hostname"`
	Tags            string `json:"tags"`
	Date            string `json:"date"`
	LastUpdated     string `json:"last_updated"`
	Org             struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"org"`
}

func (r *DatadogAlertWebhookRequest) ToAlert() Alert {
	return Alert{
		Source:         SourceDatadog,
		ThreadKey:      SourceDatadog + ":" + r.AlertCycleKey,
		EventID:        r.ID,
		Status:         r.GetAlertStatus(),
		Title:          r.AlertTitle,
		Summary:        r.AlertStatus,
		Message:        r.GetCleanMessage(),
		Link:           r.Link,
		MonitorLink:    r.GetMonitorLink(),
		Snapshot:       r.Snapshot,
		Query:          r.AlertQuery,
		Scope:          r.AlertScope,
		Hostname:       r.Hostname,
		OrgName:        r.Org.Name,
		Priority:       r.AlertPriority,
		EventTime:      r.GetEventTime(),
		SlackChannelID: r.GetChannelID(),
	}
}

func (r *DatadogAlertWebhookRequest) GetAlertStatus() AlertStatus {
	switch r.AlertTransition {
	case "Triggered", "Re-Triggered", "Renotify":
		return AlertStatusTriggered
	case "Warn", "Re-Warn":
		return AlertStatusWarn
	case "No Data", "Re-No Data":
		return AlertStatusNoData
	case "Recovered":
		return AlertStatusRecovered
	default:
		return "Unknown"
	}
}

// GetEventTime parses $DATE (epoch milliseconds), falls back to now
func (r *DatadogAlertWebhookRequest) GetEventTime() time.Time {
	ms, err := strconv.ParseInt(r.Date, 10, 64)
	if err != nil || ms <= 0 {
		return time.Now()
	}
	return time.UnixMilli(ms)
}

// GetMonitorLink builds the monitor page URL on the same Datadog site as $LINK
func (r *DatadogAlertWebhookRequest) GetMonitorLink() string {
	u, err := url.Parse(r.Link)
	if err != nil || u.Host == "" || r.AlertID == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s/monitors/%s", u.Scheme, u.Host, r.AlertID)
}

// GetCleanMessage removes Datadog notification noise: @-mentions and the auto-generated link list
func (r *DatadogAlertWebhookRequest) GetCleanMessage() string {
	var lines []string
	blank := false
	for _, line := range strings.Split(r.Message, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "@"),
			strings.Contains(trimmed, "link_source=monitor_notif"):
			continue
		case trimmed == "":
			// collapse consecutive blank lines
			if blank {
				continue
			}
			blank = true
		default:
			blank = false
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (r *DatadogAlertWebhookRequest) GetChannelID() string {
	tags := strings.Split(r.Tags, ",")

	for _, tag := range tags {
		k := strings.Split(tag, ":")
		if len(k) == 2 && k[0] == channelIDDatadogTag {
			return strings.ToUpper(k[1])
		}
	}

	return ""
}
