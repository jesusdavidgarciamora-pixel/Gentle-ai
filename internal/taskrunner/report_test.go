package taskrunner

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/model"
)

func TestPrintReportSuccess(t *testing.T) {
	var buf bytes.Buffer
	report := &Report{
		Task:        "test task",
		Status:      "success",
		Iterations:  5,
		Duration:    30 * time.Second,
		FinalOutput: "Task completed successfully",
		EngineUsed:  model.AgentClaudeCode,
		WorkDir:     "/tmp/test",
	}

	PrintReport(&buf, report)

	output := buf.String()

	// Should contain DONE marker
	if !strings.Contains(output, "DONE") {
		t.Error("expected DONE marker in output")
	}

	// Should contain task
	if !strings.Contains(output, "test task") {
		t.Error("expected task in output")
	}

	// Should contain iterations
	if !strings.Contains(output, "5") {
		t.Error("expected iterations in output")
	}

	// Should contain engine
	if !strings.Contains(output, string(model.AgentClaudeCode)) {
		t.Error("expected engine in output")
	}

	// Should contain summary
	if !strings.Contains(output, "Task completed successfully") {
		t.Error("expected summary in output")
	}
}

func TestPrintReportFailed(t *testing.T) {
	var buf bytes.Buffer
	report := &Report{
		Task:        "failing task",
		Status:      "failed",
		Iterations:  10,
		Duration:    1 * time.Minute,
		FinalOutput: "Could not complete",
		EngineUsed:  model.AgentOpenCode,
		WorkDir:     "/tmp",
	}

	PrintReport(&buf, report)

	output := buf.String()

	// Should contain FAILED marker
	if !strings.Contains(output, "FAILED") {
		t.Error("expected FAILED marker in output")
	}
}

func TestPrintVerboseReport(t *testing.T) {
	var buf bytes.Buffer
	report := &Report{
		Task:       "verbose task",
		Status:     "success",
		Iterations: 3,
		Duration:   10 * time.Second,
		Steps: []StepRecord{
			{Iteration: 1, Action: Action{Type: ActionShell}, Duration: 1 * time.Second},
			{Iteration: 2, Action: Action{Type: ActionWriteFile}, Duration: 2 * time.Second, Error: "some error"},
			{Iteration: 3, Action: Action{Type: ActionDone}, Duration: 500 * time.Millisecond},
		},
		FinalOutput: "Done",
		EngineUsed:  model.AgentClaudeCode,
		WorkDir:     "/tmp",
	}

	PrintVerboseReport(&buf, report)

	output := buf.String()

	// Should contain steps section
	if !strings.Contains(output, "Steps:") {
		t.Error("expected Steps section in verbose output")
	}

	// Should contain step details
	if !strings.Contains(output, "shell") {
		t.Error("expected step types in output")
	}
}

func TestPrintReportMultilineSummary(t *testing.T) {
	var buf bytes.Buffer
	report := &Report{
		Task:        "multiline task",
		Status:      "success",
		Iterations:  1,
		Duration:    5 * time.Second,
		FinalOutput: "Line 1\nLine 2\nLine 3",
		EngineUsed:  model.AgentGeminiCLI,
		WorkDir:     "/tmp",
	}

	PrintReport(&buf, report)

	output := buf.String()

	// Each line should be indented
	if !strings.Contains(output, "Line 1") {
		t.Error("expected Line 1 in output")
	}
	if !strings.Contains(output, "Line 2") {
		t.Error("expected Line 2 in output")
	}
}

func TestTruncateOutputShort(t *testing.T) {
	input := "short string"
	result := TruncateOutput(input, 100)

	if result != input {
		t.Errorf("expected unchanged string for short input, got %q", result)
	}
}

func TestTruncateOutputLong(t *testing.T) {
	input := strings.Repeat("a", 1000)
	result := TruncateOutput(input, 100)

	if len(result) >= len(input) {
		t.Error("expected truncated output")
	}

	if !strings.Contains(result, "truncated") {
		t.Error("expected truncation marker")
	}
}
