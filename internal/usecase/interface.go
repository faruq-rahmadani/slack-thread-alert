package usecase

import (
	"context"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
)

type AlertUseCaseItf interface {
	Handle(ctx context.Context, alert *domain.Alert) error
}

// ThreadRepositoryItf stores alert cycle -> Slack thread mapping and processed events
type ThreadRepositoryItf interface {
	// GetThread returns domain.ErrThreadNotFound when the key is unknown or expired
	GetThread(ctx context.Context, key string) (*domain.Thread, error)
	SaveThread(ctx context.Context, thread *domain.Thread) error
	DeleteThread(ctx context.Context, key string) error

	// MarkEventProcessed returns false when the event was already processed (duplicate / retry)
	MarkEventProcessed(ctx context.Context, eventID string) (bool, error)
	// UnmarkEventProcessed lets a failed event be processed again on retry
	UnmarkEventProcessed(ctx context.Context, eventID string) error
}

// SlackRepositoryItf sends alerts to Slack
type SlackRepositoryItf interface {
	// PostSummary posts the parent message and returns its ts (thread_ts for replies)
	PostSummary(ctx context.Context, channelID string, alert *domain.Alert) (string, error)
	// UpdateSummary edits the thread's parent message to show the latest status
	UpdateSummary(ctx context.Context, thread *domain.Thread, alert *domain.Alert) error
	// ReplyDetail posts the full detail of one event in the thread
	ReplyDetail(ctx context.Context, channelID, threadTS string, alert *domain.Alert) error
}
