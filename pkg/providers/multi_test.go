// file: pkg/providers/multi_test.go
// version: 1.0.0
// guid: 6b1e9c04-8a37-4d52-9f60-2c5d0a7b3841

package providers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type alwaysFailProvider struct{}

func (alwaysFailProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return nil, fmt.Errorf("fail")
}

// TestFetchFromAllHonorsCancelledContext verifies FetchFromAll returns promptly
// with the context error instead of blocking on inter-provider delays when the
// context is already cancelled.
func TestFetchFromAllHonorsCancelledContext(t *testing.T) {
	RegisterFactory("failtest", func() Provider { return alwaysFailProvider{} })
	RegisterInstance(Instance{ID: "failtest-1", Name: "failtest", Enabled: true})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	start := time.Now()
	_, _, err := FetchFromAll(ctx, "/media/movie.mkv", "en", "")
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("FetchFromAll did not abort promptly: %v", elapsed)
	}
}
