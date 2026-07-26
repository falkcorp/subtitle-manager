package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
)

func TestDiscordNotifier(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var m map[string]string
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		got = m["content"]
	}))
	defer srv.Close()

	n := DiscordNotifier{WebhookURL: srv.URL, Client: srv.Client()}
	if err := n.Notify(context.Background(), "hello"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if got != "hello" {
		t.Fatalf("unexpected message: %s", got)
	}
}

func TestTelegramNotifier(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var m map[string]string
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		got = m["text"]
	}))
	defer srv.Close()

	n := TelegramNotifier{BotToken: "t", ChatID: "1", Client: srv.Client(), APIBase: srv.URL}
	if err := n.Notify(context.Background(), "hi"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if got != "hi" {
		t.Fatalf("unexpected message: %s", got)
	}
}

func TestSMTPNotifier(t *testing.T) {
	var addr, from string
	var to []string
	var body []byte
	n := SMTPNotifier{
		Addr: "smtp:25",
		From: "a@example.com",
		To:   []string{"b@example.com"},
		Send: func(a string, _ smtp.Auth, f string, t []string, msg []byte) error {
			addr = a
			from = f
			to = t
			body = msg
			return nil
		},
	}
	if err := n.Notify(context.Background(), "hi there"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if addr != "smtp:25" || from != "a@example.com" || len(to) != 1 || to[0] != "b@example.com" {
		t.Fatalf("unexpected params")
	}
	// From/To/MIME headers are required by many MTAs; a Subject-only message is
	// widely rejected as spam.
	head, text, ok := strings.Cut(string(body), "\r\n\r\n")
	if !ok {
		t.Fatalf("message has no header/body separator: %q", body)
	}
	for _, want := range []string{
		"From: a@example.com",
		"To: b@example.com",
		"Subject: Subtitle Manager",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	} {
		if !strings.Contains(head, want) {
			t.Errorf("header block is missing %q:\n%s", want, head)
		}
	}
	if text != "hi there" {
		t.Errorf("body = %q, want the message unmodified", text)
	}
}

// TestSMTPNotifierRejectsHeaderInjection verifies the message body cannot
// introduce headers, and that a CR/LF smuggled into an address cannot either.
//
// The body is placed after the blank line that terminates the headers, so
// newlines in it start body lines rather than headers. The addresses are the
// fields where injection would actually matter, so those are stripped.
func TestSMTPNotifierRejectsHeaderInjection(t *testing.T) {
	var body []byte
	n := SMTPNotifier{
		Addr: "smtp:25",
		From: "a@example.com\r\nBcc: attacker@example.com",
		To:   []string{"b@example.com"},
		Send: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			body = msg
			return nil
		},
	}
	if err := n.Notify(context.Background(), "line one\r\nBcc: other@example.com"); err != nil {
		t.Fatalf("notify: %v", err)
	}

	// Injection means a new header *line*. The smuggled text surviving inside
	// the From value is inert, so assert on line starts rather than substring
	// presence.
	head, _, _ := strings.Cut(string(body), "\r\n\r\n")
	for _, line := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Errorf("a Bcc header line was injected:\n%s", head)
		}
	}
	if strings.Count(head, "\r\n") != 4 {
		t.Errorf("header block does not have exactly the 5 intended headers:\n%s", head)
	}
}

