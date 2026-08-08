import { computed, ref } from 'vue';
import { createGlobalState } from '@vueuse/core';
import type { JobView, RunView } from '../client/types.gen';
import { getJobs, getRuns, getHeatmap, pauseJob, resumeJob, postJob, postJobs } from '../client/sdk.gen';
import type { DayStat } from '../client/types.gen';

export type EventInfo = {
  idle: boolean;
  run?: RunView;
  jobs?: JobView[];
};

export const useJobs = createGlobalState(() => {
  const idle = ref(true);
  const jobs = ref<JobView[]>([]);
  const error = ref<string | null>(null);
  const loading = ref(false);
  const search = ref('');
  const heatmaps = ref(new Map<string, DayStat[]>());

  const busy = computed(() => loading.value || !idle.value);

  const filteredJobs = computed(() => {
    const q = search.value.toLowerCase();
    if (!q) return jobs.value;
    return jobs.value.filter((j) => j.name.toLowerCase().includes(q));
  });

  const stats = computed(() => {
    const all = jobs.value;
    const active = all.filter((j) => !j.disabled).length;
    const paused = all.filter((j) => j.disabled).length;
    let failedRuns = 0;
    let totalRuns = 0;
    for (const days of Array.from(heatmaps.value.values())) {
      for (const d of days) {
        failedRuns += d.failed;
        totalRuns += d.total;
      }
    }
    return { total: all.length, active, paused, failedRuns, totalRuns };
  });

  function setJobs(newJobs: JobView[]) {
    jobs.value = [...newJobs].sort((a, b) => a.name.localeCompare(b.name));
  }

  function getJob(slug: string): JobView | undefined {
    return jobs.value.find((j) => j.slug === slug);
  }

  function parseEventInfo(info: string | null): void {
    if (!info) return;
    const parsed: EventInfo = JSON.parse(info);
    idle.value = parsed.idle;

    if (parsed.jobs) {
      setJobs(parsed.jobs);
    }

    if (!parsed.run) return;
    const job = jobs.value.find((j) => j.name === parsed.run!.job_name);
    if (!job) return;
    const runs = [...(job.runs ?? [])];
    const idx = runs.findIndex((r) => r.id === parsed.run!.id);
    if (idx !== -1) {
      runs[idx] = parsed.run;
    } else {
      runs.push(parsed.run);
    }
    job.runs = runs.slice(-20);
  }

  async function fetchJobs() {
    error.value = null;
    loading.value = true;
    try {
      const result = await getJobs();
      if (result.data) setJobs(result.data);
    } catch (err: any) {
      error.value = err.toString();
    } finally {
      loading.value = false;
    }
  }

  async function fetchRuns(slug: string, limit = 20, includeLogs = false): Promise<RunView[]> {
    try {
      const result = await getRuns({ path: { job_name: slug }, query: { limit, include_logs: includeLogs } });
      return result.data ?? [];
    } catch (err: any) {
      error.value = err.toString();
      return [];
    }
  }

  async function fetchRunLogs(slug: string, limit = 5): Promise<RunView[]> {
    return fetchRuns(slug, limit, true);
  }

  async function fetchHeatmap(slug: string, days = 90): Promise<DayStat[]> {
    try {
      const result = await getHeatmap({ path: { job_name: slug }, query: { days } });
      const data = result.data ?? [];
      heatmaps.value.set(slug, data);
      return data;
    } catch {
      return [];
    }
  }

  async function fetchAllHeatmaps(days = 90) {
    await Promise.all(jobs.value.map((j) => fetchHeatmap(j.slug, days)));
  }

  async function pause(slug: string) {
    await pauseJob({ path: { name: slug } });
    await fetchJobs();
  }

  async function resume(slug: string) {
    await resumeJob({ path: { name: slug } });
    await fetchJobs();
  }

  async function runJob(name: string) {
    await postJob({ path: { name } });
  }

  async function runAll() {
    await postJobs();
  }

  return {
    idle,
    jobs,
    search,
    filteredJobs,
    loading,
    busy,
    error,
    stats,
    heatmaps,
    getJob,
    parseEventInfo,
    fetchJobs,
    fetchRuns,
    fetchRunLogs,
    fetchHeatmap,
    fetchAllHeatmaps,
    pause,
    resume,
    runJob,
    runAll,
  };
});
