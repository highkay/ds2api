package riskguard

import (
	"net/http"
	"strings"
	"testing"
)

type testRuntimeLimits struct {
	maxPromptChars int
	maxRefFiles    int
}

func (l testRuntimeLimits) RuntimeMaxPromptChars() int {
	return l.maxPromptChars
}

func (l testRuntimeLimits) RuntimeMaxRefFilesPerRequest() int {
	return l.maxRefFiles
}

func TestCheckCompletionRejectsOversizedPrompt(t *testing.T) {
	err := CheckCompletion(testRuntimeLimits{maxPromptChars: 4, maxRefFiles: 8}, "hello", nil)
	status, code, message, ok := ErrorDetail(err)
	if !ok {
		t.Fatalf("expected violation, got %v", err)
	}
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=%d", status, http.StatusRequestEntityTooLarge)
	}
	if code != "prompt_too_large" {
		t.Fatalf("code=%q want=prompt_too_large", code)
	}
	if !strings.Contains(message, "limit is 4") {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestCheckCompletionRejectsTooManyRefFiles(t *testing.T) {
	err := CheckCompletion(testRuntimeLimits{maxPromptChars: 100, maxRefFiles: 2}, "ok", []string{"f1", "f2", "f3"})
	status, code, _, ok := ErrorDetail(err)
	if !ok {
		t.Fatalf("expected violation, got %v", err)
	}
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=%d", status, http.StatusRequestEntityTooLarge)
	}
	if code != "too_many_ref_files" {
		t.Fatalf("code=%q want=too_many_ref_files", code)
	}
}

func TestCheckCompletionAllowsRequestWithinLimits(t *testing.T) {
	if err := CheckCompletion(testRuntimeLimits{maxPromptChars: 10, maxRefFiles: 2}, "hello", []string{"f1", "f2"}); err != nil {
		t.Fatalf("expected request within limits, got %v", err)
	}
}
