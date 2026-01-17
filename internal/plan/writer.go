package plan

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// UpdateStep updates a step's status in the plan file
func UpdateStep(path string, stepNum int, result StepResult) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	updated := updateStepInContent(string(content), stepNum, result)

	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

func updateStepInContent(content string, stepNum int, result StepResult) string {
	lines := strings.Split(content, "\n")
	var output []string

	currentStep := 0
	inNotesSection := false
	notesStepNum := 0
	foundNotesSection := false
	notesSectionExists := false

	// First pass: update the checkbox in the plan section
	for i, line := range lines {
		if matches := stepLineRegex.FindStringSubmatch(line); matches != nil {
			currentStep++
			if currentStep == stepNum {
				// Update the checkbox
				newMarker := " "
				if result.Success {
					newMarker = "x"
				} else {
					newMarker = "!"
				}
				lines[i] = updateCheckbox(line, newMarker)
			}
		}

		// Check if notes section exists
		if notesSectionRegex.MatchString(line) {
			notesSectionExists = true
		}
	}

	// Second pass: update or create notes section
	for i, line := range lines {
		// Check if we're entering a notes section for our step
		if matches := notesSectionRegex.FindStringSubmatch(line); matches != nil {
			num := parseStepNumber(matches[1])
			notesStepNum = num
			inNotesSection = true
			if num == stepNum {
				foundNotesSection = true
			}
		}

		// Update notes content if in the right section
		if inNotesSection && notesStepNum == stepNum {
			if statusRegex.MatchString(line) {
				status := "completed"
				if !result.Success {
					status = "failed"
				}
				lines[i] = fmt.Sprintf("**Status**: %s", status)
			} else if lastRunRegex.MatchString(line) {
				lines[i] = fmt.Sprintf("**Last Run**: %s", time.Now().Format("2006-01-02 15:04:05"))
			} else if notesRegex.MatchString(line) {
				notes := summarizeOutput(result.Output)
				if !result.Success && result.Reason != "" {
					notes = fmt.Sprintf("Failed: %s", result.Reason)
				}
				lines[i] = fmt.Sprintf("**Notes**: %s", notes)
			}
		}

		// Check if we're leaving the notes section
		if inNotesSection && strings.HasPrefix(line, "### Step") && notesStepNum != parseStepNumber(notesSectionRegex.FindStringSubmatch(line)[1]) {
			inNotesSection = false
		}

		output = append(output, lines[i])
	}

	// If notes section doesn't exist at all, we need to create the entire Notes section
	if !notesSectionExists {
		output = append(output, "")
		output = append(output, "## Notes")
		output = append(output, "")
		output = append(output, createNotesSectionForStep(stepNum, result))
	} else if !foundNotesSection {
		// Find where to insert the new notes section (at end of ## Notes)
		for i := len(output) - 1; i >= 0; i-- {
			if strings.HasPrefix(output[i], "## Notes") || strings.HasPrefix(output[i], "### Step") {
				// Insert after the last notes-related line
				insertIdx := i + 1
				for insertIdx < len(output) && (strings.TrimSpace(output[insertIdx]) != "" || strings.HasPrefix(strings.TrimSpace(output[insertIdx]), "**")) {
					insertIdx++
				}
				newSection := createNotesSectionForStep(stepNum, result)
				output = insertAfter(output, insertIdx, newSection)
				break
			}
		}
	}

	return strings.Join(output, "\n")
}

func updateCheckbox(line, marker string) string {
	re := regexp.MustCompile(`\[([ x!])\]`)
	return re.ReplaceAllString(line, fmt.Sprintf("[%s]", marker))
}

func createNotesSectionForStep(stepNum int, result StepResult) string {
	status := "completed"
	if !result.Success {
		status = "failed"
	}
	notes := summarizeOutput(result.Output)
	if !result.Success && result.Reason != "" {
		notes = fmt.Sprintf("Failed: %s", result.Reason)
	}

	return fmt.Sprintf(`### Step %d
**Status**: %s
**Last Run**: %s
**Notes**: %s`, stepNum, status, time.Now().Format("2006-01-02 15:04:05"), notes)
}

func insertAfter(slice []string, index int, value string) []string {
	if index >= len(slice) {
		return append(slice, "", value)
	}
	result := make([]string, 0, len(slice)+2)
	result = append(result, slice[:index]...)
	result = append(result, "", value)
	result = append(result, slice[index:]...)
	return result
}

func summarizeOutput(output string) string {
	// Extract meaningful summary from output
	// Look for STEP_COMPLETE or meaningful last lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return "(no output)"
	}

	// Find the last non-empty meaningful line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "STEP_") {
			// Truncate if too long
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}

	return "Completed successfully"
}

// WriteFile writes a plan to a file
func WriteFile(path string, plan *Plan) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Project: %s\n\n", plan.ProjectName))

	// Write context section if present
	if plan.Context != "" {
		sb.WriteString("## Context\n\n")
		sb.WriteString(plan.Context)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Plan\n\n")

	for _, step := range plan.Steps {
		marker := " "
		switch step.Status {
		case StatusCompleted:
			marker = "x"
		case StatusFailed:
			marker = "!"
		}
		sb.WriteString(fmt.Sprintf("- [%s] Step %d: %s\n", marker, step.Number, step.Description))
	}

	sb.WriteString("\n## Notes\n")

	for _, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("\n### Step %d\n", step.Number))

		status := string(step.Status)
		sb.WriteString(fmt.Sprintf("**Status**: %s\n", status))

		lastRun := "N/A"
		if step.LastRun != nil {
			lastRun = step.LastRun.Format("2006-01-02 15:04:05")
		}
		sb.WriteString(fmt.Sprintf("**Last Run**: %s\n", lastRun))

		notes := "(none)"
		if step.Notes != "" {
			notes = step.Notes
		}
		sb.WriteString(fmt.Sprintf("**Notes**: %s\n", notes))
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// SavePlan reads, parses, and rewrites the plan file (useful after modifications)
func SavePlan(path string, plan *Plan) error {
	return WriteFile(path, plan)
}

// Scanner helper for parsing
func scanLines(content string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(content))
}
