//go:build live_deepseek_probe

package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"ds2api/internal/promptcompat"
	"ds2api/internal/sse"
)

type liveProbeProfile string

const (
	liveProbeCurrentAndroid liveProbeProfile = "current-android"
	liveProbeCandidateWeb   liveProbeProfile = "candidate-web"
)

type liveProbeHTTPResult struct {
	Transport string
	Status    int
	Code      int
	BizCode   int
	Msg       string
	BizMsg    string
	Preview   string
}

type liveProbeEffectSummary struct {
	Profile            liveProbeProfile
	Total              int
	OK                 int
	Failures           int
	RiskEvents         int
	LoginFailures      int
	SessionFailures    int
	PowFailures        int
	CompletionFailures int
	Status403          int
	Status429          int
	EmptyBody          int
	Categories         map[string]int
}

type liveProbeCompletionBodyResult struct {
	Status       int
	Bytes        int
	Kind         string
	ParsedLines  int
	ContentBytes int
	Done         bool
	Preview      string
	ErrorMessage string
}

func TestLiveDeepSeekFourStageProbe(t *testing.T) {
	store, acc, ok, source := liveProbeAccount(t)
	if !ok {
		t.Skip("set DS2API_TEST_EMAIL or DS2API_TEST_MOBILE plus DS2API_TEST_PASSWORD, or provide DS2API_CONFIG_PATH/DS2API_CONFIG_JSON with a password-backed account")
	}

	client := NewClient(store, nil)
	profiles := liveProbeSelectedProfiles()
	requireSelected := liveProbeRequiresSelectedProfile()
	accountID := liveProbeMaskIdentifier(acc.Identifier())
	t.Logf("source=%s account=%s profiles=%s", source, accountID, liveProbeProfilesString(profiles))

	successes := 0
	for _, profile := range profiles {
		profile := profile
		t.Run(string(profile), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), liveProbeTimeout())
			defer cancel()

			var err error
			switch profile {
			case liveProbeCurrentAndroid:
				err = runLiveProbeCurrentAndroid(ctx, t, client, acc)
			case liveProbeCandidateWeb:
				err = runLiveProbeCandidateWeb(ctx, t, client, acc)
			default:
				err = fmt.Errorf("unknown probe profile %q", profile)
			}
			if err != nil {
				t.Logf("profile=%s result=fail account=%s error=%v", profile, accountID, err)
				if requireSelected {
					t.Fatal(err)
				}
				return
			}
			successes++
			t.Logf("profile=%s result=ok account=%s", profile, accountID)
		})
	}
	if successes == 0 {
		t.Fatalf("all selected live probe profiles failed for account=%s", accountID)
	}
}

func TestLiveDeepSeekBanRiskABProbe(t *testing.T) {
	store, acc, ok, source := liveProbeAccount(t)
	if !ok {
		t.Skip("set DS2API_TEST_EMAIL or DS2API_TEST_MOBILE plus DS2API_TEST_PASSWORD, or provide DS2API_CONFIG_PATH/DS2API_CONFIG_JSON with a password-backed account")
	}

	client := NewClient(store, nil)
	profiles := liveProbeSelectedProfiles()
	requireSelected := liveProbeRequiresSelectedProfile()
	accountID := liveProbeMaskIdentifier(acc.Identifier())
	iterations := liveProbeIterations()
	sleep := liveProbeSleep()
	t.Logf("source=%s account=%s profiles=%s iterations=%d sleep=%s", source, accountID, liveProbeProfilesString(profiles), iterations, sleep)

	successes := 0
	for _, profile := range profiles {
		profile := profile
		t.Run(string(profile), func(t *testing.T) {
			summary := newLiveProbeEffectSummary(profile)
			for i := 1; i <= iterations; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), liveProbeTimeout())
				err := runLiveProbeProfile(ctx, t, client, acc, profile)
				cancel()

				summary.record(err)
				if err != nil {
					category, risk := liveProbeFailureCategory(err)
					stage := liveProbeFailureStage(err)
					t.Logf("profile=%s iteration=%d/%d result=fail stage=%s category=%s risk=%t error=%v", profile, i, iterations, stage, category, risk, err)
				} else {
					successes++
					t.Logf("profile=%s iteration=%d/%d result=ok", profile, i, iterations)
				}

				if i < iterations && sleep > 0 {
					time.Sleep(sleep)
				}
			}
			t.Logf("profile=%s ban_risk_summary %s", profile, summary.String())
			if requireSelected && summary.OK == 0 {
				t.Fatalf("selected profile had no successful completion: %s", summary.String())
			}
		})
	}
	if successes == 0 {
		t.Fatalf("no successful completion in any selected profile; compare the per-profile ban_risk_summary logs before changing runtime request shape")
	}
}

