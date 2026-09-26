//go:build !headless

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jdbnet/icetray/config"
	"github.com/jdbnet/icetray/exportbundle"
	"github.com/jdbnet/icetray/images"
	"github.com/jdbnet/icetray/logger"
	"github.com/jdbnet/icetray/metadata"
	"github.com/jdbnet/icetray/player"
	"github.com/jdbnet/icetray/startup"
	"github.com/jdbnet/icetray/stream"
)

// StreamView is a stream exposed to the frontend.
type StreamView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Image     string `json:"image,omitempty"`
	ImageData string `json:"imageData,omitempty"`
}

// PlaybackState describes current playback.
type PlaybackState struct {
	Playing  bool   `json:"playing"`
	Paused   bool   `json:"paused"`
	Loading  bool   `json:"loading"`
	StreamID string `json:"streamId"`
	Volume   int    `json:"volume"`
}

// SettingsView exposes app settings to the frontend.
type SettingsView struct {
	Autoplay        bool   `json:"autoplay"`
	LaunchOnLogin   bool   `json:"launchOnLogin"`
	LaunchMinimized bool   `json:"launchMinimized"`
	Volume           int `json:"volume"`
	CrossfadeSeconds int `json:"crossfadeSeconds"`
	Desktop          bool   `json:"desktop"`
	Version         string `json:"version"`
}

// App is the Wails application binding layer.
type App struct {
	wails  *application.App
	window application.Window

	cfg        *config.Config
	player     *player.Player
	supervisor *stream.Supervisor
	startupMgr startup.StartupManager

	playbackMu    sync.Mutex
	metaCancel    context.CancelFunc
	nowPlaying    metadata.NowPlaying
	currentID     string
	casting       bool
	castPaused    bool
	handoffPaused bool
}

// NewApp creates the Wails app bindings.
func NewApp(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) *App {
	a := &App{
		cfg:        cfg,
		player:     p,
		supervisor: sup,
		startupMgr: sm,
	}
	p.AddStateChangeListener(a.emitPlaybackState)
	a.applyCrossfadeFromConfig()
	return a
}

func (a *App) applyCrossfadeFromConfig() {
	player.SetCrossfadeDuration(time.Duration(a.cfg.GetCrossfadeSeconds()) * time.Second)
}

func (a *App) setWindow(window application.Window) {
	a.window = window
}

// ServiceStartup initialises Wails runtime access and optional autoplay.
func (a *App) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	a.wails = application.Get()
	bindAndroidApp(a)

	if a.cfg.GetAutoplay() {
		go func() {
			time.Sleep(500 * time.Millisecond)
			a.playbackMu.Lock()
			defer a.playbackMu.Unlock()
			if err := a.playLastStreamLocked(); err != nil {
				logger.LogError("Autoplay failed", err)
			}
			a.emitPlaybackState()
		}()
	} else if a.player.IsRunning() {
		a.currentID = a.cfg.GetLastStreamID()
		if a.currentID != "" {
			if s, ok := a.cfg.GetStreamByID(a.currentID); ok {
				a.startMetadataPoller(s.URL)
			}
		}
	}
	a.emitPlaybackState()
	return nil
}

// ServiceShutdown stops playback on application exit.
func (a *App) ServiceShutdown() error {
	a.Shutdown()
	return nil
}

func (a *App) emitPlaybackState() {
	if a.wails == nil {
		return
	}
	state := a.GetPlaybackState()
	a.wails.Event.Emit("playback:state", state)
	pushAndroidSession(a, state, a.nowPlaying)
}

func (a *App) emitNowPlaying() {
	if a.wails == nil {
		return
	}
	a.wails.Event.Emit("nowplaying:update", a.nowPlaying)
	pushAndroidSession(a, a.GetPlaybackState(), a.nowPlaying)
}

func (a *App) emitStreamsChanged() {
	if a.wails == nil {
		return
	}
	a.wails.Event.Emit("streams:changed", nil)
}

