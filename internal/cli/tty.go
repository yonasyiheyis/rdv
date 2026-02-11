package cli

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// IsInteractive reports whether the current session can safely run interactive
// TUI prompts. It is intentionally conservative to avoid duplicated renders
// in limited terminals.
func IsInteractive() bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}

	termEnv := strings.TrimSpace(os.Getenv("TERM"))
	if termEnv == "" || strings.EqualFold(termEnv, "dumb") {
		return false
	}

	return true
}
