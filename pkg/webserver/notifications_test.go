// file: pkg/webserver/notifications_test.go
// version: 1.0.0
// guid: 9e40c5b2-7a18-4f63-b2d5-0c86e1439af7
// last-edited: 2026-07-26

package webserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/notifications"
)

// useNotifyConfig applies keys to viper for one test and restores them after.
func useNotifyConfig(t *testing.T, keys map[string]any) {
	t.Helper()
	prev := map[string]any{}
	for k := range keys {
		prev[k] = viper.Get(k)
	}
	t.Cleanup(func() {
		for k, v := range prev {
			viper.Set(k, v)
		}
	})
	for k, v := range keys {
		viper.Set(k, v)
	}
}

// TestNotifyKeyPrefersNamespaced covers the compatibility shim between the two
// key spellings this feature used to disagree about.
func TestNotifyKeyPrefersNamespaced(t *testing.T) {
	useNotifyConfig(t, map[string]any{
		"notifications.discord_webhook": "https://discord.com/api/webhooks/new",
		"discord_webhook":               "https://discord.com/api/webhooks/old",
		"telegram_token":                "123456789:abcdefghijklmnop",
		"notifications.telegram_token":  nil,
	})

	if got := notifyKey("discord_webhook"); got != "https://discord.com/api/webhooks/new" {
		t.Errorf("namespaced key should win, got %q", got)
	}
	// A config written by an older build has only the bare key; dropping it on
	// upgrade would silently disable a working channel.
	if got := notifyKey("telegram_token"); got != "123456789:abcdefghijklmnop" {
		t.Errorf("bare key should be honoured as a fallback, got %q", got)
	}
}

// TestNotifyEnabledDefaultsOn verifies an absent switch means enabled.
//
// Every configuration written before the settings page existed sets only the
// webhook URL. If an unset switch meant "off", upgrading would silently stop
// delivering notifications that had been working.
func TestNotifyEnabledDefaultsOn(t *testing.T) {
	useNotifyConfig(t, map[string]any{
		"notifications.discord_enabled": nil,
		"discord_enabled":               nil,
	})
	if !notifyEnabled("discord_enabled") {
		t.Error("an unset enabled switch must default to on")
	}

	useNotifyConfig(t, map[string]any{"notifications.discord_enabled": false})
	if notifyEnabled("discord_enabled") {
		t.Error("an explicit false must disable the channel")
	}
}

// TestNotificationServiceFromConfigSMTP verifies the email tab's SMTP fields
// reach a notifier.
//
// SMTPNotifier existed and was unit-tested long before anything constructed
// one: the runtime only knew notifications.email_url. This asserts the wiring,
// not the SMTP protocol.
func TestNotificationServiceFromConfigSMTP(t *testing.T) {
	useNotifyConfig(t, map[string]any{
		"notifications.smtp_host": "mail.example.com",
		"notifications.smtp_port": 2525,
		"notifications.smtp_from": "sm@example.com",
		// Two recipients in one free-text field, as the UI collects them.
		"notifications.smtp_to":         "a@example.com, b@example.com",
		"notifications.discord_webhook": "",
		"notifications.email_url":       "",
		"notifications.apprise_url":     "",
		"notifications.telegram_token":  "",
		"discord_webhook":               "",
		"email_url":                     "",
		"apprise_url":                   "",
		"telegram_token":                "",
		"smtp_host":                     "",
	})

	svc, err := notificationServiceFromConfig()
	if err != nil {
		t.Fatalf("notificationServiceFromConfig: %v", err)
	}
	if svc == nil {
		t.Fatal("SMTP settings alone produced no service; the email tab would stay dead")
	}
	if svc.SMTP == nil {
		t.Fatal("SMTP notifier not built from smtp_* settings")
	}
	if svc.SMTP.Addr != "mail.example.com:2525" {
		t.Errorf("Addr = %q, want host:port from config", svc.SMTP.Addr)
	}
	if len(svc.SMTP.To) != 2 {
		t.Errorf("To = %v, want the comma-separated field split into two", svc.SMTP.To)
	}
	if !svc.Configured(notifications.ChannelEmail) {
		t.Error("email channel reports unconfigured despite a usable SMTP notifier")
	}
}