func newLiveProbeEffectSummary(profile liveProbeProfile) *liveProbeEffectSummary {
	return &liveProbeEffectSummary{
		Profile:    profile,
		Categories: map[string]int{},
	}
}

func (s *liveProbeEffectSummary) record(err error) {
	s.Total++
	if err == nil {
		s.OK++
		return
	}
	s.Failures++

	stage := liveProbeFailureStage(err)
	switch stage {
	case "login":
		s.LoginFailures++
	case "session":
		s.SessionFailures++
	case "pow":
		s.PowFailures++
	case "completion":
		s.CompletionFailures++
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "status=403") {
		s.Status403++
	}
	if strings.Contains(msg, "status=429") || strings.Contains(msg, "too_many_requests") {
		s.Status429++
	}
	if strings.Contains(msg, "empty first body window") || strings.Contains(msg, "empty response") {
		s.EmptyBody++
	}

	category, risk := liveProbeFailureCategory(err)
	if risk {
		s.RiskEvents++
	}
	s.Categories[category]++
}

func (s *liveProbeEffectSummary) String() string {
	riskRate := 0.0
	if s.Total > 0 {
		riskRate = float64(s.RiskEvents) * 100 / float64(s.Total)
	}
	return fmt.Sprintf("total=%d ok=%d failures=%d risk_events=%d risk_rate=%.1f%% login_fail=%d session_fail=%d pow_fail=%d completion_fail=%d status_403=%d status_429=%d empty_body=%d categories=%s",
		s.Total,
		s.OK,
		s.Failures,
		s.RiskEvents,
		riskRate,
		s.LoginFailures,
		s.SessionFailures,
		s.PowFailures,
		s.CompletionFailures,
		s.Status403,
		s.Status429,
		s.EmptyBody,
		liveProbeFormatCategories(s.Categories),
	)
}

func runLiveProbeProfile(ctx context.Context, t *testing.T, client *Client, acc config.Account, profile liveProbeProfile) error {
	switch profile {
	case liveProbeCurrentAndroid:
		return runLiveProbeCurrentAndroid(ctx, t, client, acc)
	case liveProbeCandidateWeb:
		return runLiveProbeCandidateWeb(ctx, t, client, acc)
	default:
		return fmt.Errorf("unknown probe profile %q", profile)
	}
}

func runLiveProbeCurrentAndroid(ctx context.Context, t *testing.T, client *Client, acc config.Account) error {
	profile := string(liveProbeCurrentAndroid)
	start := time.Now()
	token, err := client.Login(ctx, acc)
	if err != nil {
		return fmt.Errorf("login failed after %s: %w", time.Since(start), err)
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("login returned empty token after %s", time.Since(start))
	}
	t.Logf("profile=%s stage=login ok elapsed=%s", profile, time.Since(start))

	authCtx := &auth.RequestAuth{
		AccountID:     acc.Identifier(),
		Account:       acc,
		DeepSeekToken: token,
		TriedAccounts: map[string]bool{},
	}
	ctx = auth.WithAuth(ctx, authCtx)

	start = time.Now()
	sessionID, err := client.CreateSession(ctx, authCtx, 1)
	if err != nil {
		return fmt.Errorf("create_session failed after %s: %w", time.Since(start), err)
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("create_session returned empty id after %s", time.Since(start))
	}
	t.Logf("profile=%s stage=create_session ok elapsed=%s", profile, time.Since(start))

	start = time.Now()
	powResp, err := client.GetPowForTarget(ctx, authCtx, dsprotocol.DeepSeekCompletionTargetPath, 1)
	if err != nil {
		return fmt.Errorf("get_pow failed after %s: %w", time.Since(start), err)
	}
	if strings.TrimSpace(powResp) == "" {
		return fmt.Errorf("get_pow returned empty response after %s", time.Since(start))
	}
	t.Logf("profile=%s stage=get_pow ok elapsed=%s", profile, time.Since(start))

	payload := liveProbeCompletionPayload(sessionID)
	start = time.Now()
	resp, err := client.CallCompletion(ctx, authCtx, payload, powResp, 1)
	if err != nil {
		return fmt.Errorf("completion failed after %s: %w", time.Since(start), err)
	}
	if resp == nil {
		return fmt.Errorf("completion returned nil response after %s", time.Since(start))
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("completion body read failed after %s: %w", time.Since(start), readErr)
	}
	bodyResult := liveProbeClassifyCompletionBody(resp.StatusCode, body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("completion status=%d elapsed=%s bytes=%d preview=%q", resp.StatusCode, time.Since(start), len(body), preview(body))
	}
	if err := bodyResult.validate(); err != nil {
		return fmt.Errorf("completion invalid body after %s: %w", time.Since(start), err)
	}
	t.Logf("profile=%s stage=completion ok status=%d elapsed=%s bytes=%d kind=%s parsed=%d content_bytes=%d done=%t", profile, resp.StatusCode, time.Since(start), len(body), bodyResult.Kind, bodyResult.ParsedLines, bodyResult.ContentBytes, bodyResult.Done)
	return nil
}

