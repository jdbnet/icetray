//go:build windows && !headless

package ui

import (
	"github.com/ncruces/zenity"

	"git.jdbnet.co.uk/jamie/icetray/logger"
)

func showAddStreamDialog(initial AddStreamInput) (name, url string, ok bool) {
	if initial.Error != "" {
		zenity.Error(initial.Error, zenity.Title("Add Stream"))
	}

	name, err := zenity.Entry("Enter stream name (e.g., Lofi Radio):",
		zenity.Title("Add Stream"),
		zenity.EntryText(initial.Name),
	)
	if err != nil || name == "" {
		if err != zenity.ErrCanceled {
			logger.LogError("Failed to get stream name", err)
		}
		return "", "", false
	}

	url, err = zenity.Entry("Enter stream URL:",
		zenity.Title("Add Stream"),
		zenity.EntryText(initial.URL),
	)
	if err != nil || url == "" {
		if err != zenity.ErrCanceled {
			logger.LogError("Failed to get stream URL", err)
		}
		return "", "", false
	}

	return name, url, true
}
