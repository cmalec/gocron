<script setup lang="ts">
import { RouterView } from 'vue-router';
import AppHeader from './components/AppHeader.vue';
import { useJobs } from './stores/useJobs';
import { useEventSource } from '@vueuse/core';
import { onMounted, watch } from 'vue';
import { BackendURL } from './main';

const { parseEventInfo, fetchJobs, fetchAllHeatmaps } = useJobs();

onMounted(async () => {
  await fetchJobs();
  fetchAllHeatmaps(90);
});

const { data, close } = useEventSource(BackendURL + '/api/events?stream=status', [], {
  autoReconnect: { delay: 100 },
});
addEventListener('beforeunload', () => {
  close();
});
watch(() => data.value, parseEventInfo);
</script>

<template>
  <div class="min-h-screen bg-base-200/50">
    <AppHeader />
    <main class="container py-6">
      <RouterView v-slot="{ Component }">
        <Transition mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </div>
</template>

<style>
.v-enter-active,
.v-leave-active {
  transition: opacity 0.1s ease-out;
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
}
</style>
