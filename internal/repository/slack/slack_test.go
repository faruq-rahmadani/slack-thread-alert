package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/config"
	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	slackgo "github.com/slack-go/slack"
)

// fakeSlack records chat.postMessage / chat.update form values and replies like the Slack API
func fakeSlack(t *testing.T, ok bool) (*Repository, *[]map[string]string) {
	t.Helper()

	var calls []map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		call := map[string]string{"method": r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]}
		for k := range r.PostForm {
			call[k] = r.PostForm.Get(k)
		}
		calls = append(calls, call)

		w.Header().Set("Content-Type", "application/json")
		if !ok {
			_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
			return
		}
		ts := call["ts"]
		if ts == "" {
			ts = "1700000000.000100"
		}
		_, _ = w.Write([]byte(`{"ok":true,"channel":"` + call["channel"] + `","ts":"` + ts + `"}`))
	}))
	t.Cleanup(srv.Close)

	repo := NewRepository(config.Slack{BotToken: "xoxb-test"}, slackgo.OptionAPIURL(srv.URL+"/"))
	repo.now = func() time.Time { return time.Unix(1700000000, 0) }
	return repo, &calls
}

func testAlert() *domain.Alert {
	return &domain.Alert{
		Source:      domain.SourceDatadog,
		Status:      domain.AlertStatusTriggered,
		Title:       "CPU high",
		Summary:     "cpu over host:api was > 90.0 during the last 5m",
		Message:     "## Runbook\ncheck **api** pods",
		Link:        "https://app.datadoghq.com/event/event?id=1",
		MonitorLink: "https://app.datadoghq.com/monitors/1",
		Snapshot:    "https://snapshot.datadoghq.com/graph.png",
		Priority:    "P1",
		EventTime:   time.Unix(1700000000, 0),
	}
}

func TestRepository_PostSummary(t *testing.T) {
	repo, calls := fakeSlack(t, true)

	ts, err := repo.PostSummary(context.Background(), "C123", testAlert())
	if err != nil {
		t.Fatalf("PostSummary: %v", err)
	}
	if ts != "1700000000.000100" {
		t.Fatalf("want ts 1700000000.000100, got %q", ts)
	}

	call := (*calls)[0]
	if call["method"] != "chat.postMessage" || call["channel"] != "C123" || call["thread_ts"] != "" {
		t.Fatalf("unexpected request %+v", call)
	}
	if call["text"] != ":red_circle: *ALERT* · [P1] CPU high" {
		t.Fatalf("unexpected headline %q", call["text"])
	}
	assertContains(t, call["attachments"], colorDanger, "cpu over host:api was", "during the last 5m", "just started", "View event", "Monitor", "snapshot.datadoghq.com/graph.png")
	assertNotContains(t, call["attachments"], "Runbook")
}

func TestRepository_UpdateSummary(t *testing.T) {
	repo, calls := fakeSlack(t, true)

	alert := testAlert()
	alert.Status = domain.AlertStatusRecovered
	thread := &domain.Thread{ChannelID: "C123", ThreadTS: "1700000000.000100", StartedAt: time.Unix(1700000000, 0).Add(-6 * time.Minute)}

	if err := repo.UpdateSummary(context.Background(), thread, alert); err != nil {
		t.Fatalf("UpdateSummary: %v", err)
	}

	call := (*calls)[0]
	if call["method"] != "chat.update" || call["channel"] != "C123" || call["ts"] != "1700000000.000100" {
		t.Fatalf("summary must be updated with chat.update on the parent ts, got %+v", call)
	}
	if !strings.Contains(call["text"], "*RESOLVED*") {
		t.Fatalf("unexpected headline %q", call["text"])
	}
	assertContains(t, call["attachments"], colorGood, "resolved after 6m", "snapshot.datadoghq.com/graph.png")
}

func TestRepository_ReplyDetail(t *testing.T) {
	repo, calls := fakeSlack(t, true)

	if err := repo.ReplyDetail(context.Background(), "C123", "1700000000.000100", testAlert()); err != nil {
		t.Fatalf("ReplyDetail: %v", err)
	}

	call := (*calls)[0]
	if call["method"] != "chat.postMessage" || call["thread_ts"] != "1700000000.000100" {
		t.Fatalf("detail must be posted in the thread, got %+v", call)
	}
	assertContains(t, call["attachments"], "*Runbook*", "check *api* pods", "snapshot.datadoghq.com/graph.png")
}

func TestRepository_SlackError(t *testing.T) {
	repo, _ := fakeSlack(t, false)

	_, err := repo.PostSummary(context.Background(), "C404", testAlert())
	if err == nil || !strings.Contains(err.Error(), "channel_not_found") {
		t.Fatalf("want channel_not_found error, got %v", err)
	}
}

func assertContains(t *testing.T, s string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Errorf("want %q in %s", w, s)
		}
	}
}

func assertNotContains(t *testing.T, s string, notWant ...string) {
	t.Helper()
	for _, w := range notWant {
		if strings.Contains(s, w) {
			t.Errorf("did not want %q in %s", w, s)
		}
	}
}
