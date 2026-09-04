//go:build !android

package main

import (
	"os"
	"path/filepath"
)

func platformConfigDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "IceTray"), nil
}
