<script setup lang="ts">
import { Download, Monitor, Radio, Server, Smartphone, Terminal } from '@lucide/vue'
import GithubIcon from './components/GithubIcon.vue'
import { artifactUrl, artifacts, site } from './site'

const features = [
  {
    title: 'Modern player UI',
    text: 'Stream library, artwork, volume, and now playing info in one place.',
  },
  {
    title: 'Desktop system tray',
    text: 'Background playback with play, pause, and stop from the tray.',
  },
  {
    title: 'Embedded audio engine',
    text: 'Powered by gopxl/beep with buffering for network hiccups.',
  },
  {
    title: 'Android media session',
    text: 'Notification, lock screen, and Bluetooth controls on Android.',
  },
  {
    title: 'Icecast metadata',
    text: 'Now playing via public stats, legacy JSON, and ICY fallbacks.',
  },
  {
    title: 'Headless mode',
    text: 'Terminal-only binary for servers with a simple --stream flag.',
  },
]

const aptSnippet = `curl -fsSL ${site.aptInstallScript} | sudo bash
sudo apt install icetray`

const headlessSnippet = `./icetray-headless-linux-amd64 --stream https://icecast.example.com/stream.mp3`
</script>

<template>
  <div class="min-h-screen">
    <header
      class="sticky top-0 z-50 border-b border-zinc-800/80 bg-zinc-950/80 backdrop-blur-md"
    >
      <div
        class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-3 sm:px-6"
      >
        <a href="#" class="flex items-center gap-3">
          <img src="/icon.png" alt="" class="h-9 w-9 rounded-lg" width="36" height="36" />
          <span class="text-lg font-semibold tracking-tight">{{ site.name }}</span>
        </a>
        <nav class="flex items-center gap-5 text-sm text-zinc-400">
          <a href="#features" class="hover:text-zinc-100 transition-colors">Features</a>
          <a href="#how-it-works" class="hover:text-zinc-100 transition-colors">How it works</a>
          <a href="#download" class="hover:text-zinc-100 transition-colors">Download</a>
          <a
            :href="site.githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-zinc-300 hover:text-white transition-colors"
            aria-label="GitHub repository"
          >
            <GithubIcon :size="22" />
          </a>
        </nav>
      </div>
    </header>

    <main>
      <section class="mx-auto max-w-5xl px-4 py-16 sm:px-6 sm:py-24 text-center">
        <img
          src="/icon.png"
          alt=""
          class="mx-auto mb-6 h-24 w-24 rounded-2xl shadow-lg shadow-black/40"
          width="96"
          height="96"
        />
        <h1 class="text-4xl font-bold tracking-tight sm:text-5xl">
          Internet radio for Icecast
        </h1>
        <p class="mx-auto mt-4 max-w-2xl text-lg text-zinc-400">
          IceTray is a lightweight player for Icecast streams. Desktop builds live in the system
          tray with a Wails UI. Android uses the same layout with standard media controls.
        </p>
        <p class="mt-3 text-sm text-zinc-500">Release v{{ site.version }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <a
            href="#download"
            class="inline-flex items-center gap-2 rounded-lg bg-sky-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-sky-500 transition-colors"
          >
            <Download :size="18" />
            Download v{{ site.version }}
          </a>
          <a
            :href="site.githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-2 rounded-lg border border-zinc-700 px-5 py-2.5 text-sm font-medium text-zinc-200 hover:border-zinc-500 hover:bg-zinc-900 transition-colors"
          >
            <GithubIcon :size="18" />
            View on GitHub
          </a>
        </div>
      </section>

      <section id="features" class="border-t border-zinc-800/60 bg-zinc-900/30">
        <div class="mx-auto max-w-5xl px-4 py-16 sm:px-6">
          <h2 class="text-2xl font-semibold">Features</h2>
          <p class="mt-2 text-zinc-400">Everything you need for everyday listening.</p>
          <ul class="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            <li
              v-for="item in features"
              :key="item.title"
              class="rounded-xl border border-zinc-800 bg-zinc-950/50 p-5"
            >
              <h3 class="font-medium text-zinc-100">{{ item.title }}</h3>
              <p class="mt-2 text-sm text-zinc-400">{{ item.text }}</p>
            </li>
          </ul>
        </div>
      </section>

      <section id="how-it-works" class="border-t border-zinc-800/60">
        <div class="mx-auto max-w-5xl px-4 py-16 sm:px-6">
          <h2 class="text-2xl font-semibold">How it works</h2>
          <div class="mt-10 grid gap-8 md:grid-cols-3">
            <div class="rounded-xl border border-zinc-800 p-6">
              <Monitor class="text-sky-400" :size="28" />
              <h3 class="mt-4 font-medium">Desktop</h3>
              <p class="mt-2 text-sm text-zinc-400">
                Add Icecast URLs to your library, pick artwork, and control playback from the
                window or system tray. Optional launch on login and minimized start.
              </p>
            </div>
            <div class="rounded-xl border border-zinc-800 p-6">
              <Smartphone class="text-sky-400" :size="28" />
              <h3 class="mt-4 font-medium">Android</h3>
              <p class="mt-2 text-sm text-zinc-400">
                Same stream library on your phone with notification and lock-screen controls.
                Google Cast support for speakers on your network.
              </p>
            </div>
            <div class="rounded-xl border border-zinc-800 p-6">
              <Terminal class="text-sky-400" :size="28" />
              <h3 class="mt-4 font-medium">Headless Linux</h3>
              <p class="mt-2 text-sm text-zinc-400">
                Run without a GUI on a server or Pi. Point at a stream URL and let the embedded
                engine handle buffering and reconnects.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section id="download" class="border-t border-zinc-800/60 bg-zinc-900/30">
        <div class="mx-auto max-w-5xl px-4 py-16 sm:px-6">
          <h2 class="text-2xl font-semibold flex items-center gap-2">
            <Download :size="26" />
            Download
          </h2>
          <p class="mt-2 text-zinc-400">
            Artifacts for
            <span class="text-zinc-200">v{{ site.version }}</span>
            from
            <a
              :href="site.releaseBase"
              class="text-sky-400 hover:underline"
              target="_blank"
              rel="noopener noreferrer"
            >GitHub Releases</a>.
          </p>

          <div class="mt-12 space-y-12">
            <div>
              <h3 class="flex items-center gap-2 text-lg font-medium">
                <Monitor :size="20" class="text-zinc-400" />
                Windows
              </h3>
              <ul class="mt-4 space-y-2 text-sm">
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.winSetupAmd64)"
                  >Installer (amd64)</a>
                  <span class="text-zinc-500"> - {{ artifacts.winSetupAmd64 }}</span>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.winSetupArm64)"
                  >Installer (arm64)</a>
                  <span class="text-zinc-500"> - {{ artifacts.winSetupArm64 }}</span>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.winPortableAmd64)"
                  >Portable executable (amd64)</a>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.winPortableArm64)"
                  >Portable executable (arm64)</a>
                </li>
              </ul>
            </div>

            <div>
              <h3 class="flex items-center gap-2 text-lg font-medium">
                <Radio :size="20" class="text-zinc-400" />
                Linux (desktop)
              </h3>
              <p class="mt-2 text-sm text-zinc-400">Debian and Ubuntu via our APT repository:</p>
              <pre
                class="mt-3 overflow-x-auto rounded-lg border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-300"
              ><code>{{ aptSnippet }}</code></pre>
              <ul class="mt-4 space-y-2 text-sm">
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.debAmd64)"
                  >.deb package (amd64)</a>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.debArm64)"
                  >.deb package (arm64)</a>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.linuxAmd64)"
                  >Binary (amd64)</a>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.linuxArm64)"
                  >Binary (arm64)</a>
                </li>
              </ul>
              <p class="mt-3 text-sm text-zinc-500">
                For raw binaries on other distros, install libgtk-3-0, libwebkit2gtk-4.1-0, and
                libasound2, then run the binary to install into ~/.local.
              </p>
            </div>

            <div>
              <h3 class="flex items-center gap-2 text-lg font-medium">
                <Server :size="20" class="text-zinc-400" />
                Linux (headless)
              </h3>
              <p class="mt-2 text-sm text-zinc-400">
                No GUI or tray. Ideal for servers and automation.
              </p>
              <ul class="mt-4 space-y-2 text-sm">
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.headlessAmd64)"
                  >Binary (amd64)</a>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.headlessArm64)"
                  >Binary (arm64)</a>
                </li>
              </ul>
              <pre
                class="mt-4 overflow-x-auto rounded-lg border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-300"
              ><code>{{ headlessSnippet }}</code></pre>
            </div>

            <div>
              <h3 class="flex items-center gap-2 text-lg font-medium">
                <Smartphone :size="20" class="text-zinc-400" />
                Android
              </h3>
              <ul class="mt-4 space-y-2 text-sm">
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="site.playStoreUrl"
                    target="_blank"
                    rel="noopener noreferrer"
                  >Google Play</a>
                  <span class="text-zinc-500"> (recommended)</span>
                </li>
                <li>
                  <a
                    class="text-sky-400 hover:underline"
                    :href="artifactUrl(artifacts.androidApk)"
                  >APK download</a>
                  <span class="text-zinc-500">
                    - for devices without Play Store (v{{ site.version }})
                  </span>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-zinc-800/60">
      <div
        class="mx-auto flex max-w-5xl flex-col items-center justify-between gap-4 px-4 py-8 sm:flex-row sm:px-6 text-sm text-zinc-500"
      >
        <p>
          Run your own stream with
          <a
            href="https://github.com/jdbnet/gostream"
            class="text-zinc-400 hover:text-zinc-200"
            target="_blank"
            rel="noopener noreferrer"
          >GoStream</a>.
        </p>
        <a
          :href="site.githubUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-2 text-zinc-400 hover:text-zinc-200"
        >
          <GithubIcon :size="18" />
          {{ site.githubRepo }}
        </a>
      </div>
    </footer>
  </div>
</template>
