package ui

import (
	"os"

	"github.com/charmbracelet/huh"
)

// NewForm returns a form configured for reliable rendering across terminals.
// Accessible mode avoids advanced redraw behavior that can duplicate fields.
// Set RDV_TUI=1 to enable full TUI rendering.
func NewForm(groups ...*huh.Group) *huh.Form {
	accessible := true
	if os.Getenv("RDV_TUI") == "1" {
		accessible = false
	}
	return huh.NewForm(groups...).WithAccessible(accessible)
}
