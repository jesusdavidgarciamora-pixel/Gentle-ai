package taskrunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	exec := NewExecutor("/tmp", 5*time.Minute)
	if exec.WorkDir != "/tmp" {
		t.Errorf("expected workdir /tmp, got %s", exec.WorkDir)
	}
	if exec.Timeout != 5*time.Minute {
		t.Errorf("expected timeout 5m, got %v", exec.Timeout)
	}
}

func TestExecutorExecuteShell(t *testing.T) {
	exec := NewExecutor(".", 10*time.Second)
	ctx := context.Background()

	// Test successful command
	action := Action{
		Type:    ActionShell,
		Command: "echo hello world",
	}

	output, err := exec.Execute(ctx, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected output to contain 'hello world', got: %s", output)
	}
}

func TestExecutorExecuteShellEmpty(t *testing.T) {
	exec := NewExecutor(".", 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionShell,
		Command: "   ",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestExecutorWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionWriteFile,
		Path:    "test.txt",
		Content: "hello world",
	}

	output, err := exec.Execute(ctx, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Errorf("failed to read file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(content))
	}

	if !strings.Contains(output, "test.txt") {
		t.Error("expected output to mention filename")
	}
}

func TestExecutorWriteFileNested(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionWriteFile,
		Path:    "nested/dir/file.txt",
		Content: "nested content",
	}

	_, err := exec.Execute(ctx, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify nested directories were created
	content, err := os.ReadFile(filepath.Join(tmpDir, "nested/dir/file.txt"))
	if err != nil {
		t.Errorf("failed to read nested file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("expected 'nested content', got %s", string(content))
	}
}

func TestExecutorWriteFileEmptyPath(t *testing.T) {
	exec := NewExecutor(".", 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionWriteFile,
		Path:    "",
		Content: "content",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestExecutorReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	// Create a file first
	testContent := "test file content"
	err := os.WriteFile(filepath.Join(tmpDir, "readtest.txt"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	action := Action{
		Type: ActionReadFile,
		Path: "readtest.txt",
	}

	output, err := exec.Execute(ctx, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != testContent {
		t.Errorf("expected %q, got %q", testContent, output)
	}
}

func TestExecutorReadFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type: ActionReadFile,
		Path: "nonexistent.txt",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExecutorEditFile(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	// Create a file first
	testContent := "hello old world"
	err := os.WriteFile(filepath.Join(tmpDir, "edittest.txt"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	action := Action{
		Type:    ActionEditFile,
		Path:    "edittest.txt",
		OldText: "old",
		NewText: "new",
	}

	_, err = exec.Execute(ctx, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file was edited
	content, err := os.ReadFile(filepath.Join(tmpDir, "edittest.txt"))
	if err != nil {
		t.Errorf("failed to read file: %v", err)
	}
	expected := "hello new world"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}

func TestExecutorEditFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionEditFile,
		Path:    "nonexistent.txt",
		OldText: "old",
		NewText: "new",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExecutorEditFileOldTextNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	// Create a file
	err := os.WriteFile(filepath.Join(tmpDir, "edittest2.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	action := Action{
		Type:    ActionEditFile,
		Path:    "edittest2.txt",
		OldText: "notfound",
		NewText: "replacement",
	}

	_, err = exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error when old_text not found")
	}
}

func TestExecutorEditFileEmptyOldText(t *testing.T) {
	tmpDir := t.TempDir()
	exec := NewExecutor(tmpDir, 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type:    ActionEditFile,
		Path:    "test.txt",
		OldText: "",
		NewText: "replacement",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for empty old_text")
	}
}

func TestExecutorUnknownAction(t *testing.T) {
	exec := NewExecutor(".", 10*time.Second)
	ctx := context.Background()

	action := Action{
		Type: "unknown_action",
	}

	_, err := exec.Execute(ctx, action)
	if err == nil {
		t.Error("expected error for unknown action type")
	}
}

func TestFormatActionResult(t *testing.T) {
	action := Action{
		Type:    ActionShell,
		Command: "echo test",
		Reason:  "testing",
	}

	result := FormatActionResult(action, "output", nil)
	if !strings.Contains(result, "shell") {
		t.Error("expected result to contain action type")
	}
	if !strings.Contains(result, "testing") {
		t.Error("expected result to contain reason")
	}
	if !strings.Contains(result, "output") {
		t.Error("expected result to contain output")
	}
}

func TestFormatActionResultWithError(t *testing.T) {
	action := Action{
		Type:   ActionWriteFile,
		Path:   "test.txt",
		Reason: "create file",
	}

	result := FormatActionResult(action, "", os.ErrNotExist)
	if !strings.Contains(result, "ERROR") {
		t.Error("expected result to contain ERROR")
	}
}
