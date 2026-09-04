//go:build !android && !headless

package main

func bindAndroidApp(_ *App) {}

func pushAndroidSession(_ *App, _ PlaybackState, _ any) {}
