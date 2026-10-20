package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
)

func newTestRepo(now *time.Time) *ThreadRepository {
	r := NewThreadRepository(time.Hour, time.Minute)
	r.now = func() time.Time { return *now }
	return r
}

func TestThreadRepository_SaveGetDelete(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	r := newTestRepo(&now)

	if _, err := r.GetThread(ctx, "datadog:abc"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("GetThread on empty repo: want ErrThreadNotFound, got %v", err)
	}

	err := r.SaveThread(ctx, &domain.Thread{Key: "datadog:abc", ChannelID: "C1", ThreadTS: "111.222"})
	if err != nil {
		t.Fatalf("SaveThread: %v", err)
	}

	got, err := r.GetThread(ctx, "datadog:abc")
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.ChannelID != "C1" || got.ThreadTS != "111.222" || !got.CreatedAt.Equal(now) {
		t.Fatalf("GetThread: unexpected thread %+v", got)
	}

	if err := r.DeleteThread(ctx, "datadog:abc"); err != nil {
		t.Fatalf("DeleteThread: %v", err)
	}
	if _, err := r.GetThread(ctx, "datadog:abc"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("GetThread after delete: want ErrThreadNotFound, got %v", err)
	}
}

func TestThreadRepository_ThreadExpires(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	r := newTestRepo(&now)

	_ = r.SaveThread(ctx, &domain.Thread{Key: "datadog:abc", ChannelID: "C1", ThreadTS: "111.222"})

	now = now.Add(59 * time.Minute)
	if _, err := r.GetThread(ctx, "datadog:abc"); err != nil {
		t.Fatalf("GetThread before TTL: %v", err)
	}

	now = now.Add(2 * time.Minute)
	if _, err := r.GetThread(ctx, "datadog:abc"); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("GetThread after TTL: want ErrThreadNotFound, got %v", err)
	}
}

func TestThreadRepository_ReturnsCopy(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	r := newTestRepo(&now)

	thread := &domain.Thread{Key: "datadog:abc", ThreadTS: "111.222"}
	_ = r.SaveThread(ctx, thread)
	thread.ThreadTS = "changed"

	got, _ := r.GetThread(ctx, "datadog:abc")
	got.ThreadTS = "changed-again"

	got, _ = r.GetThread(ctx, "datadog:abc")
	if got.ThreadTS != "111.222" {
		t.Fatalf("stored thread was mutated from outside: %+v", got)
	}
}

func TestThreadRepository_MarkEventProcessed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	r := newTestRepo(&now)

	first, _ := r.MarkEventProcessed(ctx, "evt-1")
	retry, _ := r.MarkEventProcessed(ctx, "evt-1")
	other, _ := r.MarkEventProcessed(ctx, "evt-2")
	if !first || retry || !other {
		t.Fatalf("want first=true retry=false other=true, got %v %v %v", first, retry, other)
	}

	now = now.Add(2 * time.Minute)
	afterTTL, _ := r.MarkEventProcessed(ctx, "evt-1")
	if !afterTTL {
		t.Fatal("event should be processable again after TTL")
	}
}
