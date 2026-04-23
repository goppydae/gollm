package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventBus_SubscribePublish(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	received := make([]string, 0)
	var mu sync.Mutex

	handler := func(ev any) {
		mu.Lock()
		received = append(received, ev.(string))
		mu.Unlock()
		wg.Done()
	}

	bus.Subscribe(handler)
	bus.Subscribe(handler)

	bus.Publish("hello")

	// Wait for delivery with timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for events")
	}

	if len(received) != 2 {
		t.Errorf("expected 2 received events, got %d", len(received))
	}
	if received[0] != "hello" || received[1] != "hello" {
		t.Errorf("received wrong events: %v", received)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	count := 0
	var mu sync.Mutex
	handler := func(ev any) {
		mu.Lock()
		count++
		mu.Unlock()
	}

	unsub := bus.Subscribe(handler)
	bus.Publish("event 1")
	
	// Give some time for async delivery
	time.Sleep(100 * time.Millisecond)
	
	unsub()
	bus.Publish("event 2")
	
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
	mu.Unlock()
}
