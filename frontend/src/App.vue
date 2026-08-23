<script setup lang="ts">
import {
  Check,
  ImagePlus,
  LoaderCircle,
  Music2,
  Pause,
  Pencil,
  Play,
  Plus,
  Settings,
  Square,
  Trash2,
  Users,
  Volume2,
  X,
} from '@lucide/vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  AddStream,
  GetNowPlaying,
  GetPlaybackState,
  GetSettings,
  GetStreams,
  Pause as PausePlayback,
  PickStreamImage,
  PlayStream,
  RemoveStream,
  Resume,
  SetAutoplay,
  SetLaunchMinimized,
  SetLaunchOnLogin,
  SetVolume,
  Stop,
  UpdateStream,
} from '../wailsjs/go/main/App.js'
import type { NowPlaying, PlaybackState, SettingsView, StreamView } from './types'
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const iconSize = 18
const iconSizeLg = 22
const iconSizeSm = 15

const streams = ref<StreamView[]>([])
const playback = ref<PlaybackState>({ playing: false, paused: false, streamId: '', volume: 80 })
const settings = ref<SettingsView>({
  autoplay: false,
  launchOnLogin: false,
  launchMinimized: false,
  volume: 80,
})
const nowPlaying = ref<NowPlaying>({ station: '', title: '' })

const showModal = ref(false)
const showSettings = ref(false)
const editing = ref<StreamView | null>(null)
const formName = ref('')
const formURL = ref('')
const formError = ref('')
const loading = ref(true)

const currentStream = computed(() => streams.value.find((s) => s.id === playback.value.streamId))
const displayTitle = computed(() => nowPlaying.value.title || currentStream.value?.name || 'Nothing playing')
const displaySubtitle = computed(() => {
  if (nowPlaying.value.station && nowPlaying.value.title) return nowPlaying.value.station
  if (nowPlaying.value.genre) return nowPlaying.value.genre
  return currentStream.value?.url || 'Select a stream to begin'
})

let unsubs: Array<() => void> = []

async function refreshStreams() {
  streams.value = await GetStreams()
}

async function refreshPlayback() {
  playback.value = await GetPlaybackState()
}

async function refreshSettings() {
  settings.value = await GetSettings()
}

async function refreshNowPlaying() {
  nowPlaying.value = await GetNowPlaying()
}

async function loadAll() {
  loading.value = true
  await Promise.all([refreshStreams(), refreshPlayback(), refreshSettings(), refreshNowPlaying()])
  loading.value = false
}

function openAdd() {
  editing.value = null
  formName.value = ''
  formURL.value = ''
  formError.value = ''
  showModal.value = true
}

function openEdit(stream: StreamView) {
  editing.value = stream
  formName.value = stream.name
  formURL.value = stream.url
  formError.value = ''
  showModal.value = true
}

async function saveStream() {
  formError.value = ''
  try {
    if (editing.value) {
      await UpdateStream(editing.value.id, formName.value, formURL.value)
    } else {
      await AddStream(formName.value, formURL.value)
    }
    showModal.value = false
    await refreshStreams()
  } catch (e: any) {
    formError.value = e?.message || 'Failed to save stream'
  }
}

async function deleteStream(stream: StreamView) {
  if (!confirm(`Remove "${stream.name}"?`)) return
  await RemoveStream(stream.id)
  await refreshStreams()
  await refreshPlayback()
}

async function uploadImage(stream: StreamView) {
  try {
    const updated = await PickStreamImage(stream.id)
    if (!updated?.id) return
    const idx = streams.value.findIndex((s) => s.id === stream.id)
    if (idx >= 0) {
      streams.value[idx] = { ...streams.value[idx], ...updated }
    }
  } catch {
    // user cancelled picker
  }
}

async function play(stream: StreamView) {
  await PlayStream(stream.id)
  await refreshPlayback()
  await refreshNowPlaying()
}