func runLiveProbeCandidateWeb(ctx context.Context, t *testing.T, client *Client, acc config.Account) error {
	profile := string(liveProbeCandidateWeb)
	start := time.Now()
	token, loginResult, err := client.liveProbeWebLogin(ctx, acc)
	if err != nil {
		return fmt.Errorf("login failed after %s: %w", time.Since(start), err)
	}
	t.Logf("profile=%s stage=login ok transport=%s status=%d code=%d biz_code=%d elapsed=%s", profile, loginResult.Transport, loginResult.Status, loginResult.Code, loginResult.BizCode, time.Since(start))

	authCtx := &auth.RequestAuth{
		AccountID:     acc.Identifier(),
		Account:       acc,
		DeepSeekToken: token,
		TriedAccounts: map[string]bool{},
	}
	ctx = auth.WithAuth(ctx, authCtx)

	start = time.Now()
	sessionID, sessionResult, err := client.liveProbeWebCreateSession(ctx, acc, token)
	if err != nil {
		return fmt.Errorf("create_session failed after %s: %w", time.Since(start), err)
	}
	t.Logf("profile=%s stage=create_session ok status=%d code=%d biz_code=%d elapsed=%s", profile, sessionResult.Status, sessionResult.Code, sessionResult.BizCode, time.Since(start))

	start = time.Now()
	powResp, powResult, err := client.liveProbeWebGetPow(ctx, acc, token)
	if err != nil {
		return fmt.Errorf("get_pow failed after %s: %w", time.Since(start), err)
	}
	t.Logf("profile=%s stage=get_pow ok status=%d code=%d biz_code=%d elapsed=%s", profile, powResult.Status, powResult.Code, powResult.BizCode, time.Since(start))

	start = time.Now()
	bodyResult, err := client.liveProbeWebCompletion(ctx, acc, token, powResp, liveProbeCompletionPayload(sessionID))
	if err != nil {
		return fmt.Errorf("completion failed after %s: %w", time.Since(start), err)
	}
	if bodyResult.Status != http.StatusOK {
		return fmt.Errorf("completion status=%d elapsed=%s bytes=%d preview=%q", bodyResult.Status, time.Since(start), bodyResult.Bytes, bodyResult.Preview)
	}
	if err := bodyResult.validate(); err != nil {
		return fmt.Errorf("completion invalid body after %s: %w", time.Since(start), err)
	}
	t.Logf("profile=%s stage=completion ok status=%d elapsed=%s bytes=%d kind=%s parsed=%d content_bytes=%d done=%t", profile, bodyResult.Status, time.Since(start), bodyResult.Bytes, bodyResult.Kind, bodyResult.ParsedLines, bodyResult.ContentBytes, bodyResult.Done)
	return nil
}

