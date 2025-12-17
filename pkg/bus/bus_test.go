package bus

import (
	"context"
	"testing"

	"github.com/fluxor-io/fluxor/pkg/types"
)

func TestBus(t *testing.T) {
	b := NewBus()
	topic := "test.topic"

	// Test Subscribe and Publish
	mbox1 := make(types.Mailbox, 1)
	mbox2 := make(types.Mailbox, 1)

	b.Subscribe(topic, "component1", mbox1)
	b.Subscribe(topic, "component2", mbox2)

	msg := types.Message{Topic: topic, Payload: "hello"}
	b.Publish(topic, msg)

	receivedMsg1 := <-mbox1
	if receivedMsg1.Payload != "hello" {
		t.Errorf("mbox1 expected 'hello', got %s", receivedMsg1.Payload)
	}

	receivedMsg2 := <-mbox2
	if receivedMsg2.Payload != "hello" {
		t.Errorf("mbox2 expected 'hello', got %s", receivedMsg2.Payload)
	}

	// Test Unsubscribe
	b.Unsubscribe(topic, "component1", mbox1)
	b.Publish(topic, msg)

	select {
	case <-mbox1:
		t.Error("mbox1 should not receive message after unsubscribe")
	default:
	}

	receivedMsg2 = <-mbox2
	if receivedMsg2.Payload != "hello" {
		t.Errorf("mbox2 expected 'hello', got %s", receivedMsg2.Payload)
	}
}

func TestBus_RequestReply(t *testing.T) {
	b := NewBus()
	topic := "test.request"

	// Create a responder
	mbox := make(types.Mailbox, 1)
	b.Subscribe(topic, "responder", mbox)

	go func() {
		req := <-mbox
		reply := types.Message{Topic: req.ReplyTo, Payload: "Re: " + req.Payload.(string), CorrelationID: req.CorrelationID}
		b.Send(req.ReplyTo, reply)
	}()

	// Send a request
	req := types.Message{Payload: "ping"}
	resp, err := b.Request(context.Background(), topic, req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Payload != "Re: ping" {
		t.Errorf("Expected 'Re: ping', got %s", resp.Payload)
	}
}
