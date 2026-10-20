package alert

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	"github.com/faruqrahmadani/slack-thread-alert/internal/usecase"
)

type UseCase struct {
	threadRepo usecase.ThreadRepositoryItf
	slackRepo  usecase.SlackRepositoryItf
}

func NewUseCase(threadRepo usecase.ThreadRepositoryItf, slackRepo usecase.SlackRepositoryItf) *UseCase {
	return &UseCase{
		threadRepo: threadRepo,
		slackRepo:  slackRepo,
	}
}

// Handle keeps one Slack thread per alert cycle: the parent message is a summary
// of the latest status, every event (triggered, warn, recovered) is replied as detail in the thread.
func (u *UseCase) Handle(ctx context.Context, alert *domain.Alert) (err error) {
	if err := alert.Validate(); err != nil {
		return err
	}

	// Skip duplicates, e.g. Datadog retrying the same event
	if alert.EventID != "" {
		isNew, markErr := u.threadRepo.MarkEventProcessed(ctx, alert.EventID)
		if markErr != nil {
			return fmt.Errorf("mark event processed: %w", markErr)
		}
		if !isNew {
			log.Printf("skip duplicate event: event_id=%s thread_key=%s", alert.EventID, alert.ThreadKey)
			return nil
		}

		// Allow the retry to process it again when this attempt fails.
		// err is the named return value, do not shadow it in this function.
		defer func() {
			if err == nil {
				return
			}
			if unmarkErr := u.threadRepo.UnmarkEventProcessed(ctx, alert.EventID); unmarkErr != nil {
				log.Printf("error unmarking event %s: %v", alert.EventID, unmarkErr)
			}
		}()
	}

	thread, err := u.threadRepo.GetThread(ctx, alert.ThreadKey)
	switch {
	case errors.Is(err, domain.ErrThreadNotFound):
		return u.startThread(ctx, alert)
	case err != nil:
		return fmt.Errorf("get thread: %w", err)
	default:
		return u.replyThread(ctx, thread, alert)
	}
}

// startThread posts the summary as parent message, then the event detail in its thread
func (u *UseCase) startThread(ctx context.Context, alert *domain.Alert) error {
	threadTS, err := u.slackRepo.PostSummary(ctx, alert.SlackChannelID, alert)
	if err != nil {
		return fmt.Errorf("post summary: %w", err)
	}

	// Save before replying, so a retry after a failed reply updates this parent instead of posting a new one.
	// Recovered without a known thread (e.g. after restart): nothing left to follow up.
	if alert.Status != domain.AlertStatusRecovered {
		err = u.threadRepo.SaveThread(ctx, &domain.Thread{
			Key:       alert.ThreadKey,
			ChannelID: alert.SlackChannelID,
			ThreadTS:  threadTS,
			StartedAt: alert.EventTime,
		})
		if err != nil {
			return fmt.Errorf("save thread: %w", err)
		}
	}

	if err := u.slackRepo.ReplyDetail(ctx, alert.SlackChannelID, threadTS, alert); err != nil {
		return fmt.Errorf("reply detail: %w", err)
	}

	return nil
}

// replyThread updates the summary to the latest status, replies the detail and closes the thread once recovered
func (u *UseCase) replyThread(ctx context.Context, thread *domain.Thread, alert *domain.Alert) error {
	// Use the stored channel, the thread lives there even if the alert routing changed
	if err := u.slackRepo.UpdateSummary(ctx, thread, alert); err != nil {
		return fmt.Errorf("update summary: %w", err)
	}

	if err := u.slackRepo.ReplyDetail(ctx, thread.ChannelID, thread.ThreadTS, alert); err != nil {
		return fmt.Errorf("reply detail: %w", err)
	}

	if alert.Status != domain.AlertStatusRecovered {
		return nil
	}

	if err := u.threadRepo.DeleteThread(ctx, alert.ThreadKey); err != nil {
		return fmt.Errorf("delete thread: %w", err)
	}

	return nil
}
