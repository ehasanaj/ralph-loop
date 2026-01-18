package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ClaudeAgent implements the Agent interface for claude CLI
type ClaudeAgent struct {
	opts Options
}

// NewClaudeAgent creates a new claude agent
func NewClaudeAgent(opts Options) *ClaudeAgent {
	return &ClaudeAgent{opts: opts}
}

// Name returns the agent's name
func (a *ClaudeAgent) Name() string {
	return "claude"
}

// Run executes claude with the given prompt
func (a *ClaudeAgent) Run(ctx context.Context, prompt string, output io.Writer) (string, error) {
	// Build command args
	// --dangerously-skip-permissions bypasses all permission prompts
	args := []string{"-p", "--dangerously-skip-permissions"}

	// Add model flag if specified
	if a.opts.Model != "" {
		args = append(args, "--model", a.opts.Model)
	}

	args = append(args, prompt)
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = nil // Prevent hanging on user input prompts
	cmd.Env = append(cmd.Environ(), "CI=true", "NONINTERACTIVE=1") // Signal non-interactive mode

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Diagnostic: show that we're starting
	if output != nil {
		fmt.Fprintln(output, "[ralph-loop] Starting claude agent...")
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	if output != nil {
		fmt.Fprintf(output, "[ralph-loop] claude started (PID: %d)\n", cmd.Process.Pid)
	}

	// Collect output while streaming - with proper synchronization
	var fullOutput strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			fullOutput.WriteString(line)
			fullOutput.WriteString("\n")
			mu.Unlock()
			if output != nil {
				fmt.Fprintln(output, line)
			}
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			fullOutput.WriteString(line)
			fullOutput.WriteString("\n")
			mu.Unlock()
			if output != nil {
				fmt.Fprintln(output, line)
			}
		}
	}()

	// Wait for goroutines to finish reading all output
	wg.Wait()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			if output != nil {
				fmt.Fprintln(output, "[ralph-loop] claude cancelled")
			}
			return fullOutput.String(), ctx.Err()
		}
		// Non-zero exit is not necessarily an error for our purposes
		// The output parsing will determine success/failure
		if output != nil {
			fmt.Fprintf(output, "[ralph-loop] claude exited with error: %v\n", err)
		}
	} else {
		if output != nil {
			fmt.Fprintln(output, "[ralph-loop] claude completed")
		}
	}

	return fullOutput.String(), nil
}
