package taskrunner

import (
	"strings"
	"testing"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/model"
)

func TestEngramSaverSaveReport(t *testing.T) {
	var savedTitle, savedContent, savedTopicKey string

	saveFunc := func(title, content, topicKey string) error {
		savedTitle = title
		savedContent = content
		savedTopicKey = topicKey
		return nil
	}

	saver := NewEngramSaver(saveFunc)

	report := &Report{
		Task:        "test task for engram",
		Status:      "success",
		Iterations:  5,
		Duration:    30 * time.Second,
		Steps:       []StepRecord{},
		FinalOutput: "All done",
		EngineUsed:  model.AgentClaudeCode,
		WorkDir:     "/tmp/test",
	}

	err := saver.SaveReport(report)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify title contains task
	if !strings.Contains(savedTitle, "Task Run:") {
		t.Error("expected title to contain 'Task Run:'")
	}

	// Verify topic key format
	if !strings.HasPrefix(savedTopicKey, "taskrunner/run/") {
		t.Errorf("expected topic key to start with 'taskrunner/run/', got %q", savedTopicKey)
	}

	// Verify content contains report data
	if !strings.Contains(savedContent, "## Task Execution Report") {
		t.Error("expected content to contain report header")
	}
	if !strings.Contains(savedContent, "test task for engram") {
		t.Error("expected content to contain task")
	}
	if !strings.Contains(savedContent, "success") {
		t.Error("expected content to contain status")
	}
}

func TestEngramSaverNilSaveFunc(t *testing.T) {
	saver := NewEngramSaver(nil)

	report := &Report{
		Task:   "test",
		Status: "success",
	}

	err := saver.SaveReport(report)
	if err == nil {
		t.Error("expected error when saveFunc is nil")
	}
}

func TestBuildEngramContent(t *testing.T) {
	report := &Report{
		Task:        "build test",
		Status:      "success",
		Iterations:  3,
		Duration:    15 * time.Second,
		Steps: []StepRecord{
			{Iteration: 1, Action: Action{Type: ActionShell}, Duration: 1 * time.Second},
			{Iteration: 2, Action: Action{Type: ActionWriteFile}, Duration: 2 * time.Second, Error: "test error"},
			{Iteration: 3, Action: Action{Type: ActionDone}, Duration: 500 * time.Millisecond},
		},
		FinalOutput: "Completed",
		EngineUsed:  model.AgentOpenCode,
		WorkDir:     "/tmp",
	}

	content := buildEngramContent(report)

	// Verify markdown structure
	if !strings.Contains(content, "## Task Execution Report") {
		t.Error("expected header")
	}
	if !strings.Contains(content, "## Summary") {
		t.Error("expected summary section")
	}
	if !strings.Contains(content, "## Steps") {
		t.Error("expected steps section")
	}
	if !strings.Contains(content, "## Raw Data (JSON)") {
		t.Error("expected raw data section")
	}

	// Verify steps table
	if !strings.Contains(content, "|") {
		t.Error("expected markdown table")
	}

	// Verify error marker
	if !strings.Contains(content, "✗") {
		t.Error("expected error marker in table")
	}
}

func TestBuildEngramContentNoSteps(t *testing.T) {
	report := &Report{
		Task:        "simple task",
		Status:      "success",
		Iterations:  1,
		Duration:    5 * time.Second,
		Steps:       []StepRecord{},
		FinalOutput: "Done",
		EngineUsed:  model.AgentClaudeCode,
		WorkDir:     "/tmp",
	}

	content := buildEngramContent(report)

	// Should not have steps section
	if strings.Contains(content, "## Steps") {
		t.Error("should not have steps section when no steps")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"short", 100, "short"},
		{strings.Repeat("a", 200), 50, strings.Repeat("a", 47) + "..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
		}
	}
}

func TestTruncateExactLength(t *testing.T) {
	input := "exact"
	result := truncate(input, 5)
	if result != input {
		t.Errorf("expected unchanged for exact length, got %q", result)
	}
}
