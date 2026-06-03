package shared

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteOpenAIAccountPoolBusyError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteOpenAIAccountPoolBusyError(rec)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["type"] != "rate_limit_error" {
		t.Fatalf("error.type=%v want=rate_limit_error", errObj["type"])
	}
	if errObj["code"] != AccountPoolBusyErrorCode {
		t.Fatalf("error.code=%v want=%s", errObj["code"], AccountPoolBusyErrorCode)
	}
	if errObj["message"] != AccountPoolBusyErrorMessage {
		t.Fatalf("error.message=%v want=%s", errObj["message"], AccountPoolBusyErrorMessage)
	}
}
