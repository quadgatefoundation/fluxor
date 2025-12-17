package bus

import (
	"context"
	"errors"
	"sync"

	"github.com/fluxor-io/fluxor/pkg/types"
	"github.com/google/uuid"
)

var ErrNoSubscribers = errors.New("bus: no subscribers for topic")

// Bus is the interface for the event bus.
type Bus interface {
	types.Bus
	SetReactorProvider(rp ReactorProvider)
}

type subscription struct {
	componentName string
	mailbox       types.Mailbox
}

type localBus struct {
	subscribers    map[string][]*subscription
	mu             sync.RWMutex
	reactorProvider ReactorProvider
}

func NewBus() Bus {
	return &localBus{
		subscribers: make(map[string][]*subscription),
	}
}

func (b *localBus) SetReactorProvider(rp ReactorProvider) {
	b.reactorProvider = rp
}

func (b *localBus) Publish(topic string, msg types.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for _, sub := range subscribers {
			// This is a fire-and-forget, non-blocking send.
			// If the subscriber's mailbox is full, the message is dropped.
			select {
			case sub.mailbox <- msg:
			default:
			}
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

	if b.reactorProvider == nil {
		return errors.New("bus: reactor provider not set")
	}
	reactor, found := b.reactorProvider.GetReactor(selectedSub.componentName)
	if !found {
		// This indicates a configuration error in the system.
		return errors.New("bus: reactor not found for component: " + selectedSub.componentName)
	}

	// Execute the send on the recipient's reactor to ensure serial processing.
	return reactor.Execute(func() {
		// Non-blocking send to the component's mailbox.
		select {
		case selectedSub.mailbox <- msg:
		default:
			// The recipient's mailbox is full, we don't block the sender's reactor.
			// The error from Execute (ErrBackpressure) will be propagated to the sender.
		}
	})
}

func (b *localBus) Request(ctx context.Context, topic string, msg types.Message) (types.Message, error) {
	replyTopic := newReplyTopic()
	msg.ReplyTo = replyTopic

	replyMailbox := make(types.Mailbox, 1)
	// Reply subscriptions are not associated with a component reactor.
	// The requester blocks on the reply channel directly.
	if err := b.Subscribe(replyTopic, "", replyMailbox); err != nil {
		return types.Message{}, err
	}
	defer b.Unsubscribe(replyTopic, "", replyMailbox)

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
