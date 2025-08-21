package cli

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether stdout is a TTY.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
