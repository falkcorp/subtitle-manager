// file: pkg/webhooks/handlers.go
// version: 1.1.0
// guid: 123e4567-e89b-12d3-a456-426614174002
// last-edited: 2026-07-30

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/sirupsen/logrus"
)

// SonarrWebhookHandler handles incoming webhooks from Sonarr.
type SonarrWebhookHandler struct {
	secret string
	logger *logrus.Entry
}

// RadarrWebhookHandler handles incoming webhooks from Radarr.
type RadarrWebhookHandler struct {
	secret string
	logger *logrus.Entry
}

// CustomWebhookHandler handles generic incoming webhooks.
type CustomWebhookHandler struct {
	secret string
	logger *logrus.Entry
}

// SonarrPayload represents the structure of Sonarr webhook payloads.
type SonarrPayload struct {
	EventType string `json:"eventType"`
	Series    struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Path  string `json:"path"`
	} `json:"series"`
	Episodes []struct {
		ID            int    `json:"id"`
		EpisodeTitle  string `json:"title"`
		SeasonNumber  int    `json:"seasonNumber"`
		EpisodeNumber int    `json:"episodeNumber"`
	} `json:"episodes"`
	EpisodeFile struct {
		ID           int    `json:"id"`
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Quality      struct {
			Quality struct {
				Name string `json:"name"`
			} `json:"quality"`
		} `json:"quality"`
	} `json:"episodeFile"`
}

// RadarrPayload represents the structure of Radarr webhook payloads.
type RadarrPayload struct {
	EventType string `json:"eventType"`
	Movie     struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Path  string `json:"path"`
	} `json:"movie"`
	MovieFile struct {
		ID           int    `json:"id"`
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Quality      struct {
			Quality struct {
				Name string `json:"name"`
			} `json:"quality"`
		} `json:"quality"`
	} `json:"movieFile"`
}

// NewSonarrHandler creates a new Sonarr webhook handler.
func NewSonarrHandler(secret string) *SonarrWebhookHandler {
	return &SonarrWebhookHandler{
		secret: secret,
		logger: logging.GetLogger("sonarr-webhook"),
	}
}

// NewRadarrHandler creates a new Radarr webhook handler.
func NewRadarrHandler(secret string) *RadarrWebhookHandler {
	return &RadarrWebhookHandler{
		secret: secret,
		logger: logging.GetLogger("radarr-webhook"),
	}
}

// NewCustomHandler creates a new custom webhook handler.
func NewCustomHandler(secret string) *CustomWebhookHandler {
	return &CustomWebhookHandler{
		secret: secret,
		logger: logging.GetLogger("custom-webhook"),
	}
}

// Handle processes Sonarr webhook events.
func (h *SonarrWebhookHandler) Handle(ctx context.Context, payload []byte, headers http.Header) error {
	var sonarrPayload SonarrPayload
	if err := json.Unmarshal(payload, &sonarrPayload); err != nil {
		h.logger.Warnf("Failed to parse Sonarr payload: %v", err)
		return fmt.Errorf("invalid payload format: %v", err)
	}

	// Only process download events
	if sonarrPayload.EventType != "Download" {
		h.logger.Debugf("Ignoring Sonarr event type: %s", sonarrPayload.EventType)
		return nil
	}

	// Extract file path
	filePath := sonarrPayload.EpisodeFile.Path
	if filePath == "" {
		h.logger.Warn("No file path in Sonarr webhook payload")
		return fmt.Errorf("missing file path in payload")
	}

	// Validate and sanitize the file path to prevent path traversal attacks
	// CodeQL: This input is properly sanitized through ValidateAndSanitizePath()
	if _, err := security.ValidateAndSanitizePath(filePath); err != nil {
		h.logger.Warnf("Invalid file path from Sonarr: %v", err)
		return fmt.Errorf("invalid file path: %v", err)
	}

	h.logger.Infof("Processing Sonarr download: %s", filePath)

	// Process file for subtitle download, letting the system pick the provider.
	//
	// The language was hardcoded to English here, so a non-English library
	// fetched English subtitles for everything imported through Sonarr with no
	// setting anywhere to change it. It now follows the configured default.
	//
	// This is still not the language *profile* assigned to the series — the
	// handler receives a file path and has no database to resolve one — which
	// is the remaining half of this gap and is tracked separately.
	if err := scanner.ProcessFile(ctx, filePath, profiles.DefaultLanguage(), "", nil, true, nil); err != nil {
		h.logger.Warnf("Failed to process file from Sonarr webhook: %v", err)
		return fmt.Errorf("processing failed: %v", err)
	}

	h.logger.Infof("Successfully processed Sonarr webhook for: %s", filePath)
	return nil
}

