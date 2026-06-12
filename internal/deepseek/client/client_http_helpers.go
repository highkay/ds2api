package client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

func readResponseBody(resp *http.Response) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	var reader io.Reader = resp.Body
	switch encoding {
	case "gzip":
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gz.Close() }()
		reader = gz
	case "br":
		reader = brotli.NewReader(resp.Body)
	}
	return io.ReadAll(reader)
}

func preview(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 160 {
		return s[:160]
	}
	return s
}

func previewJSONMap(body map[string]any) string {
	if len(body) == 0 {
		return "{}"
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(redactPreviewValue(body)); err != nil {
		return "<unavailable>"
	}
	return preview(bytes.TrimSpace(buf.Bytes()))
}

func redactPreviewValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			if sensitivePreviewKey(k) {
				out[k] = "<redacted>"
				continue
			}
			out[k] = redactPreviewValue(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, redactPreviewValue(value))
		}
		return out
	default:
		return typed
	}
}

func sensitivePreviewKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(k, "token") ||
		strings.Contains(k, "password") ||
		strings.Contains(k, "authorization") ||
		strings.Contains(k, "cookie") ||
		strings.Contains(k, "secret") ||
		strings.Contains(k, "credential")
}

func (c *Client) jsonHeaders(headers map[string]string) map[string]string {
	out := cloneStringMap(headers)
	out["Content-Type"] = "application/json"
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
