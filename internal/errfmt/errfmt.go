package errfmt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/auth"
)

// Format returns a user-friendly error message with hints.
func Format(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	var authErr *api.AuthError
	var notFoundErr *api.NotFoundError
	var rateLimitErr *api.RateLimitError
	var apiErr *api.APIError

	switch {
	case errors.Is(err, auth.ErrNoAPIKey):
		sb.WriteString("Error: Not authenticated.\n\n")
		sb.WriteString("To authenticate:\n")
		sb.WriteString("  rr auth login\n\n")
		sb.WriteString("Or set environment variable:\n")
		sb.WriteString("  export RR_API_KEY=your-api-key\n")
		return sb.String()

	case errors.As(err, &authErr):
		sb.WriteString("Error: Authentication failed.\n\n")
		fmt.Fprintf(&sb, "Details: %s\n\n", authErr.Message)
		sb.WriteString("Check your API key:\n")
		sb.WriteString("  rr auth status\n")
		return sb.String()

	case errors.As(err, &notFoundErr):
		fmt.Fprintf(&sb, "Error: %s\n", notFoundErr.Message)
		return sb.String()

	case errors.As(err, &rateLimitErr):
		sb.WriteString("Error: Rate limited by API.\n\n")
		fmt.Fprintf(&sb, "Retry after: %d seconds\n", int(rateLimitErr.RetryAfter.Seconds()))
		return sb.String()

	case errors.As(err, &apiErr):
		fmt.Fprintf(&sb, "Error: API error (%d)\n\n", apiErr.StatusCode)
		fmt.Fprintf(&sb, "Message: %s\n", apiErr.Message)
		if apiErr.Details != "" {
			fmt.Fprintf(&sb, "Details: %s\n", apiErr.Details)
		}
		return sb.String()

	default:
		return fmt.Sprintf("Error: %s\n", err.Error())
	}
}
