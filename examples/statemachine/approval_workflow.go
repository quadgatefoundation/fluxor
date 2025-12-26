package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/statemachine"
)

// ApprovalWorkflowExample demonstrates a multi-level approval state machine.
//
// States: draft → pending_l1 → pending_l2 → pending_l3 → approved
//         Any pending state can go to rejected or revision_requested

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Fluxor app
	app, err := fluxor.NewMainVerticleWithOptions("", fluxor.MainVerticleOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Create state machine verticle
	smVerticle := statemachine.NewStateMachineVerticle()

	// Deploy the verticle
	if _, err := app.Vertx().DeployVerticle(smVerticle); err != nil {
		log.Fatalf("Failed to deploy state machine verticle: %v", err)
	}

	// Create and register the approval workflow state machine
	approvalSM := createApprovalWorkflowStateMachine()
	if err := smVerticle.RegisterStateMachine(approvalSM, nil); err != nil {
		log.Fatalf("Failed to register state machine: %v", err)
	}

	log.Println("Approval Workflow State Machine deployed successfully")

	// Create a state machine client
	client := statemachine.NewStateMachineClient(app.Vertx().EventBus())

	// Example: Create a document approval instance
	docID, err := client.CreateInstance(ctx, "approval-workflow", map[string]interface{}{
		"documentId":   "DOC-2024-001",
		"documentType": "Purchase Order",
		"amount":       15000.00,
		"submittedBy":  "alice@example.com",
		"department":   "Engineering",
		"description":  "New server hardware purchase",
	})
	if err != nil {
		log.Fatalf("Failed to create document instance: %v", err)
	}

	log.Printf("Created document instance: %s", docID)

	// Simulate approval workflow
	go simulateApprovalWorkflow(ctx, client, docID)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Approval Workflow State Machine is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")
	if err := app.Stop(); err != nil {
		log.Printf("Error stopping app: %v", err)
	}
}

