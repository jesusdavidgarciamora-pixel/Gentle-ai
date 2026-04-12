package taskrunner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ErrorLesson represents a learned lesson from a failed step.
type ErrorLesson struct {
	ErrorPattern string    `json:"error_pattern"`
	Context      string    `json:"context"`
	Solution     string    `json:"solution"`
	Timestamp    time.Time `json:"timestamp"`
	WorkDir      string    `json:"work_dir"`
	TaskType     string    `json:"task_type"`
}

// LessonStore handles persistence and retrieval of error lessons.
type LessonStore struct {
	SaveFunc   func(title, content, topicKey string) error
	SearchFunc func(query string) ([]LessonResult, error)
}

// LessonResult represents a search result from Engram.
type LessonResult struct {
	Title   string
	Content string
	TopicKey string
}

// NewLessonStore creates a new lesson store.
func NewLessonStore(
	saveFunc func(title, content, topicKey string) error,
	searchFunc func(query string) ([]LessonResult, error),
) *LessonStore {
	return &LessonStore{
		SaveFunc:   saveFunc,
		SearchFunc: searchFunc,
	}
}

// ExtractLessons extracts error lessons from a report.
func ExtractLessons(report *Report) []ErrorLesson {
	var lessons []ErrorLesson

	for _, step := range report.Steps {
		if step.Error == "" {
			continue
		}

		// Extract the error pattern (first line or first 100 chars)
		errorPattern := extractErrorPattern(step.Error)

		// Determine context from action
		context := describeAction(step.Action)

		// Infer solution from subsequent successful steps
		solution := inferSolution(report.Steps, step.Iteration)

		lesson := ErrorLesson{
			ErrorPattern: errorPattern,
			Context:      context,
			Solution:     solution,
			Timestamp:    time.Now(),
			WorkDir:      report.WorkDir,
			TaskType:     classifyTask(report.Task),
		}

		lessons = append(lessons, lesson)
	}

	return lessons
}

// SaveLessons saves extracted lessons to Engram.
func (ls *LessonStore) SaveLessons(lessons []ErrorLesson) error {
	if ls.SaveFunc == nil {
		return nil // Silently skip if no save function
	}

	for i, lesson := range lessons {
		topicKey := fmt.Sprintf("taskrunner/lessons/%d-%s",
			lesson.Timestamp.Unix(),
			sanitizeTopic(lesson.ErrorPattern),
		)

		title := fmt.Sprintf("Lesson: %s", truncate(lesson.ErrorPattern, 50))
		content := formatLesson(lesson)

		if err := ls.SaveFunc(title, content, topicKey); err != nil {
			// Log but continue - don't fail the whole task for lesson saving
			fmt.Printf("[LessonStore] Warning: failed to save lesson %d: %v\n", i, err)
		}
	}

	return nil
}

// FindRelevantLessons searches for lessons relevant to a task.
func (ls *LessonStore) FindRelevantLessons(task string, workDir string) ([]ErrorLesson, error) {
	if ls.SearchFunc == nil {
		return nil, nil // Silently skip if no search function
	}

	// Search by task type
	taskType := classifyTask(task)
	query := fmt.Sprintf("taskrunner/lessons %s", taskType)

	results, err := ls.SearchFunc(query)
	if err != nil {
		return nil, err
	}

	var lessons []ErrorLesson
	for _, result := range results {
		lesson, err := parseLesson(result.Content)
		if err != nil {
			continue // Skip malformed lessons
		}

		// Filter by relevance (same workDir or task type)
		if isRelevant(lesson, workDir, taskType) {
			lessons = append(lessons, lesson)
		}
	}

	return lessons, nil
}

