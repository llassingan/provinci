package sse

import (
	"sync"
	"testing"
	"time"
)

func TestSubscribeAndPublish(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch := broker.Subscribe("instance-1")

	broker.Publish("instance-1", SSEEvent{
		Type:    "status",
		Status:  "running",
		Message: "hello",
	})

	select {
	case evt := <-ch:
		if evt.Type != "status" || evt.Status != "running" || evt.Message != "hello" {
			t.Fatalf("unexpected event: %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestUnsubscribe(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch := broker.Subscribe("instance-1")
	broker.Unsubscribe("instance-1", ch)

	// channel should be closed after unsubscribe
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed")
	}

	// publishing should not panic
	broker.Publish("instance-1", SSEEvent{Type: "status"})
}

func TestNonBlockingPublish(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch := broker.Subscribe("instance-1")

	// fill up the 64-buffer channel without reading
	for i := 0; i < 64; i++ {
		broker.Publish("instance-1", SSEEvent{Type: "status", Message: "fill"})
	}

	// this should NOT block (select/default drop)
	done := make(chan struct{})
	go func() {
		broker.Publish("instance-1", SSEEvent{Type: "status", Message: "dropped"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on full channel")
	}

	for i := 0; i < 64; i++ {
		evt := <-ch
		if evt.Message == "dropped" {
			t.Fatal("slow subscriber should not receive overflow event")
		}
	}
}

func TestConcurrentPublishSubscribe(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	var wg sync.WaitGroup
	n := 10

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ch := broker.Subscribe("shared")
			time.Sleep(10 * time.Millisecond)
			broker.Unsubscribe("shared", ch)
		}(i)
	}

	var pubWg sync.WaitGroup
	for i := 0; i < 5; i++ {
		pubWg.Add(1)
		go func() {
			defer pubWg.Done()
			for j := 0; j < 100; j++ {
				broker.Publish("shared", SSEEvent{Type: "status"})
			}
		}()
	}

	wg.Wait()
	pubWg.Wait()
}

func TestMultipleInstances(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch1 := broker.Subscribe("inst-a")
	ch2 := broker.Subscribe("inst-b")

	broker.Publish("inst-a", SSEEvent{Type: "status", Message: "a"})
	broker.Publish("inst-b", SSEEvent{Type: "status", Message: "b"})

	evt1 := <-ch1
	evt2 := <-ch2

	if evt1.Message != "a" {
		t.Fatalf("inst-a got wrong message: %s", evt1.Message)
	}
	if evt2.Message != "b" {
		t.Fatalf("inst-b got wrong message: %s", evt2.Message)
	}
}

func TestCloseCleansUp(t *testing.T) {
	broker := NewEventBroker()

	ch := broker.Subscribe("instance-1")
	broker.Close()

	// channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed after broker.Close()")
	}
}

func TestDefaultTimestamp(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch := broker.Subscribe("inst")
	broker.Publish("inst", SSEEvent{Type: "status"})

	evt := <-ch
	if evt.Timestamp == 0 {
		t.Fatal("timestamp should not be zero")
	}
}

func TestInstanceIDAutoSet(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch := broker.Subscribe("test-inst")
	broker.Publish("test-inst", SSEEvent{Type: "status"})

	evt := <-ch
	if evt.InstanceID != "test-inst" {
		t.Fatalf("expected InstanceID 'test-inst', got '%s'", evt.InstanceID)
	}
}