// GetStreams returns all saved streams with optional embedded image data.
func (a *App) GetStreams() []StreamView {
	streams := a.cfg.GetStreams()
	out := make([]StreamView, 0, len(streams))
	for _, s := range streams {
		view := StreamView{
			ID:    s.ID,
			Name:  s.Name,
			URL:   s.URL,
			Image: s.Image,
		}
		if s.Image != "" {
			path := images.ImagePath(a.cfg.ImagesDir(), s.Image)
			if data, err := os.ReadFile(path); err == nil {
				view.ImageData = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
			}
		}
		out = append(out, view)
	}
	return out
}

// AddStream creates a new stream.
func (a *App) AddStream(name, url string) (StreamView, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" {
		return StreamView{}, errInvalidInput("name and URL are required")
	}
	s, err := a.cfg.AddStream(name, url)
	if err != nil {
		return StreamView{}, err
	}
	a.emitStreamsChanged()
	return StreamView{ID: s.ID, Name: s.Name, URL: s.URL}, nil
}

// UpdateStream updates stream name and URL.
func (a *App) UpdateStream(id, name, url string) error {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" {
		return errInvalidInput("name and URL are required")
	}
	if err := a.cfg.UpdateStream(id, name, url); err != nil {
		return err
	}
	a.emitStreamsChanged()
	return nil
}

// ReorderStreams saves a new stream list order.
func (a *App) ReorderStreams(ids []string) error {
	if err := a.cfg.ReorderStreams(ids); err != nil {
		return err
	}
	a.emitStreamsChanged()
	return nil
}

// RemoveStream deletes a stream and its artwork.
func (a *App) RemoveStream(id string) error {
	removed, err := a.cfg.RemoveStreamByID(id)
	if err != nil {
		return err
	}
	images.DeleteStreamImage(a.cfg.ImagesDir(), removed.Image)
	a.emitStreamsChanged()
	a.emitPlaybackState()
	return nil
}

// PickStreamImage opens a file dialog and saves artwork for a stream.
func (a *App) PickStreamImage(streamID string) (StreamView, error) {
	if a.wails == nil {
		return StreamView{}, errInvalidInput("application not ready")
	}
	path, err := a.wails.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:          "Choose stream artwork",
		CanChooseFiles: true,
		Filters: []application.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.webp"},
		},
	}).PromptForSingleSelection()
	if err != nil {
		return StreamView{}, err
	}
	if path == "" {
		return StreamView{}, errDialogCancelled
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return StreamView{}, err
	}

	filename, err := images.SaveStreamImage(a.cfg.ImagesDir(), streamID, data)
	if err != nil {
		return StreamView{}, err
	}
	if err := a.cfg.SetStreamImage(streamID, filename); err != nil {
		return StreamView{}, err
	}

	a.emitStreamsChanged()
	s, ok := a.cfg.GetStreamByID(streamID)
	if !ok {
		return StreamView{}, errInvalidInput("stream not found")
	}
	view := StreamView{ID: s.ID, Name: s.Name, URL: s.URL, Image: s.Image}
	saved, err := os.ReadFile(images.ImagePath(a.cfg.ImagesDir(), filename))
	if err == nil {
		view.ImageData = "data:image/png;base64," + base64.StdEncoding.EncodeToString(saved)
	}
	return view, nil
}

// ImportStreamsResult reports how many streams were added from a bundle.
type ImportStreamsResult struct {
	Imported int `json:"imported"`
}

// ExportStreams writes streams, artwork, and portable settings to a zip file.
func (a *App) ExportStreams() error {
	if a.wails == nil {
		return errInvalidInput("application not ready")
	}
	path, err := a.wails.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    "Export streams",
		Filename: "icetray-streams.zip",
		Filters: []application.FileFilter{
			{DisplayName: "IceTray backup", Pattern: "*.zip"},
		},
	}).PromptForSingleSelection()
	if err != nil {
		return err
	}
	if path == "" {
		return errDialogCancelled
	}
	if !strings.HasSuffix(strings.ToLower(path), ".zip") {
		path += ".zip"
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data := a.cfg.SnapshotForExport()
	if err := exportbundle.Write(file, data, a.cfg.ImagesDir()); err != nil {
		return err
	}
	return file.Close()
}

