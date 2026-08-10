//go:build linux && !headless

package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"git.jdbnet.co.uk/jamie/icetray/assets"
)

func runAddStreamDialogMain(initial AddStreamInput) (name, url string, ok bool) {
	var result struct {
		name string
		url  string
		ok   bool
	}

	a := app.NewWithID("uk.co.jdbnet.icetray")
	a.Settings().SetTheme(newMintTheme())

	icon := fyne.NewStaticResource("icon.png", assets.Icon)
	a.SetIcon(icon)

	w := a.NewWindow("Add Stream")
	w.SetIcon(icon)
	w.Resize(fyne.NewSize(460, 300))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	title := widget.NewLabelWithStyle("Add a radio stream", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Save an Icecast or HTTP audio stream to your tray menu.")

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Lofi Radio")
	nameEntry.SetText(initial.Name)

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("https://example.com/stream.mp3")
	urlEntry.SetText(initial.URL)

	errorLabel := widget.NewLabel("")
	errorLabel.Importance = widget.DangerImportance
	if initial.Error != "" {
		errorLabel.SetText(initial.Error)
	} else {
		errorLabel.Hide()
	}

	setError := func(message string) {
		if message == "" {
			errorLabel.Hide()
			return
		}
		errorLabel.SetText(message)
		errorLabel.Show()
	}

	closeDialog := func() {
		w.Close()
		a.Quit()
	}

	submit := func() {
		streamName := strings.TrimSpace(nameEntry.Text)
		streamURL := strings.TrimSpace(urlEntry.Text)

		if streamName == "" {
			setError("Please enter a stream name.")
			nameEntry.FocusGained()
			return
		}
		if streamURL == "" {
			setError("Please enter a stream URL.")
			urlEntry.FocusGained()
			return
		}
		if !strings.HasPrefix(streamURL, "http://") && !strings.HasPrefix(streamURL, "https://") {
			setError("Stream URL must start with http:// or https://.")
			urlEntry.FocusGained()
			return
		}

		result.name = streamName
		result.url = streamURL
		result.ok = true
		closeDialog()
	}

	addButton := widget.NewButton("Add Stream", submit)
	addButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", closeDialog)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		cancelButton,
		addButton,
	)

	form := container.NewVBox(
		title,
		subtitle,
		widget.NewLabel("Name"),
		nameEntry,
		widget.NewLabel("URL"),
		urlEntry,
		errorLabel,
		buttons,
	)

	w.SetContent(container.NewPadded(form))
	w.SetCloseIntercept(closeDialog)

	nameEntry.OnSubmitted = func(_ string) {
		urlEntry.FocusGained()
	}
	urlEntry.OnSubmitted = func(_ string) {
		submit()
	}

	w.Show()
	a.Run()

	return result.name, result.url, result.ok
}
