package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/config"
)

type captureLoginDoer struct {
	method  string
	url     string
	headers http.Header
	body    []byte
}

func (d *captureLoginDoer) Do(req *http.Request) (*http.Response, error) {
	d.method = req.Method
	d.url = req.URL.String()
	d.headers = req.Header.Clone()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	d.body = body
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			`{"code":0,"msg":"","data":{"biz_code":0,"biz_msg":"","biz_data":{"user":{"token":"token-from-email-login"}}}}`,
		)),
	}, nil
}

func TestLoginPrefersEmailPayloadWhenEmailPresent(t *testing.T) {
	doer := &captureLoginDoer{}
	c := &Client{
		regular:   doer,
		stream:    doer,
		fallback:  &http.Client{},
		fallbackS: &http.Client{},
	}

	token, err := c.Login(context.Background(), config.Account{
		Email:    "global@example.com",
		Mobile:   "+8613800138000",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}
	if token != "token-from-email-login" {
		t.Fatalf("unexpected token: %q", token)
	}
	if doer.method != http.MethodPost {
		t.Fatalf("unexpected method: %q", doer.method)
	}

	var payload map[string]any
	if err := json.Unmarshal(doer.body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["email"] != "global@example.com" {
		t.Fatalf("expected email payload, got %#v", payload)
	}
	if _, ok := payload["mobile"]; ok {
		t.Fatalf("did not expect mobile in email login payload, got %#v", payload)
	}
	if _, ok := payload["area_code"]; ok {
		t.Fatalf("did not expect area_code in email login payload, got %#v", payload)
	}
	if payload["password"] != "secret" {
		t.Fatalf("expected password preserved, got %#v", payload)
	}
}
