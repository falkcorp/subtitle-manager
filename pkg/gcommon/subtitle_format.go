// file: pkg/gcommon/subtitle_format.go
// version: 1.1.1
// guid: 1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d

package gcommon

import (
	"strings"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
)

// SubtitleFormatHelper provides utilities for working with gcommon subtitle format enums
type SubtitleFormatHelper struct{}

// NewSubtitleFormatHelper creates a new subtitle format helper
func NewSubtitleFormatHelper() *SubtitleFormatHelper {
	return &SubtitleFormatHelper{}
}

// FormatFromExtension determines the subtitle format from a file extension
func (h *SubtitleFormatHelper) FormatFromExtension(filename string) common.SubtitleFormat {
	// Extract the file extension properly
	filename = strings.ToLower(filename)
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex == -1 {
		return common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED
	}
	ext := filename[dotIndex+1:]

	switch ext {
	case "srt":
		return common.SubtitleFormat_SUBTITLE_FORMAT_SRT
	case "vtt", "webvtt":
		return common.SubtitleFormat_SUBTITLE_FORMAT_VTT
	case "ass":
		return common.SubtitleFormat_SUBTITLE_FORMAT_ASS
	case "ssa":
		return common.SubtitleFormat_SUBTITLE_FORMAT_SSA
	case "ttml", "xml":
		return common.SubtitleFormat_SUBTITLE_FORMAT_TTML
	case "scc":
		return common.SubtitleFormat_SUBTITLE_FORMAT_SCC
	case "sbv":
		return common.SubtitleFormat_SUBTITLE_FORMAT_SBV
	default:
		return common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED
	}
}

// ExtensionFromFormat returns the file extension for a given subtitle format
func (h *SubtitleFormatHelper) ExtensionFromFormat(format common.SubtitleFormat) string {
	switch format {
	case common.SubtitleFormat_SUBTITLE_FORMAT_SRT:
		return "srt"
	case common.SubtitleFormat_SUBTITLE_FORMAT_VTT:
		return "vtt"
	case common.SubtitleFormat_SUBTITLE_FORMAT_ASS:
		return "ass"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SSA:
		return "ssa"
	case common.SubtitleFormat_SUBTITLE_FORMAT_TTML:
		return "ttml"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SCC:
		return "scc"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SBV:
		return "sbv"
	default:
		return ""
	}
}

// IsSupported checks if a subtitle format is supported
func (h *SubtitleFormatHelper) IsSupported(format common.SubtitleFormat) bool {
	return format != common.SubtitleFormat_SUBTITLE_FORMAT_UNSPECIFIED
}

// GetAllSupportedFormats returns all supported subtitle formats
func (h *SubtitleFormatHelper) GetAllSupportedFormats() []common.SubtitleFormat {
	return []common.SubtitleFormat{
		common.SubtitleFormat_SUBTITLE_FORMAT_SRT,
		common.SubtitleFormat_SUBTITLE_FORMAT_VTT,
		common.SubtitleFormat_SUBTITLE_FORMAT_ASS,
		common.SubtitleFormat_SUBTITLE_FORMAT_SSA,
		common.SubtitleFormat_SUBTITLE_FORMAT_TTML,
		common.SubtitleFormat_SUBTITLE_FORMAT_SCC,
		common.SubtitleFormat_SUBTITLE_FORMAT_SBV,
	}
}

// GetFormatDisplayName returns a human-readable name for the format
func (h *SubtitleFormatHelper) GetFormatDisplayName(format common.SubtitleFormat) string {
	switch format {
	case common.SubtitleFormat_SUBTITLE_FORMAT_SRT:
		return "SubRip (SRT)"
	case common.SubtitleFormat_SUBTITLE_FORMAT_VTT:
		return "WebVTT"
	case common.SubtitleFormat_SUBTITLE_FORMAT_ASS:
		return "Advanced SubStation Alpha (ASS)"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SSA:
		return "SubStation Alpha (SSA)"
	case common.SubtitleFormat_SUBTITLE_FORMAT_TTML:
		return "Timed Text Markup Language (TTML)"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SCC:
		return "Scenarist Closed Caption (SCC)"
	case common.SubtitleFormat_SUBTITLE_FORMAT_SBV:
		return "YouTube Subtitle Format (SBV)"
	default:
		return "Unknown Format"
	}
}
