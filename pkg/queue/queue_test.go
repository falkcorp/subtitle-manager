// file: pkg/queue/queue_test.go
// version: 1.1.1
// guid: 123e4567-e89b-12d3-a456-426614174003
package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	gqueue "github.com/falkcorp/gcommon/v2/pkg/queuepb/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJob implements the Job interface for testing.
type mockJob struct {
	id          string
	jobType     JobType
	description string
	executeFunc func(ctx context.Context) error
}

func (m *mockJob) ID() string {
	return m.id
}

func (m *mockJob) Type() JobType {
	return m.jobType
}

func (m *mockJob) Description() string {
	return m.description
}

func (m *mockJob) Execute(ctx context.Context) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}
	return nil
}

func (m *mockJob) QueueMessage() (*gqueue.QueueMessage, error) {
	qm := &gqueue.QueueMessage{}
	qm.SetId(m.id)
	return qm, nil
}

func TestNewQueue(t *testing.T) {
	q := NewQueue(5)
	assert.NotNil(t, q)
	assert.Equal(t, 5, q.workers)
	assert.False(t, q.IsRunning())
}

func TestQueueStartStop(t *testing.T) {
	q := NewQueue(2)

	// Test starting the queue
	err := q.Start()
	require.NoError(t, err)
	assert.True(t, q.IsRunning())

	// Test starting an already running queue
	err = q.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stopping the queue
	err = q.Stop()
	require.NoError(t, err)
	assert.False(t, q.IsRunning())

	// Test stopping a stopped queue
	err = q.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestQueueAddJob(t *testing.T) {
	q := NewQueue(1)

	// Test adding job to stopped queue
	job := &mockJob{id: "test-1", jobType: JobTypeSingleFile, description: "test job"}
	_, err := q.Add(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start queue and add job
	err = q.Start()
	require.NoError(t, err)
	defer q.Stop()

	taskID, err := q.Add(job)
	require.NoError(t, err)
	assert.Equal(t, "test-1", taskID)
}

func TestQueueProcessJob(t *testing.T) {
	q := NewQueue(1)
	err := q.Start()
	require.NoError(t, err)
	defer q.Stop()

	// executed is closed from the worker goroutine when the job runs; the test
	// goroutine waits on it. Using a channel (instead of a plain bool + sleep)
	// provides the happens-before needed to avoid a data race under `go test
	// -race`, and lets the test proceed as soon as the job completes.
	executed := make(chan struct{})
	job := &mockJob{
		id:          "test-job",
		jobType:     JobTypeSingleFile,
		description: "test execution",
		executeFunc: func(ctx context.Context) error {
			close(executed)
			return nil
		},
	}

	_, err = q.Add(job)
	require.NoError(t, err)

	// Wait for the job to be processed.
	select {
	case <-executed:
	case <-time.After(2 * time.Second):
		t.Fatal("job should have been executed within the timeout")
	}
}

func TestQueueStatus(t *testing.T) {
	q := NewQueue(3)

	status := q.Status()
	assert.False(t, status.Running)
	assert.Equal(t, 3, status.Workers)
	assert.Equal(t, 0, status.QueueLength)

	err := q.Start()
	require.NoError(t, err)
	defer q.Stop()

	status = q.Status()
	assert.True(t, status.Running)
	assert.Equal(t, 3, status.Workers)
}

func TestSingleFileJob(t *testing.T) {
	job := NewSingleFileJob(
		"/input.srt",
		"/output.srt",
		"en",
		"google",
		"api-key",
		"gpt-key",
		"localhost:8080",
	)

	assert.NotEmpty(t, job.ID())
	assert.Equal(t, JobTypeSingleFile, job.Type())
	assert.Contains(t, job.Description(), "/input.srt")
	assert.Contains(t, job.Description(), "en")
	assert.Contains(t, job.Description(), "/output.srt")

	// Test execution (will fail since file doesn't exist, but we check it's called)
	err := job.Execute(context.Background())
	assert.Error(t, err) // Expected since files don't exist
}

func TestBatchFilesJob(t *testing.T) {
	job := NewBatchFilesJob(
		[]string{"/file1.srt", "/file2.srt"},
		"en",
		"google",
		"api-key",
		"gpt-key",
		"localhost:8080",
		2,
	)

	assert.NotEmpty(t, job.ID())
	assert.Equal(t, JobTypeBatchFiles, job.Type())
	assert.Contains(t, job.Description(), "2 files")
	assert.Contains(t, job.Description(), "en")

	// Test execution (will fail since files don't exist, but we check it's called)
	err := job.Execute(context.Background())
	assert.Error(t, err) // Expected since files don't exist
}

func TestGlobalQueue(t *testing.T) {
	// Reset global queue
	SetQueue(nil)

	q1 := GetQueue()
	assert.NotNil(t, q1)

	q2 := GetQueue()
	assert.Same(t, q1, q2, "GetQueue should return the same instance")

	// Test setting a custom queue
	customQueue := NewQueue(10)
	SetQueue(customQueue)

	q3 := GetQueue()
	assert.Same(t, customQueue, q3, "GetQueue should return the custom queue")
}

func TestQueueConcurrency(t *testing.T) {
	q := NewQueue(2)
	err := q.Start()
	require.NoError(t, err)
	defer q.Stop()

	executionCount := 0
	var execMutex sync.Mutex

	// Add multiple jobs
	for i := 0; i < 5; i++ {
		job := &mockJob{
			id:          fmt.Sprintf("concurrent-job-%d", i),
			jobType:     JobTypeSingleFile,
			description: fmt.Sprintf("concurrent test job %d", i),
			executeFunc: func(ctx context.Context) error {
				execMutex.Lock()
				executionCount++
				execMutex.Unlock()
				time.Sleep(50 * time.Millisecond) // Simulate work
				return nil
			},
		}

		_, err := q.Add(job)
		require.NoError(t, err)
	}

	// Wait for all jobs to complete
	time.Sleep(500 * time.Millisecond)

	execMutex.Lock()
	finalCount := executionCount
	execMutex.Unlock()

	assert.Equal(t, 5, finalCount, "all jobs should have been executed")
}

// TestQueueRestartCycle tests that the queue can be restarted after being stopped.
func TestQueueRestartCycle(t *testing.T) {
	q := NewQueue(1)

	// First cycle: Start -> Stop
	err := q.Start()
	require.NoError(t, err)
	assert.True(t, q.IsRunning())

	err = q.Stop()
	require.NoError(t, err)
	assert.False(t, q.IsRunning())

	// Second cycle: Start -> Stop (should work after previous stop)
	err = q.Start()
	require.NoError(t, err)
	assert.True(t, q.IsRunning())

	// Test that jobs can be processed in the restarted queue
	job := &mockJob{
		id:          "restart-test-job",
		jobType:     JobTypeSingleFile,
		description: "restart cycle test job",
		executeFunc: func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	_, err = q.Add(job)
	require.NoError(t, err)

	// Give time for job to process
	time.Sleep(100 * time.Millisecond)

	err = q.Stop()
	require.NoError(t, err)
	assert.False(t, q.IsRunning())
}
