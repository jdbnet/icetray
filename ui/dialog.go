//go:build !headless

package ui

// AddStreamInput holds optional prefill values and an error message for the dialog.
type AddStreamInput struct {
	Name  string
	URL   string
	Error string
}

// ShowAddStreamDialog opens a dialog for adding a stream.
// Returns ok=false when the user cancels or closes the window.
func ShowAddStreamDialog(initial AddStreamInput) (name, url string, ok bool) {
	return showAddStreamDialog(initial)
}
