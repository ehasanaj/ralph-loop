package loop

import "time"

// Config holds configuration for the loop runner
type Config struct {
	Timeout       time.Duration // Per-step timeout (default: 30m)
	MaxRetries    int           // Max retry attempts per step (default: 3)
	RetryDelay    time.Duration // Initial delay between retries (default: 5s)
	BackoffFactor float64       // Multiplier for exponential backoff (default: 2.0)
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Timeout:       30 * time.Minute,
		MaxRetries:    3,
		RetryDelay:    5 * time.Second,
		BackoffFactor: 2.0,
	}
}