// ImportStreams loads streams from an IceTray zip backup.
func (a *App) ImportStreams(replace bool) (ImportStreamsResult, error) {
	if a.wails == nil {
		return ImportStreamsResult{}, errInvalidInput("application not ready")
	}
	path, err := a.wails.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:          "Import streams",
		CanChooseFiles: true,
		Filters: []application.FileFilter{
			{DisplayName: "IceTray backup", Pattern: "*.zip"},
		},
	}).PromptForSingleSelection()
	if err != nil {
		return ImportStreamsResult{}, err
	}
	if path == "" {
		return ImportStreamsResult{}, errDialogCancelled
	}

	bundle, err := exportbundle.Read(path)
	if err != nil {
		return ImportStreamsResult{}, errInvalidInput(err.Error())
	}

	a.playbackMu.Lock()
	a.stopPlaybackLocked()
	a.currentID = ""
	a.nowPlaying = metadata.NowPlaying{}
	a.playbackMu.Unlock()

	count, err := a.cfg.ApplyBundle(bundle, replace)
	if err != nil {
		return ImportStreamsResult{}, err
	}

	a.emitNowPlaying()
	a.emitStreamsChanged()
	a.emitPlaybackState()
	return ImportStreamsResult{Imported: count}, nil
}

// PlayStream starts playback for a stream by ID.
func (a *App) PlayStream(id string) error {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()
	return a.playStreamLocked(id)
}

func (a *App) stopPlaybackLocked() {
	a.stopMetadataPoller()
	a.player.Stop()
	a.supervisor.Stop()
}

func (a *App) playStreamLocked(id string) error {
	s, ok := a.cfg.GetStreamByID(id)
	if !ok {
		return errInvalidInput("stream not found")
	}

	if a.outputPlayingLocked() && a.currentID == id {
		return nil
	}

	if a.casting {
		if a.currentID != id {
			a.stopMetadataPoller()
			a.startMetadataPoller(s.URL)
		}
		a.player.Stop()
		a.supervisor.Stop()
		_ = a.cfg.SetLastStreamID(id)
		a.currentID = id
		a.castPaused = false
		a.handoffPaused = false
		a.emitPlaybackState()
		return nil
	}

	if a.supervisor.IsRunning() && a.player.IsRunning() && a.currentID != "" {
		a.stopMetadataPoller()
		a.supervisor.SwitchStream(s.URL)
		_ = a.cfg.SetLastStreamID(id)
		a.currentID = id
		a.castPaused = false
		a.handoffPaused = false
		a.startMetadataPoller(s.URL)
		a.emitPlaybackState()
		return nil
	}

	a.stopPlaybackLocked()

	if err := a.player.Play(s.URL); err != nil {
		return err
	}
	a.supervisor.Start(s.URL)
	a.player.SetVolume(a.playbackVolume())
	_ = a.cfg.SetLastStreamID(id)
	a.currentID = id
	a.castPaused = false
	a.handoffPaused = false
	a.startMetadataPoller(s.URL)
	a.emitPlaybackState()
	return nil
}

// Pause pauses playback.
func (a *App) Pause() error {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()

	if a.casting {
		if a.castPaused {
			return nil
		}
		a.castPaused = true
		a.emitPlaybackState()
		return nil
	}
	if a.handoffPaused {
		return nil
	}

	if err := a.player.Pause(); err != nil {
		return err
	}
	a.emitPlaybackState()
	return nil
}

// Resume resumes playback.
func (a *App) Resume() error {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()

	if a.casting {
		if !a.castPaused {
			return nil
		}
		a.castPaused = false
		a.emitPlaybackState()
		return nil
	}
	if a.handoffPaused && a.currentID != "" {
		a.handoffPaused = false
		return a.playStreamLocked(a.currentID)
	}

	if err := a.player.Resume(); err != nil {
		return err
	}
	a.emitPlaybackState()
	return nil
}

