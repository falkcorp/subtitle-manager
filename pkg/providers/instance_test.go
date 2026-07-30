package providers

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/providers/mocks"
	"github.com/jdfalk/subtitle-manager/pkg/tagging"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
	"github.com/stretchr/testify/mock"
)

// TestFetchFromAllInstances verifies that provider instances are selected in
// priority order and that backoff is tracked per instance.
func TestFetchFromAllInstances(t *testing.T) {
	// Setup mock provider factory returning different mocks sequentially.
	cnt := 0
	m1 := mocks.NewMockProvider(t)
	m2 := mocks.NewMockProvider(t)
	RegisterFactory("mock", func() Provider {
		cnt++
		if cnt == 1 {
			return m1
		}
		return m2
	})
	t.Cleanup(func() {
		delete(factories, "mock")
	})

	// Register two provider instances with different priorities.
	RegisterInstance(Instance{ID: "p1", Name: "mock", Priority: 1, Enabled: true})
	RegisterInstance(Instance{ID: "p2", Name: "mock", Priority: 0, Enabled: true})
	t.Cleanup(func() {
		instancesMu.Lock()
		instances = map[string]Instance{}
		instancesMu.Unlock()
		backoffMu.Lock()
		backoffMap = map[string]time.Time{}
		backoffMu.Unlock()
	})

	m1.On("Fetch", mock.Anything, "file.mkv", "en").Return([]byte(nil), errors.New("fail"))
	// A real subtitle: provider responses are validated, so "ok" would now be
	// rejected as not-a-subtitle and the fetch would report nothing found.
	okSub := []byte("1\n00:00:01,000 --> 00:00:02,000\nok\n")
	m2.On("Fetch", mock.Anything, "file.mkv", "en").Return(okSub, nil)

	data, id, err := FetchFromAll(context.Background(), "file.mkv", "en", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(data, []byte("ok")) || id != "p2" {
		t.Fatalf("unexpected result %s %s", data, id)
	}

}

func TestInstance_TaggedEntity(t *testing.T) {
	// Test that Instance implements TaggedEntity interface
	instance := &Instance{
		ID:       "test-provider-1",
		Name:     "Test Provider",
		Priority: 10,
		Tags:     []string{"reliable", "fast"},
		Enabled:  true,
	}

	// Test TaggedEntity interface methods
	if instance.GetEntityType() != "provider" {
		t.Errorf("Expected entity type 'provider', got '%s'", instance.GetEntityType())
	}

	if instance.GetEntityID() != "test-provider-1" {
		t.Errorf("Expected entity ID 'test-provider-1', got '%s'", instance.GetEntityID())
	}

	tags := instance.GetTags()
	expectedTags := []string{"reliable", "fast"}
	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}

	for i, expectedTag := range expectedTags {
		if i >= len(tags) || tags[i] != expectedTag {
			t.Errorf("Expected tag '%s' at index %d, got '%s'", expectedTag, i, tags[i])
		}
	}

	// Test SetTags
	newTags := []string{"updated", "tags"}
	instance.SetTags(newTags)

	updatedTags := instance.GetTags()
	if len(updatedTags) != len(newTags) {
		t.Errorf("Expected %d updated tags, got %d", len(newTags), len(updatedTags))
	}

	for i, expectedTag := range newTags {
		if i >= len(updatedTags) || updatedTags[i] != expectedTag {
			t.Errorf("Expected updated tag '%s' at index %d, got '%s'", expectedTag, i, updatedTags[i])
		}
	}
}

