package verticles

import (
	"bufio"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/examples/tcp-http-load-balancing/contracts"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/tcp"
	"github.com/fluxorio/fluxor/pkg/web"
)

// MasterVerticle acts as the load balancer and gateway
// It accepts connections via both HTTP and TCP, and distributes
// work to worker verticles using round-robin load balancing
type MasterVerticle struct {
	*core.BaseVerticle

	// Worker management
	workerIDs []string
	counter   uint64

	// Servers
	httpServer *web.FastHTTPServer
	tcpServer  *tcp.TCPServer

	// Configuration
	httpAddr string
	tcpAddr  string

	// Metrics
	totalProcessed int64
	httpProcessed  int64
	tcpProcessed   int64

	logger core.Logger
}

// NewMasterVerticle creates a new master verticle
func NewMasterVerticle(workerIDs []string, httpAddr, tcpAddr string) *MasterVerticle {
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	if tcpAddr == "" {
		tcpAddr = ":9090"
	}

	return &MasterVerticle{
		BaseVerticle: core.NewBaseVerticle("master"),
		workerIDs:    workerIDs,
		httpAddr:     httpAddr,
		tcpAddr:      tcpAddr,
		logger:       core.NewDefaultLogger(),
	}
}

// doStart initializes the master - starts HTTP and TCP servers
func (v *MasterVerticle) doStart(ctx core.FluxorContext) error {
	v.logger.Info("[Master] Starting with workers:", v.workerIDs)

	// 1. Start HTTP Server
	if err := v.startHTTPServer(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// 2. Start TCP Server
	if err := v.startTCPServer(ctx); err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	// 3. Register master status consumer
	v.Consumer(contracts.MasterAddress).Handler(func(c core.FluxorContext, msg core.Message) error {
		status := contracts.MasterStatus{
			WorkerCount:    len(v.workerIDs),
			ActiveWorkers:  v.workerIDs,
			TotalProcessed: atomic.LoadInt64(&v.totalProcessed),
			HTTPAddr:       v.httpAddr,
			TCPAddr:        v.tcpAddr,
		}
		return msg.Reply(status)
	})

	v.logger.Info("[Master] Started successfully")
	v.logger.Infof("[Master] HTTP Server: %s", v.httpAddr)
	v.logger.Infof("[Master] TCP Server: %s", v.tcpAddr)

	return nil
}

// doStop gracefully stops the master
func (v *MasterVerticle) doStop(ctx core.FluxorContext) error {
	v.logger.Info("[Master] Stopping...")

	var errs []error

	if v.httpServer != nil {
		if err := v.httpServer.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("HTTP stop error: %w", err))
		}
	}

	if v.tcpServer != nil {
		if err := v.tcpServer.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("TCP stop error: %w", err))
		}
	}

	v.logger.Infof("[Master] Stopped. Total processed: %d (HTTP: %d, TCP: %d)",
		atomic.LoadInt64(&v.totalProcessed),
		atomic.LoadInt64(&v.httpProcessed),
		atomic.LoadInt64(&v.tcpProcessed))

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// startHTTPServer initializes and starts the HTTP server with REST endpoints
func (v *MasterVerticle) startHTTPServer(ctx core.FluxorContext) error {
	cfg := web.DefaultFastHTTPServerConfig(v.httpAddr)
	v.httpServer = web.NewFastHTTPServer(ctx.Vertx(), cfg)

	router := v.httpServer.FastRouter()

	// GET /health - Health check endpoint
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":   "healthy",
			"workers":  len(v.workerIDs),
			"http":     v.httpAddr,
			"tcp":      v.tcpAddr,
			"requests": atomic.LoadInt64(&v.totalProcessed),
		})
	})

	// GET /status - Detailed master status
	router.GETFast("/status", func(c *web.FastRequestContext) error {
		return c.JSON(200, contracts.MasterStatus{
			WorkerCount:    len(v.workerIDs),
			ActiveWorkers:  v.workerIDs,
			TotalProcessed: atomic.LoadInt64(&v.totalProcessed),
			HTTPAddr:       v.httpAddr,
			TCPAddr:        v.tcpAddr,
		})
	})

	// GET /process - Process a request via HTTP (query param: data)
	router.GETFast("/process", func(c *web.FastRequestContext) error {
		data := c.Query("data")
		if data == "" {
			data = "default-payload"
		}

		priority := 1 // Default priority
		if c.Query("priority") != "" {
			fmt.Sscanf(c.Query("priority"), "%d", &priority)
		}

		// Select worker using round-robin
		workerAddr := v.nextWorkerAddress()

		req := contracts.WorkRequest{
			ID:       fmt.Sprintf("http-%d", time.Now().UnixNano()),
			Payload:  data,
			Source:   "http",
			Priority: priority,
		}

		// Request to worker via EventBus
		reply, err := c.EventBus.Request(workerAddr, req, 10*time.Second)
		if err != nil {
			v.logger.Errorf("[Master] Worker request failed: %v", err)
			return c.JSON(503, map[string]string{
				"error":  "worker_unavailable",
				"detail": err.Error(),
			})
		}

		var resp contracts.WorkResponse
		if err := reply.DecodeBody(&resp); err != nil {
			return c.JSON(500, map[string]string{"error": "invalid_response"})
		}

		atomic.AddInt64(&v.totalProcessed, 1)
		atomic.AddInt64(&v.httpProcessed, 1)

		return c.JSON(200, resp)
	})

	// POST /process - Process a request via HTTP (JSON body)
	router.POSTFast("/process", func(c *web.FastRequestContext) error {
		var req contracts.WorkRequest
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(400, map[string]string{"error": "invalid_request_body"})
		}

		if req.ID == "" {
			req.ID = fmt.Sprintf("http-%d", time.Now().UnixNano())
		}
		req.Source = "http"

		// Select worker using round-robin
		workerAddr := v.nextWorkerAddress()

		// Request to worker via EventBus
		reply, err := c.EventBus.Request(workerAddr, req, 10*time.Second)
		if err != nil {
			v.logger.Errorf("[Master] Worker request failed: %v", err)
			return c.JSON(503, map[string]string{
				"error":  "worker_unavailable",
				"detail": err.Error(),
			})
		}

		var resp contracts.WorkResponse
		if err := reply.DecodeBody(&resp); err != nil {
			return c.JSON(500, map[string]string{"error": "invalid_response"})
		}

		atomic.AddInt64(&v.totalProcessed, 1)
		atomic.AddInt64(&v.httpProcessed, 1)

		return c.JSON(200, resp)
	})

	// GET /workers - List all workers
	router.GETFast("/workers", func(c *web.FastRequestContext) error {
		workers := make([]map[string]interface{}, 0, len(v.workerIDs))
		for _, id := range v.workerIDs {
			workers = append(workers, map[string]interface{}{
				"id":      id,
				"address": fmt.Sprintf("%s.%s", contracts.WorkerAddress, id),
			})
		}
		return c.JSON(200, map[string]interface{}{
			"count":   len(v.workerIDs),
			"workers": workers,
		})
	})

	// Start HTTP server in goroutine (non-blocking)
	go func() {
		v.logger.Infof("[Master] HTTP Server starting on %s", v.httpAddr)
		if err := v.httpServer.Start(); err != nil {
			v.logger.Errorf("[Master] HTTP Server error: %v", err)
		}
	}()

	return nil
}

