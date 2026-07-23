// file: pkg/scanner/whisper_fallback.go
// version: 1.0.0
// guid: 5e1c8a37-0b46-4d92-9f28-3a7d6c40e159
// last-edited: 2026-07-23

package scanner

import (
	"context"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/transcriber"
)

// whisperFallback transcribes mediaPath with Whisper when no subtitle provider
// produced one, matching Bazarr's "Whisper fallback". It returns (data, true)
// on success or (nil, false) to leave the original fetch failure in place. It is
// active only when whisper.fallback_enabled is set and a Whisper backend is
// configured (a self-hosted whisper.transcribe_url, or an OpenAI key).
func whisperFallback(ctx context.Context, mediaPath, lang string) ([]byte, bool) {
	if !viper.GetBool("whisper.fallback_enabled") {
		return nil, false
	}
	apiKey := viper.GetString("openai_api_key")
	if viper.GetString("whisper.transcribe_url") == "" && apiKey == "" {
		return nil, false
	}
	// WhisperTranscribe prefers whisper.transcribe_url (native /asr, no key) and
	// otherwise uses the OpenAI-compatible API with apiKey.
	data, err := transcriber.WhisperTranscribe(mediaPath, lang, apiKey)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
