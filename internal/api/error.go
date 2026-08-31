package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Error struct {
	StatusCode int
	Method     string
	URL        string
	Message    string
	TraceID    string
	Body       string
}

func (e *Error) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.Body)
	}
	if message == "" {
		message = "unbekannter API-Fehler"
	}
	if e.TraceID != "" {
		return fmt.Sprintf("Lexware API: HTTP %d: %s (Trace-ID: %s)", e.StatusCode, message, e.TraceID)
	}
	return fmt.Sprintf("Lexware API: HTTP %d: %s", e.StatusCode, message)
}

func parseError(status int, method, requestURL string, body []byte) error {
	apiErr := &Error{
		StatusCode: status,
		Method:     method,
		URL:        requestURL,
		Body:       strings.TrimSpace(string(body)),
	}

	var regular struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		TraceID string `json:"traceId"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
		IssueList []struct {
			Key    string `json:"i18nKey"`
			Source string `json:"source"`
			Type   string `json:"type"`
		} `json:"IssueList"`
	}
	if json.Unmarshal(body, &regular) == nil {
		apiErr.TraceID = regular.TraceID
		apiErr.Message = regular.Message
		if apiErr.Message == "" {
			apiErr.Message = regular.Error
		}
		if len(regular.Details) > 0 {
			parts := make([]string, 0, len(regular.Details))
			for _, detail := range regular.Details {
				if detail.Field != "" {
					parts = append(parts, detail.Field+": "+detail.Message)
				} else if detail.Message != "" {
					parts = append(parts, detail.Message)
				}
			}
			if len(parts) > 0 {
				apiErr.Message = strings.Join(parts, "; ")
			}
		}
		if len(regular.IssueList) > 0 {
			parts := make([]string, 0, len(regular.IssueList))
			for _, issue := range regular.IssueList {
				part := issue.Key
				if issue.Source != "" {
					part += " (" + issue.Source + ")"
				}
				parts = append(parts, part)
			}
			apiErr.Message = strings.Join(parts, "; ")
		}
	}
	return apiErr
}
