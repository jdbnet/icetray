package main

import (
	_ "embed"
	"strings"
)

// Set at link time with: -X main.version=1.2.8
var version string

//go:embed VERSION
var versionFile string

func appVersion() string {
	if v := strings.TrimSpace(version); v != "" {
		return v
	}
	if v := strings.TrimSpace(versionFile); v != "" {
		return v
	}
	return "dev"
}
