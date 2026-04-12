package taskrunner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EngramSaver handles persistence to Engram memory.
type EngramSaver struct {
	SaveFunc func(title, content, topicKey string) error
}

// NewEngramSaver creates a new Engram saver.
func NewEngramSaver(saveFunc func(title, content, topicKey string) error) *EngramSaver {
	return &EngramSaver{SaveFunc: saveFunc}
}

// SaveReport saves the report to Engram.
func (e *EngramSaver) SaveReport(report *Report) error {
	if e.SaveFunc == nil {
		return fmt.Errorf("no save function provided")
	}

	timestamp := time.Now().Format("20060102-150405")
	topicKey := fmt.Sprintf("taskrunner/run/%s", timestamp)

	title := fmt.Sprintf("Task Run: %s", truncate(report.Task, 50))
	content := buildEngramContent(report)

	return e.SaveFunc(title, content, topicKey)
}

// buildEngramContent formats the report for Engram storage.
func buildEngramContent(report *Report) string {
	var sb strings.Builder

	sb.WriteString("## Task Execution Report\n\n")
	sb.WriteString(fmt.Sprintf("**Task:** %s\n", report.Task))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", report.Status))
	sb.WriteString(fmt.Sprintf("**Iterations:** %d\n", report.Iterations))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", report.Duration.Round(time.Second)))
	sb.WriteString(fmt.Sprintf("**Engine:** %s\n", report.EngineUsed))
	sb.WriteString(fmt.Sprintf("**WorkDir:** %s\n", report.WorkDir))
	sb.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString(report.FinalOutput)
	sb.WriteString("\n\n")

	if len(report.Steps) > 0 {
		sb.WriteString("## Steps\n\n")
		sb.WriteString("| # | Action | Duration | Error |\n")
		sb.WriteString("|---|--------|----------|-------|\n")
		for _, step := range report.Steps {
			errMark := ""
			if step.Error != "" {
				errMark = "✗"
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
				step.Iteration,
				step.Action.Type,
				step.Duration.Round(time.Millisecond),
				errMark,
			))
		}
		sb.WriteString("\n")
	}

	// JSON for machine parsing
	sb.WriteString("## Raw Data (JSON)\n\n```json\n")
	jsonData, _ := json.MarshalIndent(report, "", "  ")
	sb.Write(jsonData)
	sb.WriteString("\n```\n")

	return sb.String()
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// DefaultEngramSaveFunc is a placeholder that can be replaced with actual MCP call.
var DefaultEngramSaveFunc = func(title, content, topicKey string) error {
	// This is a placeholder - in real usage, this would call mem_save MCP tool
	// For now, we just print to stderr for debugging
	fmt.Printf("[Engram] Would save to %s: %s\n", topicKey, title)
	return nil
}

// SaveReportToEngram saves a report using the default saver.
func SaveReportToEngram(report *Report) error {
	saver := NewEngramSaver(DefaultEngramSaveFunc)
	return saver.SaveReport(report)
}
