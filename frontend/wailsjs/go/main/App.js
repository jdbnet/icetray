export function GetStreams() {
  return window.go.main.App.GetStreams()
}

export function AddStream(name, url) {
  return window.go.main.App.AddStream(name, url)
}

export function UpdateStream(id, name, url) {
  return window.go.main.App.UpdateStream(id, name, url)
}

export function RemoveStream(id) {
  return window.go.main.App.RemoveStream(id)
}

export function PickStreamImage(id) {
  return window.go.main.App.PickStreamImage(id)
}

export function PlayStream(id) {
  return window.go.main.App.PlayStream(id)
}

export function Pause() {
  return window.go.main.App.Pause()
}

export function Resume() {
  return window.go.main.App.Resume()
}

export function Stop() {
  return window.go.main.App.Stop()
}

export function GetPlaybackState() {
  return window.go.main.App.GetPlaybackState()
}

export function SetVolume(vol) {
  return window.go.main.App.SetVolume(vol)
}

export function GetSettings() {
  return window.go.main.App.GetSettings()
}

export function SetAutoplay(enabled) {
  return window.go.main.App.SetAutoplay(enabled)
}

export function SetLaunchOnLogin(enabled) {
  return window.go.main.App.SetLaunchOnLogin(enabled)
}

export function SetLaunchMinimized(enabled) {
  return window.go.main.App.SetLaunchMinimized(enabled)
}

export function GetNowPlaying() {
  return window.go.main.App.GetNowPlaying()
}
