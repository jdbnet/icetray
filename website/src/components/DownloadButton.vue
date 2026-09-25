<script setup lang="ts">
import { Download } from '@lucide/vue'

withDefaults(
  defineProps<{
    href: string
    label: string
    external?: boolean
    variant?: 'primary' | 'secondary'
  }>(),
  { external: true, variant: 'secondary' },
)
</script>

<template>
  <a
    :href="href"
    :target="external ? '_blank' : undefined"
    :rel="external ? 'noopener noreferrer' : undefined"
    class="inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors"
    :class="
      variant === 'primary'
        ? 'bg-sky-600 text-white hover:bg-sky-500 shadow-sm shadow-sky-900/30'
        : 'border border-zinc-700 bg-zinc-950/50 text-zinc-100 hover:border-zinc-500 hover:bg-zinc-900'
    "
  >
    <Download :size="16" class="shrink-0 opacity-90" />
    <span>{{ label }}</span>
  </a>
</template>
