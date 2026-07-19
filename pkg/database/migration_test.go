// file: pkg/database/migration_test.go
// version: 1.0.0
// guid: 7b4d4003-442f-4395-b208-c375162f6fea

package database

import (
	"testing"

	database "github.com/falkcorp/gcommon/v2/pkg/databasepb/v2"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestDatabaseMigration ensures gcommon types use the opaque API correctly.
func TestDatabaseMigration(t *testing.T) {
	// Test that we can create and manipulate a Row with values
	rec := &database.Row{}

	// Row works with values as Any protobufs
	testValue, err := anypb.New(&timestamppb.Timestamp{})
	if err != nil {
		t.Fatalf("failed to create Any value: %v", err)
	}

	rec.SetValues([]*anypb.Any{testValue})

	if len(rec.GetValues()) != 1 {
		t.Errorf("expected 1 value, got %d", len(rec.GetValues()))
	}
}
