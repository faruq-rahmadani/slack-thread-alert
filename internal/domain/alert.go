package domain

import (
	"errors"
	"fmt"
	"time"
)

type AlertStatus string

const (
	AlertStatusTriggered AlertStatus = "Triggered"
	AlertStatusWarn      AlertStatus = "Warn"
	AlertStatusNoData    AlertStatus = "No Data"
	AlertStatusRecovered AlertStatus = "Recovered"
)

type Alert struct {
	Source         string      // "datadog", "grafana"
	ThreadKey      string      // datadog: alert_cycle_key, grafana: fingerprint
	EventID        string      // buat dedup retry
	Status         AlertStatus // Triggered, Recovered, Warn, ...
	Title          string
	Summary        string // one line of what happened, e.g. "cpu over host:a was > 90.0 during the last 5m"
	Message        string // full message, markdown
	Link           string // event link
	MonitorLink    string // link to the monitor / rule definition
	Snapshot       string // graph image URL
	Query          string
	Scope          string
	Hostname       string
	OrgName        string
	Priority       string
	EventTime      time.Time // when the event happened at the source
	SlackChannelID string
	Tags           []string
}

// ErrInvalidAlert means the alert cannot be processed, retrying will not help
var ErrInvalidAlert = errors.New("invalid alert")

// Validate checks the fields required to post an alert into a Slack thread
func (a *Alert) Validate() error {
	switch {
	case a.ThreadKey == "":
		return fmt.Errorf("%w: thread key is required", ErrInvalidAlert)
	case a.SlackChannelID == "":
		return fmt.Errorf("%w: slack channel id is required", ErrInvalidAlert)
	}
	return nil
}