// Stop stops playback.
func (a *App) Stop() error {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()

	a.stopPlaybackLocked()
	a.currentID = ""
	a.castPaused = false
	a.handoffPaused = false
	a.nowPlaying = metadata.NowPlaying{}
	a.emitNowPlaying()
	a.emitPlaybackState()
	return nil
}

// GetPlaybackState returns current playback state.
func (a *App) GetPlaybackState() PlaybackState {
	if a.casting {
		has := a.currentID != ""
		return PlaybackState{
			Playing:  has && !a.castPaused,
			Paused:   has && a.castPaused,
			Loading:  false,
			StreamID: a.currentID,
			Volume:   a.cfg.GetVolume(),
		}
	}
	if a.handoffPaused && a.currentID != "" && !a.player.IsRunning() {
		return PlaybackState{
			Playing:  false,
			Paused:   true,
			Loading:  false,
			StreamID: a.currentID,
			Volume:   a.cfg.GetVolume(),
		}
	}
	return PlaybackState{
		Playing:  a.player.IsPlaying(),
		Paused:   a.player.IsRunning() && a.player.IsPaused(),
		Loading:  a.player.IsBuffering(),
		StreamID: a.currentID,
		Volume:   a.cfg.GetVolume(),
	}
}

func (a *App) outputPlayingLocked() bool {
	if a.casting {
		return a.currentID != "" && !a.castPaused
	}
	return a.player.IsPlaying()
}

// SetCasting switches local oto output off while a Cast session is active.
func (a *App) SetCasting(enabled bool) {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()
	if enabled {
		if a.casting {
			a.emitPlaybackState()
			return
		}
		a.casting = true
		wasPaused := (a.player.IsRunning() && a.player.IsPaused()) || a.handoffPaused
		a.handoffPaused = false
		a.castPaused = wasPaused && a.currentID != ""
		a.player.Stop()
		a.supervisor.Stop()
		if a.currentID != "" && a.metaCancel == nil {
			if s, ok := a.cfg.GetStreamByID(a.currentID); ok {
				a.startMetadataPoller(s.URL)
			}
		}
		a.emitPlaybackState()
		return
	}
	if !a.casting {
		return
	}
	a.casting = false
	paused := a.castPaused
	a.castPaused = false
	id := a.currentID
	if id == "" {
		a.emitPlaybackState()
		return
	}
	if paused {
		a.handoffPaused = true
		a.emitPlaybackState()
		return
	}
	_ = a.playStreamLocked(id)
}

// SetVolume sets volume 0-100. Android uses the device volume only.
func (a *App) SetVolume(vol int) error {
	if runtime.GOOS == "android" {
		return nil
	}
	if err := a.cfg.SetVolume(vol); err != nil {
		return err
	}
	if err := a.player.SetVolume(vol); err != nil {
		return err
	}
	a.emitPlaybackState()
	return nil
}

func (a *App) playbackVolume() int {
	if runtime.GOOS == "android" {
		return 100
	}
	return a.cfg.GetVolume()
}

// GetSettings returns app settings.
func (a *App) GetSettings() SettingsView {
	return SettingsView{
		Autoplay:        a.cfg.GetAutoplay(),
		LaunchOnLogin:   a.cfg.GetLaunchOnLogin(),
		LaunchMinimized: a.cfg.GetLaunchMinimized(),
		Volume:           a.cfg.GetVolume(),
		CrossfadeSeconds: a.cfg.GetCrossfadeSeconds(),
		Desktop:          runtime.GOOS != "android",
		Version:         appVersion(),
	}
}

// SetAutoplay toggles autoplay on startup.
func (a *App) SetAutoplay(enabled bool) error {
	return a.cfg.SetAutoplay(enabled)
}

// SetLaunchOnLogin toggles launch on login.
func (a *App) SetLaunchOnLogin(enabled bool) error {
	if enabled {
		if err := a.startupMgr.Enable(); err != nil {
			return err
		}
	} else {
		if err := a.startupMgr.Disable(); err != nil {
			return err
		}
	}
	return a.cfg.SetLaunchOnLogin(enabled)
}

