// Package events provides a typed event bus for the agent.
package events

import (
	"sync"

	"github.com/google/uuid"
)

// Handler is a function that handles an event.
type Handler func(any)

// EventBus is a robust typed event bus with async delivery.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]*subscriber
	closed      bool
}

type subscriber struct {
	id string
	ch chan any
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]*subscriber),
	}
}

// Publish sends an event to all subscribers asynchronously.
func (b *EventBus) Publish(event any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	for _, sub := range b.subscribers {
		sub.ch <- event
	}
}

// Subscribe registers a handler and returns an unsubscribe function.
// Each subscriber gets its own goroutine for async execution.
func (b *EventBus) Subscribe(fn Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New().String()
	sub := &subscriber{
		id: id,
		ch: make(chan any, 1024),
	}
	b.subscribers[id] = sub

	go func() {
		for ev := range sub.ch {
			fn(ev)
		}
	}()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if s, ok := b.subscribers[id]; ok {
			close(s.ch)
			delete(b.subscribers, id)
		}
	}
}

// Close shuts down the event bus and all subscribers.
func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for id, sub := range b.subscribers {
		close(sub.ch)
		delete(b.subscribers, id)
	}
}
