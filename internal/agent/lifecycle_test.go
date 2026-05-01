package agent

import (
	"testing"
)

func TestStateMachine_Transition(t *testing.T) {
	var lastTransition StateTransition
	sm := NewStateMachine(StateIdle, func(st StateTransition) {
		lastTransition = st
	})

	if sm.Current() != StateIdle {
		t.Errorf("expected initial state to be Idle, got %s", sm.Current())
	}

	err := sm.Transition(StateThinking)
	if err != nil {
		t.Errorf("unexpected error on transition: %v", err)
	}

	if sm.Current() != StateThinking {
		t.Errorf("expected state to be Thinking, got %s", sm.Current())
	}

	if lastTransition.From != StateIdle || lastTransition.To != StateThinking {
		t.Errorf("incorrect transition record: %+v", lastTransition)
	}
}

func TestStateMachine_SameState(t *testing.T) {
	called := false
	sm := NewStateMachine(StateIdle, func(st StateTransition) {
		called = true
	})

	err := sm.Transition(StateIdle)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called {
		t.Error("callback should not be called for same-state transition")
	}
}

func TestStateMachine_InvalidTransition(t *testing.T) {
	sm := NewStateMachine(StateIdle, nil)

	// Idle -> Aborting is invalid
	err := sm.Transition(StateAborting)
	if err == nil {
		t.Error("expected error for Idle -> Aborting transition, got nil")
	}

	// Move to Thinking
	_ = sm.Transition(StateThinking)

	// Thinking -> Executing is valid
	err = sm.Transition(StateExecuting)
	if err != nil {
		t.Errorf("unexpected error for Thinking -> Executing: %v", err)
	}
}
