// file: pkg/subtitles/reflink_darwin.go
// version: 1.0.0
// guid: 7f1a92c4-5d68-4b03-a71e-c8d40b53e926
// last-edited: 2026-08-12

package subtitles

import "golang.org/x/sys/unix"

// reflink clones src to dst on APFS via clonefile(2), which shares the
// underlying blocks copy-on-write. It fails if dst already exists, which
// matches CloneFile's no-overwrite guarantee.
func reflink(src, dst string) error {
	if err := unix.Clonefile(src, dst, 0); err != nil {
		// HFS+, a network mount, or a cross-volume destination: nothing is
		// wrong, this filesystem just cannot share blocks.
		if err == unix.ENOTSUP || err == unix.EXDEV || err == unix.EINVAL || err == unix.EOPNOTSUPP {
			return errReflinkUnsupported
		}
		return err
	}
	return nil
}
