// file: pkg/security/security.go
// version: 1.4.0
// guid: efe90a08-389d-4157-a46e-8a57bfc1181a
// last-edited: 2026-08-04
package security

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// allowTempDirPaths controls the temp-directory escape hatch in
// ValidateAndSanitizePath. It is true only in a test binary.
//
// It is a variable rather than a direct testing.Testing() call at the use site
// so the production behaviour is observable: a test can set it to false and
// assert that a temp path is refused. Without that, the closed case would be
// untestable by construction — every test runs with the hatch open.
var allowTempDirPaths = testing.Testing()

// ValidateURL checks if raw is a safe HTTP or HTTPS URL.
// It blocks known metadata services and dangerous ports.
// The sanitized URL string is returned when valid.
func ValidateURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("only HTTP and HTTPS schemes are allowed, got: %s", u.Scheme)
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}

	hostname := strings.ToLower(u.Hostname())
	blockedHosts := []string{
		"169.254.169.254",
		"metadata.google.internal",
		"metadata",
	}
	for _, blocked := range blockedHosts {
		if hostname == blocked {
			return "", fmt.Errorf("hostname %s is not allowed", hostname)
		}
	}

	if u.Port() != "" {
		blockedPorts := []string{"22", "23", "3389", "5900", "5901"}
		port := u.Port()
		for _, blocked := range blockedPorts {
			if port == blocked {
				return "", fmt.Errorf("port %s is not allowed", port)
			}
		}
	}

	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}

	return u.String(), nil
}

// GetAllowedBaseDirs returns directories considered safe for file browsing.
func GetAllowedBaseDirs() []string {
	var dirs []string
	if mediaDir := viper.GetString("media_directory"); mediaDir != "" {
		dirs = append(dirs, mediaDir)
	}
	if subDir := viper.GetString("subtitle_directory"); subDir != "" {
		dirs = append(dirs, subDir)
	}

	if info, err := os.Stat("/media"); err == nil && info.IsDir() {
		if entries, err := os.ReadDir("/media"); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					dirs = append(dirs, filepath.Join("/media", entry.Name()))
				}
			}
		}
	}

	commonDirs := []string{
		"/mnt/media", "/home/media", "/var/media",
		"/Movies", "/TV", "/Videos",
	}
	if runtime.GOOS == "windows" {
		commonDirs = []string{
			"C:\\Media", "C:\\Movies", "C:\\TV", "D:\\Media", "D:\\Movies", "D:\\TV",
		}
	}
	for _, dir := range commonDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, home)
	}
	return dirs
}

// ValidateAndSanitizePath cleans userPath and ensures it resides in an allowed directory.
// An absolute, sanitized path is returned on success.
func ValidateAndSanitizePath(userPath string) (string, error) {
	cleanPath := filepath.Clean(userPath)
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("path must be absolute")
	}
	absPath := filepath.Clean(cleanPath)
	allowedBaseDirs := GetAllowedBaseDirs()

	if cleanPath == "/" || cleanPath == "." ||
		(runtime.GOOS == "windows" && len(filepath.VolumeName(cleanPath)) == 2 &&
			strings.Trim(filepath.Clean(cleanPath), "\\") == filepath.VolumeName(cleanPath)) {
		for _, baseDir := range allowedBaseDirs {
			if absPath == baseDir || strings.HasPrefix(baseDir, absPath) {
				return absPath, nil
			}
		}
	}

	for _, baseDir := range allowedBaseDirs {
		relPath, err := filepath.Rel(baseDir, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			if strings.Contains(relPath, "..") {
				return "", fmt.Errorf("path traversal detected: %s", cleanPath)
			}
			return absPath, nil
		}
	}

	// Temp-directory escape hatch, for tests only.
	//
	// This used to be unconditional, which quietly voided `allowed_base_dirs` in
	// production: /tmp is world-writable, so any path an attacker could steer
	// under it was accepted no matter how the operator had configured the
	// allowed roots. Nothing gated it — no build tag, no config key.
	//
	// It exists because the suite builds fixtures under t.TempDir() and 68 test
	// files depend on those paths validating. testing.Testing() reports whether
	// this is a test binary, so the hatch is open exactly where it is needed and
	// closed everywhere else, without every one of those tests having to
	// configure an allowed root. A production binary never takes this branch.
	//
	// The traversal check below is kept even so: a test fixture path containing
	// ".." should still be rejected rather than normalised into acceptance.
	if tempDir := os.TempDir(); allowTempDirPaths && tempDir != "" {
		relPath, err := filepath.Rel(tempDir, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			if strings.Contains(relPath, "..") {
				return "", fmt.Errorf("path traversal detected: %s", cleanPath)
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("path not in allowed directories: %s", cleanPath)
}

// SanitizePath validates and sanitizes a path while preserving the input type.
// This is a small generic wrapper around ValidateAndSanitizePath so that code
// working with custom string types can easily sanitize path values without
// additional conversions.
func SanitizePath[T ~string](p T) (T, error) {
	sanitized, err := ValidateAndSanitizePath(string(p))
	if err != nil {
		return p, err
	}
	return T(sanitized), nil
}

// SanitizeRelativePath cleans a relative path, preventing path traversal.
func SanitizeRelativePath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("path must be relative")
	}
	if strings.HasPrefix(p, "\\") || strings.HasPrefix(p, "/") || strings.Contains(p, ":") {
		return "", fmt.Errorf("invalid path")
	}
	clean := filepath.Clean(p)
	if strings.Contains(clean, "..") || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("path traversal detected: %s", p)
	}
	return filepath.ToSlash(clean), nil
}