// TestNotificationTestHandler covers the responses the settings page relies on
// to tell an unconfigured channel from a failing one.
func TestNotificationTestHandler(t *testing.T) {
	useNotifyConfig(t, map[string]any{
		"notifications.discord_webhook": "",
		"notifications.telegram_token":  "",
		"notifications.email_url":       "",
		"notifications.apprise_url":     "",
		"notifications.smtp_host":       "",
		"discord_webhook":               "",
		"telegram_token":                "",
		"email_url":                     "",
		"apprise_url":                   "",
		"smtp_host":                     "",
	})
	h := notificationTestHandler()

	for name, tc := range map[string]struct {
		method, path string
		want         int
	}{
		// Offered by the UI, no sender exists. Reporting success here is the
		// exact failure this endpoint is meant to prevent.
		"pushover is honest about being unimplemented": {
			http.MethodPost, "/api/notifications/test/push", http.StatusNotImplemented},
		"generic webhook points at its own subsystem": {
			http.MethodPost, "/api/notifications/test/webhook", http.StatusNotImplemented},
		"unknown channel": {
			http.MethodPost, "/api/notifications/test/carrier-pigeon", http.StatusBadRequest},
		"no channel given": {
			http.MethodPost, "/api/notifications/test/", http.StatusBadRequest},
		"unconfigured channel is a 400, not a fake success": {
			http.MethodPost, "/api/notifications/test/discord", http.StatusBadRequest},
		"GET is rejected": {
			http.MethodGet, "/api/notifications/test/discord", http.StatusMethodNotAllowed},
	} {
		t.Run(name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(tc.method, tc.path,
				strings.NewReader(`{"message":"hello"}`)))
			if rr.Code != tc.want {
				t.Errorf("got %d, want %d (body %s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// TestNotificationTestHandlerUnderBaseURL verifies the channel is still parsed
// when the server runs under a base_url.
//
// The route is mounted at prefix+"/api/notifications/test/". Stripping the
// literal "/api/notifications/test" would match nothing once prefix is
// non-empty, so every channel would resolve as unknown and the feature would be
// dead for anyone behind a subpath — while every test using an empty prefix
// passed.
func TestNotificationTestHandlerUnderBaseURL(t *testing.T) {
	useNotifyConfig(t, map[string]any{
		"notifications.discord_webhook": "",
		"discord_webhook":               "",
	})
	rr := httptest.NewRecorder()
	notificationTestHandler().ServeHTTP(rr, httptest.NewRequest(
		http.MethodPost, "/subtitles/api/notifications/test/discord",
		strings.NewReader(`{"message":"hi"}`)))

	// discord is a known channel that happens to be unconfigured here, so the
	// correct answer is "not configured" (400). "unknown channel" would also be
	// a 400, so assert on the text to tell the two apart.
	if body := rr.Body.String(); strings.Contains(body, "unknown notification channel") {
		t.Errorf("channel not parsed under a base_url: %s", body)
	}
}

// TestNotificationTestHandlerDelivers verifies a configured channel actually
// sends, and that a rejecting provider is reported as a failure.
//
// Apprise is the channel used here because it is the only one whose URL is
// deliberately not run through the webhook allowlist — it is self-hosted
// infrastructure — which is what makes it reachable from an httptest server.
func TestNotificationTestHandlerDelivers(t *testing.T) {
	var gotBody string
	var status int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		w.WriteHeader(status)
	}))
	defer srv.Close()

	base := map[string]any{
		"notifications.apprise_url":     srv.URL,
		"notifications.discord_webhook": "",
		"notifications.telegram_token":  "",
		"notifications.email_url":       "",
		"notifications.smtp_host":       "",
		"apprise_url":                   "",
		"discord_webhook":               "",
		"telegram_token":                "",
		"email_url":                     "",
		"smtp_host":                     "",
	}

	t.Run("delivers the message", func(t *testing.T) {
		useNotifyConfig(t, base)
		status = http.StatusOK
		rr := httptest.NewRecorder()
		notificationTestHandler().ServeHTTP(rr, httptest.NewRequest(
			http.MethodPost, "/api/notifications/test/apprise",
			strings.NewReader(`{"message":"ping from the test"}`)))

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200 (body %s)", rr.Code, rr.Body.String())
		}
		if !strings.Contains(gotBody, "ping from the test") {
			t.Errorf("provider received %q, which does not carry the message", gotBody)
		}
	})

	t.Run("a rejecting provider is a failure, not a success", func(t *testing.T) {
		useNotifyConfig(t, base)
		status = http.StatusUnauthorized
		rr := httptest.NewRecorder()
		notificationTestHandler().ServeHTTP(rr, httptest.NewRequest(
			http.MethodPost, "/api/notifications/test/apprise",
			strings.NewReader(`{"message":"hi"}`)))

		if rr.Code == http.StatusOK {
			t.Error("provider returned 401 but the endpoint reported success; " +
				"the operator would believe notifications work")
		}
	})
}

// TestNotificationTestHandlerRejectsUnsafeWebhook verifies the SSRF guard is on
// the path this endpoint takes.
//
// The endpoint fetches an operator-supplied URL on demand, so it would be an
// SSRF primitive if it constructed the Service directly instead of going
// through notifications.New, which enforces HTTPS, rejects private and
// link-local addresses, and applies a provider allowlist.
func TestNotificationTestHandlerRejectsUnsafeWebhook(t *testing.T) {
	for name, target := range map[string]string{
		"link-local metadata endpoint": "https://169.254.169.254/latest/meta-data/",
		"private address":              "https://192.168.1.10/hook",
		"plain http":                   "http://discord.com/api/webhooks/x",
		"host outside the allowlist":   "https://evil.example.com/hook",
	} {
		t.Run(name, func(t *testing.T) {
			useNotifyConfig(t, map[string]any{
				"notifications.discord_webhook": target,
				"discord_webhook":               "",
			})
			rr := httptest.NewRecorder()
			notificationTestHandler().ServeHTTP(rr, httptest.NewRequest(
				http.MethodPost, "/api/notifications/test/discord",
				strings.NewReader(`{"message":"hi"}`)))

			if rr.Code == http.StatusOK {
				t.Fatalf("%s was accepted; the endpoint would fetch it on demand", target)
			}
			if rr.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400 (body %s)", rr.Code, rr.Body.String())
			}
		})
	}
}
