// Package events provides a typed event bus for the agent.
package events

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// subscriberChanSize is the per-subscriber buffer depth. Large enough to absorb
// bursts from a streaming LLM turn without blocking the agent loop.
const subscriberChanSize = 4096

// Handler is a function that handles an event.
type Handler func(any)

// EventBus is a robust typed event bus with async delivery.
//
// A single mutex (not RWMutex) guards all state so that Publish, Subscribe,
// and Close can never race. Publish uses non-blocking channel sends so a slow
// subscriber never stalls the agent loop — events are dropped rather than
// blocking the publisher.
type EventBus struct {
	mu          sync.Mutex
	subscribers map[string]*subscriber
	closed      bool
}

type subscriber struct {
	id      string
	ch      chan any
	closed  bool
	dropped atomic.Int64
}

// Subscription is returned by Subscribe and provides a way to check whether
// events were dropped and to unsubscribe.
type Subscription struct {
	sub         *subscriber
	unsubscribe func()
}

// Unsubscribe removes this subscription from the bus.
func (s *Subscription) Unsubscribe() { s.unsubscribe() }

// DroppedCount returns the number of events dropped for this subscriber since
// it was created. A non-zero value means the subscriber's buffer was full and
// some events were silently discarded.
func (s *Subscription) DroppedCount() int64 { return s.sub.dropped.Load() }

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]*subscriber),
	}
}

// Publish sends an event to all subscribers. The send is non-blocking: if a
// subscriber's buffer is full the event is dropped for that subscriber rather
// than stalling the caller. The first drop per subscriber is noted by
// incrementing its dropped counter.
func (b *EventBus) Publish(event any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	for _, sub := range b.subscribers {
		select {
		case sub.ch <- event:
		default:
			// subscriber is too slow — drop rather than block the agent loop
			sub.dropped.Add(1)
		}
	}
}

// Subscribe registers a handler and returns a Subscription. Each subscriber
// gets its own goroutine for async execution.
func (b *EventBus) Subscribe(fn Handler) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New().String()
	sub := &subscriber{
		id: id,
		ch: make(chan any, subscriberChanSize),
	}
	b.subscribers[id] = sub

	go func() {
		for ev := range sub.ch {
			fn(ev)
		}
	}()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if s, ok := b.subscribers[id]; ok && !s.closed {
			s.closed = true
			close(s.ch)
			delete(b.subscribers, id)
		}
	}
	return &Subscription{sub: sub, unsubscribe: unsub}
}

// Close shuts down the event bus and all subscribers.
func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for id, sub := range b.subscribers {
		if !sub.closed {
			sub.closed = true
			close(sub.ch)
		}
		delete(b.subscribers, id)
	}
}
