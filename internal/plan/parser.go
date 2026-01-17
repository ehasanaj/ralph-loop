package plan

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	// Matches: - [ ] Step 1: Description or - [x] Step 2: Description or - [!] Step 3: Description or - [-] Step 4: Description
	stepLineRegex = regexp.MustCompile(`^-\s+\[([ x!\-])\]\s+(?:Step\s+(\d+):\s+)?(.+)$`)

	// Matches: # Project: Name
	projectNameRegex = regexp.MustCompile(`^#\s+Project:\s+(.+)$`)

	// Matches: ### Step 1
	notesSectionRegex = regexp.MustCompile(`^###\s+Step\s+(\d+)$`)

	// Matches: **Status**: pending/completed/failed
	statusRegex = regexp.MustCompile(`^\*\*Status\*\*:\s+(\w+)$`)

	// Matches: **Last Run**: 2026-01-17 10:30:00
	lastRunRegex = regexp.MustCompile(`^\*\*Last Run\*\*:\s+(.+)$`)

	// Matches: **Notes**: content
	notesRegex = regexp.MustCompile(`^\*\*Notes\*\*:\s+(.*)$`)

	// Matches: **Retries**: N
	retriesRegex = regexp.MustCompile(`^\*\*Retries\*\*:\s+(\d+)$`)

	// Matches: ## Context
	contextSectionRegex = regexp.MustCompile(`^##\s+Context\s*$`)

	// Matches: ## Plan or ## Notes (to detect end of context section)
	sectionHeaderRegex = regexp.MustCompile(`^##\s+\w+`)
)

// ParseFile reads and parses a plan markdown file
func ParseFile(path string) (*Plan, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}
	return Parse(string(content))
}

// Parse parses markdown content into a Plan struct
func Parse(content string) (*Plan, error) {
	plan := &Plan{
		RawContent: content,
		Steps:      make([]Step, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	stepNumber := 0
	notesMap := make(map[int]*stepNotes)

	var currentNoteStep int
	var inNotesSection bool
	var inContextSection bool
	var contextLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for project name
		if matches := projectNameRegex.FindStringSubmatch(line); matches != nil {
			plan.ProjectName = strings.TrimSpace(matches[1])
			continue
		}

		// Check for context section header
		if contextSectionRegex.MatchString(line) {
			inContextSection = true
			continue
		}

		// If in context section, collect lines until next ## header
		if inContextSection {
			if sectionHeaderRegex.MatchString(line) {
				// End of context section, save collected content
				plan.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
				inContextSection = false
				contextLines = nil
				// Don't continue - let other matchers process this line
			} else {
				contextLines = append(contextLines, line)
				continue
			}
		}

		// Check for step definition in Plan section
		if matches := stepLineRegex.FindStringSubmatch(line); matches != nil {
			stepNumber++
			status := parseCheckbox(matches[1])
			description := strings.TrimSpace(matches[3])

			plan.Steps = append(plan.Steps, Step{
				Number:      stepNumber,
				Description: description,
				Status:      status,
			})
			continue
		}

		// Check for notes section header
		if matches := notesSectionRegex.FindStringSubmatch(line); matches != nil {
			num := parseStepNumber(matches[1])
			currentNoteStep = num
			inNotesSection = true
			if notesMap[num] == nil {
				notesMap[num] = &stepNotes{}
			}
			continue
		}

		// Parse notes section content
		if inNotesSection && currentNoteStep > 0 {
			notes := notesMap[currentNoteStep]

			if matches := statusRegex.FindStringSubmatch(line); matches != nil {
				notes.status = matches[1]
				continue
			}

			if matches := lastRunRegex.FindStringSubmatch(line); matches != nil {
				notes.lastRun = matches[1]
				continue
			}

			if matches := notesRegex.FindStringSubmatch(line); matches != nil {
				notes.notes = matches[1]
				continue
			}

			if matches := retriesRegex.FindStringSubmatch(line); matches != nil {
				fmt.Sscanf(matches[1], "%d", &notes.retryCount)
				continue
			}

			// Check if we've left the notes section (next header)
			if strings.HasPrefix(line, "#") {
				inNotesSection = false
				currentNoteStep = 0
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning plan: %w", err)
	}

	// Apply notes to steps
	for i := range plan.Steps {
		stepNum := plan.Steps[i].Number
		if notes, ok := notesMap[stepNum]; ok {
			if notes.lastRun != "" && notes.lastRun != "N/A" {
				if t, err := time.Parse("2006-01-02 15:04:05", notes.lastRun); err == nil {
					plan.Steps[i].LastRun = &t
				}
			}
			if notes.notes != "(none)" {
				plan.Steps[i].Notes = notes.notes
			}
			plan.Steps[i].RetryCount = notes.retryCount
		}
	}

	return plan, nil
}

type stepNotes struct {
	status     string
	lastRun    string
	notes      string
	retryCount int
}

func parseCheckbox(marker string) StepStatus {
	switch marker {
	case "x":
		return StatusCompleted
	case "!":
		return StatusFailed
	case "-":
		return StatusSkipped
	default:
		return StatusPending
	}
}

func parseStepNumber(s string) int {
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}
