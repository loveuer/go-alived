package vrrp

import "sync"

// State represents the VRRP instance state.
type State int

const (
	StateInit State = iota
	StateBackup
	StateMaster
	StateFault
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateInit:
		return "INIT"
	case StateBackup:
		return "BACKUP"
	case StateMaster:
		return "MASTER"
	case StateFault:
		return "FAULT"
	default:
		return "UNKNOWN"
	}
}

// StateMachine manages VRRP state transitions with thread-safe callbacks.
type StateMachine struct {
	currentState         State
	mu                   sync.RWMutex
	stateChangeCallbacks []func(old, new State)
}

// NewStateMachine creates a new StateMachine with the specified initial state.
func NewStateMachine(initialState State) *StateMachine {
	return &StateMachine{
		currentState:         initialState,
		stateChangeCallbacks: make([]func(old, new State), 0),
	}
}

// GetState returns the current state.
func (sm *StateMachine) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// SetState transitions to a new state and triggers registered callbacks.
func (sm *StateMachine) SetState(newState State) {
	sm.mu.Lock()
	oldState := sm.currentState
	if oldState == newState {
		sm.mu.Unlock()
		return
	}
	sm.currentState = newState
	callbacks := make([]func(old, new State), len(sm.stateChangeCallbacks))
	copy(callbacks, sm.stateChangeCallbacks)
	sm.mu.Unlock()

	for _, callback := range callbacks {
		callback(oldState, newState)
	}
}

// OnStateChange registers a callback to be invoked on state changes.
func (sm *StateMachine) OnStateChange(callback func(old, new State)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stateChangeCallbacks = append(sm.stateChangeCallbacks, callback)
}
