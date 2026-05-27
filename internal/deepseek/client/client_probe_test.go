package client

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	dsprotocol "ds2api/internal/deepseek/protocol"
)

func TestValidateTokenUsesCurrentUserEndpoint(t *testing.T) {
	var gotURL string
	var gotAuth string
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		gotAuth = req.Header.Get("Authorization")
		return probeResponse(http.StatusOK, `{"code":0,"data":{"biz_code":0}}`, req), nil
	})}

	result, err := c.ValidateToken(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected token valid, got %#v", result)
	}
	if gotURL != dsprotocol.DeepSeekCurrentUserURL {
		t.Fatalf("unexpected endpoint: %s", gotURL)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
}

func TestValidateTokenMarksInvalidStatus(t *testing.T) {
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		return probeResponse(http.StatusUnauthorized, `{"code":401,"msg":"invalid token","data":{"biz_code":401}}`, req), nil
	})}

	result, err := c.ValidateToken(context.Background(), "bad")
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid token, got %#v", result)
	}
	if result.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", result.HTTPStatus)
	}
	if !strings.Contains(result.Message, "invalid token") {
		t.Fatalf("expected invalid token message, got %q", result.Message)
	}
}

func TestGetAccountCapabilitiesParsesModelConfigsArray(t *testing.T) {
	var gotURL string
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		return probeResponse(http.StatusOK, `{
			"code":0,
			"data":{"biz_code":0,"biz_data":{"settings":{"model_configs":{"value":[
				{"model_type":"chat","switchable":true},
				{"model_type":"vision","switchable":true},
				{"model_type":"expert","switchable":false}
			]}}}}
		}`, req), nil
	})}

	result, err := c.GetAccountCapabilities(context.Background(), "tok", "u@example.com")
	if err != nil {
		t.Fatalf("GetAccountCapabilities error: %v", err)
	}
	if result.Vision == nil || !*result.Vision {
		t.Fatalf("expected vision switchable, got %#v", result.Vision)
	}
	if !reflect.DeepEqual(result.Models, []string{"chat", "expert", "vision"}) {
		t.Fatalf("unexpected models: %#v", result.Models)
	}
	if result.Source != "client_settings" {
		t.Fatalf("unexpected source: %q", result.Source)
	}
	if !strings.HasPrefix(gotURL, dsprotocol.DeepSeekClientSettingsURL+"?") {
		t.Fatalf("unexpected endpoint: %s", gotURL)
	}
	if !strings.Contains(gotURL, "scope=model") || !strings.Contains(gotURL, "did=") {
		t.Fatalf("expected scope and did query, got %s", gotURL)
	}
}

func TestGetAccountCapabilitiesParsesModelConfigsString(t *testing.T) {
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		return probeResponse(http.StatusOK, `{
			"code":0,
			"data":{"biz_code":0,"biz_data":{"settings":{"model_configs":{"value":"[{\"model_type\":\"vision\",\"switchable\":\"false\"}]"}}}}
		}`, req), nil
	})}

	result, err := c.GetAccountCapabilities(context.Background(), "tok", "u@example.com")
	if err != nil {
		t.Fatalf("GetAccountCapabilities error: %v", err)
	}
	if result.Vision == nil || *result.Vision {
		t.Fatalf("expected vision switchable=false, got %#v", result.Vision)
	}
	if !reflect.DeepEqual(result.Models, []string{"vision"}) {
		t.Fatalf("unexpected models: %#v", result.Models)
	}
}

func TestGetAccountCapabilitiesMarksVisionUnavailableWhenAbsent(t *testing.T) {
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		return probeResponse(http.StatusOK, `{
			"code":0,
			"data":{"biz_code":0,"biz_data":{"settings":{"model_configs":{"value":[
				{"model_type":"chat","switchable":true},
				{"model_type":"expert","switchable":true}
			]}}}}
		}`, req), nil
	})}

	result, err := c.GetAccountCapabilities(context.Background(), "tok", "u@example.com")
	if err != nil {
		t.Fatalf("GetAccountCapabilities error: %v", err)
	}
	if result.Vision == nil || *result.Vision {
		t.Fatalf("expected vision unavailable, got %#v", result.Vision)
	}
	if !reflect.DeepEqual(result.Models, []string{"chat", "expert"}) {
		t.Fatalf("unexpected models: %#v", result.Models)
	}
}

func TestGetAccountMuteStatusParsesNestedChatMute(t *testing.T) {
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		return probeResponse(http.StatusOK, `{
			"code":0,
			"data":{"biz_code":0,"biz_data":{"chat":{"is_muted":1,"mute_until":1234.5}}}
		}`, req), nil
	})}

	result, err := c.GetAccountMuteStatus(context.Background(), "tok")
	if err != nil {
		t.Fatalf("GetAccountMuteStatus error: %v", err)
	}
	if !result.Muted || result.MuteUntil != 1234.5 {
		t.Fatalf("unexpected mute status: %#v", result)
	}
	if result.Source != "users_current" {
		t.Fatalf("unexpected source: %q", result.Source)
	}
}

func TestGetAccountMuteStatusReturnsMutedBizFailure(t *testing.T) {
	c := &Client{regular: doerFunc(func(req *http.Request) (*http.Response, error) {
		return probeResponse(http.StatusOK, `{
			"code":0,
			"data":{"biz_code":5,"biz_msg":"muted","biz_data":{"chat":{"is_muted":true,"mute_until":"4567"}}}
		}`, req), nil
	})}

	result, err := c.GetAccountMuteStatus(context.Background(), "tok")
	if err != nil {
		t.Fatalf("GetAccountMuteStatus error: %v", err)
	}
	if !result.Muted || result.MuteUntil != 4567 {
		t.Fatalf("unexpected mute status: %#v", result)
	}
	if result.BizCode != 5 {
		t.Fatalf("expected biz_code 5, got %d", result.BizCode)
	}
}

func probeResponse(status int, body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
