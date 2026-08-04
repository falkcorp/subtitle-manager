// file: pkg/webserver/diskusage_windows.go
// version: 1.0.0
// guid: 8b52c07e-1a4d-4f39-95c6-0e73d2b8a614
// last-edited: 2026-08-04

//go:build windows

package webserver

import (
	"golang.org/x/sys/windows"
	"os"
)

// diskUsage reports the free and total bytes of the volume holding path.
//
// Windows has no statfs; GetDiskFreeSpaceEx is the equivalent. Both values are
// zero when the volume cannot be queried, matching the Unix implementation —
// the caller reports these as informational stats and should degrade the
// numbers rather than fail the request.
//
// The "free" figure is the caller-available free bytes rather than the total
// free bytes, which differ when disk quotas are in force. Available is the more
// useful of the two for "how much room is left for subtitles".
func diskUsage(path string) (free, total uint64) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var availableToCaller, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &availableToCaller, &totalBytes, &totalFree); err != nil {
		return 0, 0
	}
	return availableToCaller, totalBytes
}

// systemRoot is the path whose volume the system stats describe.
//
// "/" is not a volume on Windows. SystemDrive is normally "C:", which
// GetDiskFreeSpaceEx needs as a directory path, hence the separator.
func systemRoot() string {
	if drive := os.Getenv("SystemDrive"); drive != "" {
		return drive + `\`
	}
	return `C:\`
}
