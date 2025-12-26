package fsm

import (
	"context"
	"testing"
)

func TestBugTrackerFSM(t *testing.T) {
	// Define States
	const (
		StateOpen     State = "Open"
		StateAssigned State = "Assigned"
		StateResolved State = "Resolved"
		StateClosed   State = "Closed"
	)

	// Define Events
	const (
		EventAssign Event = "Assign"
		EventResolve Event = "Resolve"
		EventClose   Event = "Close"
		EventReopen  Event = "Reopen"
	)

	// Create Machine
	sm := New("bug-123", StateOpen)

	// Context for actions
	ctx := context.Background()

	// Track actions
	logs := make([]string, 0)
	logAction := func(msg string) Action {
		return func(ctx context.Context, _ TransitionContext) error {
			logs = append(logs, msg)
			return nil
		}
	}

	// Configure
	sm.Configure(StateOpen).
		Permit(EventAssign, StateAssigned).
		OnExit(logAction("Exiting Open"))

	sm.Configure(StateAssigned).
		Permit(EventResolve, StateResolved).
		Permit(EventClose, StateClosed).
		OnEntry(logAction("Entering Assigned"))

	sm.Configure(StateResolved).
		Permit(EventClose, StateClosed).
		Permit(EventReopen, StateOpen)

	sm.Configure(StateClosed).
		Permit(EventReopen, StateOpen).
		OnEntry(logAction("Entering Closed"))

	// Test 1: Initial State
	if sm.CurrentState() != StateOpen {
		t.Errorf("Expected Open, got %s", sm.CurrentState())
	}

	// Test 2: Transition Open -> Assigned
	future := sm.Fire(ctx, EventAssign, "user:john")
	state, err := future.Await(ctx)
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}
	if state != StateAssigned {
		t.Errorf("Expected Assigned, got %s", state)
	}

	// Verify Actions
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}
	if logs[0] != "Exiting Open" {
		t.Errorf("Order mismatch: %v", logs)
	}
	if logs[1] != "Entering Assigned" {
		t.Errorf("Order mismatch: %v", logs)
	}

	// Test 3: Invalid Transition
	future = sm.Fire(ctx, EventReopen, nil) // Cannot Reopen from Assigned
	_, err = future.Await(ctx)
	if err == nil {
		t.Error("Expected error for invalid transition")
	}

	// Test 4: Guarded Transition (mock)
	// Add a new config for testing guard
	sm.Configure(StateAssigned).
		PermitIf("GuardTest", StateClosed, func(ctx context.Context, t TransitionContext) bool {
			return t.Data == "admin"
		})
	
	// Should fail (not admin)
	future = sm.Fire(ctx, "GuardTest", "user")
	_, err = future.Await(ctx)
	if err == nil {
		t.Error("Expected guard failure")
	}

	// Should succeed (admin)
	future = sm.Fire(ctx, "GuardTest", "admin")
	state, err = future.Await(ctx)
	if err != nil {
		t.Fatalf("Guard transition failed: %v", err)
	}
	if state != StateClosed {
		t.Errorf("Expected Closed, got %s", state)
	}
}

func TestInternalTransition(t *testing.T) {
	sm := New("test", "A")
	count := 0
	
	sm.Configure("A").
		InternalTransition("Inc", func(ctx context.Context, _ TransitionContext) error {
			count++
			return nil
		}).
		OnEntry(func(ctx context.Context, _ TransitionContext) error {
			t.Error("OnEntry should not be called for internal transition")
			return nil
		}).
		OnExit(func(ctx context.Context, _ TransitionContext) error {
			t.Error("OnExit should not be called for internal transition")
			return nil
		})

	ctx := context.Background()
	
	// Fire internal transition
	state, err := sm.Fire(ctx, "Inc", nil).Await(ctx)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if state != "A" {
		t.Errorf("State changed: %s", state)
	}
	if count != 1 {
		t.Errorf("Action not executed")
	}
}
