package shared

import "net/http"

const UpstreamEmptyOutputCode = "upstream_empty_output"

func ShouldWriteUpstreamEmptyOutputError(text string) bool {
	return text == ""
}

func UpstreamEmptyOutputDetail(contentFilter bool, text, thinking string) (int, string, string) {
	_ = text
	if contentFilter {
		return http.StatusBadRequest, "Upstream content filtered the response and returned no output.", "content_filter"
	}
	if thinking != "" {
		return http.StatusTooManyRequests, "Upstream account hit a rate limit and returned reasoning without visible output.", UpstreamEmptyOutputCode
	}
	return http.StatusTooManyRequests, "Upstream account hit a rate limit and returned empty output.", UpstreamEmptyOutputCode
}

func ShouldPenalizeUpstreamEmptyOutput(status int, code string) bool {
	return status == http.StatusTooManyRequests && code == UpstreamEmptyOutputCode
}

func WriteUpstreamEmptyOutputError(w http.ResponseWriter, text, thinking string, contentFilter bool) bool {
	if !ShouldWriteUpstreamEmptyOutputError(text) {
		return false
	}
	status, message, code := UpstreamEmptyOutputDetail(contentFilter, text, thinking)
	WriteOpenAIErrorWithCode(w, status, message, code)
	return true
}
