//go:build android

package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func platformConfigDir() (string, error) {
	var base string
	for i := 0; i < 50; i++ {
		base = application.Mobile.StoragePath()
		if base != "" {
			return filepath.Join(base, "IceTray"), nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return "", fmt.Errorf("android storage path unavailable")
}
