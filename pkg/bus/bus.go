package bus

import (
	"context"
	"fmt"
	"github.com/example/goreactor/pkg/runtime"
	"sync"
	"time"
)

type Message interface{}

type LocalBus struct {
	rt         *runtime.Runtime
	mu         sync.RWMutex
	consumers map[string][]func(Message)
}

func NewLocalBus(rt *runtime.Runtime) *LocalBus {
	return &LocalBus{
		rt:        rt,
		consumers: make(map[string][]func(Message)),
	}
}

func (b *LocalBus) Consumer(address string, handler func(Message)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consumers[address] = append(b.consumers[address], handler)
}

func (b *LocalBus) Publish(address string, payload any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, handler := range b.consumers[address] {
		h := handler
		r := b.rt.ReactorForKey(address)
		r.Post(func() {
			h(payload)
		})
	}
}

func (b *LocalBus) Send(address string, payload any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.consumers[address]) > 0 {
		h := b.consumers[address][0]
		r := b.rt.ReactorForKey(address)
		r.Post(func() {
			h(payload)
		})
	}
}

func (b *LocalBus) Request(ctx context.Context, address string, payload any, timeout time.Duration) (any, error) {
	return nil, fmt.Errorf("not implemented") // To be implemented
}
