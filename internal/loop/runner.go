package loop

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eraldohasanaj/ralph-loop/internal/agent"
	"github.com/eraldohasanaj/ralph-loop/internal/plan"
	"github.com/eraldohasanaj/ralph-loop/internal/prompt"
)

// Runner orchestrates the main execution loop
type Runner struct {
	agent    agent.Agent
	planPath string
}

// NewRunner creates a new loop runner
func NewRunner(a agent.Agent, planPath string) *Runner {
	return &Runner{
		agent:    a,
		planPath: planPath,
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

		// Print status
		fmt.Printf("\n=== Running Step %d: %s ===\n", step.Number, step.Description)
		if step.Status == plan.StatusFailed {
			fmt.Println("(Retrying failed step)")
		}
		fmt.Println()

		// Build prompt
		promptText := prompt.Build(p, step)

		// Run agent
		output, err := r.agent.Run(ctx, promptText, os.Stdout)
		if err != nil {
			if ctx.Err() != nil {
				// Cancelled - save current state and exit
				return r.saveInterruptedState(step)
			}
			return fmt.Errorf("agent execution failed: %w", err)
		}

		// Parse result
		result := prompt.ParseResult(output)

		// Update plan
		if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
			return fmt.Errorf("failed to update plan: %w", err)
		}

		// Print result
		if result.Success {
			fmt.Printf("\n=== Step %d completed successfully ===\n", step.Number)
		} else {
			fmt.Printf("\n=== Step %d failed: %s ===\n", step.Number, result.Reason)
			fmt.Println("Will retry on next iteration...")
		}
	}
}

func (r *Runner) saveInterruptedState(step *plan.Step) error {
	fmt.Printf("\nSaving state for Step %d before exit...\n", step.Number)
	result := plan.StepResult{
		Success: false,
		Output:  "Interrupted by user",
		Reason:  "Interrupted by user (Ctrl+C)",
	}
	if err := plan.UpdateStep(r.planPath, step.Number, result); err != nil {
		return fmt.Errorf("failed to save interrupted state: %w", err)
	}
	fmt.Println("State saved. Run ralph-loop again to continue.")
	return nil
}
