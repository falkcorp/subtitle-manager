// file: pkg/security/languagecode_test.go
// version: 1.0.0
// guid: 8c05e731-4a92-4be6-90d1-27f3ba6c48e0
// last-edited: 2026-08-12

package security

import "testing"

// TestValidateLanguageCodeAcceptsRegionalTags covers hyphenated codes.
//
// The validator was alphanumeric-only, so it rejected every BCP-47 regional
// tag — pt-BR, zh-Hans, sr-Latn — even though those are ordinary subtitle
// languages, and rejected the en-es form used to name a bilingual file.
//
// A hyphen is safe here: it is not a path separator, cannot form "..", and the
// surrounding path construction still validates the finished path.
func TestValidateLanguageCodeAcceptsRegionalTags(t *testing.T) {
	for _, lang := range []string{"en", "es", "eo", "pt-BR", "zh-Hans", "sr-Latn", "en-es"} {
		if err := ValidateLanguageCode(lang); err != nil {
			t.Errorf("ValidateLanguageCode(%q) = %v, want nil", lang, err)
		}
	}
}

// TestValidateLanguageCodeRejectsUnsafe keeps the guarantees that matter: the
// value is interpolated into a filename, so nothing may introduce a path
// separator, a traversal, or a hidden extension. A hyphen is allowed only
// between characters, never at an edge, so a code can never begin a filename
// component with "-" or end one with a dangling separator.
func TestValidateLanguageCodeRejectsUnsafe(t *testing.T) {
	for _, tc := range []struct {
		lang, why string
	}{
		{"", "empty"},
		{"en/es", "path separator"},
		{"..", "traversal"},
		{"../en", "traversal with separator"},
		{"en.srt", "dot could forge an extension"},
		{"-en", "leading hyphen"},
		{"en-", "trailing hyphen"},
		{"-", "hyphen alone"},
		{"en es", "space"},
		{"en\\es", "backslash separator"},
		{"en\x00es", "null byte"},
		{"abcdefghijk", "too long"},
	} {
		if err := ValidateLanguageCode(tc.lang); err == nil {
			t.Errorf("ValidateLanguageCode(%q) = nil, want an error (%s)", tc.lang, tc.why)
		}
	}
}
