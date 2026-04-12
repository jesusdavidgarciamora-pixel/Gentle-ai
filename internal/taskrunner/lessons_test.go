package taskrunner

import (
	"strings"
	"testing"
	"time"
)

func TestExtractLessons(t *testing.T) {
	report := &Report{
		Task:   "test task",
		Status: "success",
		Steps: []StepRecord{
			{
				Iteration: 1,
				Action:    Action{Type: ActionShell, Command: "npm install", Reason: "install deps"},
				Error:     "npm: command not found",
			},
			{
				Iteration: 2,
				Action:    Action{Type: ActionShell, Command: "yarn install", Reason: "try yarn"},
				Output:    "success",
			},
			{
				Iteration: 3,
				Action:    Action{Type: ActionDone, Summary: "Done"},
			},
		},
		WorkDir: "/tmp/test",
	}

	lessons := ExtractLessons(report)

	if len(lessons) != 1 {
		t.Errorf("expected 1 lesson, got %d", len(lessons))
	}

	if len(lessons) > 0 {
		lesson := lessons[0]
		if !strings.Contains(lesson.ErrorPattern, "npm: command not found") {
			t.Errorf("expected error pattern to contain 'npm: command not found', got %s", lesson.ErrorPattern)
		}
		if lesson.Context == "" {
			t.Error("expected non-empty context")
		}
		if lesson.Solution == "" {
			t.Error("expected solution from subsequent successful step")
		}
		if lesson.WorkDir != "/tmp/test" {
			t.Errorf("expected workdir /tmp/test, got %s", lesson.WorkDir)
		}
	}
}

func TestExtractLessonsNoErrors(t *testing.T) {
	report := &Report{
		Task:   "test task",
		Status: "success",
		Steps: []StepRecord{
			{
				Iteration: 1,
				Action:    Action{Type: ActionShell, Command: "echo hello"},
				Output:    "hello",
			},
		},
	}

	lessons := ExtractLessons(report)

	if len(lessons) != 0 {
		t.Errorf("expected 0 lessons for successful run, got %d", len(lessons))
	}
}

func TestClassifyTask(t *testing.T) {
	tests := []struct {
		task     string
		expected string
	}{
		{"create a test file", "testing"},  // "test" has higher priority
		{"add new feature", "creation"},
		{"fix the bug in login", "bugfix"},
		{"refactor the code", "refactoring"},
		{"write tests for auth", "testing"},
		{"install dependencies", "setup"},
		{"setup the project", "setup"},
		{"create API endpoint", "api"},  // "api" has higher priority than "create"
		{"do something", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.task, func(t *testing.T) {
			got := classifyTask(tt.task)
			if got != tt.expected {
				t.Errorf("classifyTask(%q) = %q, want %q", tt.task, got, tt.expected)
			}
		})
	}
}

func TestExtractErrorPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple error", "simple error"},
		{"first line\nsecond line", "first line"},
		{strings.Repeat("a", 200), strings.Repeat("a", 100) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(len(tt.input), 10)], func(t *testing.T) {
			got := extractErrorPattern(tt.input)
			if got != tt.expected {
				t.Errorf("extractErrorPattern() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSanitizeTopic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Test@123", "test123"},
		{"a_b-c", "a_b-c"},
		{strings.Repeat("a", 100), strings.Repeat("a", 50)},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(len(tt.input), 10)], func(t *testing.T) {
			got := sanitizeTopic(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeTopic(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDescribeAction(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{Action{Type: ActionShell, Command: "echo hello"}, "running command: echo hello"},
		{Action{Type: ActionWriteFile, Path: "/tmp/test.txt"}, "writing file: /tmp/test.txt"},
		{Action{Type: ActionReadFile, Path: "/tmp/read.txt"}, "reading file: /tmp/read.txt"},
		{Action{Type: ActionEditFile, Path: "/tmp/edit.txt"}, "editing file: /tmp/edit.txt"},
		{Action{Type: ActionDone}, "done"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action.Type), func(t *testing.T) {
			got := describeAction(tt.action)
			if !strings.Contains(got, tt.expected) && tt.expected != "" {
				t.Errorf("describeAction() = %q, expected to contain %q", got, tt.expected)
			}
		})
	}
}

func TestInferSolution(t *testing.T) {
	steps := []StepRecord{
		{Iteration: 1, Action: Action{Type: ActionShell, Command: "npm install"}, Error: "failed"},
		{Iteration: 2, Action: Action{Type: ActionShell, Command: "yarn install"}, Output: "success"},
		{Iteration: 3, Action: Action{Type: ActionDone}},
	}

	solution := inferSolution(steps, 1)
	if solution == "" {
		t.Error("expected solution from step 2")
	}
	if !strings.Contains(solution, "yarn") {
		t.Errorf("expected solution to mention yarn, got %s", solution)
	}
}

func TestInferSolutionNoSuccess(t *testing.T) {
	steps := []StepRecord{
		{Iteration: 1, Action: Action{Type: ActionShell}, Error: "failed"},
		{Iteration: 2, Action: Action{Type: ActionFailed}},
	}

	solution := inferSolution(steps, 1)
	if solution != "" {
		t.Errorf("expected empty solution, got %s", solution)
	}
}

func TestFormatLessonsForPrompt(t *testing.T) {
	lessons := []ErrorLesson{
		{
			ErrorPattern: "npm: command not found",
			Context:      "running command: npm install",
			Solution:     "running command: yarn install",
			Timestamp:    time.Now(),
		},
		{
			ErrorPattern: "file not found",
			Context:      "reading file: config.json",
			Solution:     "",
			Timestamp:    time.Now(),
		},
	}

	result := FormatLessonsForPrompt(lessons)

	if !strings.Contains(result, "LESSONS LEARNED") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "npm: command not found") {
		t.Error("expected first lesson")
	}
	if !strings.Contains(result, "file not found") {
		t.Error("expected second lesson")
	}
	if !strings.Contains(result, "Solution:") {
		t.Error("expected solution label")
	}
}

func TestFormatLessonsForPromptEmpty(t *testing.T) {
	result := FormatLessonsForPrompt([]ErrorLesson{})
	if result != "" {
		t.Errorf("expected empty string for no lessons, got %s", result)
	}
}

func TestLessonStoreSaveLessons(t *testing.T) {
	var savedCount int
	saveFunc := func(title, content, topicKey string) error {
		savedCount++
		return nil
	}

	store := NewLessonStore(saveFunc, nil)

	lessons := []ErrorLesson{
		{ErrorPattern: "error 1", Timestamp: time.Now()},
		{ErrorPattern: "error 2", Timestamp: time.Now()},
	}

	err := store.SaveLessons(lessons)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if savedCount != 2 {
		t.Errorf("expected 2 saves, got %d", savedCount)
	}
}

func TestLessonStoreFindRelevantLessons(t *testing.T) {
	searchFunc := func(query string) ([]LessonResult, error) {
		return []LessonResult{
			{
				Title:   "Lesson 1",
				Content: formatLessonForTest(ErrorLesson{
					ErrorPattern: "npm error",
					TaskType:     "setup",
					WorkDir:      "/tmp/test",
					Timestamp:    time.Now(),
				}),
				TopicKey: "taskrunner/lessons/1",
			},
		}, nil
	}

	store := NewLessonStore(nil, searchFunc)

	lessons, err := store.FindRelevantLessons("install deps", "/tmp/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Errorf("expected 1 lesson, got %d", len(lessons))
	}
}

func TestIsRelevant(t *testing.T) {
	now := time.Now()
	old := now.Add(-10 * 24 * time.Hour)

	tests := []struct {
		name     string
		lesson   ErrorLesson
		workDir  string
		taskType string
		expected bool
	}{
		{
			name:     "same task type",
			lesson:   ErrorLesson{TaskType: "setup", Timestamp: old},
			workDir:  "/other",
			taskType: "setup",
			expected: true,
		},
		{
			name:     "same workdir",
			lesson:   ErrorLesson{WorkDir: "/tmp/test", Timestamp: old, TaskType: "other"},
			workDir:  "/tmp/test",
			taskType: "general",
			expected: true,
		},
		{
			name:     "recent lesson",
			lesson:   ErrorLesson{Timestamp: now, TaskType: "other", WorkDir: "/other"},
			workDir:  "/tmp",
			taskType: "general",
			expected: true,
		},
		{
			name:     "old unrelated lesson",
			lesson:   ErrorLesson{Timestamp: old, TaskType: "other", WorkDir: "/other"},
			workDir:  "/tmp",
			taskType: "general",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRelevant(tt.lesson, tt.workDir, tt.taskType)
			if got != tt.expected {
				t.Errorf("isRelevant() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func formatLessonForTest(lesson ErrorLesson) string {
	return "## Error Lesson\n\n**Error Pattern:** " + lesson.ErrorPattern +
		"\n\n## Raw Data (JSON)\n\n```json\n" +
		`{"error_pattern":"` + lesson.ErrorPattern + `","task_type":"` + lesson.TaskType + `","work_dir":"` + lesson.WorkDir + `","timestamp":"` + lesson.Timestamp.Format(time.RFC3339) + `"}` +
		"\n```\n"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