// TestNew verifies that New creates a service with proper validation.
func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		discordURL     string
		telegramToken  string
		telegramChatID string
		emailURL       string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "valid configuration",
			discordURL:     "https://discord.com/api/webhooks/123/abc",
			telegramToken:  "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
			telegramChatID: "123456",
			emailURL:       "https://hooks.slack.com/services/abc/def/ghi",
			expectError:    false,
		},
		{
			name:           "empty URLs allowed",
			discordURL:     "",
			telegramToken:  "",
			telegramChatID: "",
			emailURL:       "",
			expectError:    false,
		},
		{
			name:          "invalid Discord URL - not HTTPS",
			discordURL:    "http://discord.com/api/webhooks/123/abc",
			expectError:   true,
			errorContains: "only HTTPS URLs are allowed",
		},
		{
			name:          "invalid Discord URL - private IP",
			discordURL:    "https://192.168.1.1/webhook",
			expectError:   true,
			errorContains: "webhooks to private/internal addresses are not allowed",
		},
		{
			name:          "invalid Discord URL - unauthorized domain",
			discordURL:    "https://malicious.com/webhook",
			expectError:   true,
			errorContains: "webhook domain not in allowed list",
		},
		{
			name:          "invalid Telegram token",
			telegramToken: "invalid-token",
			expectError:   true,
			errorContains: "invalid Telegram bot token format",
		},
		{
			name:          "invalid email URL",
			emailURL:      "https://127.0.0.1/webhook",
			expectError:   true,
			errorContains: "webhooks to private/internal addresses are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.discordURL, tt.telegramToken, tt.telegramChatID, tt.emailURL)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if svc == nil {
					t.Fatal("expected service to be created, got nil")
				}
				if svc.DiscordWebhook != tt.discordURL {
					t.Errorf("expected Discord webhook %q, got %q", tt.discordURL, svc.DiscordWebhook)
				}
				if svc.TelegramToken != tt.telegramToken {
					t.Errorf("expected Telegram token %q, got %q", tt.telegramToken, svc.TelegramToken)
				}
				if svc.TelegramChatID != tt.telegramChatID {
					t.Errorf("expected Telegram chat ID %q, got %q", tt.telegramChatID, svc.TelegramChatID)
				}
				if svc.EmailURL != tt.emailURL {
					t.Errorf("expected email URL %q, got %q", tt.emailURL, svc.EmailURL)
				}
			}
		})
	}
}

// TestIsValidTelegramToken tests Telegram token validation.
func TestIsValidTelegramToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{
			name:  "valid token",
			token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
			valid: true,
		},
		{
			name:  "valid token with numbers and letters",
			token: "987654321:XYZ123abc456DEF789ghi",
			valid: true,
		},
		{
			name:  "empty token",
			token: "",
			valid: false,
		},
		{
			name:  "too short",
			token: "123:abc",
			valid: false,
		},
		{
			name:  "no colon",
			token: "123456789ABCdefGHIjklMNOpqrsTUVwxyz",
			valid: false,
		},
		{
			name:  "multiple colons",
			token: "123:456:789",
			valid: false,
		},
		{
			name:  "short auth token",
			token: "123456789:abc",
			valid: false,
		},
		{
			name:  "dangerous pattern - path traversal",
			token: "123456789:../etc/passwd",
			valid: false,
		},
		{
			name:  "dangerous pattern - script tag",
			token: "123456789:<script>alert(1)</script>",
			valid: false,
		},
		{
			name:  "dangerous pattern - quotes",
			token: "123456789:token\"with'quotes",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTelegramToken(tt.token)
			if result != tt.valid {
				t.Errorf("expected %v, got %v for token %q", tt.valid, result, tt.token)
			}
		})
	}
}

