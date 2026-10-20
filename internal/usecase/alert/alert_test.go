package alert

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	"github.com/faruqrahmadani/slack-thread-alert/internal/repository/memory"
)

type slackCall struct {
	method    string // "post", "update" or "reply"
	channelID string
	threadTS  string
	status    domain.AlertStatus
}

type fakeSlack struct {
	calls []slackCall
	err   error
	seq   int
}

func (f *fakeSlack) PostSummary(_ context.Context, channelID string, alert *domain.Alert) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.seq++
	f.calls = append(f.calls, slackCall{method: "post", channelID: channelID, status: alert.Status})
	return fmt.Sprintf("ts-%d", f.seq), nil
}

func (f *fakeSlack) UpdateSummary(_ context.Context, thread *domain.Thread, alert *domain.Alert) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, slackCall{method: "update", channelID: thread.ChannelID, threadTS: thread.ThreadTS, status: alert.Status})
	return nil
}

func (f *fakeSlack) ReplyDetail(_ context.Context, channelID, threadTS string, alert *domain.Alert) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, slackCall{method: "reply", channelID: channelID, threadTS: threadTS, status: alert.Status})
	return nil
}

func setup() (*UseCase, *memory.ThreadRepository, *fakeSlack) {
	threadRepo := memory.NewThreadRepository(time.Hour, time.Hour)
	slack := &fakeSlack{}
	return NewUseCase(threadRepo, slack), threadRepo, slack
}

func newAlert(eventID string, status domain.AlertStatus) *domain.Alert {
	return &domain.Alert{
		Source:         domain.SourceDatadog,
		ThreadKey:      "datadog:cycle-1",
		EventID:        eventID,
		Status:         status,
		Title:          "CPU high",
		SlackChannelID: "C1",
	}
}

func TestHandle_FullCycle(t *testing.T) {
	ctx := context.Background()
	uc, threadRepo, slack := setup()

	steps := []*domain.Alert{
		newAlert("e1", domain.AlertStatusTriggered),
		newAlert("e2", domain.AlertStatusWarn),
		newAlert("e3", domain.AlertStatusRecovered),
	}
	for _, a := range steps {
		if err := uc.Handle(ctx, a); err != nil {
			t.Fatalf("Handle(%s): %v", a.EventID, err)
		}
	}

	want := []slackCall{
		{method: "post", channelID: "C1", status: domain.AlertStatusTriggered},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusTriggered},
		{method: "update", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusWarn},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusWarn},
		{method: "update", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusRecovered},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusRecovered},
	}
	assertCalls(t, slack.calls, want)

	if _, err := threadRepo.GetThread(ctx, "datadog:cycle-1"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("thread should be deleted after recovered, got %v", err)
	}
}

func TestHandle_NewCycleAfterRecoveredStartsNewThread(t *testing.T) {
	ctx := context.Background()
	uc, _, slack := setup()

	_ = uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered))
	_ = uc.Handle(ctx, newAlert("e2", domain.AlertStatusRecovered))
	_ = uc.Handle(ctx, newAlert("e3", domain.AlertStatusTriggered))

	got := slack.calls[len(slack.calls)-2:]
	want := []slackCall{
		{method: "post", channelID: "C1", status: domain.AlertStatusTriggered},
		{method: "reply", channelID: "C1", threadTS: "ts-2", status: domain.AlertStatusTriggered},
	}
	assertCalls(t, got, want)
}

func TestHandle_ReplyUsesStoredChannel(t *testing.T) {
	ctx := context.Background()
	uc, _, slack := setup()

	_ = uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered))

	moved := newAlert("e2", domain.AlertStatusRecovered)
	moved.SlackChannelID = "C2"
	_ = uc.Handle(ctx, moved)

	for _, call := range slack.calls[2:] {
		if call.channelID != "C1" {
			t.Fatalf("update/reply should go to the thread's channel C1, got %+v", call)
		}
	}
}

func TestHandle_DuplicateEventSkipped(t *testing.T) {
	ctx := context.Background()
	uc, _, slack := setup()

	_ = uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered))
	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered)); err != nil {
		t.Fatalf("duplicate should not error: %v", err)
	}

	if len(slack.calls) != 2 {
		t.Fatalf("duplicate event should not reach Slack, got %+v", slack.calls)
	}
}

func TestHandle_SlackErrorAllowsRetry(t *testing.T) {
	ctx := context.Background()
	uc, threadRepo, slack := setup()

	slack.err = errors.New("slack down")
	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered)); err == nil {
		t.Fatal("want error when Slack fails")
	}
	if _, err := threadRepo.GetThread(ctx, "datadog:cycle-1"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("thread must not be saved when Slack fails, got %v", err)
	}

	// Datadog retries the same event
	slack.err = nil
	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	want := []slackCall{
		{method: "post", channelID: "C1", status: domain.AlertStatusTriggered},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusTriggered},
	}
	assertCalls(t, slack.calls, want)
}

func TestHandle_RecoveredWithoutThread(t *testing.T) {
	ctx := context.Background()
	uc, threadRepo, slack := setup()

	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusRecovered)); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	want := []slackCall{
		{method: "post", channelID: "C1", status: domain.AlertStatusRecovered},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusRecovered},
	}
	assertCalls(t, slack.calls, want)
	if _, err := threadRepo.GetThread(ctx, "datadog:cycle-1"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("recovered without thread should not be saved, got %v", err)
	}
}

func TestHandle_InvalidAlert(t *testing.T) {
	ctx := context.Background()
	uc, _, slack := setup()

	noChannel := newAlert("e1", domain.AlertStatusTriggered)
	noChannel.SlackChannelID = ""
	noKey := newAlert("e2", domain.AlertStatusTriggered)
	noKey.ThreadKey = ""

	for _, a := range []*domain.Alert{noChannel, noKey} {
		if err := uc.Handle(ctx, a); !errors.Is(err, domain.ErrInvalidAlert) {
			t.Fatalf("Handle(%s): want ErrInvalidAlert, got %v", a.EventID, err)
		}
	}
	if len(slack.calls) != 0 {
		t.Fatalf("invalid alert should not reach Slack, got %+v", slack.calls)
	}
}

func assertCalls(t *testing.T, got, want []slackCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("want %d slack calls, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d: want %+v, got %+v", i, want[i], got[i])
		}
	}
}

// failOnReply fails ReplyDetail once, to simulate Slack failing after the parent was posted
type failOnReply struct {
	*fakeSlack
	failed bool
}

func (f *failOnReply) ReplyDetail(ctx context.Context, channelID, threadTS string, alert *domain.Alert) error {
	if !f.failed {
		f.failed = true
		return errors.New("slack timeout")
	}
	return f.fakeSlack.ReplyDetail(ctx, channelID, threadTS, alert)
}

func TestHandle_RetryAfterFailedReplyKeepsSingleParent(t *testing.T) {
	ctx := context.Background()
	slack := &failOnReply{fakeSlack: &fakeSlack{}}
	uc := NewUseCase(memory.NewThreadRepository(time.Hour, time.Hour), slack)

	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered)); err == nil {
		t.Fatal("want error when reply fails")
	}
	if err := uc.Handle(ctx, newAlert("e1", domain.AlertStatusTriggered)); err != nil {
		t.Fatalf("retry: %v", err)
	}

	want := []slackCall{
		{method: "post", channelID: "C1", status: domain.AlertStatusTriggered},
		{method: "update", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusTriggered},
		{method: "reply", channelID: "C1", threadTS: "ts-1", status: domain.AlertStatusTriggered},
	}
	assertCalls(t, slack.calls, want)
}
