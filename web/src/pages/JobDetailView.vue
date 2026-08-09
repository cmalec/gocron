<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useJobs } from '../stores/useJobs';
import { statusLabel, statusBadgeClass, timeAgo, timeUntil } from '../status';
import { GetColor } from '../severity';
import ActivityHeatmap from '../components/ActivityHeatmap.vue';
import DurationSparkline from '../components/DurationSparkline.vue';
import JobFormDialog from '../components/JobFormDialog.vue';
import type { RunView } from '../client/types.gen';

const route = useRoute();
const router = useRouter();
const { getJob, fetchJobs, fetchRuns, fetchRunLogs, fetchHeatmap, heatmaps, busy, pause, resume, runJob, deleteJob } = useJobs();

const showEdit = ref(false);
const showDeleteConfirm = ref(false);
const deleting = ref(false);

async function onEditClose(saved: boolean) {
  showEdit.value = false;
  if (saved) await fetchJobs();
}

async function confirmDelete() {
  if (!job.value) return;
  deleting.value = true;
  await deleteJob(job.value.slug);
  deleting.value = false;
  showDeleteConfirm.value = false;
  await fetchJobs();
  router.push('/');
}

const slug = computed(() => String(route.params.id ?? ''));
const job = computed(() => getJob(slug.value));

const runs = ref<RunView[]>([]);
const expandedRun = ref<number | null>(null);
const runLogs = ref<RunView | null>(null);
const loadingLogs = ref(false);
const logFilter = ref<'all' | 'errors'>('all');
const autoRefresh = ref(true);

const heatmapDays = ref(180);

async function loadRuns() {
  if (!slug.value) return;
  const list = await fetchRuns(slug.value, 30, false);
  runs.value = [...list].reverse();
}

async function toggleRun(run: RunView) {
  if (expandedRun.value === run.id) {
    expandedRun.value = null;
    runLogs.value = null;
    return;
  }
  expandedRun.value = run.id;
  loadingLogs.value = true;
  const withLogs = await fetchRunLogs(slug.value, 30);
  runLogs.value = withLogs.find((r) => r.id === run.id) ?? null;
  loadingLogs.value = false;
}

const filteredLogs = computed(() => {
  const logs = runLogs.value?.logs ?? [];
  if (logFilter.value === 'errors') return logs.filter((l) => l.severity_id >= 3);
  return logs;
});

onMounted(async () => {
  await loadRuns();
  fetchHeatmap(slug.value, heatmapDays.value);
});

watch(heatmapDays, (d) => fetchHeatmap(slug.value, d));

// refresh runs when a run event arrives for this job
watch(
  () => job.value?.runs?.length,
  () => {
    if (autoRefresh.value) loadRuns();
  }
);
</script>

