package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eraldohasanaj/ralph-loop/internal/agent"
	"github.com/eraldohasanaj/ralph-loop/internal/loop"
	"github.com/eraldohasanaj/ralph-loop/internal/plan"
)

// splitLines splits a string into lines
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

var (
	// Version info (set via ldflags)
	version = "dev"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ralph-loop",
	Short: "Meta-orchestrator for AI coding agents",
	Long: `ralph-loop runs AI coding agents (opencode or claude) in a loop to implement a plan.

It solves the context-bloat problem by:
  - Maintaining state in a plan.md file (source of truth)
  - Running one AI session per step
  - Updating the plan with results after each step
  - Starting a fresh session for the next step`,
	Version: version,
}

// Run command
var (
	runAgentType string
	runPlanPath  string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start or continue executing the plan",
	Long: `Run the plan loop, executing each step with the specified AI agent.

The loop will:
  1. Parse the plan file to find the next pending or failed step
  2. Build a prompt with the step context
  3. Run the AI agent with the prompt
  4. Parse the output for STEP_COMPLETE or STEP_FAILED markers
  5. Update the plan file with results
  6. Continue to the next step or retry if failed

Press Ctrl+C to gracefully stop the loop.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse agent type
		agentType, err := agent.ParseAgentType(runAgentType)
		if err != nil {
			return err
		}

		// Create agent
		a, err := agent.New(agentType)
		if err != nil {
			return err
		}

		// Check plan file exists
		if _, err := os.Stat(runPlanPath); os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'ralph-loop init' to create one", runPlanPath)
		}

		// Create and run the loop
		runner := loop.NewRunner(a, runPlanPath)

		fmt.Printf("Starting ralph-loop with %s agent\n", a.Name())
		fmt.Printf("Plan file: %s\n", runPlanPath)
		fmt.Println("Press Ctrl+C to stop gracefully")

		return runner.Run()
	},
}

// Init command
var (
	initOutputPath string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new plan template",
	Long: `Create a new plan.md template file.

The template includes:
  - Project name placeholder
  - Example steps
  - Notes section structure

Edit the generated file to add your actual implementation steps.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := plan.CreateTemplate(initOutputPath); err != nil {
			return err
		}
		fmt.Printf("Created plan template: %s\n", initOutputPath)
		fmt.Println("Edit the file to add your project name and steps.")
		return nil
	},
}

// Status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current plan status",
	Long:  `Display the current status of all steps in the plan.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := plan.ParseFile(runPlanPath)
		if err != nil {
			return fmt.Errorf("failed to parse plan: %w", err)
		}

		fmt.Printf("Project: %s\n", p.ProjectName)

		if p.Context != "" {
			fmt.Println("\nContext:")
			// Indent context lines for readability
			for _, line := range splitLines(p.Context) {
				fmt.Printf("  %s\n", line)
			}
		}

		fmt.Println("\nSteps:")

		completed := 0
		failed := 0
		pending := 0

		for _, step := range p.Steps {
			status := "[ ]"
			switch step.Status {
			case plan.StatusCompleted:
				status = "[x]"
				completed++
			case plan.StatusFailed:
				status = "[!]"
				failed++
			default:
				pending++
			}
			fmt.Printf("  %s Step %d: %s\n", status, step.Number, step.Description)
		}

		fmt.Printf("\nSummary: %d completed, %d failed, %d pending\n", completed, failed, pending)

		if p.IsComplete() {
			fmt.Println("\nAll steps completed!")
		} else if next := p.NextStep(); next != nil {
			fmt.Printf("\nNext step: Step %d - %s\n", next.Number, next.Description)
		}

		return nil
	},
}

func init() {
	// Run command flags
	runCmd.Flags().StringVarP(&runAgentType, "agent", "a", "claude", "AI agent to use (opencode, claude, or codex)")
	runCmd.Flags().StringVarP(&runPlanPath, "plan", "p", "plan.md", "Path to the plan file")

	// Init command flags
	initCmd.Flags().StringVarP(&initOutputPath, "output", "o", "plan.md", "Output path for the plan template")

	// Status command uses same plan path flag
	statusCmd.Flags().StringVarP(&runPlanPath, "plan", "p", "plan.md", "Path to the plan file")

	// Add commands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
}
