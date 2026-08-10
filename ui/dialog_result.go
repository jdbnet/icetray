package ui

import (
	"encoding/json"
	"os"
)

// DialogResult is written by the dialog subprocess for the tray process to read.
type DialogResult struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	OK   bool   `json:"ok"`
}

func writeDialogResult(path string, name, url string, ok bool) error {
	data, err := json.Marshal(DialogResult{Name: name, URL: url, OK: ok})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func readDialogResult(path string) (name, url string, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}

	var result DialogResult
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", false, err
	}

	return result.Name, result.URL, result.OK, nil
}
