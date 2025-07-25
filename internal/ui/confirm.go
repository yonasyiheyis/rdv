package ui

import (
	"github.com/charmbracelet/huh"
)

func Confirm(title string) (bool, error) {
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(&ok),
		),
	)
	return ok, f.Run()
}
