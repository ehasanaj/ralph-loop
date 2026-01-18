package loop

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PromptDetector wraps an io.Writer and monitors output for feedback prompts
type PromptDetector struct {
	writer      io.Writer
	mu          sync.Mutex
	recentLines []string // Buffer of recent lines for context
	lastLineAt  time.Time
	warned      bool
	checkTicker *time.Ticker
	done        chan struct{}
}

const maxRecentLines = 10 // Keep last 10 lines for context

// Common patterns that indicate the agent is waiting for user input
var promptPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\[y/n\]`),
	regexp.MustCompile(`(?i)\[yes/no\]`),
	regexp.MustCompile(`(?i)press enter`),
	regexp.MustCompile(`(?i)press any key`),
	regexp.MustCompile(`(?i)do you want to (continue|proceed|confirm)`),
	regexp.MustCompile(`(?i)would you like to`),
	regexp.MustCompile(`(?i)are you sure`),
	regexp.MustCompile(`(?i)confirm\?`),
	regexp.MustCompile(`(?i)proceed\?`),
	regexp.MustCompile(`(?i)continue\?`),
	regexp.MustCompile(`^\s*\?\s+`), // inquirer-style "? " prompts
	regexp.MustCompile(`(?i)waiting for (input|response|confirmation)`),
	regexp.MustCompile(`(?i)enter .* to continue`),
	regexp.MustCompile(`(?i)type .* to confirm`),
}

// NewPromptDetector creates a new PromptDetector wrapping the given writer
func NewPromptDetector(w io.Writer) *PromptDetector {
	pd := &PromptDetector{
		writer: w,
		done:   make(chan struct{}),
	}

	// Start a background goroutine to detect stalls
	pd.checkTicker = time.NewTicker(10 * time.Second)
	go pd.monitorStalls()

	return pd
}

// Write implements io.Writer
func (pd *PromptDetector) Write(p []byte) (n int, err error) {
	// Write to underlying writer first
	n, err = pd.writer.Write(p)
	if err != nil {
		return n, err
	}

	// Check for prompt patterns in the output
	text := string(p)
	lines := strings.Split(text, "\n")

	pd.mu.Lock()
	defer pd.mu.Unlock()

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Add to recent lines buffer
		pd.recentLines = append(pd.recentLines, line)
		if len(pd.recentLines) > maxRecentLines {
			pd.recentLines = pd.recentLines[1:]
		}
		pd.lastLineAt = time.Now()

		// Check if line matches any prompt pattern
		if pd.matchesPromptPattern(trimmed) {
			pd.showWarning()
		}
	}

	return n, nil
}

// matchesPromptPattern checks if a line matches any known prompt pattern
func (pd *PromptDetector) matchesPromptPattern(line string) bool {
	for _, pattern := range promptPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// showWarning displays a warning that the agent may be waiting for input
func (pd *PromptDetector) showWarning() {
	if pd.warned {
		return // Only warn once per detection
	}
	pd.warned = true

	fmt.Fprintln(pd.writer, "")
	fmt.Fprintln(pd.writer, "╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(pd.writer, "║  WARNING: Agent is asking for user input!                                ║")
	fmt.Fprintln(pd.writer, "║  The agent should be running autonomously without prompts.               ║")
	fmt.Fprintln(pd.writer, "║  Press Ctrl+C to cancel - the step will be retried automatically.        ║")
	fmt.Fprintln(pd.writer, "╠══════════════════════════════════════════════════════════════════════════╣")
	fmt.Fprintln(pd.writer, "║  AGENT IS ASKING:                                                        ║")
	fmt.Fprintln(pd.writer, "╟──────────────────────────────────────────────────────────────────────────╢")

	// Show recent lines as context for the question
	contextLines := pd.getQuestionContext()
	for _, line := range contextLines {
		// Pad or truncate line to fit in the box
		displayLine := formatBoxLine(line, 74)
		fmt.Fprintf(pd.writer, "║  %s  ║\n", displayLine)
	}

	fmt.Fprintln(pd.writer, "╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(pd.writer, "")
}

// getQuestionContext extracts relevant lines that form the question
func (pd *PromptDetector) getQuestionContext() []string {
	if len(pd.recentLines) == 0 {
		return []string{"(no context available)"}
	}

	// Get the last few lines, filtering out empty ones and limiting context
	var context []string
	startIdx := len(pd.recentLines) - 5 // Last 5 lines max
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(pd.recentLines); i++ {
		line := strings.TrimSpace(pd.recentLines[i])
		if line != "" {
			context = append(context, line)
		}
	}

	if len(context) == 0 {
		return []string{"(no context available)"}
	}

	return context
}

// formatBoxLine pads or truncates a line to fit in a box of given width
func formatBoxLine(line string, width int) string {
	// Remove any ANSI escape codes for length calculation
	cleanLine := stripAnsi(line)

	if len(cleanLine) > width {
		return line[:width-3] + "..."
	}

	// Pad with spaces
	padding := width - len(cleanLine)
	return line + strings.Repeat(" ", padding)
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// monitorStalls checks for output stalls that might indicate waiting for input
func (pd *PromptDetector) monitorStalls() {
	for {
		select {
		case <-pd.done:
			return
		case <-pd.checkTicker.C:
			pd.mu.Lock()
			if !pd.lastLineAt.IsZero() && time.Since(pd.lastLineAt) > 30*time.Second && !pd.warned {
				// No output for 30 seconds - might be waiting for input
				pd.warned = true

				fmt.Fprintln(pd.writer, "")
				fmt.Fprintln(pd.writer, "╔══════════════════════════════════════════════════════════════════════════╗")
				fmt.Fprintln(pd.writer, "║  WARNING: No output for 30+ seconds - agent may be stalled!              ║")
				fmt.Fprintln(pd.writer, "║  The agent might be waiting for input or processing a large task.        ║")
				fmt.Fprintln(pd.writer, "║  Press Ctrl+C to cancel - the step will be retried automatically.        ║")
				fmt.Fprintln(pd.writer, "╠══════════════════════════════════════════════════════════════════════════╣")
				fmt.Fprintln(pd.writer, "║  LAST OUTPUT:                                                            ║")
				fmt.Fprintln(pd.writer, "╟──────────────────────────────────────────────────────────────────────────╢")

				// Show recent lines as context
				contextLines := pd.getQuestionContext()
				for _, line := range contextLines {
					displayLine := formatBoxLine(line, 74)
					fmt.Fprintf(pd.writer, "║  %s  ║\n", displayLine)
				}

				fmt.Fprintln(pd.writer, "╚══════════════════════════════════════════════════════════════════════════╝")
				fmt.Fprintln(pd.writer, "")
			}
			pd.mu.Unlock()
		}
	}
}

// Close stops the background monitoring
func (pd *PromptDetector) Close() {
	pd.checkTicker.Stop()
	close(pd.done)
}

// Reset clears the warning state for a new step
func (pd *PromptDetector) Reset() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.warned = false
	pd.recentLines = nil
	pd.lastLineAt = time.Time{}
}