func liveProbeAccount(t *testing.T) (*config.Store, config.Account, bool, string) {
	email := strings.TrimSpace(os.Getenv("DS2API_TEST_EMAIL"))
	mobile := strings.TrimSpace(os.Getenv("DS2API_TEST_MOBILE"))
	password := strings.TrimSpace(os.Getenv("DS2API_TEST_PASSWORD"))
	if password != "" && (email != "" || mobile != "") {
		return nil, config.Account{Email: email, Mobile: mobile, Password: password}, true, "env"
	}

	store, err := config.LoadStoreWithError()
	if err != nil {
		if strings.TrimSpace(os.Getenv("DS2API_CONFIG_PATH")) == "" && strings.TrimSpace(os.Getenv("DS2API_CONFIG_JSON")) == "" {
			if rootConfig, ok := liveProbeRepoRootConfigPath(); ok {
				t.Setenv("DS2API_CONFIG_PATH", rootConfig)
				store, err = config.LoadStoreWithError()
			} else {
				t.Logf("config-backed live probe account not loaded: no DS2API_CONFIG_PATH/DS2API_CONFIG_JSON or repo-root config.json; first load error: %v", err)
				return nil, config.Account{}, false, ""
			}
		}
		if err != nil {
			t.Logf("config-backed live probe account not loaded: %v", err)
			return nil, config.Account{}, false, ""
		}
	}
	filter := strings.TrimSpace(os.Getenv("DS2API_PROBE_ACCOUNT"))
	if filter == "" {
		filter = strings.TrimSpace(os.Getenv("DS2API_TEST_ACCOUNT"))
	}
	accountIndex, hasAccountIndex := liveProbeAccountIndex()
	for i, acc := range store.Accounts() {
		if !acc.IsActive() || strings.TrimSpace(acc.Password) == "" || strings.TrimSpace(acc.Identifier()) == "" {
			continue
		}
		if hasAccountIndex && i != accountIndex {
			continue
		}
		if filter != "" && !liveProbeAccountMatches(acc, filter) {
			continue
		}
		return store, acc, true, "config"
	}
	if hasAccountIndex {
		t.Logf("no password-backed config account matched DS2API_PROBE_ACCOUNT_INDEX=%d", accountIndex)
	}
	if filter != "" {
		t.Logf("no password-backed config account matched DS2API_PROBE_ACCOUNT=%q", filter)
	}
	return store, config.Account{}, false, "config"
}

func liveProbeAccountIndex() (int, bool) {
	raw := strings.TrimSpace(os.Getenv("DS2API_PROBE_ACCOUNT_INDEX"))
	if raw == "" {
		return 0, false
	}
	index, err := strconv.Atoi(raw)
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}

func liveProbeAccountMatches(acc config.Account, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	values := []string{acc.Identifier(), acc.Email, acc.Mobile, acc.Name, acc.Remark, acc.ProxyID}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == filter {
			return true
		}
	}
	return false
}

func liveProbeSelectedProfiles() []liveProbeProfile {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DS2API_PROBE_PROFILE"))) {
	case "current", "android", string(liveProbeCurrentAndroid):
		return []liveProbeProfile{liveProbeCurrentAndroid}
	case "web", "candidate", string(liveProbeCandidateWeb):
		return []liveProbeProfile{liveProbeCandidateWeb}
	default:
		return []liveProbeProfile{liveProbeCurrentAndroid, liveProbeCandidateWeb}
	}
}

func liveProbeRequiresSelectedProfile() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DS2API_PROBE_PROFILE"))) {
	case "", "all":
		return false
	default:
		return true
	}
}

func liveProbeProfilesString(profiles []liveProbeProfile) string {
	parts := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		parts = append(parts, string(profile))
	}
	return strings.Join(parts, ",")
}

func liveProbeTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("DS2API_PROBE_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 150
	}
	return time.Duration(seconds) * time.Second
}

