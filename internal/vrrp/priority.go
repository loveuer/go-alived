package vrrp

import (
	"sync"
	"time"
)

// PriorityCalculator manages VRRP priority with support for dynamic adjustment.
type PriorityCalculator struct {
	basePriority    uint8
	currentPriority uint8
	mu              sync.RWMutex
}

// NewPriorityCalculator creates a new PriorityCalculator with the specified base priority.
func NewPriorityCalculator(basePriority uint8) *PriorityCalculator {
	return &PriorityCalculator{
		basePriority:    basePriority,
		currentPriority: basePriority,
	}
}

// GetPriority returns the current priority.
func (pc *PriorityCalculator) GetPriority() uint8 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.currentPriority
}

// DecreasePriority decreases the current priority by the specified amount.
// The priority will not go below 0.
func (pc *PriorityCalculator) DecreasePriority(amount uint8) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.currentPriority > amount {
		pc.currentPriority -= amount
	} else {
		pc.currentPriority = 0
	}
}

// IncreasePriority increases the current priority by the specified amount.
// The priority will not exceed 255 or the base priority.
func (pc *PriorityCalculator) IncreasePriority(amount uint8) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	newPriority := pc.currentPriority + amount
	if newPriority > pc.basePriority {
		newPriority = pc.basePriority
	}
	if newPriority < pc.currentPriority { // overflow check
		newPriority = pc.basePriority
	}
	pc.currentPriority = newPriority
}

// ResetPriority resets the priority to the base value.
func (pc *PriorityCalculator) ResetPriority() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.currentPriority = pc.basePriority
}

// SetBasePriority sets a new base priority and resets current priority to match.
func (pc *PriorityCalculator) SetBasePriority(priority uint8) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.basePriority = priority
	pc.currentPriority = priority
}

// ShouldBecomeMaster determines if the local node should become master
// based on priority comparison and IP address tie-breaking.
func ShouldBecomeMaster(localPriority, remotePriority uint8, localIP, remoteIP string) bool {
	if localPriority > remotePriority {
		return true
	}

	if localPriority == remotePriority {
		return localIP > remoteIP
	}

	return false
}

// CalculateMasterDownInterval calculates the master down interval
// according to VRRP specification: (3 * Advertisement_Interval).
func CalculateMasterDownInterval(advertInt uint8) time.Duration {
	return time.Duration(3*int(advertInt)) * time.Second
}

// CalculateSkewTime calculates the skew time for master down timer
// according to VRRP specification: ((256 - Priority) / 256).
func CalculateSkewTime(priority uint8) time.Duration {
	skew := float64(256-int(priority)) / 256.0
	return time.Duration(skew * float64(time.Second))
}
