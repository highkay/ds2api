package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	dsprotocol "ds2api/internal/deepseek/protocol"
)

type TokenValidationResult struct {
	Valid      bool
	HTTPStatus int
	Code       int
	BizCode    int
	Message    string
}

type AccountCapabilities struct {
	Vision    *bool
	Models    []string
	CheckedAt int64
	Source    string
}

type AccountMuteStatus struct {
	Muted      bool
	MuteUntil  float64
	CheckedAt  int64
	HTTPStatus int
	Code       int
	BizCode    int
	Message    string
	Source     string
}

func (c *Client) ValidateToken(ctx context.Context, token string) (*TokenValidationResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return &TokenValidationResult{Valid: false, Message: "empty token"}, nil
	}
	clients := c.requestClientsFromContext(ctx)
	body, status, err := c.getJSONWithStatus(ctx, clients.regular, dsprotocol.DeepSeekCurrentUserURL, c.authHeaders(token))
	if err != nil {
		return nil, err
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(body)
	valid := status == http.StatusOK && code == 0 && bizCode == 0 && !isTokenInvalid(status, code, bizCode, msg, bizMsg)
	result := &TokenValidationResult{Valid: valid, HTTPStatus: status, Code: code, BizCode: bizCode}
	if !valid {
		result.Message = failureMessage(msg, bizMsg, fmt.Sprintf("HTTP %d", status))
	}
	return result, nil
}

func (c *Client) GetAccountMuteStatus(ctx context.Context, token string) (*AccountMuteStatus, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("empty token")
	}
	clients := c.requestClientsFromContext(ctx)
	body, status, err := c.getJSONWithStatus(ctx, clients.regular, dsprotocol.DeepSeekCurrentUserURL, c.authHeaders(token))
	if err != nil {
		return nil, err
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(body)
	info := extractMuteInfo(body)
	result := &AccountMuteStatus{
		Muted:      info.Muted,
		MuteUntil:  info.Until,
		CheckedAt:  time.Now().Unix(),
		HTTPStatus: status,
		Code:       code,
		BizCode:    bizCode,
		Message:    failureMessage(msg, bizMsg, fmt.Sprintf("HTTP %d", status)),
		Source:     "users_current",
	}
	if info.Muted {
		return result, nil
	}
	if status != http.StatusOK || code != 0 || bizCode != 0 || isTokenInvalid(status, code, bizCode, msg, bizMsg) {
		return nil, fmt.Errorf("current user failed: %s", result.Message)
	}
	result.Message = ""
	return result, nil
}

func (c *Client) GetAccountCapabilities(ctx context.Context, token string, accountID string) (*AccountCapabilities, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("empty token")
	}
	endpoint, err := url.Parse(dsprotocol.DeepSeekClientSettingsURL)
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("did", DeviceID(accountID))
	q.Set("scope", "model")
	endpoint.RawQuery = q.Encode()

	clients := c.requestClientsFromContext(ctx)
	body, status, err := c.getJSONWithStatus(ctx, clients.regular, endpoint.String(), c.authHeaders(token))
	if err != nil {
		return nil, err
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(body)
	if status != http.StatusOK || code != 0 || bizCode != 0 || isTokenInvalid(status, code, bizCode, msg, bizMsg) {
		return nil, fmt.Errorf("client settings failed: %s", failureMessage(msg, bizMsg, fmt.Sprintf("HTTP %d", status)))
	}

	configs := extractModelConfigs(body)
	modelSet := map[string]struct{}{}
	var vision *bool
	seenVision := false
	for _, item := range configs {
		modelType := stringFromProbeAny(item["model_type"])
		if modelType == "" {
			modelType = stringFromProbeAny(item["modelType"])
		}
		if modelType != "" {
			modelSet[modelType] = struct{}{}
		}
		if modelType == "vision" {
			seenVision = true
			if value, ok := boolFromProbeAny(item["switchable"]); ok {
				vision = boolPtr(value)
			}
		}
	}
	if len(configs) > 0 && !seenVision {
		vision = boolPtr(false)
	}
	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return &AccountCapabilities{
		Vision:    vision,
		Models:    models,
		CheckedAt: time.Now().Unix(),
		Source:    "client_settings",
	}, nil
}

func extractModelConfigs(body map[string]any) []map[string]any {
	data, _ := body["data"].(map[string]any)
	bizData, _ := data["biz_data"].(map[string]any)
	settings, _ := bizData["settings"].(map[string]any)
	modelConfigs, _ := settings["model_configs"].(map[string]any)
	switch raw := modelConfigs["value"].(type) {
	case []any:
		out := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case string:
		var decoded []map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			return decoded
		}
		var rawItems []any
		if err := json.Unmarshal([]byte(raw), &rawItems); err != nil {
			return nil
		}
		out := make([]map[string]any, 0, len(rawItems))
		for _, item := range rawItems {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func stringFromProbeAny(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func boolFromProbeAny(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	case float64:
		return x != 0, true
	case int:
		return x != 0, true
	}
	return false, false
}

func boolPtr(v bool) *bool {
	return &v
}
