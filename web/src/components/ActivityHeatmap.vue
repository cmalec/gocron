<script setup lang="ts">
import { computed } from 'vue';
import type { DayStat } from '../client/types.gen';

const props = withDefaults(
  defineProps<{
    data: DayStat[];
    days?: number;
    cellSize?: number;
    cellGap?: number;
  }>(),
  { days: 90, cellSize: 12, cellGap: 3 }
);

type Cell = {
  date: string;
  level: number;
  cls: string;
  tooltip: string;
};

const statsByDay = computed(() => {
  const map = new Map<string, DayStat>();
  for (const d of props.data ?? []) map.set(d.day, d);
  return map;
});

function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function cellFor(dateStr: string): Cell {
  const stat = statsByDay.value.get(dateStr);
  if (!stat || stat.total === 0) {
    return { date: dateStr, level: 0, cls: 'fill-base-300/60', tooltip: `${dateStr}: no runs` };
  }
  if (stat.failed === 0) {
    const level = Math.min(4, 1 + Math.floor(Math.log2(stat.total)));
    return {
      date: dateStr,
      level,
      cls: ['', 'fill-success/40', 'fill-success/60', 'fill-success/80', 'fill-success'][level],
      tooltip: `${dateStr}: ${stat.succeeded} succeeded`,
    };
  }
  const ratio = stat.failed / stat.total;
  const level = ratio >= 0.75 ? 4 : ratio >= 0.5 ? 3 : ratio >= 0.25 ? 2 : 1;
  return {
    date: dateStr,
    level,
    cls: ['', 'fill-error/40', 'fill-error/60', 'fill-error/80', 'fill-error'][level],
    tooltip: `${dateStr}: ${stat.failed}/${stat.total} failed`,
  };
}

const weeks = computed(() => {
  const result: { month: string | null; cells: Cell[] }[] = [];
  const today = new Date();
  const start = new Date();
  start.setDate(today.getDate() - (props.days - 1));
  // align to Sunday
  start.setDate(start.getDate() - start.getDay());

  let currentWeek: Cell[] = [];
  let lastMonth = -1;
  let weekMonth: string | null = null;

  const cursor = new Date(start);
  while (cursor <= today) {
    if (cursor.getDay() === 0 && currentWeek.length > 0) {
      result.push({ month: weekMonth, cells: currentWeek });
      currentWeek = [];
      weekMonth = null;
    }
    if (weekMonth === null && cursor.getMonth() !== lastMonth) {
      lastMonth = cursor.getMonth();
      weekMonth = cursor.toLocaleString('default', { month: 'short' });
    }
    currentWeek.push(cellFor(toISODate(cursor)));
    cursor.setDate(cursor.getDate() + 1);
  }
  if (currentWeek.length > 0) result.push({ month: weekMonth, cells: currentWeek });
  return result;
});

const dayLabels = ['', 'Mon', '', 'Wed', '', 'Fri', ''];
</script>

<template>
  <div class="flex gap-2">
    <div class="flex flex-col justify-between py-0.5 text-[10px] text-base-content/50 select-none" :style="{ gap: cellGap + 'px' }">
      <span v-for="(label, i) in dayLabels" :key="i" :style="{ height: cellSize + 'px', lineHeight: cellSize + 'px' }">{{ label }}</span>
    </div>
    <div class="flex overflow-x-auto" :style="{ gap: cellGap + 'px' }">
      <div v-for="(week, wi) in weeks" :key="wi" class="flex flex-col" :style="{ gap: cellGap + 'px' }">
        <div class="text-[10px] text-base-content/50 select-none" :style="{ height: '14px' }">{{ week.month ?? '' }}</div>
        <svg v-for="cell in week.cells" :key="cell.date" :width="cellSize" :height="cellSize" class="rounded-[3px]">
          <rect :width="cellSize" :height="cellSize" :class="cell.cls" rx="2.5">
            <title>{{ cell.tooltip }}</title>
          </rect>
        </svg>
      </div>
    </div>
  </div>
</template>
