<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
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
  type NowPlaying,
  type PlaybackState,
  type SettingsView,
  type StreamView,
} from '../wailsjs/go/main/App.ts'
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const streams = ref<StreamView[]>([])
const playback = ref<PlaybackState>({ playing: false, paused: false, streamId: '', volume: 80 })
const settings = ref<SettingsView>({ autoplay: false, launchOnLogin: false, volume: 80 })
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
    const idx = streams.value.findIndex((s) => s.id === stream.id)
    if (idx >= 0) streams.value[idx] = updated
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
    await Pause()
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

async function toggleAutoplay() {
  const next = !settings.value.autoplay
  await SetAutoplay(next)
  settings.value.autoplay = next
}

async function toggleLaunchOnLogin() {
  const next = !settings.value.launchOnLogin
  await SetLaunchOnLogin(next)
  settings.value.launchOnLogin = next
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
          class="rounded-lg border border-white/10 px-3 py-2 text-sm text-zinc-300 hover:bg-white/5"
          @click="showSettings = !showSettings"
        >
          Settings
        </button>
        <button
          class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-black hover:bg-emerald-400"
          @click="openAdd"
        >
          Add stream
        </button>
      </div>
    </header>

    <div v-if="showSettings" class="border-b border-white/10 bg-white/5 px-6 py-4">
      <div class="flex flex-wrap gap-6 text-sm">
        <label class="flex items-center gap-2">
          <input type="checkbox" :checked="settings.autoplay" @change="toggleAutoplay" />
          Autoplay on startup
        </label>
        <label class="flex items-center gap-2">
          <input type="checkbox" :checked="settings.launchOnLogin" @change="toggleLaunchOnLogin" />
          Launch on login
        </label>
      </div>
    </div>

    <main class="flex-1 overflow-y-auto px-6 py-6">
      <div v-if="loading" class="text-zinc-400">Loading...</div>
      <div v-else-if="streams.length === 0" class="rounded-2xl border border-dashed border-white/10 p-12 text-center">
        <p class="text-lg text-zinc-300">No streams yet</p>
        <p class="mt-2 text-sm text-zinc-500">Add your first Icecast stream to get started.</p>
        <button class="mt-6 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-black" @click="openAdd">
          Add stream
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
            <div class="aspect-square w-full overflow-hidden bg-zinc-900">
              <img
                v-if="stream.imageData"
                :src="stream.imageData"
                :alt="stream.name"
                class="h-full w-full object-cover transition duration-300 group-hover:scale-105"
              />
              <div v-else class="flex h-full items-center justify-center bg-gradient-to-br from-zinc-800 to-zinc-900 text-4xl text-zinc-600">
                ♪
              </div>
            </div>
            <div class="p-3">
              <h2 class="truncate font-medium">{{ stream.name }}</h2>
              <p class="truncate text-xs text-zinc-500">{{ stream.url }}</p>
            </div>
          </button>
          <div class="absolute right-2 top-2 flex gap-1 opacity-0 transition group-hover:opacity-100">
            <button class="rounded-md bg-black/60 px-2 py-1 text-xs" @click.stop="uploadImage(stream)">Art</button>
            <button class="rounded-md bg-black/60 px-2 py-1 text-xs" @click.stop="openEdit(stream)">Edit</button>
            <button class="rounded-md bg-black/60 px-2 py-1 text-xs text-red-300" @click.stop="deleteStream(stream)">Del</button>
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
          <div v-else class="flex h-full items-center justify-center text-zinc-500">♪</div>
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{{ displayTitle }}</p>
          <p class="truncate text-sm text-zinc-400">{{ displaySubtitle }}</p>
          <p v-if="nowPlaying.listeners" class="text-xs text-zinc-500">{{ nowPlaying.listeners }} listeners</p>
        </div>
        <div class="flex items-center gap-2">
          <button class="rounded-full bg-white/10 px-4 py-2 text-sm hover:bg-white/20" @click="togglePlay">
            {{ playback.playing ? 'Pause' : 'Play' }}
          </button>
          <button class="rounded-full bg-white/10 px-4 py-2 text-sm hover:bg-white/20" @click="stopPlayback">Stop</button>
        </div>
        <div class="hidden w-40 items-center gap-2 md:flex">
          <span class="text-xs text-zinc-500">VOL</span>
          <input
            type="range"
            min="0"
            max="100"
            :value="playback.volume"
            class="w-full accent-emerald-400"
            @input="onVolumeInput"
          />
        </div>
      </div>
    </footer>

    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
      <div class="w-full max-w-md rounded-2xl border border-white/10 bg-zinc-900 p-6 shadow-2xl">
        <h3 class="text-lg font-semibold">{{ editing ? 'Edit stream' : 'Add stream' }}</h3>
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
          <button class="rounded-lg px-4 py-2 text-sm text-zinc-400 hover:text-white" @click="showModal = false">Cancel</button>
          <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-black" @click="saveStream">Save</button>
        </div>
      </div>
    </div>
  </div>
</template>
