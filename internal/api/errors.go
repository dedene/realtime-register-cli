package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// APIError represents a generic API error response.
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}

// AuthError represents a 401/403 authentication error.
type AuthError struct {
	APIError
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

// RateLimitError represents a 429 rate limit error.
type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: retry after %ds", int(e.RetryAfter.Seconds()))
}

// NotFoundError represents a 404 resource not found error.
type NotFoundError struct {
	APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Message)
}

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s %s", e.Field, e.Message)
}

// NewAPIError parses a JSON error response body and returns the appropriate typed error.
func NewAPIError(statusCode int, body []byte) error {
	msg, details := parseAPIErrorBody(body)

	base := APIError{
		StatusCode: statusCode,
		Message:    msg,
		Details:    details,
	}

	switch statusCode {
	case 401, 403:
		return &AuthError{APIError: base}
	case 404:
		return &NotFoundError{APIError: base}
	case 429:
		return &RateLimitError{APIError: base, RetryAfter: parseRetryAfter(body)}
	default:
		return &base
	}
}

func parseAPIErrorBody(body []byte) (message, details string) {
	rawBody := strings.TrimSpace(string(body))
	if rawBody == "" {
		return "unknown error", ""
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return "unknown error", rawBody
	}

	msg := firstNonEmpty(
		nestedString(raw, "error", "message"),
		stringValue(raw["message"]),
		stringValue(raw["error_description"]),
	)
	if msg == "" {
		msg = "unknown error"
	}

	var detailLines []string
	if nestedError, ok := raw["error"].(map[string]any); ok {
		detailLines = append(detailLines, formattedDetails(nestedError["details"])...)
	}
	detailLines = append(detailLines, formattedDetails(raw["detail"])...)
	detailLines = append(detailLines, formattedDetails(raw["details"])...)
	detailLines = append(detailLines, formatRecordErrors(raw["errors"])...)
	detailLines = append(detailLines, formatRecordConflicts(raw["conflicts"])...)

	joined := strings.Join(uniqueNonEmpty(detailLines), "\n")
	if joined == "" && msg == "unknown error" {
		joined = rawBody
	}

	return msg, joined
}

func nestedString(raw map[string]any, outer, inner string) string {
	value, ok := raw[outer].(map[string]any)
	if !ok {
		return ""
	}
	return stringValue(value[inner])
}

func formattedDetails(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return []string{compactJSON(v)}
	}
}

func formatRecordErrors(value any) []string {
	records, ok := value.(map[string]any)
	if !ok {
		return formattedDetails(value)
	}

	keys := sortedKeys(records)
	lines := make([]string, 0, len(keys))
	for _, recordIndex := range keys {
		recordValue, ok := records[recordIndex]
		if !ok {
			continue
		}

		fields, ok := recordValue.(map[string]any)
		if !ok {
			lines = append(lines, fmt.Sprintf("record %s: %s", recordIndex, stringValue(recordValue)))
			continue
		}

		for _, field := range sortedKeys(fields) {
			lines = append(lines, fmt.Sprintf("record %s.%s: %s", recordIndex, field, stringValue(fields[field])))
		}
	}
	return lines
}

func formatRecordConflicts(value any) []string {
	conflicts, ok := value.(map[string]any)
	if !ok {
		return formattedDetails(value)
	}

	keys := sortedKeys(conflicts)
	lines := make([]string, 0, len(keys))
	for _, recordIndex := range keys {
		indices, ok := conflicts[recordIndex].([]any)
		if !ok {
			lines = append(lines, fmt.Sprintf("record %s conflicts: %s", recordIndex, stringValue(conflicts[recordIndex])))
			continue
		}

		parts := make([]string, 0, len(indices))
		for _, index := range indices {
			parts = append(parts, stringValue(index))
		}
		lines = append(lines, fmt.Sprintf("record %s conflicts with records %s", recordIndex, strings.Join(parts, ", ")))
	}
	return lines
}

func compactJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func stringValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			part := stringValue(item)
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) == len(v) && len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
		return compactJSON(v)
	default:
		return compactJSON(v)
	}
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])
		switch {
		case leftErr == nil && rightErr == nil:
			return left < right
		case leftErr == nil:
			return true
		case rightErr == nil:
			return false
		default:
			return keys[i] < keys[j]
		}
	})
	return keys
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// parseRetryAfter attempts to extract retry-after seconds from the response body.
func parseRetryAfter(body []byte) time.Duration {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0
	}

	if v, ok := raw["retry_after"]; ok {
		if val, ok := v.(float64); ok {
			return time.Duration(val) * time.Second
		}
	}
	return 0
}