<template>
  <div v-if="job" class="space-y-6">
    <!-- Header -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold">{{ job.name }}</h1>
        <div class="mt-1 flex flex-wrap items-center gap-3 text-sm text-base-content/60">
          <span class="font-mono bg-base-100 border border-base-300 rounded px-2 py-0.5">{{ job.disable_cron ? 'manual only' : job.cron }}</span>
          <span v-if="job.disabled" class="badge badge-warning">paused</span>
          <span v-if="!job.disable_cron && !job.disabled" class="text-base-content/50">next run {{ timeUntil(job.next_run_unix) }} · {{ job.next_run }}</span>
        </div>
      </div>
      <div class="flex gap-2">
        <button class="btn btn-sm btn-ghost" @click="showEdit = true" title="Edit job">
          <span class="icon-[fa7-solid--pen] size-3.5"></span>
          Edit
        </button>
        <button class="btn btn-sm btn-ghost text-error" @click="showDeleteConfirm = true" title="Delete job">
          <span class="icon-[fa7-solid--trash] size-3.5"></span>
        </button>
        <button class="btn btn-sm" :class="job.disabled ? 'btn-success' : 'btn-warning btn-soft'" @click="job.disabled ? resume(job.slug) : pause(job.slug)">
          <span :class="job.disabled ? 'icon-[fa7-solid--play]' : 'icon-[fa7-solid--pause]'" class="size-3.5"></span>
          {{ job.disabled ? 'Resume' : 'Pause' }}
        </button>
        <button class="btn btn-sm btn-primary" :disabled="busy" @click="runJob(job.name)">
          <span class="icon-[fa7-solid--bolt] size-3.5"></span>
          Run now
        </button>
      </div>
    </div>

    <JobFormDialog v-if="showEdit" :job="job" @close="onEditClose" />

    <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="showDeleteConfirm = false">
      <div class="card bg-base-100 w-full max-w-sm shadow-xl">
        <div class="card-body p-6">
          <h2 class="card-title">Delete job?</h2>
          <p class="text-sm text-base-content/70">
            <strong>{{ job.name }}</strong> will be removed from the config file. Its run history stays in the database until cleaned up.
          </p>
          <div class="card-actions justify-end mt-2">
            <button class="btn btn-ghost btn-sm" @click="showDeleteConfirm = false">Cancel</button>
            <button class="btn btn-error btn-sm" :disabled="deleting" @click="confirmDelete">
              <span v-if="deleting" class="loading loading-spinner loading-xs"></span>
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Activity -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-5 space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="font-semibold">Activity</h2>
          <div class="join">
            <button
              v-for="d in [30, 90, 180, 365]"
              :key="d"
              class="btn btn-xs join-item"
              :class="heatmapDays === d ? 'btn-active btn-primary' : 'btn-ghost'"
              @click="heatmapDays = d"
            >
              {{ d }}d
            </button>
          </div>
        </div>
        <ActivityHeatmap :data="heatmaps.get(job.slug) ?? []" :days="heatmapDays" :cell-size="13" :cell-gap="3" />
        <div class="flex items-center gap-4 text-xs text-base-content/50">
          <span class="flex items-center gap-1.5"><span class="inline-block size-3 rounded-sm bg-success"></span> all succeeded</span>
          <span class="flex items-center gap-1.5"><span class="inline-block size-3 rounded-sm bg-error"></span> failures</span>
          <span class="flex items-center gap-1.5"><span class="inline-block size-3 rounded-sm bg-base-300/60"></span> no runs</span>
          <span class="ml-auto">intensity = run count / failure ratio</span>
        </div>
      </div>
    </div>

    <!-- Duration sparkline -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-5">
        <h2 class="font-semibold mb-3">Run durations</h2>
        <DurationSparkline :runs="runs" :height="48" />
      </div>
    </div>

    <!-- Runs -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-5">
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold">Recent runs</h2>
          <label class="flex items-center gap-2 text-xs text-base-content/60 cursor-pointer">
            <input type="checkbox" v-model="autoRefresh" class="checkbox checkbox-xs" />
            auto-refresh
          </label>
        </div>

        <div v-if="runs.length === 0" class="text-sm text-base-content/50 py-6 text-center">No runs recorded yet.</div>

        <div class="divide-y divide-base-300/70">
          <div v-for="run in runs" :key="run.id">
            <button class="w-full flex items-center gap-4 py-2.5 px-2 hover:bg-base-200/60 rounded text-left" @click="toggleRun(run)">
              <span class="badge badge-sm w-20 justify-center" :class="statusBadgeClass(run.status_id)">{{ statusLabel(run.status_id) }}</span>
              <span class="text-sm flex-1">{{ run.start_time }}</span>
              <span class="text-xs text-base-content/50 hidden sm:inline">{{ timeAgo(run.start_time_unix) }}</span>
              <span class="text-xs font-mono text-base-content/60 w-16 text-right">{{ run.duration || '—' }}</span>
              <span
                class="icon-[fa7-solid--chevron-down] size-3 text-base-content/40 transition-transform"
                :class="expandedRun === run.id ? 'rotate-180' : ''"
              ></span>
            </button>

            <div v-if="expandedRun === run.id" class="px-2 pb-3">
              <div v-if="loadingLogs" class="flex justify-center py-4"><span class="loading loading-spinner loading-sm"></span></div>
              <template v-else>
                <div class="flex justify-end gap-1 mb-1.5">
                  <button class="btn btn-xs" :class="logFilter === 'all' ? 'btn-active' : 'btn-ghost'" @click="logFilter = 'all'">All</button>
                  <button class="btn btn-xs" :class="logFilter === 'errors' ? 'btn-active' : 'btn-ghost'" @click="logFilter = 'errors'">
                    Warnings & errors
                  </button>
                </div>
                <div class="rounded-lg bg-neutral text-neutral-content font-mono text-xs overflow-x-auto max-h-96 overflow-y-auto">
                  <div v-if="filteredLogs.length === 0" class="p-4 text-neutral-content/50">No logs for this run.</div>
                  <div v-for="(log, i) in filteredLogs" :key="i" class="flex gap-3 px-3 py-1 hover:bg-white/5">
                    <span class="text-neutral-content/40 shrink-0 select-none">{{ log.created_at_time }}</span>
                    <span :class="GetColor(log.severity_id)" class="whitespace-pre-wrap break-all">{{ log.message }}</span>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="text-center py-16 text-base-content/50">
    <span class="loading loading-spinner loading-lg"></span>
  </div>
</template>
