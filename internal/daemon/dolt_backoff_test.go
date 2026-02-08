package daemon

import (
	"testing"
	"time"
)

func TestAdvanceBackoff(t *testing.T) {
	m := &DoltServerManager{
		config: &DoltServerConfig{
			RestartDelay:    5 * time.Second,
			MaxRestartDelay: 5 * time.Minute,
		},
		logger: func(format string, v ...interface{}) {},
	}

	// First advance: 5s -> 10s
	m.advanceBackoff()
	if m.currentDelay != 10*time.Second {
		t.Errorf("expected 10s, got %v", m.currentDelay)
	}

	// Second advance: 10s -> 20s
	m.advanceBackoff()
	if m.currentDelay != 20*time.Second {
		t.Errorf("expected 20s, got %v", m.currentDelay)
	}

	// Third: 20s -> 40s
	m.advanceBackoff()
	if m.currentDelay != 40*time.Second {
		t.Errorf("expected 40s, got %v", m.currentDelay)
	}

	// Fourth: 40s -> 80s
	m.advanceBackoff()
	if m.currentDelay != 80*time.Second {
		t.Errorf("expected 80s, got %v", m.currentDelay)
	}

	// Fifth: 80s -> 160s
	m.advanceBackoff()
	if m.currentDelay != 160*time.Second {
		t.Errorf("expected 160s, got %v", m.currentDelay)
	}

	// Sixth: 160s -> 300s (capped at 5min)
	m.advanceBackoff()
	if m.currentDelay != 5*time.Minute {
		t.Errorf("expected 5m0s (cap), got %v", m.currentDelay)
	}

	// Stays capped
	m.advanceBackoff()
	if m.currentDelay != 5*time.Minute {
		t.Errorf("expected 5m0s (still capped), got %v", m.currentDelay)
	}
}

func TestGetBackoffDelay_InitialValue(t *testing.T) {
	m := &DoltServerManager{
		config: &DoltServerConfig{
			RestartDelay: 5 * time.Second,
		},
		logger: func(format string, v ...interface{}) {},
	}

	// Before any advances, should return base delay
	delay := m.getBackoffDelay()
	if delay != 5*time.Second {
		t.Errorf("expected initial delay 5s, got %v", delay)
	}
}

func TestPruneRestartTimes(t *testing.T) {
	now := time.Now()
	m := &DoltServerManager{
		config: &DoltServerConfig{
			RestartWindow: 10 * time.Minute,
		},
		logger: func(format string, v ...interface{}) {},
		restartTimes: []time.Time{
			now.Add(-15 * time.Minute), // Outside window
			now.Add(-11 * time.Minute), // Outside window
			now.Add(-5 * time.Minute),  // Inside window
			now.Add(-1 * time.Minute),  // Inside window
		},
	}

	m.pruneRestartTimes(now)

	if len(m.restartTimes) != 2 {
		t.Errorf("expected 2 times after pruning, got %d", len(m.restartTimes))
	}
}

func TestMaybeResetBackoff(t *testing.T) {
	m := &DoltServerManager{
		config: &DoltServerConfig{
			HealthyResetInterval: 5 * time.Minute,
		},
		logger:       func(format string, v ...interface{}) {},
		currentDelay: 40 * time.Second,
		restartTimes: []time.Time{time.Now()},
		escalated:    true,
	}

	// First call sets lastHealthyTime
	m.maybeResetBackoff()
	if m.currentDelay != 40*time.Second {
		t.Error("should not reset on first healthy check")
	}

	// Simulate time passing (set lastHealthyTime to 6 minutes ago)
	m.lastHealthyTime = time.Now().Add(-6 * time.Minute)
	m.maybeResetBackoff()

	if m.currentDelay != 0 {
		t.Errorf("expected delay reset to 0, got %v", m.currentDelay)
	}
	if m.restartTimes != nil {
		t.Error("expected restartTimes to be nil after reset")
	}
	if m.escalated {
		t.Error("expected escalated to be false after reset")
	}
}

func TestMaybeResetBackoff_NoResetIfNotLongEnough(t *testing.T) {
	m := &DoltServerManager{
		config: &DoltServerConfig{
			HealthyResetInterval: 5 * time.Minute,
		},
		logger:          func(format string, v ...interface{}) {},
		currentDelay:    40 * time.Second,
		lastHealthyTime: time.Now().Add(-2 * time.Minute), // Only 2 min healthy
		restartTimes:    []time.Time{time.Now()},
	}

	m.maybeResetBackoff()

	if m.currentDelay != 40*time.Second {
		t.Errorf("should not reset after only 2 minutes, got delay %v", m.currentDelay)
	}
}

func TestDefaultConfig_BackoffFields(t *testing.T) {
	cfg := DefaultDoltServerConfig("/tmp/test")

	if cfg.MaxRestartDelay != 5*time.Minute {
		t.Errorf("expected MaxRestartDelay 5m, got %v", cfg.MaxRestartDelay)
	}
	if cfg.MaxRestartsInWindow != 5 {
		t.Errorf("expected MaxRestartsInWindow 5, got %d", cfg.MaxRestartsInWindow)
	}
	if cfg.RestartWindow != 10*time.Minute {
		t.Errorf("expected RestartWindow 10m, got %v", cfg.RestartWindow)
	}
	if cfg.HealthyResetInterval != 5*time.Minute {
		t.Errorf("expected HealthyResetInterval 5m, got %v", cfg.HealthyResetInterval)
	}
}
