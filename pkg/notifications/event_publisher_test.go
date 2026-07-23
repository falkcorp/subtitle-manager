// file: pkg/notifications/event_publisher_test.go
// version: 1.0.0
// guid: e0b3d7a4-9c15-4f28-b6a0-1d5e8c2f0947

package notifications

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/events"
)

// TestEventPublisherSendsToApprise verifies a download event is delivered to a
// (self-hosted) Apprise endpoint with the message body.
func TestEventPublisherSendsToApprise(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer srv.Close()

	svc := &Service{AppriseURL: srv.URL, client: &http.Client{Timeout: 5 * time.Second}}
	pub := NewEventPublisher(svc)

	pub.PublishSubtitleDownloaded(context.Background(), events.SubtitleDownloadedData{
		FilePath: "/media/movie.mkv", Language: "en", Provider: "opensubtitles",
	})

	if !strings.Contains(gotBody, "movie.mkv") || !strings.Contains(gotBody, "Downloaded") {
		t.Fatalf("apprise did not receive the expected body: %q", gotBody)
	}
}

// TestEventPublisherGating verifies per-kind suppression flags.
func TestEventPublisherGating(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer srv.Close()

	svc := &Service{AppriseURL: srv.URL, client: &http.Client{Timeout: 5 * time.Second}}
	pub := NewEventPublisher(svc)
	pub.NotifyOnDownload = false // suppress downloads

	pub.PublishSubtitleDownloaded(context.Background(), events.SubtitleDownloadedData{FilePath: "x"})
	if hits != 0 {
		t.Fatalf("expected download suppressed, got %d hits", hits)
	}
	pub.PublishSubtitleFailed(context.Background(), events.SubtitleFailedData{FilePath: "x", Error: "boom"})
	if hits != 1 {
		t.Fatalf("expected failure delivered, got %d hits", hits)
	}
}
