//go:build !darwin && !linux

// file: pkg/subtitles/reflink_other.go
// version: 1.0.0
// guid: 5e40a218-3c7b-49d6-b1f2-6a908c74d3be
// last-edited: 2026-08-12

package subtitles

// reflink is unavailable on this platform (Windows ReFS aside, which Go has no
// portable API for), so CloneFile always falls back to copying.
func reflink(src, dst string) error { return errReflinkUnsupported }
