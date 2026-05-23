package middleware

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	successes   int
	threshold   int
	timeout     time.Duration
	openedAt    time.Time
	halfOpenMax int
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 1,
		state:       StateClosed,
	}
}

func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.openedAt) >= cb.timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			fmt.Println("circuit breaker: half-open, probing upstream")
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.state = StateClosed
			fmt.Println("circuit breaker: closed, upstream recovered")
		}
	}
}

func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.openedAt = time.Now()
		fmt.Println("circuit breaker: open, probe failed")
		return
	}

	if cb.failures >= cb.threshold {
		cb.state = StateOpen
		cb.openedAt = time.Now()
		fmt.Printf("circuit breaker: open after %d failures\n", cb.failures)
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
