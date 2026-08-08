<script setup lang="ts">
import { useJobs } from '../stores/useJobs';
import { statusLabel, statusDotClass, timeAgo, timeUntil } from '../status';
import ActivityHeatmap from '../components/ActivityHeatmap.vue';
import { useRouter } from 'vue-router';

const router = useRouter();
const { filteredJobs, stats, heatmaps, idle, busy, pause, resume, runJob } = useJobs();

function lastRun(job: import('../client/types.gen').JobView) {
  const runs = job.runs ?? [];
  return runs.length ? runs[runs.length - 1] : null;
}

async function togglePause(job: import('../client/types.gen').JobView, event: Event) {
  event.stopPropagation();
  if (job.disabled) {
    await resume(job.slug);
  } else {
    await pause(job.slug);
  }
}

async function run(event: Event, name: string) {
  event.stopPropagation();
  await runJob(name);
}
</script>

<template>
  <div class="space-y-6">
    <!-- Stats -->
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="text-xs text-base-content/50 uppercase tracking-wide">Jobs</div>
          <div class="text-2xl font-bold">{{ stats.total }}</div>
        </div>
      </div>
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="text-xs text-base-content/50 uppercase tracking-wide">Active</div>
          <div class="text-2xl font-bold text-success">{{ stats.active }}</div>
        </div>
      </div>
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="text-xs text-base-content/50 uppercase tracking-wide">Paused</div>
          <div class="text-2xl font-bold" :class="stats.paused ? 'text-warning' : ''">{{ stats.paused }}</div>
        </div>
      </div>
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="text-xs text-base-content/50 uppercase tracking-wide">Failed runs (90d)</div>
          <div class="text-2xl font-bold" :class="stats.failedRuns ? 'text-error' : ''">{{ stats.failedRuns }}</div>
        </div>
      </div>
    </div>

    <div v-if="!idle" class="alert alert-info text-sm">
      <span class="loading loading-spinner loading-xs"></span>
      A job is currently running…
    </div>

    <!-- Job cards -->
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="job in filteredJobs"
        :key="job.slug"
        @click="router.push('/jobs/' + job.slug)"
        class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow cursor-pointer border border-transparent hover:border-base-300"
      >
        <div class="card-body p-5 space-y-3">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <h2 class="font-semibold truncate">{{ job.name }}</h2>
              <div class="text-xs text-base-content/50 font-mono mt-0.5">
                <template v-if="job.disable_cron">manual only</template>
                <template v-else>{{ job.cron }}</template>
              </div>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              <span v-if="job.disabled" class="badge badge-warning badge-sm">paused</span>
              <template v-else-if="lastRun(job)">
                <span class="badge badge-sm gap-1.5" :class="statusDotClass(lastRun(job)!.status_id).replace('bg-', 'badge-').replace(' animate-pulse', '')">
                  {{ statusLabel(lastRun(job)!.status_id) }}
                </span>
              </template>
              <span v-else class="badge badge-ghost badge-sm">never run</span>
            </div>
          </div>

          <ActivityHeatmap :data="heatmaps.get(job.slug) ?? []" :days="90" :cell-size="10" :cell-gap="2" />

          <div class="flex items-center justify-between text-xs text-base-content/60">
            <span>
              <template v-if="lastRun(job)">Last: {{ timeAgo(lastRun(job)!.start_time_unix) }}</template>
              <template v-else>No runs yet</template>
            </span>
            <span v-if="!job.disable_cron && !job.disabled">Next: {{ timeUntil(job.next_run_unix) }}</span>
          </div>

          <div class="card-actions justify-end pt-1" @click.stop>
            <button class="btn btn-xs" :class="job.disabled ? 'btn-success btn-soft' : 'btn-warning btn-soft'" @click="togglePause(job, $event)">
              <span :class="job.disabled ? 'icon-[fa7-solid--play]' : 'icon-[fa7-solid--pause]'" class="size-3"></span>
              {{ job.disabled ? 'Resume' : 'Pause' }}
            </button>
            <button class="btn btn-xs btn-primary btn-soft" :disabled="busy" @click="run($event, job.name)">
              <span class="icon-[fa7-solid--bolt] size-3"></span>
              Run now
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="filteredJobs.length === 0" class="text-center py-16 text-base-content/50">
      <span class="icon-[fa7-solid--inbox] size-10 mb-3 block mx-auto opacity-40"></span>
      No jobs match your filter.
    </div>
  </div>
</template>