async function togglePlay() {
  if (playback.value.playing) {
    await PausePlayback()
  } else if (playback.value.paused) {
    await Resume()
  } else if (currentStream.value) {
    await PlayStream(currentStream.value.id)
  } else if (streams.value.length > 0) {
    await PlayStream(streams.value[0].id)
  }
  await refreshPlayback()
}

async function stopPlayback() {
  await Stop()
  nowPlaying.value = { station: '', title: '' }
  await refreshPlayback()
}

async function onVolumeInput(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  playback.value.volume = value
  await SetVolume(value)
}

async function setAutoplay(enabled: boolean) {
  await SetAutoplay(enabled)
  settings.value.autoplay = enabled
}

async function setLaunchOnLogin(enabled: boolean) {
  await SetLaunchOnLogin(enabled)
  settings.value.launchOnLogin = enabled
}

async function setLaunchMinimized(enabled: boolean) {
  await SetLaunchMinimized(enabled)
  settings.value.launchMinimized = enabled
}

onMounted(async () => {
  await loadAll()
  unsubs.push(
    EventsOn('playback:state', (state: PlaybackState) => {
      playback.value = state
    }),
    EventsOn('nowplaying:update', (np: NowPlaying) => {
      nowPlaying.value = np
    }),
    EventsOn('streams:changed', () => {
      refreshStreams()
    }),
  )
})

onUnmounted(() => {
  unsubs.forEach((u) => u())
})
</script>

