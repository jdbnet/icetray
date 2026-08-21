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
  streamId: string
  volume: number
}

export interface SettingsView {
  autoplay: boolean
  launchOnLogin: boolean
  volume: number
}

export interface NowPlaying {
  station: string
  title: string
  genre?: string
  listeners?: number
}

export {
  AddStream,
  GetNowPlaying,
  GetPlaybackState,
  GetSettings,
  GetStreams,
  Pause,
  PickStreamImage,
  PlayStream,
  RemoveStream,
  Resume,
  SetAutoplay,
  SetLaunchOnLogin,
  SetVolume,
  Stop,
  UpdateStream,
} from './App.js'
