package config

import (
	"strings"
	"testing"
)

func TestValidateConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "admin",
			cfg:  Config{Admin: AdminConfig{JWTExpireHours: 721}},
			want: "admin.jwt_expire_hours",
		},
		{
			name: "runtime relation",
			cfg: Config{Runtime: RuntimeConfig{
				AccountMaxInflight: 8,
				GlobalMaxInflight:  4,
			}},
			want: "runtime.global_max_inflight must be >= runtime.account_max_inflight",
		},
		{
			name: "runtime prompt block rule empty terms",
			cfg: Config{Runtime: RuntimeConfig{
				PromptBlockRules: []PromptBlockRule{{
					Name: "stock extraction",
				}},
			}},
			want: "runtime.prompt_block_rules[0].contains_all",
		},
		{
			name: "runtime prompt block rule blank term",
			cfg: Config{Runtime: RuntimeConfig{
				PromptBlockRules: []PromptBlockRule{{
					ContainsAll: []string{"rag_search", "   "},
				}},
			}},
			want: "runtime.prompt_block_rules[0].contains_all[1]",
		},
		{
			name: "responses",
			cfg:  Config{Responses: ResponsesConfig{StoreTTLSeconds: 10}},
			want: "responses.store_ttl_seconds",
		},
		{
			name: "embeddings",
			cfg:  Config{Embeddings: EmbeddingsConfig{Provider: "   "}},
			want: "embeddings.provider",
		},
		{
			name: "auto delete",
			cfg:  Config{AutoDelete: AutoDeleteConfig{Mode: "maybe"}},
			want: "auto_delete.mode",
		},
		{
			name: "history split",
			cfg: Config{HistorySplit: HistorySplitConfig{
				TriggerAfterTurns: intPtr(0),
			}},
			want: "history_split.trigger_after_turns",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(tc.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q in error, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateConfigAcceptsLegacyAutoDeleteSessions(t *testing.T) {
	if err := ValidateConfig(Config{AutoDelete: AutoDeleteConfig{Sessions: true}}); err != nil {
		t.Fatalf("expected legacy auto_delete.sessions config to remain valid, got %v", err)
	}
}

func TestValidateRuntimeConfigAcceptsZeroAccountMaxQueue(t *testing.T) {
	if err := ValidateRuntimeConfig(RuntimeConfig{AccountMaxInflight: 1, AccountMaxQueue: 0, GlobalMaxInflight: 1}); err != nil {
		t.Fatalf("expected zero account_max_queue to be valid, got %v", err)
	}
}

func TestValidateRuntimeConfigAcceptsPromptBlockRule(t *testing.T) {
	err := ValidateRuntimeConfig(RuntimeConfig{
		PromptBlockRules: []PromptBlockRule{{
			Name:        "stock extraction",
			ContainsAll: []string{"股票标的提取助手", "rag_search", "rq_web_search"},
			Message:     "route elsewhere",
		}},
	})
	if err != nil {
		t.Fatalf("expected prompt block rule to be valid, got %v", err)
	}
}

func intPtr(v int) *int { return &v }
