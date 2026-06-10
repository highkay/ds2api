package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/auth"
	"ds2api/internal/config"
)

func TestExtractCreateSessionIDSupportsLegacyShape(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"biz_data": map[string]any{
				"id": "legacy-session-id",
			},
		},
	}

	if got := extractCreateSessionID(resp); got != "legacy-session-id" {
		t.Fatalf("expected legacy session id, got %q", got)
	}
}

func TestExtractCreateSessionIDSupportsNestedChatSessionShape(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"biz_data": map[string]any{
				"chat_session": map[string]any{
					"id":         "nested-session-id",
					"model_type": "default",
				},
			},
		},
	}

	if got := extractCreateSessionID(resp); got != "nested-session-id" {
		t.Fatalf("expected nested session id, got %q", got)
	}
}

func TestLoginReturnsAccountMutedForMutedBizResponse(t *testing.T) {
	c := &Client{
		regular: doerFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"code":0,"msg":"","data":{"biz_code":5,"biz_msg":"user_is_banned","biz_data":{"is_muted":true,"mute_until":4567}}}`,
				)),
				Request: req,
			}, nil
		}),
		fallback: &http.Client{},
	}

	_, err := c.Login(context.Background(), config.Account{
		Email:    "muted@example.com",
		Password: "secret",
	})
	if !errors.Is(err, auth.ErrAccountMuted) {
		t.Fatalf("expected auth.ErrAccountMuted, got %v", err)
	}
	if !IsAccountMutedError(err) {
		t.Fatalf("expected IsAccountMutedError to classify %v", err)
	}
}

func TestIsAccountMutedErrorDetectsRequestFailure(t *testing.T) {
	err := &RequestFailure{Op: "get pow", Kind: FailureAccountMuted, Message: "account is muted"}
	if !IsAccountMutedError(err) {
		t.Fatalf("expected account muted request failure to be classified")
	}
}
