package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPostJSONWithStatusUsesProvidedFallbackClient(t *testing.T) {
	var fallbackCalled bool
	client := &Client{}
	primary := failingDoer{err: errors.New("primary failed")}
	fallbackDoer := doerFunc(func(req *http.Request) (*http.Response, error) {
		fallbackCalled = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	})

	resp, status, err := client.postJSONWithStatus(
		context.Background(),
		primary,
		fallbackDoer,
		"https://example.com/api",
		map[string]string{"x-test": "1"},
		map[string]any{"foo": "bar"},
	)
	if err != nil {
		t.Fatalf("postJSONWithStatus error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status=%d want=%d", status, http.StatusOK)
	}
	if !fallbackCalled {
		t.Fatal("expected provided fallback doer to be called")
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("unexpected response body: %#v", resp)
	}
}

func TestPostJSONWithStatusReturnsDecodeErrorWithPreview(t *testing.T) {
	client := &Client{}
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`<html>blocked by upstream</html>`)),
			Request:    req,
		}, nil
	})

	_, status, err := client.postJSONWithStatus(
		context.Background(),
		doer,
		doer,
		"https://example.com/api",
		nil,
		map[string]any{"foo": "bar"},
	)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if status != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", status, http.StatusForbidden)
	}
	if got := err.Error(); !strings.Contains(got, "decode json response failed") || !strings.Contains(got, "blocked by upstream") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetJSONWithStatusReturnsDecodeErrorWithPreview(t *testing.T) {
	client := &Client{}
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`too many requests`)),
			Request:    req,
		}, nil
	})

	_, status, err := client.getJSONWithStatus(
		context.Background(),
		doer,
		"https://example.com/api",
		nil,
	)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if status != http.StatusTooManyRequests {
		t.Fatalf("status=%d want=%d", status, http.StatusTooManyRequests)
	}
	if got := err.Error(); !strings.Contains(got, "decode json response failed") || !strings.Contains(got, "too many requests") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}
