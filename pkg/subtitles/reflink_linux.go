// file: pkg/subtitles/reflink_linux.go
// version: 1.0.0
// guid: 0c6d38b7-9e21-4f5a-8340-be27d1f9a405
// last-edited: 2026-08-12

package subtitles

import (
	"os"

	"golang.org/x/sys/unix"
)

// reflink clones src to dst using the FICLONE ioctl, supported by btrfs, XFS
// and OpenZFS 2.2+ block cloning. ext4 does not support it and returns
// EOPNOTSUPP, which is reported as unsupported so the caller copies instead.
func reflink(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// O_EXCL preserves CloneFile's no-overwrite guarantee.
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := unix.IoctlFileClone(int(out.Fd()), int(in.Fd())); err != nil {
		// Remove the empty file we just created so the copy fallback can make
		// it itself; leaving it would trip that fallback's own O_EXCL.
		out.Close()
		_ = os.Remove(dst)
		if err == unix.EOPNOTSUPP || err == unix.ENOTTY || err == unix.EXDEV || err == unix.EINVAL {
			return errReflinkUnsupported
		}
		return err
	}
	return nil
}