func TestInstancePriorities(t *testing.T) {
	// Clear any existing instances
	instancesMu.Lock()
	instances = make(map[string]Instance)
	instancesMu.Unlock()

	// Register instances with different priorities
	highPriorityInstance := Instance{
		ID:       "high-priority",
		Name:     "High Priority Provider",
		Priority: 90,
		Tags:     []string{"premium"},
		Enabled:  true,
	}

	mediumPriorityInstance := Instance{
		ID:       "medium-priority",
		Name:     "Medium Priority Provider",
		Priority: 50,
		Tags:     []string{"standard"},
		Enabled:  true,
	}

	lowPriorityInstance := Instance{
		ID:       "low-priority",
		Name:     "Low Priority Provider",
		Priority: 10,
		Tags:     []string{"free"},
		Enabled:  true,
	}

	// Register instances
	RegisterInstance(highPriorityInstance)
	RegisterInstance(mediumPriorityInstance)
	RegisterInstance(lowPriorityInstance)

	// Test GetInstance retrieval
	retrieved, exists := GetInstance("high-priority")
	if !exists {
		t.Fatal("High priority instance not found")
	}

	if retrieved.Priority != 90 {
		t.Errorf("Expected priority 90, got %d", retrieved.Priority)
	}

	// Test ListInstances returns all instances
	allInstances := ListInstances()
	if len(allInstances) != 3 {
		t.Errorf("Expected 3 instances, got %d", len(allInstances))
	}

	// Test priority ordering (ListInstances should return sorted by priority)
	expectedOrder := []string{"high-priority", "medium-priority", "low-priority"}
	for i, expectedID := range expectedOrder {
		if i >= len(allInstances) || allInstances[i].ID != expectedID {
			t.Errorf("Expected instance '%s' at index %d, got '%s'", expectedID, i, allInstances[i].ID)
		}
	}
}

func TestInstancesByTags(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()
	tm := tagging.NewTagManager(db)

	tag, err := tm.CreateTag("fast", "user", "provider", "", "")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	tagID, _ := strconv.ParseInt(tag.ID, 10, 64)

	instancesMu.Lock()
	instances = map[string]Instance{}
	instancesMu.Unlock()
	RegisterInstance(Instance{ID: "p1", Name: "mock", Priority: 5, Enabled: true})
	RegisterInstance(Instance{ID: "p2", Name: "mock", Priority: 1, Enabled: true})

	testutil.MustNoError(t, "tag p2", database.AssignTagToEntity(db, tagID, "provider", "p2"))

	insts, err := InstancesByTags(tm, []string{"fast"})
	if err != nil {
		t.Fatalf("InstancesByTags error: %v", err)
	}
	if len(insts) != 1 || insts[0].ID != "p2" {
		t.Fatalf("expected only p2, got %v", insts)
	}
}

func TestFetchFromTagged(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()
	tm := tagging.NewTagManager(db)

	tag, _ := tm.CreateTag("fast", "user", "provider", "", "")
	tagID, _ := strconv.ParseInt(tag.ID, 10, 64)

	m2 := mocks.NewMockProvider(t)
	RegisterFactory("mock", func() Provider { return m2 })
	t.Cleanup(func() {
		delete(factories, "mock")
		instancesMu.Lock()
		instances = map[string]Instance{}
		instancesMu.Unlock()
		backoffMu.Lock()
		backoffMap = map[string]time.Time{}
		backoffMu.Unlock()
	})

	RegisterInstance(Instance{ID: "p1", Name: "mock", Priority: 1, Enabled: true})
	RegisterInstance(Instance{ID: "p2", Name: "mock", Priority: 0, Enabled: true})

	testutil.MustNoError(t, "tag p2", database.AssignTagToEntity(db, tagID, "provider", "p2"))

	// A real subtitle: provider responses are validated, so "ok" would now be
	// rejected as not-a-subtitle and the fetch would report nothing found.
	okSub := []byte("1\n00:00:01,000 --> 00:00:02,000\nok\n")
	m2.On("Fetch", mock.Anything, "file.mkv", "en").Return(okSub, nil)

	data, id, err := FetchFromTagged(context.Background(), "file.mkv", "en", "", []string{"fast"}, tm)
	if err != nil {
		t.Fatalf("FetchFromTagged err: %v", err)
	}
	if id != "p2" || !bytes.Contains(data, []byte("ok")) {
		t.Fatalf("unexpected result %s %s", data, id)
	}
}
