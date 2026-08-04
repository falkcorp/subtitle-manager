// file: pkg/webserver/diskusage_unix.go
// version: 1.0.0
// guid: 2d81f4a6-73c0-4b19-8e52-6f0a9d31bc47
// last-edited: 2026-08-04

//go:build !windows

package webserver

import "syscall"

// diskUsage reports the free and total bytes of the filesystem holding path.
//
// Both are zero when the filesystem cannot be queried. The caller reports these
// as informational system stats, so a failure here should degrade the numbers
// rather than fail the request.
func diskUsage(path string) (free, total uint64) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, 0
	}
	// Bsize is int64 on Linux and int32 on Darwin, hence the conversion.
	bsize := uint64(fs.Bsize)
	return fs.Bfree * bsize, fs.Blocks * bsize
}

// systemRoot is the path whose filesystem the system stats describe.
func systemRoot() string { return "/" }
