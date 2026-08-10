//go:build !linux || headless

package main

func runAddStreamDialogIfRequested(resultPath, name, url, errMsg string) bool {
	return false
}
