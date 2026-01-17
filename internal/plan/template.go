package plan

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultTemplate = `# Project: [Your Project Name]

## Context

Add background information about your project here.
Include key technologies, constraints, or any context the AI should know.

## Plan

- [ ] Step 1: Describe your first task here
- [ ] Step 2: Describe your second task here
- [ ] Step 3: Add more steps as needed

## Notes

### Step 1
**Status**: pending
**Last Run**: N/A
**Notes**: (none)

### Step 2
**Status**: pending
**Last Run**: N/A
**Notes**: (none)

### Step 3
**Status**: pending
**Last Run**: N/A
**Notes**: (none)
`

// CreateTemplate creates a new plan template file
func CreateTemplate(path string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(defaultTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// CreateTemplateWithSteps creates a plan template with the given steps
func CreateTemplateWithSteps(path string, projectName string, steps []string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	plan := &Plan{
		ProjectName: projectName,
		Steps:       make([]Step, len(steps)),
	}

	for i, desc := range steps {
		plan.Steps[i] = Step{
			Number:      i + 1,
			Description: desc,
			Status:      StatusPending,
		}
	}

	return WriteFile(path, plan)
}
