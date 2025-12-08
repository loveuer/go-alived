package vrrp

import (
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateInit State = iota
	StateBackup
	StateMaster
	StateFault
)

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

type StateMachine struct {
	currentState     State
	previousState    State
	mu               sync.RWMutex
	stateChangeCallbacks []func(old, new State)
}

func NewStateMachine(initialState State) *StateMachine {
	return &StateMachine{
		currentState:  initialState,
		previousState: StateInit,
		stateChangeCallbacks: make([]func(old, new State), 0),
	}
}

func (sm *StateMachine) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

func (sm *StateMachine) SetState(newState State) {
	sm.mu.Lock()
	oldState := sm.currentState
	sm.previousState = oldState
	sm.currentState = newState
	callbacks := sm.stateChangeCallbacks
	sm.mu.Unlock()

	for _, callback := range callbacks {
		callback(oldState, newState)
	}
}

func (sm *StateMachine) OnStateChange(callback func(old, new State)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stateChangeCallbacks = append(sm.stateChangeCallbacks, callback)
}

type Timer struct {
	duration time.Duration
	timer    *time.Timer
	callback func()
	mu       sync.Mutex
}

func NewTimer(duration time.Duration, callback func()) *Timer {
	return &Timer{
		duration: duration,
		callback: callback,
	}
}

func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}

	t.timer = time.AfterFunc(t.duration, t.callback)
}

func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}

	t.timer = time.AfterFunc(t.duration, t.callback)
}

func (t *Timer) SetDuration(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.duration = duration
}

type PriorityCalculator struct {
	basePriority    uint8
	currentPriority uint8
	mu              sync.RWMutex
}

func NewPriorityCalculator(basePriority uint8) *PriorityCalculator {
	return &PriorityCalculator{
		basePriority:    basePriority,
		currentPriority: basePriority,
	}
}

func (pc *PriorityCalculator) GetPriority() uint8 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.currentPriority
}

func (pc *PriorityCalculator) DecreasePriority(amount uint8) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.currentPriority > amount {
		pc.currentPriority -= amount
	} else {
		pc.currentPriority = 0
	}
}

func (pc *PriorityCalculator) ResetPriority() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.currentPriority = pc.basePriority
}

func (pc *PriorityCalculator) SetBasePriority(priority uint8) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.basePriority = priority
	pc.currentPriority = priority
}

func ShouldBecomeMaster(localPriority, remotePriority uint8, localIP, remoteIP string) bool {
	if localPriority > remotePriority {
		return true
	}

	if localPriority == remotePriority {
		return localIP > remoteIP
	}

	return false
}

func CalculateMasterDownInterval(advertInt uint8) time.Duration {
	return time.Duration(3*int(advertInt)) * time.Second
}

func CalculateSkewTime(priority uint8) time.Duration {
	skew := float64(256-int(priority)) / 256.0
	return time.Duration(skew * float64(time.Second))
}

type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

type StateHistory struct {
	transitions []StateTransition
	maxSize     int
	mu          sync.RWMutex
}

func NewStateHistory(maxSize int) *StateHistory {
	return &StateHistory{
		transitions: make([]StateTransition, 0, maxSize),
		maxSize:     maxSize,
	}
}

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

	if len(sh.transitions) > sh.maxSize {
		sh.transitions = sh.transitions[1:]
	}
}

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