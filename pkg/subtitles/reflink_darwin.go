// file: pkg/subtitles/reflink_darwin.go
// version: 2.0.0
// guid: 7f1a92c4-5d68-4b03-a71e-c8d40b53e926
// last-edited: 2026-08-12

package subtitles

import (
	"os"

	"golang.org/x/sys/unix"
)

// reflink clones srcName to dstName on APFS via clonefileat(2), which shares
// the underlying blocks copy-on-write. It fails if dstName exists, matching
// CloneFileIn's no-overwrite guarantee.
//
// clonefileat rather than clonefile so the operation is relative to a
// directory descriptor taken from root: the names never leave the confined
// directory, and no reconstructed absolute path is handed to a syscall.
func reflink(root *os.Root, srcName, dstName string) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	dirfd := int(dir.Fd())

	if err := unix.Clonefileat(dirfd, srcName, dirfd, dstName, 0); err != nil {
		// HFS+, a network mount, or a cross-volume destination: nothing is
		// wrong, this filesystem just cannot share blocks.
		if err == unix.ENOTSUP || err == unix.EXDEV || err == unix.EINVAL || err == unix.EOPNOTSUPP {
			return errReflinkUnsupported
		}
		return err
	}
	return nil
}
