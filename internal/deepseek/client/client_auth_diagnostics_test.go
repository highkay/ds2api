package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/config"
)

func TestLoginMissingTokenIncludesSanitizedBodyPreview(t *testing.T) {
	c := &Client{
		regular: doerFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"code":0,"msg":"","data":{"biz_code":0,"biz_msg":"","biz_data":{"user":{"token":"","refresh_token":"refresh-secret","name":"tester"}}}}`,
				)),
				Request: req,
			}, nil
		}),
		fallback: &http.Client{},
	}

	_, err := c.Login(context.Background(), config.Account{
		Email:    "tester@example.com",
		Password: "password-secret",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing login token: body=") {
		t.Fatalf("expected body preview, got %q", msg)
	}
	if strings.Contains(msg, "refresh-secret") || strings.Contains(msg, "password-secret") {
		t.Fatalf("expected secrets to be redacted, got %q", msg)
	}
	if !strings.Contains(msg, "<redacted>") {
		t.Fatalf("expected redaction marker, got %q", msg)
	}
}

func TestLoginFailureIncludesCodeAndBodyPreview(t *testing.T) {
	c := &Client{
		regular: doerFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"code":1,"msg":"need captcha","data":{"biz_code":1001,"biz_msg":"captcha_required"}}`,
				)),
				Request: req,
			}, nil
		}),
		fallback: &http.Client{},
	}

	_, err := c.Login(context.Background(), config.Account{
		Email:    "tester@example.com",
		Password: "password-secret",
	})
	if err == nil {
		t.Fatal("expected login failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "code=1") || !strings.Contains(msg, "need captcha") || !strings.Contains(msg, "body=") {
		t.Fatalf("expected diagnostic login error, got %q", msg)
	}
}
