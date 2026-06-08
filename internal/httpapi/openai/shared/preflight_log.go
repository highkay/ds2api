package shared

import (
	"net/http"
	"strings"

	"ds2api/internal/config"
)

func LogLocalRequestRejection(r *http.Request, status int, code, message string, attrs ...any) {
	args := []any{
		"trace_id", RequestTraceID(r),
		"status", status,
		"code", strings.TrimSpace(code),
		"message", strings.TrimSpace(message),
	}
	args = append(args, attrs...)
	config.Logger.Warn("[openai_preflight] rejected request", args...)
}
