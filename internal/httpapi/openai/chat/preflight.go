package chat

import (
	"net/http"
	"unicode/utf8"

	"ds2api/internal/httpapi/openai/files"
	"ds2api/internal/promptcompat"
	"ds2api/internal/riskguard"
)

func (h *Handler) preflightParsedChatRequest(w http.ResponseWriter, r *http.Request, req map[string]any) bool {
	if files.HasInlineUploadPayload(req) {
		return true
	}
	stdReq, err := promptcompat.NormalizeOpenAIChatRequest(h.Store, req, requestTraceID(r))
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return h.writeCompletionRiskRejection(w, r, stdReq)
}

func (h *Handler) writeCompletionRiskRejection(w http.ResponseWriter, r *http.Request, stdReq promptcompat.StandardRequest) bool {
	err := riskguard.CheckCompletion(h.Store, stdReq.FinalPrompt, stdReq.RefFileIDs)
	if err == nil {
		return true
	}
	status, code, message, _ := riskguard.ErrorDetail(err)
	if status == http.StatusRequestEntityTooLarge {
		logOpenAILocalRequestRejection(
			r,
			status,
			code,
			message,
			"prompt_chars", utf8.RuneCountInString(stdReq.FinalPrompt),
			"ref_file_count", len(stdReq.RefFileIDs),
		)
	}
	writeOpenAIError(w, status, message)
	return false
}
