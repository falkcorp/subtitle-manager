// file: pkg/webserver/translate.go
// version: 1.0.0
// guid: f0523cff-b15b-4527-aa90-0b65326f73f9

package webserver

import (
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/metrics"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

// translateHandler handles translating an uploaded subtitle file.
//
// POST requests must use multipart/form-data containing a "file" part and
// a "lang" field specifying the target language. Optional fields "service"
// and "grpc" override the configured translation service and gRPC address.
// The translated SRT data is returned with content type "application/x-subrip".
func translateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			metrics.TranslationRequests.WithLabelValues("", "", "bad_request").Inc()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			metrics.TranslationRequests.WithLabelValues("", "", "bad_request").Inc()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		lang := r.FormValue("lang")
		if lang == "" {
			metrics.TranslationRequests.WithLabelValues("", "", "bad_request").Inc()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		service := r.FormValue("service")
		if service == "" {
			service = viper.GetString("translate_service")
			if service == "" {
				service = "google"
			}
		}
		grpcAddr := r.FormValue("grpc")
		if grpcAddr == "" {
			grpcAddr = viper.GetString("grpc_addr")
		}
		gKey := viper.GetString("google_api_key")
		gptKey := viper.GetString("openai_api_key")

		in, err := os.CreateTemp("", "in-*"+filepath.Ext(hdr.Filename))
		if err != nil {
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() { _ = in.Close() }()
		defer os.Remove(in.Name())
		if _, err := io.Copy(in, file); err != nil {
			_ = in.Close()
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		out, err := os.CreateTemp("", "out-*.srt")
		if err != nil {
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() { _ = out.Close() }()
		defer os.Remove(out.Name())
		if err := out.Close(); err != nil {
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := subtitles.TranslateFileToSRT(in.Name(), out.Name(), lang, service, gKey, gptKey, grpcAddr); err != nil {
			// Log it. The error was discarded here, so a failed translation
			// produced a bare 500 with nothing recorded anywhere — leaving no
			// way to tell a bad API key from an unreachable service from a
			// malformed subtitle.
			logging.GetLogger("webserver").Warnf("translate %s->%s via %s: %v",
				in.Name(), out.Name(), service, err)
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data, err := os.ReadFile(out.Name())
		if err != nil {
			metrics.TranslationRequests.WithLabelValues(service, lang, "error").Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		metrics.TranslationRequests.WithLabelValues(service, lang, "success").Inc()
		w.Header().Set("Content-Type", "application/x-subrip")
		_, _ = w.Write(data)
	})
}
