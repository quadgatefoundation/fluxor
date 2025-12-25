package contracts

// EventBus address definitions
const (
	// WorkerAddress is the base address for worker verticles
	WorkerAddress = "worker.process"

	// MasterAddress is the address for master health checks
	MasterAddress = "master.health"
)

// WorkRequest represents a job request sent to workers
type WorkRequest struct {
	ID       string `json:"id"`
	Payload  string `json:"payload"`
	Source   string `json:"source"` // "tcp" or "http"
	Priority int    `json:"priority"`
}

// WorkResponse represents the result from a worker
type WorkResponse struct {
	ID       string `json:"id"`
	Result   string `json:"result"`
	Worker   string `json:"worker"`
	Duration int64  `json:"duration_ms"`
}

// WorkerStatus represents a worker's health status
type WorkerStatus struct {
	ID        string `json:"id"`
	Active    bool   `json:"active"`
	JobsCount int64  `json:"jobs_count"`
}

// MasterStatus represents master's health status
type MasterStatus struct {
	WorkerCount       int      `json:"worker_count"`
	ActiveWorkers     []string `json:"active_workers"`
	TotalProcessed    int64    `json:"total_processed"`
	HTTPAddr          string   `json:"http_addr"`
	TCPAddr           string   `json:"tcp_addr"`
}
