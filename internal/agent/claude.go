package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ClaudeAgent implements the Agent interface for claude CLI
type ClaudeAgent struct{}

// NewClaudeAgent creates a new claude agent
func NewClaudeAgent() *ClaudeAgent {
	return &ClaudeAgent{}
}

// Name returns the agent's name
func (a *ClaudeAgent) Name() string {
	return "claude"
}

// Run executes claude with the given prompt
func (a *ClaudeAgent) Run(ctx context.Context, prompt string, output io.Writer) (string, error) {
	// claude -p "<prompt>"
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	// Collect output while streaming
	var fullOutput strings.Builder

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
		for scanner.Scan() {
			line := scanner.Text()
			fullOutput.WriteString(line)
			fullOutput.WriteString("\n")
			if output != nil {
				fmt.Fprintln(output, line)
			}
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
		for scanner.Scan() {
			line := scanner.Text()
			fullOutput.WriteString(line)
			fullOutput.WriteString("\n")
			if output != nil {
				fmt.Fprintln(output, line)
			}
		}
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			return fullOutput.String(), ctx.Err()
		}
		// Non-zero exit is not necessarily an error for our purposes
		// The output parsing will determine success/failure
	}

	return fullOutput.String(), nil
}
