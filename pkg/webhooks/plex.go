// file: pkg/webhooks/plex.go
// version: 1.0.0
// guid: d6b1f39c-4e07-4a52-9c8b-2f0a7d54e916
// last-edited: 2026-07-23

package webhooks

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// plexPayload is the subset of a Plex webhook payload we care about. Plex sends
// a JSON document (as the "payload" field of a multipart/form-data request)
// describing library and playback events.
type plexPayload struct {
	Event    string `json:"event"`
	Metadata struct {
		LibrarySectionType string `json:"librarySectionType"`
		Title              string `json:"title"`
		Media              []struct {
			Part []struct {
				File string `json:"file"`
			} `json:"Part"`
		} `json:"Media"`
	} `json:"Metadata"`
}

// firstFile returns the first media part file path in the payload, or "".
func (p *plexPayload) firstFile() string {
	for _, m := range p.Metadata.Media {
		for _, part := range m.Part {
			if part.File != "" {
				return part.File
			}
		}
	}
	return ""
}

// PlexHandler accepts Plex webhook events and, on "library.new" (new media
// added to a library), fetches subtitles for the newly-added file. Plex sends
// the event as the "payload" field of a multipart/form-data POST; a raw JSON
// body is also accepted for flexibility. Events other than library.new are
// acknowledged and ignored.
func PlexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		logger := logging.GetLogger("plex-webhook")

		raw := plexRawPayload(r)
		if raw == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var p plexPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			logger.Warnf("failed to parse Plex payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Only new library items trigger a subtitle fetch.
		if p.Event != "library.new" {
			logger.Debugf("ignoring Plex event: %s", p.Event)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		file := p.firstFile()
		if file == "" {
			logger.Warnf("no media file in Plex library.new for %q", p.Metadata.Title)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		lang := viper.GetString("webhooks.plex.language")
		if lang == "" {
			lang = "en"
		}

		handle(w, r, event{Path: file, Lang: lang})
	})
}

// plexRawPayload extracts the Plex payload JSON from either a multipart form
// ("payload" field, how Plex actually posts) or a raw JSON request body.
func plexRawPayload(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		// 1 MiB is plenty for a Plex payload; it references a file, not content.
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			return ""
		}
		return r.FormValue("payload")
	}
	// Fall back to a raw JSON body.
	dec := json.NewDecoder(r.Body)
	var buf json.RawMessage
	if err := dec.Decode(&buf); err != nil {
		return ""
	}
	return string(buf)
}
