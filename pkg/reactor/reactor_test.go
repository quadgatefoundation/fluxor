package reactor

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestReactor_SequentialExecution(t *testing.T) {
	reactor := New(10)
	reactor.Start()
	defer reactor.Stop(context.Background())

	var result []int
	var wg sync.WaitGroup
	wg.Add(5)

	for i := 0; i < 5; i++ {
		val := i
		reactor.Post(func() {
			result = append(result, val)
			wg.Done()
		})
	}

	wg.Wait()

	if len(result) != 5 {
		t.Fatalf("Expected result length 5, got %d", len(result))
	}

	for i, v := range result {
		if v != i {
			t.Errorf("Expected result[%d] to be %d, got %d", i, i, v)
		}
	}
}

func TestReactor_Backpressure(t *testing.T) {
	reactor := New(1)
	reactor.Start()
	defer reactor.Stop(context.Background())

	blocker := make(chan struct{})

	// Post a task that blocks
	err := reactor.Post(func() {
		<-blocker
	})
	if err != nil {
		t.Fatalf("Post should not have failed: %v", err)
	}

	// Post another task, which should fail with ErrBackpressure
	err = reactor.Post(func() {})
	if err != ErrBackpressure {
		t.Fatalf("Expected ErrBackpressure, got %v", err)
	}

	// Unblock the first task
	close(blocker)
}

func TestReactor_Stop(t *testing.T) {
	reactor := New(1)
	reactor.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := reactor.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	err := reactor.Post(func() {})
	if err != ErrStopped {
		t.Fatalf("Expected ErrStopped, got %v", err)
	}
}
