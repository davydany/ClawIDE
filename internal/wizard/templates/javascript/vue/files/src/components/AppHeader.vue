<script setup lang="ts">
import { RouterLink } from 'vue-router';
</script>

<template>
  <nav class="border-b border-gray-200 bg-white">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="flex h-16 items-center justify-between">
        <RouterLink to="/" class="text-xl font-semibold text-gray-900">
          App
        </RouterLink>
      </div>
    </div>
  </nav>
</template>
