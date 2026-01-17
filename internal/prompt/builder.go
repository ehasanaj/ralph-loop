package prompt

import (
	"fmt"
	"strings"

	"github.com/eraldohasanaj/ralph-loop/internal/plan"
)

// Build constructs the prompt for the AI agent
func Build(p *plan.Plan, step *plan.Step) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Task: Execute a Step in the Implementation Plan\n\n")

	// Overall context
	sb.WriteString("## Project Overview\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n\n", p.ProjectName))

	// Project context if available
	if p.Context != "" {
		sb.WriteString("### Project Context\n")
		sb.WriteString(p.Context)
		sb.WriteString("\n\n")
	}

	// Full plan for context
	sb.WriteString("## Full Plan\n")
	for _, s := range p.Steps {
		status := "[ ]"
		switch s.Status {
		case plan.StatusCompleted:
			status = "[x]"
		case plan.StatusFailed:
			status = "[!]"
		}
		sb.WriteString(fmt.Sprintf("- %s Step %d: %s\n", status, s.Number, s.Description))
	}
	sb.WriteString("\n")

	// Current step
	sb.WriteString("## Your Current Task\n")
	sb.WriteString(fmt.Sprintf("**Step %d**: %s\n\n", step.Number, step.Description))

	// Previous notes if retrying
	if step.Status == plan.StatusFailed && step.Notes != "" {
		sb.WriteString("## Previous Attempt\n")
		sb.WriteString("This step failed previously. Here are the notes from the last attempt:\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", step.Notes))
		sb.WriteString("Please try a different approach or fix the issues mentioned above.\n\n")
	}

	// Instructions
	sb.WriteString("## Instructions\n")
	sb.WriteString("1. Focus ONLY on completing the current step (Step " + fmt.Sprintf("%d", step.Number) + ")\n")
	sb.WriteString("2. Do not work on other steps\n")
	sb.WriteString("3. When you have completed the step successfully, output exactly:\n")
	sb.WriteString("   STEP_COMPLETE\n")
	sb.WriteString("4. If you encounter an error you cannot resolve, output exactly:\n")
	sb.WriteString("   STEP_FAILED: <brief description of what went wrong>\n")
	sb.WriteString("5. Make sure STEP_COMPLETE or STEP_FAILED appears at the end of your response\n\n")

	sb.WriteString("Begin working on the step now.\n")

	return sb.String()
}

// ParseResult parses the agent output for completion markers
func ParseResult(output string) plan.StepResult {
	lines := strings.Split(output, "\n")

	// Check from the end for markers
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		if line == "STEP_COMPLETE" {
			return plan.StepResult{
				Success: true,
				Output:  output,
			}
		}

		if strings.HasPrefix(line, "STEP_FAILED:") {
			reason := strings.TrimSpace(strings.TrimPrefix(line, "STEP_FAILED:"))
			return plan.StepResult{
				Success: false,
				Output:  output,
				Reason:  reason,
			}
		}

		// Also check for markers that might have text after them
		if strings.Contains(line, "STEP_COMPLETE") {
			return plan.StepResult{
				Success: true,
				Output:  output,
			}
		}

		if strings.Contains(line, "STEP_FAILED:") {
			idx := strings.Index(line, "STEP_FAILED:")
			reason := strings.TrimSpace(line[idx+len("STEP_FAILED:"):])
			return plan.StepResult{
				Success: false,
				Output:  output,
				Reason:  reason,
			}
		}
	}

	// No marker found - treat as failure
	return plan.StepResult{
		Success: false,
		Output:  output,
		Reason:  "No STEP_COMPLETE or STEP_FAILED marker found in output",
	}
}
