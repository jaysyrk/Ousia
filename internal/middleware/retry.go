package middleware

import (
	"fmt"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}
}

func WithRetry(cfg RetryConfig, fn func() error) error {
	var err error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.BaseDelay * time.Duration(1<<attempt)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
			fmt.Printf("retry: attempt %d failed (%v), retrying in %s\n", attempt+1, err, delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all %d attempts failed: %w", cfg.MaxAttempts, err)
}
