<script setup lang="ts">
import { RouterView } from 'vue-router';
import AppHeader from './components/AppHeader.vue';
import JobFormDialog from './components/JobFormDialog.vue';
import { useJobs } from './stores/useJobs';
import { useEventSource } from '@vueuse/core';
import { onMounted, ref, watch } from 'vue';
import { BackendURL } from './main';

const { parseEventInfo, fetchJobs, fetchAllHeatmaps } = useJobs();
const showNewJob = ref(false);

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

async function onDialogClose(saved: boolean) {
  showNewJob.value = false;
  if (saved) {
    await fetchJobs();
    fetchAllHeatmaps(90);
  }
}
</script>

<template>
  <div class="min-h-screen bg-base-200/50">
    <AppHeader @new-job="showNewJob = true" />
    <main class="container py-6">
      <RouterView v-slot="{ Component }">
        <Transition mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <JobFormDialog v-if="showNewJob" :job="null" @close="onDialogClose" />
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