func liveProbeIterations() int {
	value := strings.TrimSpace(os.Getenv("DS2API_PROBE_ITERATIONS"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DS2API_BAN_RISK_ITERATIONS"))
	}
	iterations, err := strconv.Atoi(value)
	if err != nil || iterations <= 0 {
		return 3
	}
	if iterations > 50 {
		return 50
	}
	return iterations
}

func liveProbeSleep() time.Duration {
	value := strings.TrimSpace(os.Getenv("DS2API_PROBE_SLEEP_MS"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DS2API_BAN_RISK_SLEEP_MS"))
	}
	ms, err := strconv.Atoi(value)
	if err != nil || ms < 0 {
		ms = 2000
	}
	return time.Duration(ms) * time.Millisecond
}

func liveProbeRepoRootConfigPath() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		goMod := filepath.Join(dir, "go.mod")
		cfgPath := filepath.Join(dir, "config.json")
		if _, err := os.Stat(goMod); err == nil {
			if _, cfgErr := os.Stat(cfgPath); cfgErr == nil {
				return cfgPath, true
			}
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func liveProbeCompletionPayload(sessionID string) map[string]any {
	return promptcompat.StandardRequest{
		RequestedModel: "deepseek-chat",
		ResolvedModel:  "deepseek-chat",
		FinalPrompt:    "Reply exactly OK.",
	}.CompletionPayload(sessionID)
}

func (c *Client) liveProbeWebLogin(ctx context.Context, acc config.Account) (string, liveProbeHTTPResult, error) {
	clients := c.requestClientsForAccount(acc)
	payload, err := liveProbeWebLoginPayload(acc)
	if err != nil {
		return "", liveProbeHTTPResult{}, err
	}
	headers := liveProbeWebBaseHeaders()

	firstResp, firstStatus, firstErr := c.postJSONWithStatus(ctx, clients.fallback, clients.fallback, dsprotocol.DeepSeekLoginURL, headers, payload)
	firstResult := liveProbeResult("std_http", firstStatus, firstResp)
	if token, ok := liveProbeLoginToken(firstResp); firstErr == nil && ok {
		return token, firstResult, nil
	}

	secondResp, secondStatus, secondErr := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekLoginURL, headers, payload)
	secondResult := liveProbeResult("utls", secondStatus, secondResp)
	if token, ok := liveProbeLoginToken(secondResp); secondErr == nil && ok {
		return token, secondResult, nil
	}
	return "", secondResult, fmt.Errorf("std_http=%s; utls=%s", liveProbeFailure(firstResult, firstErr), liveProbeFailure(secondResult, secondErr))
}

func (c *Client) liveProbeWebCreateSession(ctx context.Context, acc config.Account, token string) (string, liveProbeHTTPResult, error) {
	clients := c.requestClientsForAccount(acc)
	resp, status, err := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekCreateSessionURL, liveProbeWebAuthHeaders(token), map[string]any{})
	result := liveProbeResult("utls", status, resp)
	if err != nil {
		return "", result, err
	}
	if status == http.StatusOK && result.Code == 0 && result.BizCode == 0 {
		if sessionID := extractCreateSessionID(resp); strings.TrimSpace(sessionID) != "" {
			return sessionID, result, nil
		}
	}
	return "", result, fmt.Errorf("status=%d code=%d biz_code=%d msg=%q biz_msg=%q body=%s", status, result.Code, result.BizCode, result.Msg, result.BizMsg, result.Preview)
}

func (c *Client) liveProbeWebGetPow(ctx context.Context, acc config.Account, token string) (string, liveProbeHTTPResult, error) {
	clients := c.requestClientsForAccount(acc)
	resp, status, err := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekCreatePowURL, liveProbeWebAuthHeaders(token), map[string]any{"target_path": dsprotocol.DeepSeekCompletionTargetPath})
	result := liveProbeResult("utls", status, resp)
	if err != nil {
		return "", result, err
	}
	if status != http.StatusOK || result.Code != 0 || result.BizCode != 0 {
		return "", result, fmt.Errorf("status=%d code=%d biz_code=%d msg=%q biz_msg=%q body=%s", status, result.Code, result.BizCode, result.Msg, result.BizMsg, result.Preview)
	}
	data, _ := resp["data"].(map[string]any)
	bizData, _ := data["biz_data"].(map[string]any)
	challenge, _ := bizData["challenge"].(map[string]any)
	answer, err := ComputePow(ctx, challenge)
	if err != nil {
		return "", result, err
	}
	powResp, err := BuildPowHeader(challenge, answer)
	if err != nil {
		return "", result, err
	}
	if strings.TrimSpace(powResp) == "" {
		return "", result, fmt.Errorf("empty pow response")
	}
	return powResp, result, nil
}

