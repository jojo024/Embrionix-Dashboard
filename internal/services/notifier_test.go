package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/embrionix/dashboard/internal/models"
)

func TestNotifierShouldNotify(t *testing.T) {
	n := NewNotifier("https://example.test/hook", []string{"critical", "offline"})
	if !n.ShouldNotify(models.StatusCritical) {
		t.Error("expected critical to notify")
	}
	if n.ShouldNotify(models.StatusOnline) {
		t.Error("did not expect online to notify")
	}

	disabled := NewNotifier("", []string{"critical"})
	if disabled.Enabled() || disabled.ShouldNotify(models.StatusCritical) {
		t.Error("notifier with empty URL must be disabled")
	}
}

func TestNotifierPostsSlackCompatiblePayload(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, []string{"critical"})
	n.Notify(context.Background(), models.AlertEvent{
		DeviceName: "Encap-1",
		FromStatus: models.StatusOnline,
		ToStatus:   models.StatusCritical,
		Message:    "Core temperature critical",
	})

	if _, ok := got["text"]; !ok {
		t.Error("payload missing Slack-compatible 'text' field")
	}
	if _, ok := got["event"]; !ok {
		t.Error("payload missing structured 'event' field")
	}
}
