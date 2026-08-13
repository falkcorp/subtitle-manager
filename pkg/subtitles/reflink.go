// file: pkg/subtitles/reflink.go
// version: 2.0.0
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

// CloneFileIn creates dstName with the same contents as srcName, both relative
// to root, sharing storage where the filesystem supports it.
//
// A bilingual subtitle is written twice: once under a self-describing name
// (Episode.en-es.srt) and once under the Esperanto sentinel tag
// (Episode.eo.srt) that media servers list as a selectable track. The two are
// byte-identical, so on APFS, btrfs, XFS or OpenZFS ≥2.2 they are reflinked and
// the second costs no space. Everywhere else this degrades to a plain copy —
// subtitles are tens of kilobytes, so the fallback is not worth avoiding.
//
// # Why this takes a *os.Root rather than two paths
//
// Every operation goes through root, which confines it beneath that directory
// at the OS level: a traversal in a name is refused by the kernel-facing API
// rather than by our own string checks. Taking paths instead let a caller's
// (ultimately user-derived) media path flow straight into os.Open/os.OpenFile,
// which CodeQL correctly flagged as go/path-injection — twelve high-severity
// alerts across this file and its platform backends. Names here are expected to
// be single filename components produced by filepath.Base.
//
// Deliberately not a hardlink: a hardlink shares an inode, so a tool that
// rewrote one file in place would silently rewrite the other. A reflink is
// copy-on-write, and the copy fallback is independent too, which keeps the
// behaviour identical on every platform.
//
// It never overwrites: bilingual output is additive, and the sentinel name
// could legitimately belong to a real Esperanto subtitle.
func CloneFileIn(root *os.Root, srcName, dstName string) error {
	if _, err := root.Stat(srcName); err != nil {
		return fmt.Errorf("cloning %s: %w", srcName, err)
	}
	if _, err := root.Stat(dstName); err == nil {
		return fmt.Errorf("refusing to overwrite %s", dstName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking %s: %w", dstName, err)
	}

	if err := reflink(root, srcName, dstName); err == nil {
		return nil
	} else if !errors.Is(err, errReflinkUnsupported) {
		// A real failure on a filesystem that does support reflinks — out of
		// space, permissions. Copying would most likely fail the same way, but
		// it is cheap and might not, so try. Clear any partial destination
		// first or the copy's O_EXCL will trip over it.
		_ = root.Remove(dstName)
	}
	return copyFileIn(root, srcName, dstName)
}

// copyFileIn is the portable fallback. O_EXCL keeps the no-overwrite guarantee
// even if something created dstName between the check above and here.
func copyFileIn(root *os.Root, srcName, dstName string) error {
	in, err := root.Open(srcName)
	if err != nil {
		return fmt.Errorf("opening %s: %w", srcName, err)
	}
	defer in.Close()

	out, err := root.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dstName, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		// Do not leave a truncated subtitle behind for a player to pick up.
		_ = root.Remove(dstName)
		return fmt.Errorf("copying to %s: %w", dstName, err)
	}
	if err := out.Close(); err != nil {
		_ = root.Remove(dstName)
		return fmt.Errorf("closing %s: %w", dstName, err)
	}
	return nil
}
