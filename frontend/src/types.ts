export interface StreamView {
  id: string
  name: string
  url: string
  image?: string
  imageData?: string
}

export interface PlaybackState {
  playing: boolean
  paused: boolean
  loading: boolean
  streamId: string
  volume: number
}

export interface SettingsView {
  autoplay: boolean
  launchOnLogin: boolean
  launchMinimized: boolean
  volume: number
  crossfadeSeconds: number
  desktop: boolean
  version: string
}

export interface NowPlaying {
  station: string
  title: string
  genre?: string
  listeners?: number
}

export interface WailsEvent<T> {
  data: T
}
