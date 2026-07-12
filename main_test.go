package main

import (
	"strings"
	"testing"
)

func TestGuardedPromptNoPriors(t *testing.T) {
	if got := guardedPrompt("fix the crash", nil); got != "fix the crash" {
		t.Errorf("prompt must pass through unchanged, got %q", got)
	}
}

func TestGuardedPromptWithPriors(t *testing.T) {
	got := guardedPrompt("fix the crash", []string{"claude", "gemini"})
	wantPrefix := "[usher failover] A previous agent (claude, gemini) was already working on this task"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("notice prefix missing:\n%q", got)
	}
	if !strings.Contains(got, "Inspect `git status` and the diff before continuing") {
		t.Errorf("git-status instruction missing:\n%q", got)
	}
	if !strings.HasSuffix(got, "\n\nfix the crash") {
		t.Errorf("original task must end the prompt untouched:\n%q", got)
	}
}
