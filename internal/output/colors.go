package output

import (
	"os"

	"github.com/muesli/termenv"
)

// Colors provides terminal color styling
type Colors struct {
	profile  termenv.Profile
	disabled bool
}

// NewColors creates a Colors instance
func NewColors(noColor bool) *Colors {
	if noColor {
		return &Colors{disabled: true}
	}

	// Check NO_COLOR env var
	if os.Getenv("NO_COLOR") != "" {
		return &Colors{disabled: true}
	}

	return &Colors{
		profile: termenv.ColorProfile(),
	}
}

// Enabled returns true if colors are enabled
func (c *Colors) Enabled() bool {
	return !c.disabled && c.profile != termenv.Ascii
}

// Bold returns text in bold
func (c *Colors) Bold(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Bold().String()
}

// Green returns text in green
func (c *Colors) Green(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Foreground(c.profile.Color("2")).String()
}

// Yellow returns text in yellow
func (c *Colors) Yellow(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Foreground(c.profile.Color("3")).String()
}

// Red returns text in red
func (c *Colors) Red(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Foreground(c.profile.Color("1")).String()
}

// Cyan returns text in cyan
func (c *Colors) Cyan(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Foreground(c.profile.Color("6")).String()
}

// Faint returns text in faint/dim style
func (c *Colors) Faint(s string) string {
	if !c.Enabled() {
		return s
	}
	return termenv.String(s).Faint().String()
}

// StatusColor returns the appropriate color for a status string
func (c *Colors) StatusColor(status string) string {
	switch status {
	case "active", "ok", "completed", "available":
		return c.Green(status)
	case "pending", "processing", "expiring":
		return c.Yellow(status)
	case "expired", "failed", "error", "taken":
		return c.Red(status)
	default:
		return status
	}
}
