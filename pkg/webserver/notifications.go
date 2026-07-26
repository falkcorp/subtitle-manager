// file: pkg/webserver/notifications.go
// version: 1.0.0
// guid: 4c81f27a-06b3-4e59-9d84-b71e3a05c2f6
// last-edited: 2026-07-26

package webserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"path"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/notifications"
)

// notifyKey resolves a notification setting, preferring the namespaced key and
// falling back to the bare one.
//
// Both spellings have to be read because the two halves of this feature
// disagreed. The runtime has always read namespaced keys
// ("notifications.discord_webhook"), while the settings page saves through
// POST /api/config, which viper.Set()s exactly the keys the JSON body carries —
// and NotificationSettings.jsx sent bare ones ("discord_webhook"). Nothing read
// what the page wrote, so configuring notifications in the UI appeared to
// succeed and then never delivered anything.
//
// The UI now sends namespaced keys. The fallback keeps configurations saved by
// an older build working instead of silently dropping them on upgrade.
func notifyKey(name string) string {
	if v := viper.GetString("notifications." + name); v != "" {
		return v
	}
	return viper.GetString(name)
}

// notifyEnabled reports whether a channel's "enabled" switch is on.
//
// Unset means enabled: a config that sets only discord_webhook, as every
// pre-existing one does, must keep working. The switch only has effect once the
// settings page has written it.
func notifyEnabled(name string) bool {
	for _, k := range []string{"notifications." + name, name} {
		if viper.IsSet(k) {
			return viper.GetBool(k)
		}
	}
	return true
}

// smtpFromConfig builds an SMTP notifier from the settings page's fields, or
// nil when no host is configured.
//
// pkg/notifications has had a fully unit-tested SMTPNotifier since before this
// change, but nothing ever constructed one: the runtime only knew about
// notifications.email_url, a webhook that happens to send mail. Meanwhile the
// UI's email tab collects host/port/credentials/from/to. The two never met, so
// the email tab could not work regardless of the key namespace.
func smtpFromConfig() *notifications.SMTPNotifier {
	host := notifyKey("smtp_host")
	if host == "" {
		return nil
	}
	port := viper.GetInt("notifications.smtp_port")
	if port == 0 {
		port = viper.GetInt("smtp_port")
	}
	if port == 0 {
		port = 587
	}

	var auth smtp.Auth
	user, pass := notifyKey("smtp_username"), notifyKey("smtp_password")
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	// The UI collects recipients as one free-text field; accept comma or
	// semicolon separated addresses and drop empties.
	var to []string
	for _, addr := range strings.FieldsFunc(notifyKey("smtp_to"), func(r rune) bool {
		return r == ',' || r == ';'
	}) {
		if addr = strings.TrimSpace(addr); addr != "" {
			to = append(to, addr)
		}
	}
	if len(to) == 0 {
		return nil
	}

	return &notifications.SMTPNotifier{
		Addr: fmt.Sprintf("%s:%d", host, port),
		Auth: auth,
		From: notifyKey("smtp_from"),
		To:   to,
	}
}

// notificationServiceFromConfig builds a notification Service from the current
// configuration, or (nil, nil) when no channel is configured.
//
// Both the event publisher and the test-notification endpoint go through here
// so that "test succeeded" and "events are delivered" cannot disagree — a test
// button that exercises a different code path than production is worse than no
// button, because it reports confidence it has not earned.
func notificationServiceFromConfig() (*notifications.Service, error) {
	var discord, telegramToken, telegramChat, email string
	if notifyEnabled("discord_enabled") {
		discord = notifyKey("discord_webhook")
	}
	if notifyEnabled("telegram_enabled") {
		telegramToken = notifyKey("telegram_token")
		telegramChat = notifyKey("telegram_chat_id")
	}
	if notifyEnabled("email_enabled") {
		email = notifyKey("email_url")
	}
	apprise := notifyKey("apprise_url")

	var smtpNotifier *notifications.SMTPNotifier
	if notifyEnabled("email_enabled") {
		smtpNotifier = smtpFromConfig()
	}

	if discord == "" && telegramToken == "" && email == "" && apprise == "" && smtpNotifier == nil {
		return nil, nil
	}

	// Through notifications.New rather than constructing the struct directly:
	// New runs validateWebhookURL, which enforces HTTPS, rejects private and
	// link-local addresses and applies the provider allowlist. Bypassing it
	// would turn this endpoint into an SSRF primitive, since the URL is
	// operator-supplied and the server fetches it on demand.
	svc, err := notifications.New(discord, telegramToken, telegramChat, email)
	if err != nil {
		return nil, err
	}
	// Apprise and SMTP are deliberately set after New: both are self-hosted
	// infrastructure, commonly on a private address, which is precisely what
	// the webhook allowlist exists to reject.
	svc.AppriseURL = apprise
	svc.SMTP = smtpNotifier
	return svc, nil
}

