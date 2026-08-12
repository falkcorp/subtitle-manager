// file: pkg/webserver/stacksubs.go
// version: 1.1.0
// guid: 9e51cb27-6d04-4a83-bf15-70c9e2a4d386
// last-edited: 2026-08-11

package webserver

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

// stackRequest asks for two subtitles that already exist in the library to be
// combined into one bilingual file.
//
// There is deliberately no caller-supplied output path. The destination is
// always derived from the validated primary — same directory, same base name,
// sentinel language tag, .srt — so this endpoint cannot be used to choose where
// a file gets written. It is the only write here, and narrowing it to one
// deterministic name is worth more than the flexibility.
type stackRequest struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

// stackResponse reports where the bilingual file was written.
type stackResponse struct {
	Output string `json:"output"`
}

// defaultSentinelLang is the language suffix given to a generated bilingual
// file. Esperanto is used because no real track claims it, so Plex/Jellyfin/Emby
// list the double-subs file as a distinct track instead of colliding with the
// primary language. This matches the default `dualsub` uses.
const defaultSentinelLang = "eo"

// stackSubtitlesHandler combines two existing subtitle files into a single
// bilingual "double subs" file.
//
// It takes paths rather than uploads, unlike /api/dualsub, because the files
// are already in the library — the UI passes the sidecars the browse endpoint
// reported for a media item. It also does not translate: both languages must
// already exist, so no translation service or credentials are involved.
func stackSubtitlesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req stackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Primary) == "" || strings.TrimSpace(req.Secondary) == "" {
			http.Error(w, "primary and secondary are both required", http.StatusBadRequest)
			return
		}

		// Validate before opening anything, the same as every other
		// path-taking endpoint here.
		primary, err := security.ValidateAndSanitizePath(req.Primary)
		if err != nil {
			http.Error(w, "invalid primary path", http.StatusBadRequest)
			return
		}
		secondary, err := security.ValidateAndSanitizePath(req.Secondary)
		if err != nil {
			http.Error(w, "invalid secondary path", http.StatusBadRequest)
			return
		}

		// Derived from the already-validated primary, never from the request,
		// then re-validated so the constructed name is confined to the allowed
		// base directories too.
		out, err := security.ValidateAndSanitizePath(sentinelOutputPath(primary))
		if err != nil {
			http.Error(w, "invalid output path", http.StatusBadRequest)
			return
		}
		// The write must land beside the subtitle it was built from. Any other
		// directory means the derivation went wrong, and this is the one call
		// here that creates a file.
		if filepath.Dir(out) != filepath.Dir(primary) {
			http.Error(w, "invalid output path", http.StatusBadRequest)
			return
		}

		primarySub, err := astisub.OpenFile(primary)
		if err != nil {
			http.Error(w, "cannot read primary subtitle", http.StatusBadRequest)
			return
		}
		secondarySub, err := astisub.OpenFile(secondary)
		if err != nil {
			http.Error(w, "cannot read secondary subtitle", http.StatusBadRequest)
			return
		}

		primarySub.Items = subtitles.StackTracks(primarySub.Items, secondarySub.Items)

		// Write through an os.Root rooted at the primary's directory. Root
		// confines every operation beneath it at the OS level — a traversal in
		// the name is refused by the kernel-facing API rather than by our own
		// string checks — so the one call here that creates a file cannot
		// escape the directory the primary already validated into.
		root, err := os.OpenRoot(filepath.Dir(primary))
		if err != nil {
			http.Error(w, "cannot write output", http.StatusInternalServerError)
			return
		}
		defer root.Close()

		f, err := root.Create(filepath.Base(out))
		if err != nil {
			http.Error(w, "cannot write output", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		if err := primarySub.WriteToSRT(f); err != nil {
			http.Error(w, "cannot write output", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stackResponse{Output: out})
	})
}

// sentinelOutputPath derives the bilingual filename from the primary subtitle,
// replacing a trailing language tag so "Episode.en.srt" becomes
// "Episode.eo.srt" rather than "Episode.en.eo.srt".
func sentinelOutputPath(primary string) string {
	lang := viper.GetString("dualsub.sentinel_language")
	if lang == "" {
		lang = defaultSentinelLang
	}
	ext := filepath.Ext(primary) // .srt
	base := strings.TrimSuffix(primary, ext)
	// Drop an existing two/three letter language tag if present.
	if dot := strings.LastIndex(base, "."); dot != -1 {
		if tag := base[dot+1:]; len(tag) >= 2 && len(tag) <= 3 {
			base = base[:dot]
		}
	}
	return base + "." + lang + ext
}