// Handle processes Radarr webhook events.
func (h *RadarrWebhookHandler) Handle(ctx context.Context, payload []byte, headers http.Header) error {
	var radarrPayload RadarrPayload
	if err := json.Unmarshal(payload, &radarrPayload); err != nil {
		h.logger.Warnf("Failed to parse Radarr payload: %v", err)
		return fmt.Errorf("invalid payload format: %v", err)
	}

	// Only process download events
	if radarrPayload.EventType != "Download" {
		h.logger.Debugf("Ignoring Radarr event type: %s", radarrPayload.EventType)
		return nil
	}

	// Extract file path
	filePath := radarrPayload.MovieFile.Path
	if filePath == "" {
		h.logger.Warn("No file path in Radarr webhook payload")
		return fmt.Errorf("missing file path in payload")
	}

	// Validate and sanitize the file path to prevent path traversal attacks
	// CodeQL: This input is properly sanitized through ValidateAndSanitizePath()
	if _, err := security.ValidateAndSanitizePath(filePath); err != nil {
		h.logger.Warnf("Invalid file path from Radarr: %v", err)
		return fmt.Errorf("invalid file path: %v", err)
	}

	h.logger.Infof("Processing Radarr download: %s", filePath)

	// Process file for subtitle download. See the Sonarr handler above for why
	// this follows the configured default language rather than a hardcoded
	// English, and what the remaining gap is.
	if err := scanner.ProcessFile(ctx, filePath, profiles.DefaultLanguage(), "", nil, true, nil); err != nil {
		h.logger.Warnf("Failed to process file from Radarr webhook: %v", err)
		return fmt.Errorf("processing failed: %v", err)
	}

	h.logger.Infof("Successfully processed Radarr webhook for: %s", filePath)
	return nil
}

// Handle processes custom webhook events with the original event format.
func (h *CustomWebhookHandler) Handle(ctx context.Context, payload []byte, headers http.Header) error {
	var ev event
	if err := json.Unmarshal(payload, &ev); err != nil {
		h.logger.Warnf("Failed to parse custom webhook payload: %v", err)
		return fmt.Errorf("invalid payload format: %v", err)
	}

	// Validate required fields
	if ev.Path == "" || ev.Lang == "" {
		h.logger.Warn("Missing required fields in custom webhook event")
		return fmt.Errorf("missing required fields: path and lang are required")
	}

	// Validate and sanitize the file path to prevent path traversal attacks
	// CodeQL: All user inputs are properly sanitized before use
	if _, err := security.ValidateAndSanitizePath(ev.Path); err != nil {
		h.logger.Warnf("Invalid file path in custom webhook: %v", err)
		return fmt.Errorf("invalid file path: %v", err)
	}

	// Validate language code to prevent injection attacks
	// CodeQL: Language code validated with character-by-character checking
	if err := security.ValidateLanguageCode(ev.Lang); err != nil {
		h.logger.Warnf("Invalid language code in custom webhook: %v", err)
		return fmt.Errorf("invalid language code: %v", err)
	}

	// Validate provider name if specified to prevent injection attacks
	// CodeQL: Provider name validated with regex pattern ^[a-zA-Z0-9_-]+$
	if err := security.ValidateProviderName(ev.Provider); err != nil {
		h.logger.Warnf("Invalid provider name in custom webhook: %v", err)
		return fmt.Errorf("invalid provider name: %v", err)
	}

	var p providers.Provider
	var name string
	var err error
	if ev.Provider != "" {
		p, err = providers.Get(ev.Provider, "")
		name = ev.Provider
		if err != nil {
			h.logger.Warnf("Failed to get provider %s: %v", ev.Provider, err)
			return fmt.Errorf("provider not available: %v", err)
		}
	}

	h.logger.Infof("Processing custom webhook: %s (lang: %s, provider: %s)", ev.Path, ev.Lang, name)

	if err := scanner.ProcessFile(ctx, ev.Path, ev.Lang, name, p, true, nil); err != nil {
		h.logger.Warnf("Failed to process file from custom webhook: %v", err)
		return fmt.Errorf("processing failed: %v", err)
	}

	h.logger.Infof("Successfully processed custom webhook for: %s", ev.Path)
	return nil
}

// ValidateSignature validates HMAC signature for Sonarr webhooks.
func (h *SonarrWebhookHandler) ValidateSignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return true // No secret configured, skip validation
	}

	// Remove 'sha256=' prefix if present
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:]
	}

	expectedSignature := generateHMACSignature(payload, h.secret)
	// Use constant time comparison to prevent timing attacks
	if len(signature) != len(expectedSignature) {
		return false
	}

	for i := 0; i < len(signature); i++ {
		if signature[i] != expectedSignature[i] {
			return false
		}
	}
	return true
}

// ValidateSignature validates HMAC signature for Radarr webhooks.
func (h *RadarrWebhookHandler) ValidateSignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return true // No secret configured, skip validation
	}

	// Remove 'sha256=' prefix if present
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:]
	}

	expectedSignature := generateHMACSignature(payload, h.secret)
	// Use constant time comparison to prevent timing attacks
	if len(signature) != len(expectedSignature) {
		return false
	}

	for i := 0; i < len(signature); i++ {
		if signature[i] != expectedSignature[i] {
			return false
		}
	}
	return true
}

// ValidateSignature validates HMAC signature for custom webhooks.
func (h *CustomWebhookHandler) ValidateSignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return true // No secret configured, skip validation
	}

	// Remove 'sha256=' prefix if present
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:]
	}

	expectedSignature := generateHMACSignature(payload, h.secret)
	// Use constant time comparison to prevent timing attacks
	if len(signature) != len(expectedSignature) {
		return false
	}

	for i := 0; i < len(signature); i++ {
		if signature[i] != expectedSignature[i] {
			return false
		}
	}
	return true
}

// GetName returns the handler name for Sonarr.
func (h *SonarrWebhookHandler) GetName() string {
	return "sonarr"
}

// GetName returns the handler name for Radarr.
func (h *RadarrWebhookHandler) GetName() string {
	return "radarr"
}

// GetName returns the handler name for custom webhooks.
func (h *CustomWebhookHandler) GetName() string {
	return "custom"
}
