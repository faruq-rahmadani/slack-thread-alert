package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	"github.com/labstack/echo/v4"
)

type fakeAlertUC struct {
	got *domain.Alert
}

func (f *fakeAlertUC) Handle(_ context.Context, alert *domain.Alert) error {
	f.got = alert
	return alert.Validate()
}

func callScalyr(t *testing.T, target, contentType, body string) (*httptest.ResponseRecorder, *fakeAlertUC) {
	t.Helper()
	uc := &fakeAlertUC{}
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()

	if err := NewScalyrController(uc).ScalyrAlertWebhook(echo.New().NewContext(req, rec)); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return rec, uc
}

func TestScalyrAlertWebhook_FormContentType(t *testing.T) {
	// Scalyr's default content-type, the body is still JSON
	body := `{"status":"triggered","alert_id":"scalyr-alert-1","alert_name":"HTTP 500"}`
	rec, uc := callScalyr(t, "/webhook/scalyr?channel_id=c123", echo.MIMEApplicationForm, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	if uc.got.ThreadKey != "scalyr:scalyr-alert-1" || uc.got.SlackChannelID != "C123" || uc.got.Status != domain.AlertStatusTriggered {
		t.Errorf("unexpected alert %+v", uc.got)
	}
}

func TestScalyrAlertWebhook_BodyChannelWins(t *testing.T) {
	body := `{"status":"resolved","alert_id":"scalyr-alert-1","channel_id":"C999"}`
	rec, uc := callScalyr(t, "/webhook/scalyr?channel_id=C123", echo.MIMEApplicationJSON, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	if uc.got.SlackChannelID != "C999" {
		t.Errorf("body channel_id should win, got %q", uc.got.SlackChannelID)
	}
}

func TestScalyrAlertWebhook_BadRequest(t *testing.T) {
	tests := map[string]string{
		"invalid json": `not json`,
		"no channel":   `{"status":"triggered","alert_id":"scalyr-alert-1"}`,
		"no alert id":  `{"status":"triggered","channel_id":"C1"}`,
	}
	for name, body := range tests {
		rec, _ := callScalyr(t, "/webhook/scalyr", echo.MIMEApplicationJSON, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
}
