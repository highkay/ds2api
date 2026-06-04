package riskguard

import (
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"ds2api/internal/config"
)

type Violation struct {
	Status  int
	Code    string
	Message string
}

func (v *Violation) Error() string {
	if v == nil {
		return ""
	}
	if v.Message != "" {
		return v.Message
	}
	return v.Code
}

func CheckCompletion(reader any, prompt string, refFileIDs []string) error {
	maxPromptChars := config.RuntimeMaxPromptCharsFrom(reader)
	if maxPromptChars > 0 {
		chars := utf8.RuneCountInString(prompt)
		if chars > maxPromptChars {
			return &Violation{
				Status:  http.StatusRequestEntityTooLarge,
				Code:    "prompt_too_large",
				Message: fmt.Sprintf("prompt has %d characters, limit is %d", chars, maxPromptChars),
			}
		}
	}
	maxRefFiles := config.RuntimeMaxRefFilesPerRequestFrom(reader)
	if maxRefFiles > 0 && len(refFileIDs) > maxRefFiles {
		return &Violation{
			Status:  http.StatusRequestEntityTooLarge,
			Code:    "too_many_ref_files",
			Message: fmt.Sprintf("request references %d files, limit is %d", len(refFileIDs), maxRefFiles),
		}
	}
	return nil
}

func ErrorDetail(err error) (status int, code string, message string, ok bool) {
	var violation *Violation
	if !errors.As(err, &violation) || violation == nil {
		return 0, "", "", false
	}
	status = violation.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	return status, violation.Code, violation.Error(), true
}