func createApprovalWorkflowStateMachine() *statemachine.StateMachineDefinition {
	builder := statemachine.NewBuilder("approval-workflow", "Multi-Level Approval Workflow")
	builder.WithDescription("State machine for multi-level document approval").
		WithVersion("1.0").
		WithInitialState("draft")

	// Define states
	builder.AddStates(
		// Draft: Document being prepared
		statemachine.NewState("draft", "Draft").
			WithDescription("Document is in draft state").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] Document in draft state", ctx.MachineID)
				ctx.Data["approvals"] = make([]map[string]interface{}, 0)
				return nil
			}).
			Build(),

		// Level 1 Approval: Manager approval
		statemachine.NewState("pending_l1", "Pending Level 1 Approval").
			WithDescription("Waiting for manager approval").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] Submitted for Level 1 approval (Manager)", ctx.MachineID)
				ctx.Data["l1_submitted_at"] = time.Now()
				return nil
			}).
			Build(),

		// Level 2 Approval: Director approval
		statemachine.NewState("pending_l2", "Pending Level 2 Approval").
			WithDescription("Waiting for director approval").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] Submitted for Level 2 approval (Director)", ctx.MachineID)
				ctx.Data["l2_submitted_at"] = time.Now()
				return nil
			}).
			Build(),

		// Level 3 Approval: Executive approval
		statemachine.NewState("pending_l3", "Pending Level 3 Approval").
			WithDescription("Waiting for executive approval").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] Submitted for Level 3 approval (Executive)", ctx.MachineID)
				ctx.Data["l3_submitted_at"] = time.Now()
				return nil
			}).
			Build(),

		// Approved: All approvals complete (final state)
		statemachine.NewState("approved", "Approved").
			WithDescription("Document fully approved").
			AsFinal().
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] ✅ Document fully approved!", ctx.MachineID)
				ctx.Data["approved_at"] = time.Now()
				approvals := ctx.Data["approvals"].([]map[string]interface{})
				log.Printf("[Doc %s] Total approvals: %d", ctx.MachineID, len(approvals))
				for _, approval := range approvals {
					log.Printf("  - Level %v by %v at %v", approval["level"], approval["approver"], approval["timestamp"])
				}
				return nil
			}).
			Build(),

		// Rejected: Document rejected (final state)
		statemachine.NewState("rejected", "Rejected").
			WithDescription("Document rejected").
			AsFinal().
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] ❌ Document rejected", ctx.MachineID)
				ctx.Data["rejected_at"] = time.Now()
				if reason, ok := ctx.Data["rejection_reason"]; ok {
					log.Printf("[Doc %s] Reason: %v", ctx.MachineID, reason)
				}
				return nil
			}).
			Build(),

		// Revision Requested: Sent back for changes
		statemachine.NewState("revision_requested", "Revision Requested").
			WithDescription("Changes requested, sent back to draft").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Doc %s] 📝 Revision requested", ctx.MachineID)
				if comments, ok := ctx.Data["revision_comments"]; ok {
					log.Printf("[Doc %s] Comments: %v", ctx.MachineID, comments)
				}
				return nil
			}).
			Build(),
	)

	// Define transitions
	builder.AddTransitions(
		// draft → pending_l1 (submit for approval)
		statemachine.NewTransition("submit", "draft", "pending_l1", "submit").
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				// Check if document is complete
				if ctx.Data["documentId"] == nil {
					return false, fmt.Errorf("document ID is required")
				}
				return true, nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				ctx.Data["submitted_at"] = time.Now()
				log.Printf("[Doc %s] Submitting document for approval", ctx.MachineID)
				return nil
			}).
			Build(),

		// pending_l1 → pending_l2 (L1 approved)
		statemachine.NewTransition("approve-l1", "pending_l1", "pending_l2", "approve").
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				// Check if approver has L1 authority
				role, ok := event.Data["approver_role"].(string)
				return ok && role == "manager", nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordApproval(ctx, event, 1)
				return nil
			}).
			Build(),

		// pending_l2 → pending_l3 (L2 approved, high value requires L3)
		statemachine.NewTransition("approve-l2-high-value", "pending_l2", "pending_l3", "approve").
			WithPriority(10). // Higher priority than direct approval
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				role, ok := event.Data["approver_role"].(string)
				if !ok || role != "director" {
					return false, nil
				}
				// Require L3 approval for amounts > $10,000
				amount, ok := ctx.Data["amount"].(float64)
				return ok && amount > 10000, nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordApproval(ctx, event, 2)
				log.Printf("[Doc %s] High value item, escalating to Level 3", ctx.MachineID)
				return nil
			}).
			Build(),

		// pending_l2 → approved (L2 approved, low value)
		statemachine.NewTransition("approve-l2-low-value", "pending_l2", "approved", "approve").
			WithPriority(5).
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				role, ok := event.Data["approver_role"].(string)
				if !ok || role != "director" {
					return false, nil
				}
				// Can approve directly if amount <= $10,000
				amount, ok := ctx.Data["amount"].(float64)
				return ok && amount <= 10000, nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordApproval(ctx, event, 2)
				log.Printf("[Doc %s] Low value item, approved at Level 2", ctx.MachineID)
				return nil
			}).
			Build(),

		// pending_l3 → approved (L3 approved)
		statemachine.NewTransition("approve-l3", "pending_l3", "approved", "approve").
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				role, ok := event.Data["approver_role"].(string)
				return ok && role == "executive", nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordApproval(ctx, event, 3)
				return nil
			}).
			Build(),

		// Any pending state → rejected
		statemachine.NewTransition("reject-l1", "pending_l1", "rejected", "reject").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRejection(ctx, event)
				return nil
			}).
			Build(),

		statemachine.NewTransition("reject-l2", "pending_l2", "rejected", "reject").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRejection(ctx, event)
				return nil
			}).
			Build(),

		statemachine.NewTransition("reject-l3", "pending_l3", "rejected", "reject").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRejection(ctx, event)
				return nil
			}).
			Build(),

		// Any pending state → revision_requested
		statemachine.NewTransition("revise-l1", "pending_l1", "revision_requested", "request_revision").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRevisionRequest(ctx, event)
				return nil
			}).
			Build(),

		statemachine.NewTransition("revise-l2", "pending_l2", "revision_requested", "request_revision").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRevisionRequest(ctx, event)
				return nil
			}).
			Build(),

		statemachine.NewTransition("revise-l3", "pending_l3", "revision_requested", "request_revision").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				recordRevisionRequest(ctx, event)
				return nil
			}).
			Build(),

		// revision_requested → draft (resubmit after changes)
		statemachine.NewTransition("resubmit", "revision_requested", "draft", "resubmit").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				log.Printf("[Doc %s] Resubmitting after revisions", ctx.MachineID)
				ctx.Data["revision_count"] = getRevisionCount(ctx) + 1
				return nil
			}).
			Build(),
	)

	definition, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build state machine: %v", err)
	}

	return definition
}

