package xapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type APIError struct {
	Operation  string
	Status     string
	StatusCode int
	Body       string
	RateLimit  *RateLimitInfo
}

type RateLimitInfo struct {
	Limit             int
	Remaining         int
	ResetUnix         int64
	ResetAt           string
	RetryAfterSeconds int
	limitSet          bool
	remainingSet      bool
	resetSet          bool
	retryAfterSet     bool
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("%s returned %s", e.Operation, e.Status)
	if strings.TrimSpace(e.Body) != "" {
		message += ": " + e.Body
	}
	if e.RateLimit != nil && e.RateLimit.ResetAt != "" {
		message += fmt.Sprintf(" (rate limit resets at %s)", e.RateLimit.ResetAt)
	}
	return message
}

func apiError(operation string, resp *http.Response) error {
	return &APIError{
		Operation:  operation,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Body:       readErrorBody(resp.Body),
		RateLimit:  parseRateLimit(resp.Header, time.Now()),
	}
}

func parseRateLimit(header http.Header, now time.Time) *RateLimitInfo {
	info := &RateLimitInfo{}
	seen := false

	if value, ok := parseHeaderInt(header.Get("x-rate-limit-limit")); ok {
		info.Limit = value
		info.limitSet = true
		seen = true
	}
	if value, ok := parseHeaderInt(header.Get("x-rate-limit-remaining")); ok {
		info.Remaining = value
		info.remainingSet = true
		seen = true
	}
	if value, ok := parseHeaderInt64(header.Get("x-rate-limit-reset")); ok {
		info.ResetUnix = value
		info.resetSet = true
		resetAt := time.Unix(value, 0).UTC()
		info.ResetAt = resetAt.Format(time.RFC3339)
		if seconds := int(resetAt.Sub(now).Seconds()); seconds > 0 {
			info.RetryAfterSeconds = seconds
			info.retryAfterSet = true
		}
		seen = true
	}
	if value, ok := parseHeaderInt(header.Get("retry-after")); ok {
		info.RetryAfterSeconds = value
		info.retryAfterSet = true
		seen = true
	}

	if !seen {
		return nil
	}
	if info.RetryAfterSeconds == 0 && !now.IsZero() && info.ResetUnix > 0 {
		if seconds := int(time.Unix(info.ResetUnix, 0).Sub(now).Seconds()); seconds > 0 {
			info.RetryAfterSeconds = seconds
			info.retryAfterSet = true
		}
	}
	return info
}

func (info RateLimitInfo) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{}
	if info.limitSet {
		data["limit"] = info.Limit
	}
	if info.remainingSet {
		data["remaining"] = info.Remaining
	}
	if info.resetSet {
		data["reset_unix"] = info.ResetUnix
		if info.ResetAt != "" {
			data["reset_at"] = info.ResetAt
		}
	}
	if info.retryAfterSet {
		data["retry_after_seconds"] = info.RetryAfterSeconds
	}
	return json.Marshal(data)
}

func parseHeaderInt(value string) (int, bool) {
	parsed, ok := parseHeaderInt64(value)
	return int(parsed), ok
}

func parseHeaderInt64(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
