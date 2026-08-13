// file: pkg/subtitles/reflink.go
// version: 1.0.0
// guid: 3b8e60d1-47af-4c92-9e05-1af7c2635d80
// last-edited: 2026-08-12

package subtitles

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// errReflinkUnsupported reports that this platform or filesystem cannot share
// blocks between files, so the caller should fall back to copying.
var errReflinkUnsupported = errors.New("reflink not supported")

// CloneFile creates dst with the same contents as src, sharing storage where
// the filesystem supports it.
//
// A bilingual subtitle is written twice: once under a self-describing name
// (Episode.en-es.srt) and once under the Esperanto sentinel tag
// (Episode.eo.srt) that media servers list as a selectable track. The two are
// byte-identical, so on APFS, btrfs, XFS or OpenZFS ≥2.2 they are reflinked and
// the second costs no space. Everywhere else this degrades to a plain copy —
// subtitles are tens of kilobytes, so the fallback is not worth avoiding.
//
// Deliberately not a hardlink: a hardlink shares an inode, so a tool that
// rewrote one file in place would silently rewrite the other. A reflink is
// copy-on-write, and the copy fallback is independent too, which keeps the
// behaviour identical on every platform. TestCloneFileProducesAnIndependentCopy
// pins that down.
//
// CloneFile never overwrites: bilingual output is additive, and the sentinel
// name could legitimately belong to a real Esperanto subtitle.
func CloneFile(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("cloning %s: %w", src, err)
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("refusing to overwrite %s", dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking %s: %w", dst, err)
	}

	if err := reflink(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, errReflinkUnsupported) {
		// A real failure on a filesystem that does support reflinks — out of
		// space, permissions, a cross-device destination. Copying would most
		// likely fail the same way, but it is cheap and might not, so try.
		_ = os.Remove(dst)
	}
	return copyFile(src, dst)
}

// copyFile is the portable fallback. O_EXCL keeps the no-overwrite guarantee
// even if something created dst between the check above and here.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		// Do not leave a truncated subtitle behind for a player to pick up.
		_ = os.Remove(dst)
		return fmt.Errorf("copying to %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("closing %s: %w", dst, err)
	}
	return nil
}