func recordApproval(ctx *statemachine.StateContext, event *statemachine.Event, level int) {
	approvals := ctx.Data["approvals"].([]map[string]interface{})
	approval := map[string]interface{}{
		"level":     level,
		"approver":  event.Data["approver_email"],
		"role":      event.Data["approver_role"],
		"timestamp": time.Now(),
		"comments":  event.Data["comments"],
	}
	ctx.Data["approvals"] = append(approvals, approval)
	log.Printf("[Doc %s] ✓ Level %d approved by %v", ctx.MachineID, level, event.Data["approver_email"])
}

func recordRejection(ctx *statemachine.StateContext, event *statemachine.Event) {
	ctx.Data["rejection_reason"] = event.Data["reason"]
	ctx.Data["rejected_by"] = event.Data["approver_email"]
	log.Printf("[Doc %s] Rejected by %v: %v", ctx.MachineID, event.Data["approver_email"], event.Data["reason"])
}

func recordRevisionRequest(ctx *statemachine.StateContext, event *statemachine.Event) {
	ctx.Data["revision_comments"] = event.Data["comments"]
	ctx.Data["revision_requested_by"] = event.Data["approver_email"]
	log.Printf("[Doc %s] Revision requested by %v", ctx.MachineID, event.Data["approver_email"])
}

func getRevisionCount(ctx *statemachine.StateContext) int {
	if count, ok := ctx.Data["revision_count"].(int); ok {
		return count
	}
	return 0
}

func simulateApprovalWorkflow(ctx context.Context, client *statemachine.StateMachineClient, docID string) {
	time.Sleep(1 * time.Second)

	// Step 1: Submit for approval
	log.Println("\n=== Submitting document for approval ===")
	success, err := client.SendEvent(ctx, "approval-workflow", docID, "submit", nil)
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Submitted: %v", success)

	// Step 2: Level 1 approval (Manager)
	time.Sleep(2 * time.Second)
	log.Println("\n=== Manager approving (Level 1) ===")
	success, err = client.SendEvent(ctx, "approval-workflow", docID, "approve", map[string]interface{}{
		"approver_email": "manager@example.com",
		"approver_role":  "manager",
		"comments":       "Looks good, approved.",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Approved: %v", success)

	// Step 3: Level 2 approval (Director) - will escalate to L3 due to high value
	time.Sleep(2 * time.Second)
	log.Println("\n=== Director approving (Level 2) ===")
	success, err = client.SendEvent(ctx, "approval-workflow", docID, "approve", map[string]interface{}{
		"approver_email": "director@example.com",
		"approver_role":  "director",
		"comments":       "Approved, escalating to executive.",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Approved: %v", success)

	// Step 4: Level 3 approval (Executive)
	time.Sleep(2 * time.Second)
	log.Println("\n=== Executive approving (Level 3) ===")
	success, err = client.SendEvent(ctx, "approval-workflow", docID, "approve", map[string]interface{}{
		"approver_email": "ceo@example.com",
		"approver_role":  "executive",
		"comments":       "Final approval granted.",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Approved: %v", success)

	// Query final state
	time.Sleep(1 * time.Second)
	state, err := client.QueryInstance(ctx, "approval-workflow", docID)
	if err != nil {
		log.Printf("Failed to query document: %v", err)
		return
	}
	log.Printf("\n=== Final State ===")
	log.Printf("Current state: %v", state["currentState"])
	log.Printf("Status: %v", state["status"])
}
