package agent

import (
	"fmt"
	"sync"
)

// LifecycleState identifies the current operational state of the agent.
type LifecycleState string

const (
	StateIdle       LifecycleState = "idle"
	StateThinking   LifecycleState = "thinking"
	StateExecuting  LifecycleState = "executing"
	StateCompacting LifecycleState = "compacting"
	StateAborting   LifecycleState = "aborting"
	StateError      LifecycleState = "error"
)

// StateTransition represents a transition between two states.
type StateTransition struct {
	From LifecycleState
	To   LifecycleState
}

// StateMachine manages agent states and transitions.
type StateMachine struct {
	mu           sync.RWMutex
	current      LifecycleState
	onTransition func(StateTransition)
}

// NewStateMachine creates a new state machine.
func NewStateMachine(initial LifecycleState, onTransition func(StateTransition)) *StateMachine {
	return &StateMachine{
		current:      initial,
		onTransition: onTransition,
	}
}

// Current returns the current lifecycle state.
func (s *StateMachine) Current() LifecycleState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Transition moves the state machine to a new state.
func (s *StateMachine) Transition(to LifecycleState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	from := s.current
	if from == to {
		return nil
	}

	// Basic validation of transitions
	if err := s.validateTransition(from, to); err != nil {
		return err
	}

	s.current = to
	if s.onTransition != nil {
		s.onTransition(StateTransition{From: from, To: to})
	}
	return nil
}

func (s *StateMachine) validateTransition(from, to LifecycleState) error {
	switch from {
	case StateIdle:
		// From Idle, we can start thinking, executing (for manual tool calls), or compacting.
		if to == StateThinking || to == StateExecuting || to == StateCompacting {
			return nil
		}
	case StateThinking:
		// From Thinking, we can move to execution, back to idle, or handle interruptions.
		if to == StateExecuting || to == StateIdle || to == StateAborting || to == StateError || to == StateCompacting {
			return nil
		}
	case StateExecuting:
		// From Executing, we move back to thinking or handle interruptions.
		if to == StateThinking || to == StateIdle || to == StateAborting || to == StateError || to == StateCompacting {
			return nil
		}
	case StateCompacting:
		// Compacting is a side-state; it can return to anywhere.
		return nil
	case StateAborting, StateError:
		// Once in terminal/interrupted state, must go back to Idle.
		if to == StateIdle {
			return nil
		}
	}

	return fmt.Errorf("invalid lifecycle transition: %s -> %s", from, to)
}