<template>
  <div class="flex h-full flex-col">
    <header class="flex items-center justify-between border-b border-white/10 px-6 py-4">
      <div>
        <h1 class="text-xl font-semibold tracking-tight">IceTray</h1>
        <p class="text-sm text-zinc-400">Your Icecast stations</p>
      </div>
      <div class="flex gap-2">
        <button
          class="icon-btn"
          :class="showSettings ? 'icon-btn-active' : ''"
          title="Settings"
          aria-label="Settings"
          @click="showSettings = !showSettings"
        >
          <Settings :size="iconSize" />
        </button>
        <button class="icon-btn icon-btn-primary" title="Add stream" aria-label="Add stream" @click="openAdd">
          <Plus :size="iconSize" />
        </button>
      </div>
    </header>

    <div v-if="showSettings" class="border-b border-white/10 bg-white/5 px-6 py-4">
      <p class="mb-3 text-sm font-medium text-zinc-300">Settings</p>
      <div class="space-y-4">
        <div class="setting-row">
          <div class="setting-copy">
            <p class="setting-title">Autoplay on startup</p>
            <p class="setting-desc">Play your last stream automatically when IceTray opens.</p>
          </div>
          <label class="setting-switch">
            <input
              type="checkbox"
              :checked="settings.autoplay"
              aria-label="Autoplay on startup"
              @change="setAutoplay(($event.target as HTMLInputElement).checked)"
            />
            <span class="setting-switch-track" />
          </label>
        </div>
        <div class="setting-row">
          <div class="setting-copy">
            <p class="setting-title">Launch on login</p>
            <p class="setting-desc">Start IceTray in the background when you sign in to your computer.</p>
          </div>
          <label class="setting-switch">
            <input
              type="checkbox"
              :checked="settings.launchOnLogin"
              aria-label="Launch on login"
              @change="setLaunchOnLogin(($event.target as HTMLInputElement).checked)"
            />
            <span class="setting-switch-track" />
          </label>
        </div>
        <div class="setting-row">
          <div class="setting-copy">
            <p class="setting-title">Start minimised</p>
            <p class="setting-desc">Keep the player window hidden on startup. Open it from the system tray when you need it.</p>
          </div>
          <label class="setting-switch">
            <input
              type="checkbox"
              :checked="settings.launchMinimized"
              aria-label="Start minimised"
              @change="setLaunchMinimized(($event.target as HTMLInputElement).checked)"
            />
            <span class="setting-switch-track" />
          </label>
        </div>
      </div>
    </div>

    <main class="flex-1 overflow-y-auto px-6 py-6">
      <div v-if="loading" class="flex items-center gap-2 text-zinc-400">
        <LoaderCircle :size="iconSize" class="animate-spin" />
      </div>
      <div v-else-if="streams.length === 0" class="rounded-2xl border border-dashed border-white/10 p-12 text-center">
        <Music2 :size="48" class="mx-auto text-zinc-600" />
        <p class="mt-4 text-lg text-zinc-300">No streams yet</p>
        <p class="mt-2 text-sm text-zinc-500">Add your first Icecast stream to get started.</p>
        <button class="icon-btn icon-btn-primary mx-auto mt-6" title="Add stream" aria-label="Add stream" @click="openAdd">
          <Plus :size="iconSize" />
        </button>
      </div>
      <div v-else class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
        <article
          v-for="stream in streams"
          :key="stream.id"
          class="group relative overflow-hidden rounded-2xl border border-white/10 bg-white/5 transition hover:border-emerald-400/40 hover:bg-white/10"
          :class="playback.streamId === stream.id ? 'ring-2 ring-emerald-400/60' : ''"
        >
          <button class="block w-full text-left" @click="play(stream)">
            <div class="relative aspect-square w-full overflow-hidden bg-zinc-900">
              <img
                v-if="stream.imageData"
                :src="stream.imageData"
                :alt="stream.name"
                class="h-full w-full object-cover transition duration-300 group-hover:scale-105"
              />
              <div v-else class="flex h-full items-center justify-center bg-gradient-to-br from-zinc-800 to-zinc-900 text-zinc-600">
                <Music2 :size="40" />
              </div>
              <div class="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition group-hover:opacity-100">
                <Play :size="32" class="text-white" fill="currentColor" />
              </div>
            </div>
            <div class="p-3">
              <h2 class="truncate font-medium">{{ stream.name }}</h2>
              <p class="truncate text-xs text-zinc-500">{{ stream.url }}</p>
            </div>
          </button>
          <div class="absolute right-2 top-2 flex gap-1 opacity-0 transition group-hover:opacity-100">
            <button class="card-action-btn" title="Upload artwork" aria-label="Upload artwork" @click.stop="uploadImage(stream)">
              <ImagePlus :size="iconSizeSm" />
            </button>
            <button class="card-action-btn" title="Edit stream" aria-label="Edit stream" @click.stop="openEdit(stream)">
              <Pencil :size="iconSizeSm" />
            </button>
            <button class="card-action-btn card-action-btn-danger" title="Delete stream" aria-label="Delete stream" @click.stop="deleteStream(stream)">
              <Trash2 :size="iconSizeSm" />
            </button>
          </div>
        </article>
      </div>
    </main>

    <footer class="border-t border-white/10 bg-black/40 px-6 py-4 backdrop-blur">
      <div class="flex items-center gap-4">
        <div class="h-14 w-14 shrink-0 overflow-hidden rounded-lg bg-zinc-800">
          <img
            v-if="currentStream?.imageData"
            :src="currentStream.imageData"
            class="h-full w-full object-cover"
            alt=""
          />
          <div v-else class="flex h-full items-center justify-center text-zinc-500">
            <Music2 :size="iconSizeLg" />
          </div>
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{{ displayTitle }}</p>
          <p class="truncate text-sm text-zinc-400">{{ displaySubtitle }}</p>
          <p v-if="nowPlaying.listeners" class="flex items-center gap-1 text-xs text-zinc-500">
            <Users :size="12" />
            {{ nowPlaying.listeners }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="icon-btn icon-btn-lg"
            :title="playback.playing ? 'Pause' : 'Play'"
            :aria-label="playback.playing ? 'Pause' : 'Play'"
            @click="togglePlay"
          >
            <Pause v-if="playback.playing" :size="iconSizeLg" fill="currentColor" />
            <Play v-else :size="iconSizeLg" fill="currentColor" />
          </button>
          <button class="icon-btn icon-btn-lg" title="Stop" aria-label="Stop" @click="stopPlayback">
            <Square :size="iconSize" fill="currentColor" />
          </button>
        </div>
        <div class="hidden w-44 items-center gap-2 md:flex">
          <Volume2 :size="iconSizeSm" class="shrink-0 text-zinc-500" />
          <input
            type="range"
            min="0"
            max="100"
            :value="playback.volume"
            class="w-full accent-emerald-400"
            :title="`Volume ${playback.volume}%`"
            :aria-label="`Volume ${playback.volume}%`"
            @input="onVolumeInput"
          />
        </div>
      </div>
    </footer>

    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
      <div class="w-full max-w-md rounded-2xl border border-white/10 bg-zinc-900 p-6 shadow-2xl">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit stream' : 'Add stream' }}</h3>
          <button class="icon-btn" title="Close" aria-label="Close" @click="showModal = false">
            <X :size="iconSize" />
          </button>
        </div>
        <div class="mt-4 space-y-3">
          <input
            v-model="formName"
            placeholder="Station name"
            class="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 outline-none focus:border-emerald-400"
          />
          <input
            v-model="formURL"
            placeholder="https://example.com/stream.mp3"
            class="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 outline-none focus:border-emerald-400"
          />
          <p v-if="formError" class="text-sm text-red-400">{{ formError }}</p>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <button class="icon-btn" title="Cancel" aria-label="Cancel" @click="showModal = false">
            <X :size="iconSize" />
          </button>
          <button class="icon-btn icon-btn-primary" title="Save" aria-label="Save" @click="saveStream">
            <Check :size="iconSize" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  border: 1px solid rgb(255 255 255 / 0.1);
  background: rgb(255 255 255 / 0.05);
  padding: 0.5rem;
  color: rgb(212 212 216);
  transition: background-color 150ms, color 150ms, border-color 150ms;
}

