package verticles

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/examples/tcp-http-load-balancing/contracts"
	"github.com/fluxorio/fluxor/pkg/core"
)

// WorkerVerticle processes work requests from the master
// Each worker listens on a unique address for manual load balancing
type WorkerVerticle struct {
	*core.BaseVerticle
	id        string
	logger    core.Logger
	jobsCount int64
}

// NewWorkerVerticle creates a new worker verticle
func NewWorkerVerticle(id string) *WorkerVerticle {
	return &WorkerVerticle{
		BaseVerticle: core.NewBaseVerticle(fmt.Sprintf("worker-%s", id)),
		id:           id,
		logger:       core.NewDefaultLogger(),
	}
}

// doStart initializes the worker and registers its EventBus consumer
func (v *WorkerVerticle) doStart(ctx core.FluxorContext) error {
	v.logger.Infof("[Worker-%s] Starting...", v.id)

	// Register consumer on worker-specific address for load-balanced requests
	myAddress := fmt.Sprintf("%s.%s", contracts.WorkerAddress, v.id)

	v.Consumer(myAddress).Handler(func(c core.FluxorContext, msg core.Message) error {
		var req contracts.WorkRequest
		if err := msg.DecodeBody(&req); err != nil {
			v.logger.Errorf("[Worker-%s] Invalid request body: %v", v.id, err)
			return msg.Fail(400, "Invalid request body")
		}

		v.logger.Infof("[Worker-%s] Processing job %s from %s", v.id, req.ID, req.Source)

		startTime := time.Now()

		// Simulate processing work based on priority
		processingTime := time.Duration(50+req.Priority*10) * time.Millisecond
		time.Sleep(processingTime)

		// Increment job counter
		atomic.AddInt64(&v.jobsCount, 1)

		duration := time.Since(startTime).Milliseconds()

		resp := contracts.WorkResponse{
			ID:       req.ID,
			Result:   fmt.Sprintf("Processed '%s' by worker-%s", req.Payload, v.id),
			Worker:   v.id,
			Duration: duration,
		}

		v.logger.Infof("[Worker-%s] Completed job %s in %dms", v.id, req.ID, duration)
		return msg.Reply(resp)
	})

	// Register health status consumer
	statusAddress := fmt.Sprintf("%s.%s.status", contracts.WorkerAddress, v.id)
	v.Consumer(statusAddress).Handler(func(c core.FluxorContext, msg core.Message) error {
		status := contracts.WorkerStatus{
			ID:        v.id,
			Active:    true,
			JobsCount: atomic.LoadInt64(&v.jobsCount),
		}
		return msg.Reply(status)
	})

	v.logger.Infof("[Worker-%s] Started, listening on %s", v.id, myAddress)
	return nil
}

// doStop gracefully stops the worker
func (v *WorkerVerticle) doStop(ctx core.FluxorContext) error {
	v.logger.Infof("[Worker-%s] Stopping... Processed %d jobs", v.id, atomic.LoadInt64(&v.jobsCount))
	return nil
}

// GetJobsCount returns the number of jobs processed
func (v *WorkerVerticle) GetJobsCount() int64 {
	return atomic.LoadInt64(&v.jobsCount)
}

// GetID returns the worker ID
func (v *WorkerVerticle) GetID() string {
	return v.id
}
