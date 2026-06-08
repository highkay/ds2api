package riskguard

import (
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/config"
)

type testRuntimeLimits struct {
	maxPromptChars         int
	maxRefFiles            int
	promptRiskGuardEnabled *bool
	promptBlockRules       []config.PromptBlockRule
}

func (l testRuntimeLimits) RuntimeMaxPromptChars() int {
	return l.maxPromptChars
}

func (l testRuntimeLimits) RuntimeMaxRefFilesPerRequest() int {
	return l.maxRefFiles
}

func (l testRuntimeLimits) RuntimePromptRiskGuardEnabled() bool {
	if l.promptRiskGuardEnabled == nil {
		return true
	}
	return *l.promptRiskGuardEnabled
}

func (l testRuntimeLimits) RuntimePromptBlockRules() []config.PromptBlockRule {
	return l.promptBlockRules
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

func TestCheckCompletionRejectsPromptBlockRule(t *testing.T) {
	limits := testRuntimeLimits{
		maxPromptChars: 1000,
		maxRefFiles:    8,
		promptBlockRules: []config.PromptBlockRule{{
			Name:        "stock_extraction_tools",
			ContainsAll: []string{"股票标的提取助手", "rag_search", "rq_web_search"},
			Message:     "route this extraction workload away from DeepSeek web accounts",
		}},
	}
	prompt := "你是一个专业的股票标的提取助手。请在必要时调用 rag_search 和 rq_web_search。"
	err := CheckCompletion(limits, prompt, nil)
	status, code, message, ok := ErrorDetail(err)
	if !ok {
		t.Fatalf("expected violation, got %v", err)
	}
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=%d", status, http.StatusUnprocessableEntity)
	}
	if code != "prompt_blocked" {
		t.Fatalf("code=%q want=prompt_blocked", code)
	}
	if message != "route this extraction workload away from DeepSeek web accounts" {
		t.Fatalf("message=%q", message)
	}
}

func TestCheckCompletionAllowsToolPromptWithoutFullRule(t *testing.T) {
	limits := testRuntimeLimits{
		maxPromptChars: 1000,
		maxRefFiles:    8,
		promptBlockRules: []config.PromptBlockRule{{
			ContainsAll: []string{"股票标的提取助手", "rag_search", "rq_web_search"},
		}},
	}
	prompt := "TOOL CALL FORMAT - FOLLOW EXACTLY. Available tools: rag_search and rq_web_search."
	if err := CheckCompletion(limits, prompt, nil); err != nil {
		t.Fatalf("expected tool prompt without full rule to pass, got %v", err)
	}
}

func TestCheckCompletionAllowsPromptBlockRuleWhenDisabled(t *testing.T) {
	disabled := false
	limits := testRuntimeLimits{
		maxPromptChars:         1000,
		maxRefFiles:            8,
		promptRiskGuardEnabled: &disabled,
		promptBlockRules: []config.PromptBlockRule{{
			ContainsAll: []string{"股票标的提取助手", "rag_search", "rq_web_search"},
		}},
	}
	prompt := "你是一个专业的股票标的提取助手。请调用 rag_search 和 rq_web_search。"
	if err := CheckCompletion(limits, prompt, nil); err != nil {
		t.Fatalf("expected disabled prompt guard to pass, got %v", err)
	}
}
