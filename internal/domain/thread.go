package domain

import (
	"errors"
	"time"
)

var ErrThreadNotFound = errors.New("thread not found")

// Thread links an alert cycle to the Slack thread holding its notifications
type Thread struct {
	Key       string // Alert.ThreadKey, e.g. datadog:<alert_cycle_key>
	ChannelID string
	ThreadTS  string    // Slack ts of the parent message
	StartedAt time.Time // event time of the first alert of the cycle
	CreatedAt time.Time
}