// SetLaunchMinimized toggles whether the player window starts hidden (desktop tray only).
func (a *App) SetLaunchMinimized(enabled bool) error {
	return a.cfg.SetLaunchMinimized(enabled)
}

// SetCrossfadeSeconds sets stream crossfade duration (0–8 seconds).
func (a *App) SetCrossfadeSeconds(seconds int) error {
	if err := a.cfg.SetCrossfadeSeconds(seconds); err != nil {
		return err
	}
	a.applyCrossfadeFromConfig()
	return nil
}

// GetNowPlaying returns the latest metadata snapshot.
func (a *App) GetNowPlaying() metadata.NowPlaying {
	return a.nowPlaying
}

// ShowPlayer shows and focuses the main window.
func (a *App) ShowPlayer() {
	if a.window == nil {
		return
	}
	a.window.Show()
	a.window.UnMinimise()
	a.window.Focus()
}

// QuitApp exits the application.
func (a *App) QuitApp() {
	if a.wails == nil {
		return
	}
	a.wails.Quit()
}

func (a *App) startMetadataPoller(streamURL string) {
	a.stopMetadataPoller()
	ctx, cancel := context.WithCancel(context.Background())
	a.metaCancel = cancel

	go func() {
		p := metadata.NewPoller(10*time.Second, func(np metadata.NowPlaying) {
			a.nowPlaying = np
			a.emitNowPlaying()
		})
		p.Run(ctx, streamURL)
	}()
}

func (a *App) stopMetadataPoller() {
	if a.metaCancel != nil {
		a.metaCancel()
		a.metaCancel = nil
	}
}

type invalidInputError string

var errDialogCancelled = errors.New("dialog cancelled")

func (e invalidInputError) Error() string { return string(e) }

func errInvalidInput(msg string) error {
	return invalidInputError(msg)
}

// PlayLastStream plays the last selected stream if any.
func (a *App) PlayLastStream() error {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()
	return a.playLastStreamLocked()
}

func (a *App) playLastStreamLocked() error {
	id := a.cfg.GetLastStreamID()
	if id != "" {
		return a.playStreamLocked(id)
	}
	url := a.cfg.GetLastStream()
	if url == "" {
		return nil
	}
	for _, s := range a.cfg.GetStreams() {
		if s.URL == url {
			return a.playStreamLocked(s.ID)
		}
	}
	return nil
}

// TrayPlay handles play from the system tray.
func (a *App) TrayPlay() {
	state := a.GetPlaybackState()
	if state.Paused {
		_ = a.Resume()
		return
	}
	if state.Playing {
		return
	}

	id := a.cfg.GetLastStreamID()
	if id == "" {
		url := a.cfg.GetLastStream()
		for _, s := range a.cfg.GetStreams() {
			if s.URL == url {
				id = s.ID
				break
			}
		}
	}
	if id != "" {
		_ = a.PlayStream(id)
	}
}

// TrayPause handles pause from the system tray.
func (a *App) TrayPause() {
	_ = a.Pause()
}

// TrayStop handles stop from the system tray.
func (a *App) TrayStop() {
	_ = a.Stop()
}

// Shutdown cleans up on application exit.
func (a *App) Shutdown() {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()
	a.stopPlaybackLocked()
	logger.Log("Application shutdown")
}

// GetImagePath returns the filesystem path for a stream image (for tray/debug).
func (a *App) GetImagePath(filename string) string {
	return filepath.Join(a.cfg.ImagesDir(), filename)
}

func (a *App) sessionArtworkPath() string {
	if a.currentID == "" {
		return ""
	}
	s, ok := a.cfg.GetStreamByID(a.currentID)
	if !ok || s.Image == "" {
		return ""
	}
	return images.ImagePath(a.cfg.ImagesDir(), s.Image)
}

func (a *App) sessionStationName() string {
	if a.currentID == "" {
		return "IceTray"
	}
	s, ok := a.cfg.GetStreamByID(a.currentID)
	if !ok {
		return "IceTray"
	}
	return s.Name
}