func (c *Client) liveProbeWebCompletion(ctx context.Context, acc config.Account, token string, powResp string, payload map[string]any) (liveProbeCompletionBodyResult, error) {
	clients := c.requestClientsForAccount(acc)
	headers := liveProbeWebAuthHeaders(token)
	headers["x-ds-pow-response"] = powResp
	ctx = auth.WithAuth(ctx, &auth.RequestAuth{AccountID: acc.Identifier(), Account: acc, DeepSeekToken: token})
	resp, err := c.streamPost(ctx, clients.stream, dsprotocol.DeepSeekCompletionURL, headers, payload)
	if err != nil {
		return liveProbeCompletionBodyResult{}, err
	}
	if resp == nil {
		return liveProbeCompletionBodyResult{}, fmt.Errorf("nil response")
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	result := liveProbeClassifyCompletionBody(resp.StatusCode, body)
	if readErr != nil {
		return result, readErr
	}
	return result, nil
}

func liveProbeClassifyCompletionBody(status int, body []byte) liveProbeCompletionBodyResult {
	result := liveProbeCompletionBodyResult{
		Status:  status,
		Bytes:   len(body),
		Kind:    "empty",
		Preview: preview(body),
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return result
	}
	if strings.HasPrefix(trimmed, "{") {
		parsed := map[string]any{}
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			statusResult := liveProbeResult("", status, parsed)
			result.Kind = "json"
			if statusResult.Code != 0 || statusResult.BizCode != 0 || statusResult.Msg != "" || statusResult.BizMsg != "" {
				result.Kind = "json_error"
				result.ErrorMessage = fmt.Sprintf("code=%d biz_code=%d msg=%q biz_msg=%q", statusResult.Code, statusResult.BizCode, statusResult.Msg, statusResult.BizMsg)
			}
			return result
		}
	}

	currentType := "text"
	for _, line := range strings.Split(trimmed, "\n") {
		lineResult := sse.ParseDeepSeekContentLine([]byte(line), true, currentType)
		if lineResult.NextType != "" {
			currentType = lineResult.NextType
		}
		if !lineResult.Parsed {
			continue
		}
		result.ParsedLines++
		if lineResult.ErrorMessage != "" {
			result.Kind = "sse_error"
			result.ErrorMessage = lineResult.ErrorMessage
			return result
		}
		if lineResult.ContentFilter {
			result.Kind = "content_filter"
			result.ErrorMessage = "content_filter"
			return result
		}
		for _, part := range lineResult.Parts {
			result.ContentBytes += len(strings.TrimSpace(part.Text))
		}
		if lineResult.Stop {
			result.Done = true
		}
	}
	if result.ParsedLines > 0 {
		if result.ContentBytes > 0 {
			result.Kind = "sse_content"
			return result
		}
		if result.Done {
			result.Kind = "sse_done_without_content"
			return result
		}
		result.Kind = "sse_without_content"
		return result
	}
	if liveProbeContainsAny(strings.ToLower(trimmed), "muted", "banned", "forbidden", "too many requests", "risk", "captcha", "verify") {
		result.Kind = "text_error"
		result.ErrorMessage = trimmed
		return result
	}
	result.Kind = "unknown"
	return result
}

func (r liveProbeCompletionBodyResult) validate() error {
	if r.Bytes == 0 {
		return fmt.Errorf("empty first body window")
	}
	if r.ErrorMessage != "" {
		return fmt.Errorf("kind=%s error=%s preview=%q", r.Kind, r.ErrorMessage, r.Preview)
	}
	if r.Kind != "sse_content" {
		return fmt.Errorf("kind=%s parsed=%d content_bytes=%d done=%t preview=%q", r.Kind, r.ParsedLines, r.ContentBytes, r.Done, r.Preview)
	}
	return nil
}

func liveProbeWebLoginPayload(acc config.Account) (map[string]any, error) {
	deviceID, err := liveProbeNewWebDeviceID()
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"password":  strings.TrimSpace(acc.Password),
		"device_id": deviceID,
		"os":        "web",
	}
	if email := strings.TrimSpace(acc.Email); email != "" {
		payload["email"] = email
		return payload, nil
	}
	if mobile := strings.TrimSpace(acc.Mobile); mobile != "" {
		loginMobile, areaCode := liveProbeNormalizeMobileForWeb(mobile)
		if loginMobile == "" {
			return nil, fmt.Errorf("invalid mobile account")
		}
		payload["mobile"] = loginMobile
		payload["area_code"] = areaCode
		return payload, nil
	}
	return nil, fmt.Errorf("missing email/mobile")
}

func liveProbeNewWebDeviceID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func liveProbeNormalizeMobileForWeb(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "+86"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if strings.HasPrefix(digits, "86") && len(digits) > 11 {
		return digits[2:], "+86"
	}
	return digits, "+86"
}

