// file: pkg/subtitles/reflink_linux.go
// version: 2.0.0
// guid: 0c6d38b7-9e21-4f5a-8340-be27d1f9a405
// last-edited: 2026-08-12

package subtitles

import (
	"os"

	"golang.org/x/sys/unix"
)

// reflink clones srcName to dstName using the FICLONE ioctl, supported by
// btrfs, XFS and OpenZFS 2.2+ block cloning. ext4 does not support it and
// returns EOPNOTSUPP, which is reported as unsupported so the caller copies
// instead.
//
// The ioctl works on file descriptors, and both are obtained through root, so
// no path string reaches a syscall.
func reflink(root *os.Root, srcName, dstName string) error {
	in, err := root.Open(srcName)
	if err != nil {
		return err
	}
	defer in.Close()

	// O_EXCL preserves CloneFileIn's no-overwrite guarantee.
	out, err := root.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := unix.IoctlFileClone(int(out.Fd()), int(in.Fd())); err != nil {
		// Remove the empty file just created so the copy fallback can make it
		// itself; leaving it would trip that fallback's own O_EXCL.
		out.Close()
		_ = root.Remove(dstName)
		if err == unix.EOPNOTSUPP || err == unix.ENOTTY || err == unix.EXDEV || err == unix.EINVAL {
			return errReflinkUnsupported
		}
		return err
	}
	return nil
}