// TestValidateWebhookURL tests webhook URL validation.
func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty URL",
			url:         "",
			expectError: false,
		},
		{
			name:        "valid Discord webhook",
			url:         "https://discord.com/api/webhooks/123/abc",
			expectError: false,
		},
		{
			name:        "valid Slack webhook",
			url:         "https://hooks.slack.com/services/T00/B00/XXXXXXXXXXXXXXXXXXXXXXXX",
			expectError: false,
		},
		{
			name:        "invalid scheme - HTTP",
			url:         "http://discord.com/api/webhooks/123/abc",
			expectError: true,
			errorMsg:    "only HTTPS URLs are allowed",
		},
		{
			name:        "invalid scheme - FTP",
			url:         "ftp://example.com/webhook",
			expectError: true,
			errorMsg:    "only HTTPS URLs are allowed",
		},
		{
			name:        "localhost URL",
			url:         "https://localhost/webhook",
			expectError: true,
			errorMsg:    "webhooks to private/internal addresses are not allowed",
		},
		{
			name:        "private IP - 192.168.x.x",
			url:         "https://192.168.1.100/webhook",
			expectError: true,
			errorMsg:    "webhooks to private/internal addresses are not allowed",
		},
		{
			name:        "private IP - 10.x.x.x",
			url:         "https://10.0.0.1/webhook",
			expectError: true,
			errorMsg:    "webhooks to private/internal addresses are not allowed",
		},
		{
			name:        "private IP - 172.16.x.x",
			url:         "https://172.16.0.1/webhook",
			expectError: true,
			errorMsg:    "webhooks to private/internal addresses are not allowed",
		},
		{
			name:        "link-local IP",
			url:         "https://169.254.1.1/webhook",
			expectError: true,
			errorMsg:    "webhooks to private/internal addresses are not allowed",
		},
		{
			name:        "unauthorized domain",
			url:         "https://malicious.com/webhook",
			expectError: true,
			errorMsg:    "webhook domain not in allowed list",
		},
		{
			name:        "malformed URL",
			url:         "://invalid-url",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestIsPrivateOrLocalhost tests private IP detection.
func TestIsPrivateOrLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"localhost", "localhost", true},
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv6 loopback", "::1", true},
		{"private 192.168.x.x", "192.168.1.1", true},
		{"private 10.x.x.x", "10.0.0.1", true},
		{"private 172.16.x.x", "172.16.0.1", true},
		{"private 172.31.x.x", "172.31.255.255", true},
		{"link-local", "169.254.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"domain", "discord.com", false},
		{"subdomain", "api.telegram.org", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrivateOrLocalhost(tt.host)
			if result != tt.expected {
				t.Errorf("expected %v, got %v for host %q", tt.expected, result, tt.host)
			}
		})
	}
}

// TestServiceSend tests the Send method with multiple notification targets.
func TestServiceSend(t *testing.T) {
	// Mock servers for different services
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg map[string]string
		json.NewDecoder(r.Body).Decode(&msg)
		if msg["content"] != "test message" {
			t.Errorf("Discord: expected 'test message', got %q", msg["content"])
		}
	}))
	defer discordServer.Close()

	emailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg map[string]string
		json.NewDecoder(r.Body).Decode(&msg)
		if msg["message"] != "test message" {
			t.Errorf("Email: expected 'test message', got %q", msg["message"])
		}
	}))
	defer emailServer.Close()

	// Test with Discord and Email only (since Telegram is harder to mock due to hardcoded URL)
	service := &Service{
		DiscordWebhook: discordServer.URL,
		EmailURL:       emailServer.URL,
		client:         &http.Client{},
	}

	err := service.Send(context.Background(), "test message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestServiceSendWithTelegram tests Send method with Telegram configuration.
func TestServiceSendWithTelegram(t *testing.T) {
	telegramServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg map[string]string
		json.NewDecoder(r.Body).Decode(&msg)
		if msg["text"] != "test message" {
			t.Errorf("Telegram: expected 'test message', got %q", msg["text"])
		}
		if msg["chat_id"] != "123456" {
			t.Errorf("Telegram: expected chat_id '123456', got %q", msg["chat_id"])
		}
		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer telegramServer.Close()

	// Test the Telegram notifier directly since it's easier to mock
	telegramNotifier := TelegramNotifier{
		BotToken: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
		ChatID:   "123456",
		Client:   telegramServer.Client(),
		APIBase:  telegramServer.URL,
	}

	err := telegramNotifier.Notify(context.Background(), "test message")
	if err != nil {
		t.Fatalf("expected no error from Telegram notifier, got %v", err)
	}
}

// TestNopNotifier tests the Nop notifier.
func TestNopNotifier(t *testing.T) {
	nop := Nop{}
	err := nop.Notify(context.Background(), "test message")
	if err != nil {
		t.Errorf("expected no error from Nop notifier, got %v", err)
	}
}

// TestFuncNotifier tests the Func notifier adapter.
func TestFuncNotifier(t *testing.T) {
	var receivedMsg string
	var receivedCtx context.Context

	fn := Func(func(ctx context.Context, msg string) error {
		receivedCtx = ctx
		receivedMsg = msg
		return nil
	})

	ctx := context.Background()
	err := fn.Notify(ctx, "test message")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if receivedMsg != "test message" {
		t.Errorf("expected 'test message', got %q", receivedMsg)
	}
	if receivedCtx != ctx {
		t.Error("context not passed correctly")
	}
}