// notificationTestHandler sends a test notification on a single channel.
//
//	POST /api/notifications/test/{type}
//
// {type} is one of discord, telegram, email or apprise. It reports what
// actually happened rather than a blanket 200: an unconfigured channel is a
// different answer than a provider that rejected the message, and the operator
// needs to tell them apart.
//
// The test uses saved configuration, not the values currently typed into the
// form, because the request carries only a message. Settings must be saved
// before testing. Bazarr tests the live form values; matching that would need
// the credentials in the request body, which is a meaningfully worse trade.
func notificationTestHandler() http.Handler {
	type reqBody struct {
		Message string `json:"message"`
	}
	logger := logging.GetLogger("notifications")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// path.Base rather than TrimPrefix of a literal "/api/notifications/test":
		// the route is mounted at prefix+"/api/notifications/test/", where prefix
		// comes from base_url. Behind a subpath the incoming path is
		// "/sm/api/notifications/test/discord", which the literal prefix does not
		// match — every channel would resolve as unknown and the feature would be
		// dead for anyone running under a base URL. A channel name never contains
		// a slash, so the last segment is exactly it.
		name := path.Base(r.URL.Path)
		if name == "" || name == "." || name == "/" || name == "test" {
			http.Error(w, "no notification channel given", http.StatusBadRequest)
			return
		}

		// push and webhook are offered by the settings page but have no backend
		// here, and saying so is the honest answer. Reporting success for a
		// channel that will never fire is the specific failure this endpoint
		// exists to prevent.
		switch name {
		case "push":
			http.Error(w, "push (Pushover) notifications are not implemented yet; "+
				"only the hostname is allowlisted, there is no sender",
				http.StatusNotImplemented)
			return
		case "webhook":
			http.Error(w, "generic webhooks are managed separately: configure them "+
				"under /api/webhooks/config and test with /api/webhooks/test",
				http.StatusNotImplemented)
			return
		}

		channel := notifications.Channel(name)
		switch channel {
		case notifications.ChannelDiscord, notifications.ChannelTelegram,
			notifications.ChannelEmail, notifications.ChannelApprise:
		default:
			http.Error(w, "unknown notification channel "+name, http.StatusBadRequest)
			return
		}

		var q reqBody
		// An absent or unparseable body is not fatal; the message is cosmetic.
		_ = json.NewDecoder(r.Body).Decode(&q)
		if strings.TrimSpace(q.Message) == "" {
			q.Message = "Test notification from Subtitle Manager"
		}

		svc, err := notificationServiceFromConfig()
		if err != nil {
			// The error text comes from validateWebhookURL and names the
			// rejected host, not any credential.
			http.Error(w, "notification settings rejected: "+err.Error(),
				http.StatusBadRequest)
			return
		}
		if svc == nil || !svc.Configured(channel) {
			http.Error(w, string(channel)+" is not configured; save its settings first",
				http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		if err := svc.SendTo(ctx, channel, q.Message); err != nil {
			// Log the channel and error only. The Service holds bot tokens and
			// SMTP passwords, and this is the one place tempted to dump config
			// on failure.
			logger.Warnf("test notification via %s failed: %v", channel, err)
			status := http.StatusBadGateway
			if errors.Is(err, notifications.ErrChannelNotConfigured) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}

		logger.Infof("test notification sent via %s", channel)
		writeJSON(w, map[string]any{"status": "sent", "channel": string(channel)})
	})
}
