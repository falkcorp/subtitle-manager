// file: pkg/gcommon/subtitle_format_test.go
// version: 1.0.1
// guid: 1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6e

package gcommon

import (
	"testing"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"github.com/stretchr/testify/assert"
)

// TestNewSubtitleFormatHelper tests the constructor
func TestNewSubtitleFormatHelper(t *testing.T) {
	helper := NewSubtitleFormatHelper()
	assert.NotNil(t, helper)
}

// TestFormatFromExtension tests format detection from file extensions
func TestFormatFromExtension(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	tests := []struct {
		name     string
		filename string
		expected common.SubtitleFormat
	}{
		// Standard cases
		{
			name:     "srt extension",
			filename: "movie.srt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},
		{
			name:     "vtt extension",
			filename: "movie.vtt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		},
		{
			name:     "webvtt extension",
			filename: "movie.webvtt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		},
		{
			name:     "ass extension",
			filename: "movie.ass",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
		},
		{
			name:     "ssa extension",
			filename: "movie.ssa",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
		},
		{
			name:     "ttml extension",
			filename: "movie.ttml",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
		},
		{
			name:     "xml extension",
			filename: "movie.xml",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
		},
		{
			name:     "scc extension",
			filename: "movie.scc",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
		},
		{
			name:     "sbv extension",
			filename: "movie.sbv",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
		},

		// Case sensitivity tests
		{
			name:     "uppercase SRT",
			filename: "movie.SRT",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},
		{
			name:     "mixed case VTT",
			filename: "movie.Vtt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		},
		{
			name:     "uppercase WEBVTT",
			filename: "movie.WEBVTT",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		},

		// File path tests
		{
			name:     "full path with srt",
			filename: "/path/to/movie.srt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},
		{
			name:     "complex filename",
			filename: "movie.2023.1080p.x264.srt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},

		// Just extension (with dot)
		{
			name:     "just .srt",
			filename: ".srt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},

		// Unknown/unsupported extensions
		{
			name:     "unknown extension",
			filename: "movie.txt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
		},
		{
			name:     "no extension",
			filename: "movie",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
		},
		{
			name:     "empty string",
			filename: "",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
		},
		{
			name:     "dot only",
			filename: ".",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
		},
		{
			name:     "multiple dots",
			filename: "movie.backup.srt",
			expected: common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.FormatFromExtension(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtensionFromFormat tests getting file extensions from formats
func TestExtensionFromFormat(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	tests := []struct {
		name     string
		format   common.SubtitleFormat
		expected string
	}{
		{
			name:     "SRT format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
			expected: "srt",
		},
		{
			name:     "VTT format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
			expected: "vtt",
		},
		{
			name:     "ASS format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
			expected: "ass",
		},
		{
			name:     "SSA format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
			expected: "ssa",
		},
		{
			name:     "TTML format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
			expected: "ttml",
		},
		{
			name:     "SCC format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
			expected: "scc",
		},
		{
			name:     "SBV format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
			expected: "sbv",
		},
		{
			name:     "unspecified format",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.ExtensionFromFormat(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsSupported tests format support checking
func TestIsSupported(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	tests := []struct {
		name     string
		format   common.SubtitleFormat
		expected bool
	}{
		{
			name:     "SRT is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
			expected: true,
		},
		{
			name:     "VTT is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
			expected: true,
		},
		{
			name:     "ASS is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
			expected: true,
		},
		{
			name:     "SSA is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
			expected: true,
		},
		{
			name:     "TTML is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
			expected: true,
		},
		{
			name:     "SCC is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
			expected: true,
		},
		{
			name:     "SBV is supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
			expected: true,
		},
		{
			name:     "unspecified is not supported",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.IsSupported(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetAllSupportedFormats tests getting all supported formats
func TestGetAllSupportedFormats(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	formats := helper.GetAllSupportedFormats()

	// Check that we have the expected number of formats
	assert.Len(t, formats, 7)

	// Check that all expected formats are present
	expectedFormats := []common.SubtitleFormat{
		common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
		common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
		common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
		common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
		common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
	}

	for _, expected := range expectedFormats {
		assert.Contains(t, formats, expected)
	}

	// Ensure UNSPECIFIED is not in the list
	assert.NotContains(t, formats, common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED)
}

// TestGetFormatDisplayName tests getting human-readable format names
func TestGetFormatDisplayName(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	tests := []struct {
		name     string
		format   common.SubtitleFormat
		expected string
	}{
		{
			name:     "SRT display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
			expected: "SubRip (SRT)",
		},
		{
			name:     "VTT display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
			expected: "WebVTT",
		},
		{
			name:     "ASS display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
			expected: "Advanced SubStation Alpha (ASS)",
		},
		{
			name:     "SSA display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
			expected: "SubStation Alpha (SSA)",
		},
		{
			name:     "TTML display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
			expected: "Timed Text Markup Language (TTML)",
		},
		{
			name:     "SCC display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
			expected: "Scenarist Closed Caption (SCC)",
		},
		{
			name:     "SBV display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
			expected: "YouTube Subtitle Format (SBV)",
		},
		{
			name:     "unspecified display name",
			format:   common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
			expected: "Unknown Format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetFormatDisplayName(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRoundTripConversion tests that format->extension->format conversion works
func TestRoundTripConversion(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	supportedFormats := helper.GetAllSupportedFormats()

	for _, originalFormat := range supportedFormats {
		t.Run(helper.GetFormatDisplayName(originalFormat), func(t *testing.T) {
			// Convert format to extension
			extension := helper.ExtensionFromFormat(originalFormat)
			assert.NotEmpty(t, extension)

			// Convert extension back to format
			filename := "test." + extension
			convertedFormat := helper.FormatFromExtension(filename)

			// Should match original format
			assert.Equal(t, originalFormat, convertedFormat)

			// Should be supported
			assert.True(t, helper.IsSupported(convertedFormat))
		})
	}
}

// TestEdgeCasesAndValidation tests edge cases and validation
func TestEdgeCasesAndValidation(t *testing.T) {
	helper := NewSubtitleFormatHelper()

	t.Run("extension extraction edge cases", func(t *testing.T) {
		// Test that the method handles various filename formats correctly
		testCases := []struct {
			filename string
			expected common.SubtitleFormat
		}{
			{"movie.backup.old.srt", common.SubtitleFormat_SUBTITLE_FORMAT_SRT},
			{".hidden.srt", common.SubtitleFormat_SUBTITLE_FORMAT_SRT},
			{"movie.", common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED},
			{"movie.SRT.bak", common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED},
			{"/path/with/dots.in.path/movie.vtt", common.SubtitleFormat_SUBTITLE_FORMAT_VTT},
		}

		for _, tc := range testCases {
			result := helper.FormatFromExtension(tc.filename)
			assert.Equal(t, tc.expected, result, "Failed for filename: %s", tc.filename)
		}
	})

	t.Run("all supported formats have display names", func(t *testing.T) {
		// Ensure all supported formats have non-empty display names
		supportedFormats := helper.GetAllSupportedFormats()
		for _, format := range supportedFormats {
			displayName := helper.GetFormatDisplayName(format)
			assert.NotEmpty(t, displayName)
			assert.NotEqual(t, "Unknown Format", displayName)
		}
	})

	t.Run("consistency between supported formats and methods", func(t *testing.T) {
		// Ensure all formats returned by GetAllSupportedFormats are indeed supported
		supportedFormats := helper.GetAllSupportedFormats()
		for _, format := range supportedFormats {
			assert.True(t, helper.IsSupported(format))
			assert.NotEmpty(t, helper.ExtensionFromFormat(format))
			assert.NotEqual(t, "Unknown Format", helper.GetFormatDisplayName(format))
		}
	})
}

// Benchmark tests for performance-critical operations
func BenchmarkFormatFromExtension(b *testing.B) {
	helper := NewSubtitleFormatHelper()
	testFilenames := []string{
		"movie.srt",
		"movie.vtt",
		"movie.ass",
		"movie.unknown",
		"/very/long/path/to/subtitle/file.with.many.dots.srt",
	}

	for i := 0; i < b.N; i++ {
		for _, filename := range testFilenames {
			helper.FormatFromExtension(filename)
		}
	}
}

func BenchmarkExtensionFromFormat(b *testing.B) {
	helper := NewSubtitleFormatHelper()
	testFormats := []common.SubtitleFormat{
		common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
		common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED,
	}

	for i := 0; i < b.N; i++ {
		for _, format := range testFormats {
			helper.ExtensionFromFormat(format)
		}
	}
}
