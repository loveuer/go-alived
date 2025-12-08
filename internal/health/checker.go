package health

import (
	"context"
	"time"
)

type CheckResult int

const (
	CheckResultUnknown CheckResult = iota
	CheckResultSuccess
	CheckResultFailure
)

func (r CheckResult) String() string {
	switch r {
	case CheckResultSuccess:
		return "SUCCESS"
	case CheckResultFailure:
		return "FAILURE"
	default:
		return "UNKNOWN"
	}
}

type Checker interface {
	Check(ctx context.Context) CheckResult
	Name() string
	Type() string
}

type CheckerConfig struct {
	Name     string
	Type     string
	Interval time.Duration
	Timeout  time.Duration
	Rise     int
	Fall     int
	Config   map[string]interface{}
}

type CheckerState struct {
	Name          string
	Healthy       bool
	LastResult    CheckResult
	LastCheckTime time.Time
	SuccessCount  int
	FailureCount  int
	TotalChecks   int
	ConsecutiveOK int
	ConsecutiveFail int
}

func (s *CheckerState) IsHealthy() bool {
	return s.Healthy
}

func (s *CheckerState) Update(result CheckResult, rise, fall int) bool {
	s.LastResult = result
	s.LastCheckTime = time.Now()
	s.TotalChecks++

	oldHealthy := s.Healthy

	switch result {
	case CheckResultSuccess:
		s.SuccessCount++
		s.ConsecutiveOK++
		s.ConsecutiveFail = 0

		if !s.Healthy && s.ConsecutiveOK >= rise {
			s.Healthy = true
		}

	case CheckResultFailure:
		s.FailureCount++
		s.ConsecutiveFail++
		s.ConsecutiveOK = 0

		if s.Healthy && s.ConsecutiveFail >= fall {
			s.Healthy = false
		}
	}

	return s.Healthy != oldHealthy
}

type StateChangeCallback func(name string, oldHealthy, newHealthy bool)