// FormatLessonsForPrompt formats lessons for inclusion in the system prompt.
func FormatLessonsForPrompt(lessons []ErrorLesson) string {
	if len(lessons) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n=== LESSONS LEARNED FROM PREVIOUS RUNS ===\n")
	sb.WriteString("Avoid these mistakes:\n\n")

	for i, lesson := range lessons {
		sb.WriteString(fmt.Sprintf("%d. When: %s\n", i+1, lesson.Context))
		sb.WriteString(fmt.Sprintf("   Error: %s\n", lesson.ErrorPattern))
		if lesson.Solution != "" {
			sb.WriteString(fmt.Sprintf("   Solution: %s\n", lesson.Solution))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=== END LESSONS ===\n")

	return sb.String()
}

// Helper functions

func extractErrorPattern(error string) string {
	// Take first line or first 100 chars
	lines := strings.SplitN(error, "\n", 2)
	pattern := lines[0]
	if len(pattern) > 100 {
		pattern = pattern[:100] + "..."
	}
	return pattern
}

func describeAction(action Action) string {
	switch action.Type {
	case ActionShell:
		return fmt.Sprintf("running command: %s", truncate(action.Command, 50))
	case ActionWriteFile:
		return fmt.Sprintf("writing file: %s", action.Path)
	case ActionReadFile:
		return fmt.Sprintf("reading file: %s", action.Path)
	case ActionEditFile:
		return fmt.Sprintf("editing file: %s", action.Path)
	default:
		return string(action.Type)
	}
}

func inferSolution(steps []StepRecord, errorIteration int) string {
	// Look for successful steps after the error
	// A successful step has no error AND is not a failed action type
	for _, step := range steps {
		if step.Iteration > errorIteration && step.Error == "" && step.Action.Type != ActionFailed {
			// Found a successful step after the error
			return describeAction(step.Action)
		}
	}
	return ""
}

func classifyTask(task string) string {
	taskLower := strings.ToLower(task)

	// Common task type classifications - more specific first
	if strings.Contains(taskLower, "test") || strings.Contains(taskLower, "spec") {
		return "testing"
	}
	if strings.Contains(taskLower, "refactor") {
		return "refactoring"
	}
	if strings.Contains(taskLower, "api") || strings.Contains(taskLower, "endpoint") {
		return "api"
	}
	if strings.Contains(taskLower, "fix") || strings.Contains(taskLower, "bug") {
		return "bugfix"
	}
	if strings.Contains(taskLower, "install") || strings.Contains(taskLower, "setup") {
		return "setup"
	}
	if strings.Contains(taskLower, "create") || strings.Contains(taskLower, "add") {
		return "creation"
	}

	return "general"
}

func sanitizeTopic(s string) string {
	// Remove special characters for topic key
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' {
			result.WriteRune('-')
		}
	}
	truncated := result.String()
	if len(truncated) > 50 {
		truncated = truncated[:50]
	}
	return truncated
}

func formatLesson(lesson ErrorLesson) string {
	var sb strings.Builder

	sb.WriteString("## Error Lesson\n\n")
	sb.WriteString(fmt.Sprintf("**Error Pattern:** %s\n", lesson.ErrorPattern))
	sb.WriteString(fmt.Sprintf("**Context:** %s\n", lesson.Context))
	sb.WriteString(fmt.Sprintf("**Task Type:** %s\n", lesson.TaskType))
	sb.WriteString(fmt.Sprintf("**WorkDir:** %s\n", lesson.WorkDir))
	sb.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", lesson.Timestamp.Format(time.RFC3339)))

	if lesson.Solution != "" {
		sb.WriteString("## Solution\n\n")
		sb.WriteString(lesson.Solution)
		sb.WriteString("\n")
	}

	// JSON for machine parsing
	sb.WriteString("\n## Raw Data (JSON)\n\n```json\n")
	jsonData, _ := json.MarshalIndent(lesson, "", "  ")
	sb.Write(jsonData)
	sb.WriteString("\n```\n")

	return sb.String()
}

func parseLesson(content string) (ErrorLesson, error) {
	// Try to extract JSON from the content
	startIdx := strings.Index(content, "```json")
	if startIdx == -1 {
		return ErrorLesson{}, fmt.Errorf("no JSON found")
	}

	startIdx += len("```json")
	endIdx := strings.Index(content[startIdx:], "```")
	if endIdx == -1 {
		return ErrorLesson{}, fmt.Errorf("malformed JSON block")
	}

	jsonStr := strings.TrimSpace(content[startIdx : startIdx+endIdx])

	var lesson ErrorLesson
	if err := json.Unmarshal([]byte(jsonStr), &lesson); err != nil {
		return ErrorLesson{}, err
	}

	return lesson, nil
}

func isRelevant(lesson ErrorLesson, workDir string, taskType string) bool {
	// Same task type is always relevant
	if lesson.TaskType == taskType {
		return true
	}

	// Same workDir is relevant
	if lesson.WorkDir == workDir {
		return true
	}

	// Recent lessons (< 7 days) are relevant
	if time.Since(lesson.Timestamp) < 7*24*time.Hour {
		return true
	}

	return false
}

// DefaultLessonStore returns a lesson store with placeholder functions.
func DefaultLessonStore() *LessonStore {
	return NewLessonStore(
		func(title, content, topicKey string) error {
			fmt.Printf("[LessonStore] Would save: %s to %s\n", title, topicKey)
			return nil
		},
		func(query string) ([]LessonResult, error) {
			// Return empty - in real usage this would search Engram
			return nil, nil
		},
	)
}