.icon-btn:hover {
  background: rgb(255 255 255 / 0.12);
  color: rgb(255 255 255);
}

.icon-btn-active {
  border-color: rgb(52 211 153 / 0.5);
  background: rgb(52 211 153 / 0.15);
  color: rgb(110 231 183);
}

.icon-btn-primary {
  border-color: rgb(52 211 153 / 0.4);
  background: rgb(52 211 153);
  color: rgb(0 0 0);
}

.icon-btn-primary:hover {
  background: rgb(74 222 128);
  color: rgb(0 0 0);
}

.icon-btn-lg {
  padding: 0.75rem;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.setting-copy {
  min-width: 0;
}

.setting-title {
  font-size: 0.9375rem;
  font-weight: 500;
  color: rgb(244 244 245);
}

.setting-desc {
  margin-top: 0.25rem;
  font-size: 0.8125rem;
  line-height: 1.4;
  color: rgb(113 113 122);
}

.setting-switch {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  cursor: pointer;
}

.setting-switch input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.setting-switch-track {
  display: block;
  width: 2.75rem;
  height: 1.5rem;
  border-radius: 9999px;
  background: rgb(63 63 70);
  transition: background-color 150ms;
}

.setting-switch-track::after {
  content: '';
  position: absolute;
  top: 0.1875rem;
  left: 0.1875rem;
  width: 1.125rem;
  height: 1.125rem;
  border-radius: 9999px;
  background: rgb(255 255 255);
  transition: transform 150ms;
}

.setting-switch input:checked + .setting-switch-track {
  background: rgb(52 211 153);
}

.setting-switch input:checked + .setting-switch-track::after {
  transform: translateX(1.25rem);
}

.setting-switch input:focus-visible + .setting-switch-track {
  outline: 2px solid rgb(52 211 153 / 0.6);
  outline-offset: 2px;
}

.card-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.375rem;
  background: rgb(0 0 0 / 0.65);
  padding: 0.375rem;
  color: rgb(228 228 231);
  backdrop-filter: blur(4px);
  transition: background-color 150ms, color 150ms;
}

.card-action-btn:hover {
  background: rgb(0 0 0 / 0.85);
  color: rgb(255 255 255);
}

.card-action-btn-danger:hover {
  color: rgb(252 165 165);
}
</style>
