package worker

import (
	"context"
	"errors"
)

var (
	ErrBackpressure = errors.New("worker queue is full")
)

type job struct {
	ctx context.Context
	fn  func(context.Context) (any, error)
	ret chan<- result
}

type result struct {
	val any
	err error
}

type Pool struct {
	jobs chan job
	stop chan struct{}
}

func NewPool(size int, queue int) *Pool {
	if size <= 0 {
		size = 1
	}
	if queue <= 0 {
		queue = 128
	}

	p := &Pool{
		jobs: make(chan job, queue),
		stop: make(chan struct{}),
	}

	for i := 0; i < size; i++ {
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	for {
		select {
		case j := <-p.jobs:
			val, err := j.fn(j.ctx)
			j.ret <- result{val, err}
		case <-p.stop:
			return
		}
	}
}

func (p *Pool) Stop() {
	close(p.stop)
}

func (p *Pool) Submit(ctx context.Context, j func(context.Context) (any, error)) (any, error) {
	ret := make(chan result, 1)
	select {
	case p.jobs <- job{ctx, j, ret}:
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res := <-ret:
			return res.val, res.err
		}
	default:
		return nil, ErrBackpressure
	}
}