func liveProbeWebAuthHeaders(token string) map[string]string {
	headers := liveProbeWebBaseHeaders()
	headers["authorization"] = "Bearer " + token
	return headers
}

func liveProbeWebBaseHeaders() map[string]string {
	_, offset := time.Now().Zone()
	headers := map[string]string{
		"Host":                     "chat.deepseek.com",
		"User-Agent":               "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept":                   "application/json",
		"Content-Type":             "application/json",
		"accept-charset":           "UTF-8",
		"x-app-version":            "2.0.0",
		"x-client-locale":          "en_US",
		"x-client-platform":        "web",
		"x-client-version":         "2.0.0",
		"x-client-timezone-offset": strconv.Itoa(offset),
		"Referer":                  "https://chat.deepseek.com/",
		"Origin":                   "https://chat.deepseek.com",
	}
	if dsprotocol.RangersID != "" {
		headers["x-rangers-id"] = dsprotocol.RangersID
	}
	return headers
}

func liveProbeLoginToken(resp map[string]any) (string, bool) {
	result := liveProbeResult("", 0, resp)
	if result.Code != 0 || result.BizCode != 0 {
		return "", false
	}
	data, _ := resp["data"].(map[string]any)
	bizData, _ := data["biz_data"].(map[string]any)
	user, _ := bizData["user"].(map[string]any)
	token, _ := user["token"].(string)
	token = strings.TrimSpace(token)
	return token, token != ""
}

func liveProbeResult(transport string, status int, resp map[string]any) liveProbeHTTPResult {
	code, bizCode, msg, bizMsg := extractResponseStatus(resp)
	return liveProbeHTTPResult{
		Transport: transport,
		Status:    status,
		Code:      code,
		BizCode:   bizCode,
		Msg:       msg,
		BizMsg:    bizMsg,
		Preview:   previewJSONMap(resp),
	}
}

func liveProbeFailure(result liveProbeHTTPResult, err error) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("status=%d code=%d biz_code=%d msg=%q biz_msg=%q body=%s", result.Status, result.Code, result.BizCode, result.Msg, result.BizMsg, result.Preview)
}

func liveProbeFailureStage(err error) string {
	if err == nil {
		return "ok"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "login failed") || strings.Contains(msg, "login returned"):
		return "login"
	case strings.Contains(msg, "create_session"):
		return "session"
	case strings.Contains(msg, "get_pow") || strings.Contains(msg, "pow failed") || strings.Contains(msg, "empty pow"):
		return "pow"
	case strings.Contains(msg, "completion"):
		return "completion"
	default:
		return "unknown"
	}
}

func liveProbeFailureCategory(err error) (string, bool) {
	if err == nil {
		return "ok", false
	}
	raw := err.Error()
	msg := strings.ToLower(raw)
	switch {
	case strings.Contains(msg, "password_or_user_name_is_wrong"):
		return "bad_credentials", false
	case liveProbeContainsAny(msg, "status=429", "too_many_requests", "rate limit", "too many requests"):
		return "rate_limit", true
	case liveProbeContainsAny(msg, "status=403", "forbidden"):
		return "forbidden", true
	case liveProbeContainsAny(raw, "封", "禁", "风控") || liveProbeContainsAny(msg, "banned", " ban", "muted", "blocked"):
		return "ban_or_mute", true
	case liveProbeContainsAny(msg, "captcha", "verify", "verification", "risk", "abuse", "suspicious", "security challenge"):
		return "risk_challenge", true
	case liveProbeContainsAny(msg, "empty first body window", "empty response"):
		return "empty_body", false
	case liveProbeContainsAny(msg, "context deadline exceeded", "connection reset", "connection refused", "no such host", "timeout"):
		return "network", false
	default:
		return "other", false
	}
}

func liveProbeContainsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func liveProbeFormatCategories(counts map[string]int) string {
	if len(counts) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, counts[key]))
	}
	return strings.Join(parts, ",")
}

func liveProbeMaskIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "<unknown>"
	}
	if at := strings.Index(identifier, "@"); at > 0 {
		domain := identifier[at:]
		return identifier[:1] + "***" + domain
	}
	if len(identifier) <= 4 {
		return "***"
	}
	return "***" + identifier[len(identifier)-4:]
}
