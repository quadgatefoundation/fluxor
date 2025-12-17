package bus

import (
	"context"
	"errors"
	"sync"

	"github.com/fluxor-io/fluxor/pkg/types"
	"github.com/google/uuid"
)

var ErrNoSubscribers = errors.New("bus: no subscribers for topic")

type subscription struct {
	componentName string
	mailbox       types.Mailbox
}

type localBus struct {
	subscribers map[string][]*subscription
	mu          sync.RWMutex
}

func NewBus() types.Bus {
	return &localBus{
		subscribers: make(map[string][]*subscription),
	}
}

func (b *localBus) Publish(topic string, msg types.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for _, sub := range subscribers {
			// This is a blocking send. If the subscriber's mailbox is full,
			// the publisher will block until there is space.
			sub.mailbox <- msg
		}
	}
}

func (b *localBus) Subscribe(topic, componentName string, handler types.Mailbox) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	newSub := &subscription{
		componentName: componentName,
		mailbox:       handler,
	}
	b.subscribers[topic] = append(b.subscribers[topic], newSub)
	return nil
}

func (b *localBus) Unsubscribe(topic, componentName string, handler types.Mailbox) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for i, sub := range subscribers {
			if sub.mailbox == handler && sub.componentName == componentName {
				b.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (b *localBus) Send(topic string, msg types.Message) error {
	b.mu.RLock()
	// Basic round-robin selection for load balancing
	subs, ok := b.subscribers[topic]
	if !ok || len(subs) == 0 {
		b.mu.RUnlock()
		return ErrNoSubscribers
	}
	selectedSub := subs[0] // In a real scenario, use a better strategy
	b.mu.RUnlock()

	// Non-blocking send to the component's mailbox.
	select {
	case selectedSub.mailbox <- msg:
		return nil
	default:
		return types.ErrBackpressure
	}
}

func (b *localBus) Request(ctx context.Context, topic string, msg types.Message) (types.Message, error) {
	replyTopic := newReplyTopic()
	msg.ReplyTo = replyTopic
	msg.CorrelationID = uuid.New().String()

	replyMailbox := make(types.Mailbox, 1)
	if err := b.Subscribe(replyTopic, "requestor", replyMailbox); err != nil {
		return types.Message{}, err
	}
	defer b.Unsubscribe(replyTopic, "requestor", replyMailbox)

	if err := b.Send(topic, msg); err != nil {
		return types.Message{}, err
	}

	select {
	case reply := <-replyMailbox:
		return reply, nil
	case <-ctx.Done():
		return types.Message{}, ctx.Err()
	}
}

func newReplyTopic() string {
	return "reply." + uuid.New().String()
}
