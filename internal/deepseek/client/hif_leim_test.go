package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/auth"
	dsprotocol "ds2api/internal/deepseek/protocol"
)

func hifLeimResponse(value string) *http.Response {
	body := `{"code":0,"msg":"","data":{"biz_code":0,"biz_msg":"","biz_data":{"value":"` + value + `"}}}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetHifLeimUsesFallbackAndParsesValue(t *testing.T) {
	var fallbackURL string
	var fallbackAccept string
	client := &Client{
		regular: failingDoer{err: errors.New("primary failed")},
		fallback: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			fallbackURL = req.URL.String()
			fallbackAccept = req.Header.Get("Accept")
			return hifLeimResponse("hif-value"), nil
		})},
	}

	got, err := client.GetHifLeim(context.Background(), &auth.RequestAuth{DeepSeekToken: "token"})
	if err != nil {
		t.Fatalf("GetHifLeim returned error: %v", err)
	}
	if got != "hif-value" {
		t.Fatalf("GetHifLeim value=%q want %q", got, "hif-value")
	}
	if fallbackURL != dsprotocol.DeepSeekHifLeimURL {
		t.Fatalf("fallback url=%q want %q", fallbackURL, dsprotocol.DeepSeekHifLeimURL)
	}
	if fallbackAccept != "application/json" {
		t.Fatalf("fallback Accept=%q want application/json", fallbackAccept)
	}
}

func TestCallCompletionAddsHifLeimHeader(t *testing.T) {
	var seenHifLeim string
	client := &Client{
		regular: doerFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != dsprotocol.DeepSeekHifLeimURL {
				t.Fatalf("unexpected regular request url %s", req.URL.String())
			}
			return hifLeimResponse("hif-completion"), nil
		}),
		stream: doerFunc(func(req *http.Request) (*http.Response, error) {
			seenHifLeim = req.Header.Get("x-hif-leim")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`data: {"p":"response/content","v":"ok"}` + "\n" + `data: [DONE]` + "\n")),
				Request:    req,
			}, nil
		}),
		maxRetries: 1,
	}

	resp, err := client.CallCompletion(context.Background(), &auth.RequestAuth{
		DeepSeekToken: "token",
		AccountID:     "acct",
	}, map[string]any{}, "pow-response", 1)
	if err != nil {
		t.Fatalf("CallCompletion returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if seenHifLeim != "hif-completion" {
		t.Fatalf("completion x-hif-leim=%q want %q", seenHifLeim, "hif-completion")
	}
}

func TestCallContinueAddsHifLeimHeader(t *testing.T) {
	var seenHifLeim string
	client := &Client{
		regular: doerFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != dsprotocol.DeepSeekHifLeimURL {
				t.Fatalf("unexpected regular request url %s", req.URL.String())
			}
			return hifLeimResponse("hif-continue"), nil
		}),
		stream: doerFunc(func(req *http.Request) (*http.Response, error) {
			seenHifLeim = req.Header.Get("x-hif-leim")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`data: {"p":"response/content","v":"continued"}` + "\n" + `data: [DONE]` + "\n")),
				Request:    req,
			}, nil
		}),
	}

	resp, err := client.callContinue(context.Background(), &auth.RequestAuth{
		DeepSeekToken: "token",
		AccountID:     "acct",
	}, "session-123", 99, "pow-response")
	if err != nil {
		t.Fatalf("callContinue returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if seenHifLeim != "hif-continue" {
		t.Fatalf("continue x-hif-leim=%q want %q", seenHifLeim, "hif-continue")
	}
}
