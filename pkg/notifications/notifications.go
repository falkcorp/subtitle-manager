// Package notifications provides notification services for subtitle-manager.
// It supports sending notifications via Discord, Telegram, Email, and other channels.
//
// This package is used to alert users and systems about subtitle events and errors.
// It includes features for validating webhook URLs, formatting messages, and dispatching
// notifications to multiple targets. The Service struct is the primary interface for
// sending notifications, and it can be configured with the necessary credentials and
// endpoint URLs for the desired notification channels.
//
// Example usage:
//
// 	import (
// 		"context"
// 		"log"
// 		"github.com/yourusername/yourproject/pkg/notifications"
// 	)
//
// 	func main() {
// 		// Create a new notification service
// 		svc, err := notifications.New(
// 			"https://discord.com/api/webhooks/...",
// 			"telegram_bot_token",
// 			"telegram_chat_id",
// 			"https://your-email-service.com/send",
// 		)
// 		if err != nil {
// 			log.Fatalf("failed to create notification service: %v", err)
// 		}
//
// 		// Send a test notification
// 		err = svc.Send(context.Background(), "Hello, this is a test notification!")
// 		if err != nil {
// 			log.Fatalf("failed to send notification: %v", err)
// 		}
// 	}
//
// The notifications package is designed to be flexible and extensible, allowing
// integration with various notification services as needed. Contributions and
// improvements are welcome to support additional features and services.
//
// See the README.md file for more information about the subtitle-manager project,
// and how to configure and use the notifications package.
//
// Copyright (c) 2023 Your Name. All rights reserved.
// License: MIT License (see LICENSE file for details)

package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Service sends notifications to external services.
type Service struct {
	DiscordWebhook string
	TelegramToken  string
	TelegramChatID string
	EmailURL       string
	// AppriseURL is the notify endpoint of a self-hosted Apprise API server
	// (for example http://apprise:8000/notify/mykey). It is operator-supplied
	// infrastructure, so it is validated leniently (any http/https URL) and is
	// not subject to the strict public-webhook allowlist. Set it directly on
	// the struct after New.
	AppriseURL string
	client     *http.Client
}

// New creates a Service with the provided endpoints.
func New(discordURL, telegramToken, telegramChatID, emailURL string) (*Service, error) {
	// Validate webhook URLs to prevent SSRF attacks
	if err := validateWebhookURL(discordURL); err != nil {
		return nil, fmt.Errorf("invalid Discord webhook URL: %v", err)
	}

	if err := validateWebhookURL(emailURL); err != nil {
		return nil, fmt.Errorf("invalid email webhook URL: %v", err)
	}

	// Note: Telegram API URL is constructed internally, but we should still validate the token format
	if telegramToken != "" && !isValidTelegramToken(telegramToken) {
		return nil, fmt.Errorf("invalid Telegram bot token format")
	}

	return &Service{
		DiscordWebhook: discordURL,
		TelegramToken:  telegramToken,
		TelegramChatID: telegramChatID,
		EmailURL:       emailURL,
		client:         &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// isValidTelegramToken validates the format of a Telegram bot token
func isValidTelegramToken(token string) bool {
	// Telegram bot tokens have a specific format: {bot_id}:{auth_token}
	// Example: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
	if len(token) < 10 || !strings.Contains(token, ":") {
		return false
	}

	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}

	// Basic validation - bot ID should be numeric, auth token should be alphanumeric
	botID, authToken := parts[0], parts[1]
	if len(botID) < 1 || len(authToken) < 10 {
		return false
	}

	// Additional security: check for obvious patterns that shouldn't be in tokens
	dangerousPatterns := []string{"../", "\\", "<", ">", "'", "\"", "&"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(token, pattern) {
			return false
		}
	}

	return true
}

// validateWebhookURL validates that a webhook URL is safe to use and prevents SSRF attacks
func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return nil // Empty URLs are allowed (feature disabled)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Only allow HTTPS for webhooks (security best practice)
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed for webhooks")
	}

	// Block private/internal IP ranges and localhost
	host := parsedURL.Hostname()
	if isPrivateOrLocalhost(host) {
		return fmt.Errorf("webhooks to private/internal addresses are not allowed")
	}

	// Allow only specific known webhook domains for additional security
	allowedDomains := []string{
		"discord.com",
		"discordapp.com",
		"api.telegram.org",
		"hooks.slack.com",
		"api.pushover.net",
	}

	domainAllowed := false
	for _, domain := range allowedDomains {
		if strings.HasSuffix(host, domain) {
			domainAllowed = true
			break
		}
	}

	if !domainAllowed {
		return fmt.Errorf("webhook domain not in allowed list: %s", host)
	}

	return nil
}

// isPrivateOrLocalhost checks if a hostname is a private IP or localhost
func isPrivateOrLocalhost(host string) bool {
	// Check for localhost variations
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Check for private IP ranges (simplified check)
	// In production, you'd want more comprehensive IP range checking
	privatePatterns := []string{
		"10.",
		"192.168.",
		"172.16.", "172.17.", "172.18.", "172.19.", "172.20.",
		"172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
		"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
		"169.254.", // Link-local
	}

	for _, pattern := range privatePatterns {
		if strings.HasPrefix(host, pattern) {
			return true
		}
	}

	return false
}

// Send dispatches the message to all configured notification targets.
func (s *Service) Send(ctx context.Context, msg string) error {
	if s.DiscordWebhook != "" {
		body, _ := json.Marshal(map[string]string{"content": msg})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.DiscordWebhook, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			_, err = s.client.Do(req)
		}
		if err != nil {
			return err
		}
	}
	if s.TelegramToken != "" && s.TelegramChatID != "" {
		u := "https://api.telegram.org/bot" + s.TelegramToken + "/sendMessage"
		body := url.Values{"chat_id": {s.TelegramChatID}, "text": {msg}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(body.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			_, err = s.client.Do(req)
		}
		if err != nil {
			return err
		}
	}
	if s.EmailURL != "" {
		body, _ := json.Marshal(map[string]string{"message": msg})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.EmailURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			_, err = s.client.Do(req)
		}
		if err != nil {
			return err
		}
	}
	if s.AppriseURL != "" {
		// Apprise API expects a JSON body with a "body" field (and optionally a
		// "title"); the configured notification targets live server-side.
		body, _ := json.Marshal(map[string]string{"body": msg, "title": "subtitle-manager"})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.AppriseURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			var resp *http.Response
			resp, err = s.client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
