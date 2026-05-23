package middleware

import (
	"fmt"
	"testing"
	"time"
)

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	for i := 0; i < 3; i++ {
		cb.Failure()
	}

	if cb.State() != StateOpen {
		t.Fatal("expected circuit to be open after threshold failures")
	}

	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)
	cb.Failure()

	if cb.State() != StateOpen {
		t.Fatal("expected open state")
	}

	time.Sleep(60 * time.Millisecond)

	if err := cb.Allow(); err != nil {
		t.Fatalf("expected allow after timeout, got %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Fatal("expected half-open state after timeout")
	}
}

func TestCircuitBreaker_ClosesAfterSuccess(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)
	cb.Failure()

	time.Sleep(60 * time.Millisecond)
	cb.Allow()
	cb.Success()

	if cb.State() != StateClosed {
		t.Fatal("expected closed state after success in half-open")
	}
}

func TestRetry_SucceedsEventually(t *testing.T) {
	attempts := 0
	err := WithRetry(RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("not yet")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
