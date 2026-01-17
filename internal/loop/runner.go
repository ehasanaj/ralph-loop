package loop

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eraldohasanaj/ralph-loop/internal/agent"
	"github.com/eraldohasanaj/ralph-loop/internal/plan"
	"github.com/eraldohasanaj/ralph-loop/internal/prompt"
)

// Runner orchestrates the main execution loop
type Runner struct {
	agent    agent.Agent
	planPath string
	config   Config
}

// NewRunner creates a new loop runner with default config
func NewRunner(a agent.Agent, planPath string) *Runner {
	return &Runner{
		agent:    a,
		planPath: planPath,
		config:   DefaultConfig(),
	}
}

// NewRunnerWithConfig creates a new loop runner with custom config
func NewRunnerWithConfig(a agent.Agent, planPath string, config Config) *Runner {
	return &Runner{
		agent:    a,
		planPath: planPath,
		config:   config,
	}
}

// Run executes the main loop
func (r *Runner) Run() error {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

	return r.runLoop(ctx)
}

func (r *Runner) runLoop(ctx context.Context) error {
	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Parse the plan
		p, err := plan.ParseFile(r.planPath)
		if err != nil {
			return fmt.Errorf("failed to parse plan: %w", err)
		}

		// Find next step
		step := p.NextStep()
		if step == nil {
			fmt.Println("\n=== All steps completed! ===")
			return nil
		}

		// Check max retries - skip and continue to next step
		if step.RetryCount >= r.config.MaxRetries {
			fmt.Printf("\n=== Step %d exceeded max retries (%d). Marking as skipped. ===\n",
				step.Number, r.config.MaxRetries)
			result := plan.StepResult{
				Success:    false,
				Reason:     fmt.Sprintf("Skipped after %d failed attempts", r.config.MaxRetries),
				Status:     plan.StatusSkipped,
				RetryCount: step.RetryCount,
			}
			if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
				return fmt.Errorf("failed to update plan: %w", err)
			}
			continue // Move to next step
		}

		// Apply backoff delay if retrying
		if step.Status == plan.StatusFailed && step.RetryCount > 0 {
			delay := r.calculateBackoff(step.RetryCount)
			fmt.Printf("\n=== Waiting %v before retry (attempt %d of %d)... ===\n",
				delay, step.RetryCount+1, r.config.MaxRetries)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Print status
		fmt.Printf("\n=== Running Step %d: %s ===\n", step.Number, step.Description)
		if step.Status == plan.StatusFailed {
			fmt.Printf("(Retry attempt %d of %d)\n", step.RetryCount+1, r.config.MaxRetries)
		}
		fmt.Println()

		// Build prompt
		promptText := prompt.Build(p, step)

		// Create timeout context
		stepCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)

		// Run agent
		output, err := r.agent.Run(stepCtx, promptText, os.Stdout)
		cancel()

		// Check for timeout
		if stepCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("\n=== Step %d timed out after %v ===\n", step.Number, r.config.Timeout)
			result := plan.StepResult{
				Success:    false,
				Reason:     fmt.Sprintf("Step timed out after %v", r.config.Timeout),
				RetryCount: step.RetryCount + 1,
			}
			if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
				return fmt.Errorf("failed to update plan: %w", err)
			}
			continue
		}

		if err != nil {
			if ctx.Err() != nil {
				// Parent cancelled - save current state and exit
				return r.saveInterruptedState(step)
			}
			return fmt.Errorf("agent execution failed: %w", err)
		}

		// Parse result
		result := prompt.ParseResult(output)

		// Update retry count on failure
		if !result.Success {
			result.RetryCount = step.RetryCount + 1
		} else {
			result.RetryCount = step.RetryCount
		}

		// Update plan
		if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
			return fmt.Errorf("failed to update plan: %w", err)
		}

		// Print result
		if result.Success {
			fmt.Printf("\n=== Step %d completed successfully ===\n", step.Number)
		} else {
			fmt.Printf("\n=== Step %d failed: %s ===\n", step.Number, result.Reason)
			if result.RetryCount < r.config.MaxRetries {
				fmt.Printf("Will retry (attempt %d of %d)...\n", result.RetryCount+1, r.config.MaxRetries)
			} else {
				fmt.Printf("Max retries reached (%d). Step will be skipped on next iteration.\n", r.config.MaxRetries)
			}
		}
	}
}

// calculateBackoff calculates the backoff delay for a given retry count
func (r *Runner) calculateBackoff(retryCount int) time.Duration {
	delay := r.config.RetryDelay
	for i := 0; i < retryCount-1; i++ {
		delay = time.Duration(float64(delay) * r.config.BackoffFactor)
	}
	return delay
}

func (r *Runner) saveInterruptedState(step *plan.Step) error {
	fmt.Printf("\nSaving state for Step %d before exit...\n", step.Number)
	result := plan.StepResult{
		Success:    false,
		Output:     "Interrupted by user",
		Reason:     "Interrupted by user (Ctrl+C)",
		RetryCount: step.RetryCount,
	}
	if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
		return fmt.Errorf("failed to save interrupted state: %w", err)
	}
	fmt.Println("State saved. Run ralph-loop again to continue.")
	return nil
}
