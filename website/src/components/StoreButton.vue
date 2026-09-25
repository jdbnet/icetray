<script setup lang="ts">
defineProps<{
  href: string
}>()
</script>

<template>
  <a
    :href="href"
    target="_blank"
    rel="noopener noreferrer"
    class="inline-block leading-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500"
  >
    <img
      src="/brands/google-play-badge.png"
      alt="Get it on Google Play"
      class="block h-12 w-auto sm:h-14"
      width="646"
      height="250"
      loading="lazy"
      decoding="async"
    />
  </a>
</template>
