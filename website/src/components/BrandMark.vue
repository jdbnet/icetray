<script setup lang="ts">
withDefaults(
  defineProps<{
    src: string
    alt?: string
    height?: number
    imgClass?: string
  }>(),
  { alt: '', height: 24, imgClass: '' },
)
</script>

<template>
  <img
    :src="src"
    :alt="alt"
    class="shrink-0 object-contain"
    :class="imgClass"
    :style="{ height: `${height}px`, width: 'auto' }"
    loading="lazy"
    decoding="async"
  />
</template>