// ValidateWebSocketOrigin validates the Origin header for WebSocket connections
// to prevent cross-site WebSocket hijacking attacks. It allows connections from:
// - Same origin (localhost with various ports for development)
// - Configured allowed origins from environment/config
// - Local network origins for development (127.0.0.1, localhost)
func ValidateWebSocketOrigin(origin, host string) bool {
	if origin == "" {
		// Allow empty origin for same-origin requests (some browsers)
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	hostURL, err := url.Parse("http://" + host)
	if err != nil {
		return false
	}

	// Allow same origin
	if originURL.Host == hostURL.Host {
		return true
	}

	// Allow localhost variations for development
	allowedHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
	}

	originHost := strings.Split(originURL.Host, ":")[0]
	for _, allowed := range allowedHosts {
		if originHost == allowed {
			return true
		}
	}

	// Check configured allowed origins from environment
	if allowedOrigins := viper.GetString("allowed_websocket_origins"); allowedOrigins != "" {
		for _, allowed := range strings.Split(allowedOrigins, ",") {
			allowed = strings.TrimSpace(allowed)
			if origin == allowed || originURL.Host == allowed {
				return true
			}
		}
	}

	return false
}

// ValidateLanguageCode validates that a language code contains only safe characters
// to prevent path traversal attacks. Only alphanumeric characters are allowed.
func ValidateLanguageCode(lang string) error {
	if lang == "" {
		return fmt.Errorf("language code cannot be empty")
	}

	// Maximum reasonable length for language codes (ISO 639 codes are typically 2-3 chars)
	if len(lang) > 10 {
		return fmt.Errorf("language code too long: %s", lang)
	}

	// Only allow alphanumeric characters
	for _, r := range lang {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("invalid character in language code: %c", r)
		}
	}

	return nil
}

// ValidateProviderName validates that a provider name contains only safe characters
func ValidateProviderName(provider string) error {
	if provider == "" {
		return nil // Empty provider is allowed
	}

	if len(provider) > 50 {
		return fmt.Errorf("provider name too long: %s", provider)
	}

	// Allow alphanumeric characters, underscores, and hyphens
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(provider) {
		return fmt.Errorf("invalid provider name: %s", provider)
	}

	return nil
}

// ValidateSubtitleOutputPath validates and constructs a safe subtitle output
// path of the form "<base>.<lang>.srt".
func ValidateSubtitleOutputPath(videoPath, lang string) (string, error) {
	return ValidateSubtitleOutputPathWithOptions(videoPath, lang, false)
}

// ValidateSubtitleOutputPathWithOptions validates and constructs a safe subtitle
// output path. When singleLanguage is true the language code is omitted from the
// filename ("<base>.srt" instead of "<base>.<lang>.srt"), matching Bazarr's
// "single language" naming option; lang is still validated so it can be recorded
// in the download history. A single-language layout can only hold one subtitle
// language per media file.
func ValidateSubtitleOutputPathWithOptions(videoPath, lang string, singleLanguage bool) (string, error) {
	return ValidateSubtitleOutputPathWithFormat(videoPath, lang, singleLanguage, "srt")
}

// ValidateSubtitleOutputPathWithFormat is ValidateSubtitleOutputPathWithOptions
// with a caller-chosen container extension.
//
// ext reaches a file name joined onto a media directory, so it is checked
// against an allowlist rather than sanitised: only the containers the
// application can actually write are accepted, and anything else is an error.
// Sanitising a caller-supplied path fragment invites the one case the filter
// missed; refusing everything off a short list does not.
func ValidateSubtitleOutputPathWithFormat(videoPath, lang string, singleLanguage bool, ext string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "srt", "vtt", "ass":
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	case "":
		ext = "srt"
	default:
		return "", fmt.Errorf("unsupported subtitle format: %s", ext)
	}
	return validateSubtitleOutputPath(videoPath, lang, singleLanguage, ext)
}

// validateSubtitleOutputPath builds and validates the output path. ext is
// already known-good.
func validateSubtitleOutputPath(videoPath, lang string, singleLanguage bool, ext string) (string, error) {
	// Validate inputs
	sanitizedVideoPath, err := ValidateAndSanitizePath(videoPath)
	if err != nil {
		return "", fmt.Errorf("invalid video path: %w", err)
	}

	if err := ValidateLanguageCode(lang); err != nil {
		return "", fmt.Errorf("invalid language code: %w", err)
	}

	// Construct subtitle path
	base := strings.TrimSuffix(filepath.Base(sanitizedVideoPath), filepath.Ext(sanitizedVideoPath))
	dir := filepath.Dir(sanitizedVideoPath)

	// Clean the base filename to prevent injection
	base = filepath.Clean(base)
	if strings.Contains(base, "..") {
		return "", fmt.Errorf("invalid characters in video filename")
	}

	name := base + "." + lang + "." + ext
	if singleLanguage {
		name = base + "." + ext
	}
	subtitlePath := filepath.Join(dir, name)
	subtitlePath = filepath.Clean(subtitlePath)

	// Ensure the output path is still within allowed directories
	if _, err := ValidateAndSanitizePath(subtitlePath); err != nil {
		return "", fmt.Errorf("invalid subtitle output path: %w", err)
	}

	return subtitlePath, nil
}

var labelSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// SanitizeLabel removes potentially dangerous characters from a short
// identifier. Only letters, numbers, underscores and hyphens are retained.
func SanitizeLabel(label string) string {
	return labelSanitizer.ReplaceAllString(label, "")
}
