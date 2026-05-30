package middleware

import (
	"fmt"
	"testing"
	"time"
)

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)
	cb.Failure() // open
	time.Sleep(60 * time.Millisecond)
	cb.Allow() // half open
	cb.Failure() // should return to open

	if cb.State() != StateOpen {
		t.Fatal("expected open state after failure in half-open")
	}
}

func TestRetry_FailsAllAttempts(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 2
	cfg.BaseDelay = 1 * time.Millisecond
	
	err := WithRetry(cfg, func() error {
		return fmt.Errorf("always fail")
	})
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