// startTCPServer initializes and starts the TCP server
func (v *MasterVerticle) startTCPServer(ctx core.FluxorContext) error {
	cfg := tcp.DefaultTCPServerConfig(v.tcpAddr)
	cfg.MaxQueue = 500
	cfg.Workers = 25
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second

	v.tcpServer = tcp.NewTCPServer(ctx.Vertx(), cfg)

	// Set TCP connection handler
	v.tcpServer.SetHandler(func(connCtx *tcp.ConnContext) error {
		return v.handleTCPConnection(connCtx)
	})

	// Start TCP server in goroutine (blocking call)
	go func() {
		v.logger.Infof("[Master] TCP Server starting on %s", v.tcpAddr)
		if err := v.tcpServer.Start(); err != nil {
			v.logger.Errorf("[Master] TCP Server error: %v", err)
		}
	}()

	return nil
}

// handleTCPConnection handles a single TCP connection
// Protocol: Line-based text protocol
// Request: <payload>\n
// Response: <result>\n
func (v *MasterVerticle) handleTCPConnection(connCtx *tcp.ConnContext) error {
	reader := bufio.NewReader(connCtx.Conn)

	// Read line from client
	line, err := reader.ReadString('\n')
	if err != nil {
		v.logger.Errorf("[Master] TCP read error: %v", err)
		return err
	}

	payload := strings.TrimSpace(line)
	if payload == "" {
		connCtx.Conn.Write([]byte("ERROR: empty payload\n"))
		return nil
	}

	// Handle special commands
	if payload == "PING" {
		connCtx.Conn.Write([]byte("PONG\n"))
		return nil
	}

	if payload == "STATUS" {
		status := fmt.Sprintf("MASTER: workers=%d, processed=%d\n",
			len(v.workerIDs), atomic.LoadInt64(&v.totalProcessed))
		connCtx.Conn.Write([]byte(status))
		return nil
	}

	// Select worker using round-robin
	workerAddr := v.nextWorkerAddress()

	req := contracts.WorkRequest{
		ID:       fmt.Sprintf("tcp-%d", time.Now().UnixNano()),
		Payload:  payload,
		Source:   "tcp",
		Priority: 1,
	}

	// Request to worker via EventBus
	reply, err := connCtx.EventBus.Request(workerAddr, req, 10*time.Second)
	if err != nil {
		v.logger.Errorf("[Master] TCP worker request failed: %v", err)
		connCtx.Conn.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
		return nil
	}

	var resp contracts.WorkResponse
	if err := reply.DecodeBody(&resp); err != nil {
		connCtx.Conn.Write([]byte("ERROR: invalid worker response\n"))
		return nil
	}

	atomic.AddInt64(&v.totalProcessed, 1)
	atomic.AddInt64(&v.tcpProcessed, 1)

	// Send response
	response := fmt.Sprintf("OK: worker=%s, result=%s, duration=%dms\n",
		resp.Worker, resp.Result, resp.Duration)
	connCtx.Conn.Write([]byte(response))

	return nil
}

// nextWorkerAddress returns the next worker's EventBus address using round-robin
func (v *MasterVerticle) nextWorkerAddress() string {
	idx := atomic.AddUint64(&v.counter, 1) % uint64(len(v.workerIDs))
	workerID := v.workerIDs[idx]
	return fmt.Sprintf("%s.%s", contracts.WorkerAddress, workerID)
}

// GetHTTPAddr returns the HTTP server address
func (v *MasterVerticle) GetHTTPAddr() string {
	return v.httpAddr
}

// GetTCPAddr returns the TCP server address
func (v *MasterVerticle) GetTCPAddr() string {
	return v.tcpAddr
}

// GetTotalProcessed returns the total number of processed requests
func (v *MasterVerticle) GetTotalProcessed() int64 {
	return atomic.LoadInt64(&v.totalProcessed)
}
