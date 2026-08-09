<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useJobs } from '../stores/useJobs';
import { statusLabel, statusBadgeClass, timeAgo } from '../status';
import type { ActivityRun } from '../client/types.gen';

const router = useRouter();
const { fetchActivity } = useJobs();

const activity = ref<ActivityRun[]>([]);
const loading = ref(true);

onMounted(async () => {
  activity.value = await fetchActivity(50);
  loading.value = false;
});
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-xl font-bold">Activity</h1>

    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <div v-else-if="activity.length === 0" class="text-center py-16 text-base-content/50">
      <span class="icon-[fa7-solid--clock-rotate-left] size-10 mb-3 block mx-auto opacity-40"></span>
      No runs recorded yet.
    </div>

    <div v-else class="card bg-base-100 shadow-sm">
      <div class="divide-y divide-base-300/70">
        <button
          v-for="run in activity"
          :key="run.id"
          class="w-full flex items-center gap-4 px-4 py-3 hover:bg-base-200/60 text-left"
          @click="router.push('/jobs/' + run.job_slug)"
        >
          <span class="badge badge-sm w-20 justify-center shrink-0" :class="statusBadgeClass(run.status_id)">
            {{ statusLabel(run.status_id) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="font-medium truncate">{{ run.job_name }}</div>
            <div class="text-xs text-base-content/50">{{ run.start_time }}</div>
          </div>
          <span class="text-xs text-base-content/50 hidden sm:inline shrink-0">{{ timeAgo(run.start_time_unix) }}</span>
          <span class="text-xs font-mono text-base-content/60 w-14 text-right shrink-0">{{ run.duration || '—' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
