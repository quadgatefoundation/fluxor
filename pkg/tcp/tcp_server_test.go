package tcp

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewTCPServer_FailFast_NilVertxPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for nil vertx")
		}
	}()
	_ = NewTCPServer(nil, DefaultTCPServerConfig(":0"))
}

func TestTCPServer_SetHandler_FailFast_NilPanics(t *testing.T) {
	t.Parallel()
	vertx := core.NewVertx(context.Background())
	s := NewTCPServer(vertx, DefaultTCPServerConfig("127.0.0.1:0"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for nil handler")
		}
	}()
	s.SetHandler(nil)
}

func TestTCPServer_StartStop_HandlesConnection(t *testing.T) {
	vertx := core.NewVertx(context.Background())
	cfg := DefaultTCPServerConfig("127.0.0.1:0")
	cfg.Workers = 2
	cfg.MaxQueue = 10
	cfg.ReadTimeout = 250 * time.Millisecond
	cfg.WriteTimeout = 250 * time.Millisecond

	s := NewTCPServer(vertx, cfg)

	var handled int64
	s.SetHandler(func(ctx *ConnContext) error {
		buf := make([]byte, 1)
		_, _ = ctx.Conn.Read(buf) // best-effort
		atomic.AddInt64(&handled, 1)
		return nil
	})

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- s.Start()
	}()

	// Wait until listener is up and we have an actual port.
	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr = s.ListeningAddr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		_ = s.Stop()
		t.Fatalf("server did not start listening in time")
	}

	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		_ = s.Stop()
		t.Fatalf("dial failed: %v", err)
	}
	_, _ = conn.Write([]byte{0x01})
	_ = conn.Close()

	// Wait for handler to run.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&handled) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&handled) == 0 {
		_ = s.Stop()
		t.Fatalf("expected handler to be called")
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not exit after stop")
	}
}

func TestTCPServer_FailFast_BackpressureRejectsConnections(t *testing.T) {
	vertx := core.NewVertx(context.Background())
	cfg := DefaultTCPServerConfig("127.0.0.1:0")
	cfg.Workers = 1
	cfg.MaxQueue = 1
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second

	s := NewTCPServer(vertx, cfg)

	block := make(chan struct{})
	s.SetHandler(func(ctx *ConnContext) error {
		<-block // block to keep load high
		return nil
	})

	startErrCh := make(chan error, 1)
	go func() { startErrCh <- s.Start() }()

	// Wait until listener is up and we have an actual port.
	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr = s.ListeningAddr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		close(block)
		_ = s.Stop()
		t.Fatalf("server did not start listening in time")
	}

	// Create enough connections to exceed (workers + queue) baseline and force rejection.
	var conns []net.Conn
	for i := 0; i < 10; i++ {
		c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conns = append(conns, c)
		}
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.Metrics().RejectedConnections > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.Metrics().RejectedConnections == 0 {
		for _, c := range conns {
			_ = c.Close()
		}
		close(block)
		_ = s.Stop()
		t.Fatalf("expected at least one rejected connection under backpressure")
	}

	for _, c := range conns {
		_ = c.Close()
	}
	close(block)

	if err := s.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not exit after stop")
	}
}

func TestTCPServer_FailFast_PanicIsolation(t *testing.T) {
	vertx := core.NewVertx(context.Background())
	cfg := DefaultTCPServerConfig("127.0.0.1:0")
	cfg.Workers = 1
	cfg.MaxQueue = 5
	cfg.ReadTimeout = 250 * time.Millisecond
	cfg.WriteTimeout = 250 * time.Millisecond

	s := NewTCPServer(vertx, cfg)

	var calls int64
	s.SetHandler(func(ctx *ConnContext) error {
		n := atomic.AddInt64(&calls, 1)
		if n == 1 {
			panic("boom")
		}
		return nil
	})

	startErrCh := make(chan error, 1)
	go func() { startErrCh <- s.Start() }()

	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr = s.ListeningAddr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		_ = s.Stop()
		t.Fatalf("server did not start listening in time")
	}

	// First connection triggers panic in handler.
	c1, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		_ = s.Stop()
		t.Fatalf("dial1 failed: %v", err)
	}
	_ = c1.Close()

	// Second connection should still be handled (panic isolated).
	c2, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		_ = s.Stop()
		t.Fatalf("dial2 failed: %v", err)
	}
	_ = c2.Close()

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&calls) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&calls) < 2 {
		_ = s.Stop()
		t.Fatalf("expected second call after panic, got %d", atomic.LoadInt64(&calls))
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not exit after stop")
	}
}
