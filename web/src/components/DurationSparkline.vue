<script setup lang="ts">
import { computed } from 'vue';
import type { RunView } from '../client/types.gen';
import { RunStatus } from '../status';

const props = withDefaults(defineProps<{ runs: RunView[]; height?: number }>(), { height: 40 });

const bars = computed(() => {
  const list = (props.runs ?? []).filter((r) => r.status_id !== RunStatus.Running).slice(-24);
  const durations = list.map((r) => {
    // duration string like "1m30s" — recompute from unix if available is better, but RunView has no end unix; parse string
    return parseDuration(r.duration);
  });
  const max = Math.max(...durations, 1);
  return list.map((r, i) => ({
    id: r.id,
    pct: Math.max(6, Math.round((durations[i] / max) * 100)),
    cls: r.status_id === RunStatus.Finished ? 'fill-success/70' : r.status_id === RunStatus.Canceled ? 'fill-warning/70' : 'fill-error/70',
    tooltip: `${r.start_time} — ${r.duration}`,
  }));
});

function parseDuration(d: string): number {
  if (!d) return 0;
  let seconds = 0;
  const h = /(\d+)h/.exec(d);
  const m = /(\d+)m/.exec(d);
  const s = /(\d+)s/.exec(d);
  if (h) seconds += parseInt(h[1]) * 3600;
  if (m) seconds += parseInt(m[1]) * 60;
  if (s) seconds += parseInt(s[1]);
  return seconds || 1;
}
</script>

<template>
  <div v-if="bars.length" class="flex items-end gap-[2px]" :style="{ height: height + 'px' }">
    <svg v-for="bar in bars" :key="bar.id" width="6" :height="height" class="shrink-0">
      <rect :y="height - (bar.pct / 100) * height" width="6" :height="(bar.pct / 100) * height" :class="bar.cls" rx="1.5">
        <title>{{ bar.tooltip }}</title>
      </rect>
    </svg>
  </div>
  <div v-else class="text-xs text-base-content/40" :style="{ height: height + 'px' }">No run data</div>
</template>
