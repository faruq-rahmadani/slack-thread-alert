package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/config"
	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	slackgo "github.com/slack-go/slack"
)

// Repository is the Slack implementation of usecase.SlackRepositoryItf
type Repository struct {
	client *slackgo.Client
	now    func() time.Time
}

func NewRepository(cfg config.Slack, options ...slackgo.Option) *Repository {
	return &Repository{
		client: slackgo.New(cfg.BotToken, options...),
		now:    time.Now,
	}
}

func (r *Repository) PostSummary(ctx context.Context, channelID string, alert *domain.Alert) (string, error) {
	_, ts, err := r.client.PostMessageContext(ctx, channelID, buildSummary(alert, alert.EventTime, r.now())...)
	if err != nil {
		return "", fmt.Errorf("post slack summary to %s: %w", channelID, err)
	}

	return ts, nil
}

func (r *Repository) UpdateSummary(ctx context.Context, thread *domain.Thread, alert *domain.Alert) error {
	options := buildSummary(alert, thread.StartedAt, r.now())

	if _, _, _, err := r.client.UpdateMessageContext(ctx, thread.ChannelID, thread.ThreadTS, options...); err != nil {
		return fmt.Errorf("update slack summary %s in %s: %w", thread.ThreadTS, thread.ChannelID, err)
	}

	return nil
}

func (r *Repository) ReplyDetail(ctx context.Context, channelID, threadTS string, alert *domain.Alert) error {
	options := append(buildDetail(alert), slackgo.MsgOptionTS(threadTS))

	if _, _, err := r.client.PostMessageContext(ctx, channelID, options...); err != nil {
		return fmt.Errorf("reply slack thread %s in %s: %w", threadTS, channelID, err)
	}

	return nil
}
