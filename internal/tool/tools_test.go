package tool

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestLocalCommandToolRequiresConfirmation(t *testing.T) {
	tool := &LocalCommandTool{}
	if !tool.RequiresConfirmation(map[string]interface{}{"command": "echo hello"}) {
		t.Fatalf("expected local command tool to require confirmation")
	}
}

func TestLocalCommandToolExecute(t *testing.T) {
	tool := &LocalCommandTool{}
	command := "printf hello"
	if runtime.GOOS == "windows" {
		command = "echo hello"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{"command": command})
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful execution, got error result: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Fatalf("expected output to contain hello, got: %s", result.Content)
	}
}
