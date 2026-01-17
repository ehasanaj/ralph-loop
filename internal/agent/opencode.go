package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// OpencodeAgent implements the Agent interface for opencode
type OpencodeAgent struct{}

// NewOpencodeAgent creates a new opencode agent
func NewOpencodeAgent() *OpencodeAgent {
	return &OpencodeAgent{}
}

// Name returns the agent's name
func (a *OpencodeAgent) Name() string {
	return "opencode"
}

// Run executes opencode with the given prompt
func (a *OpencodeAgent) Run(ctx context.Context, prompt string, output io.Writer) (string, error) {
	// opencode run "<prompt>"
	cmd := exec.CommandContext(ctx, "opencode", "run", prompt)
	cmd.Stdin = nil // Prevent hanging on user input prompts

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
		return "", fmt.Errorf("failed to start opencode: %w", err)
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
