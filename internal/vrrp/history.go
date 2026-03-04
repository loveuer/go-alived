package vrrp

import (
	"fmt"
	"sync"
	"time"
)

// StateTransition represents a single state transition event.
type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

// StateHistory maintains a bounded history of state transitions.
type StateHistory struct {
	transitions []StateTransition
	maxSize     int
	mu          sync.RWMutex
}

// NewStateHistory creates a new StateHistory with the specified maximum size.
func NewStateHistory(maxSize int) *StateHistory {
	return &StateHistory{
		transitions: make([]StateTransition, 0, maxSize),
		maxSize:     maxSize,
	}
}

// Add records a new state transition.
func (sh *StateHistory) Add(from, to State, reason string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	transition := StateTransition{
		From:      from,
		To:        to,
		Timestamp: time.Now(),
		Reason:    reason,
	}

	sh.transitions = append(sh.transitions, transition)

	// Maintain bounded size using ring buffer style
	if len(sh.transitions) > sh.maxSize {
		// Copy to new slice to allow garbage collection of old backing array
		newTransitions := make([]StateTransition, len(sh.transitions)-1, sh.maxSize)
		copy(newTransitions, sh.transitions[1:])
		sh.transitions = newTransitions
	}
}

// GetRecent returns the most recent n transitions.
func (sh *StateHistory) GetRecent(n int) []StateTransition {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	if n > len(sh.transitions) {
		n = len(sh.transitions)
	}

	start := len(sh.transitions) - n
	result := make([]StateTransition, n)
	copy(result, sh.transitions[start:])

	return result
}

// Len returns the number of recorded transitions.
func (sh *StateHistory) Len() int {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return len(sh.transitions)
}

// String returns a formatted string representation of the history.
func (sh *StateHistory) String() string {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	if len(sh.transitions) == 0 {
		return "No state transitions"
	}

	result := "State transition history:\n"
	for _, t := range sh.transitions {
		result += fmt.Sprintf("  %s: %s -> %s (%s)\n",
			t.Timestamp.Format("2006-01-02 15:04:05"),
			t.From, t.To, t.Reason)
	}

	return result
}
