package taskrunner

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseActionValid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ActionType
	}{
		{
			name:     "shell action",
			input:    `{"action":"shell","command":"echo hello","reason":"test"}`,
			expected: ActionShell,
		},
		{
			name:     "write file action",
			input:    `{"action":"write_file","path":"test.txt","content":"hello","reason":"create"}`,
			expected: ActionWriteFile,
		},
		{
			name:     "read file action",
			input:    `{"action":"read_file","path":"test.txt","reason":"read"}`,
			expected: ActionReadFile,
		},
		{
			name:     "edit file action",
			input:    `{"action":"edit_file","path":"test.txt","old_text":"old","new_text":"new","reason":"edit"}`,
			expected: ActionEditFile,
		},
		{
			name:     "done action",
			input:    `{"action":"done","summary":"Task completed","reason":"finished"}`,
			expected: ActionDone,
		},
		{
			name:     "failed action",
			input:    `{"action":"failed","summary":"Could not complete","reason":"error"}`,
			expected: ActionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := ParseAction(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if action.Type != tt.expected {
				t.Errorf("expected type %v, got %v", tt.expected, action.Type)
			}
		})
	}
}

func TestParseActionWithMarkdownFences(t *testing.T) {
	input := "```json\n{\"action\":\"shell\",\"command\":\"echo test\",\"reason\":\"test\"}\n```"

	action, err := ParseAction(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if action.Type != ActionShell {
		t.Errorf("expected type shell, got %v", action.Type)
	}
	if action.Command != "echo test" {
		t.Errorf("expected command 'echo test', got %q", action.Command)
	}
}

func TestParseActionInvalidJSON(t *testing.T) {
	input := "not valid json"

	_, err := ParseAction(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseActionInvalidType(t *testing.T) {
	input := `{"action":"invalid_type","reason":"test"}`

	_, err := ParseAction(input)
	if err == nil {
		t.Error("expected error for invalid action type")
	}
}

func TestParseActionEmpty(t *testing.T) {
	input := ""

	_, err := ParseAction(input)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain json",
			input:    `{"action":"shell"}`,
			expected: `{"action":"shell"}`,
		},
		{
			name:     "json with fences",
			input:    "```json\n{\"action\":\"shell\"}\n```",
			expected: `{"action":"shell"}`,
		},
		{
			name:     "json with plain fences",
			input:    "```\n{\"action\":\"shell\"}\n```",
			expected: `{"action":"shell"}`,
		},
		{
			name:     "text before and after",
			input:    "some text {\"action\":\"shell\"} more text",
			expected: `{"action":"shell"}`,
		},
		{
			name:     "no json",
			input:    "no json here",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.expected {
				t.Errorf("extractJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildTurnPrompt(t *testing.T) {
	task := "create a test file"
	history := []StepRecord{
		{
			Iteration: 1,
			Action:    Action{Type: ActionShell, Command: "ls", Reason: "list files"},
			Output:    "file1.txt file2.txt",
		},
	}
	workDir := "/tmp/test"

	prompt := BuildTurnPrompt(task, history, workDir, nil)

	// Should contain task
	if !strings.Contains(prompt, task) {
		t.Error("prompt should contain task")
	}

	// Should contain workDir
	if !strings.Contains(prompt, workDir) {
		t.Error("prompt should contain workDir")
	}

	// Should contain history
	if !strings.Contains(prompt, "HISTORY") {
		t.Error("prompt should contain HISTORY section")
	}

	// Should contain system prompt rules
	if !strings.Contains(prompt, "NEVER ask") {
		t.Error("prompt should contain autonomy rules")
	}
}

func TestBuildTurnPromptNoHistory(t *testing.T) {
	task := "simple task"
	history := []StepRecord{}
	workDir := "/tmp"

	prompt := BuildTurnPrompt(task, history, workDir, nil)

	// Should still contain task
	if !strings.Contains(prompt, task) {
		t.Error("prompt should contain task")
	}

	// Should not contain history section
	if strings.Contains(prompt, "HISTORY") {
		t.Error("prompt should not contain HISTORY when empty")
	}
}

func TestBuildTurnPromptLongHistory(t *testing.T) {
	task := "test task"
	workDir := "/tmp"

	// Create 15 history items
	history := make([]StepRecord, 15)
	for i := 0; i < 15; i++ {
		history[i] = StepRecord{
			Iteration: i + 1,
			Action:    Action{Type: ActionShell, Command: "echo " + string(rune('a'+i)), Reason: "step"},
		}
	}

	prompt := BuildTurnPrompt(task, history, workDir, nil)

	// Should indicate showing only last 10
	if !strings.Contains(prompt, "Showing last 10") {
		t.Error("prompt should indicate truncated history")
	}
}

func TestActionFields(t *testing.T) {
	action := Action{
		Type:    ActionEditFile,
		Path:    "/tmp/test.txt",
		Content: "content",
		OldText: "old",
		NewText: "new",
		Reason:  "reason",
		Summary: "summary",
	}

	// Verify all fields can be marshaled
	data, err := json.Marshal(action)
	if err != nil {
		t.Errorf("failed to marshal action: %v", err)
	}

	// Verify can be unmarshaled
	var decoded Action
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("failed to unmarshal action: %v", err)
	}

	if decoded.Type != action.Type {
		t.Error("type mismatch")
	}
	if decoded.Path != action.Path {
		t.Error("path mismatch")
	}
}
