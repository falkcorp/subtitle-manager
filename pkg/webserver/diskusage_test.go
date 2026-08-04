// file: pkg/webserver/diskusage_test.go
// version: 1.0.0
// guid: c04a8f16-5b93-4e27-a1d8-36f905e2b7c3
// last-edited: 2026-08-04

package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDiskUsageReportsSomething pins that the platform helper actually queries
// the filesystem rather than silently returning zeros.
//
// Zeros are the documented failure value, so a helper that always failed would
// still satisfy the type checker and produce a valid JSON response. Any real
// machine running this test has a non-empty root filesystem.
func TestDiskUsageReportsSomething(t *testing.T) {
	free, total := diskUsage(systemRoot())
	if total == 0 {
		t.Fatalf("diskUsage(%q) reported total=0; the query failed", systemRoot())
	}
	if free > total {
		t.Errorf("free (%d) exceeds total (%d)", free, total)
	}
}

// TestSystemHandlerReportsDisk covers the handler wiring, since the fields are
// what the UI reads.
func TestSystemHandlerReportsDisk(t *testing.T) {
	rr := httptest.NewRecorder()
	systemHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/system", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}

	var got struct {
		OS        string `json:"os"`
		DiskTotal uint64 `json:"disk_total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.OS == "" {
		t.Error("os is empty")
	}
	if got.DiskTotal == 0 {
		t.Error("disk_total is 0; the handler is not reaching the platform helper")
	}
}
