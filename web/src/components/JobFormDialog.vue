<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { JobView, JobConfigInput } from '../client/types.gen';
import { useJobs } from '../stores/useJobs';

const props = defineProps<{ job?: JobView | null }>();
const emit = defineEmits<{ close: [saved: boolean] }>();

const { createJob, updateJob } = useJobs();

const name = ref('');
const cron = ref('');
const manualOnly = ref(false);
const commandsText = ref('');
const error = ref<string | null>(null);
const saving = ref(false);

const isEdit = computed(() => !!props.job);
const title = computed(() => (isEdit.value ? 'Edit job' : 'New job'));

watch(
  () => props.job,
  (job) => {
    error.value = null;
    if (job) {
      name.value = job.name;
      cron.value = job.cron;
      manualOnly.value = job.disable_cron;
      commandsText.value = '';
    } else {
      name.value = '';
      cron.value = '0 3 * * *';
      manualOnly.value = false;
      commandsText.value = '';
    }
  },
  { immediate: true }
);

const cronPresets = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6h', value: '0 */6 * * *' },
  { label: 'Daily 3am', value: '0 3 * * *' },
  { label: 'Weekly Sun', value: '0 3 * * 0' },
];

async function save() {
  error.value = null;
  const commands = commandsText.value
    .split('\n')
    .map((c) => c.trim())
    .filter((c) => c.length > 0);

  if (!isEdit.value && !name.value.trim()) {
    error.value = 'Name is required';
    return;
  }
  if (!manualOnly.value && !cron.value.trim()) {
    error.value = 'Cron expression is required (or enable manual-only)';
    return;
  }
  if (commands.length === 0) {
    error.value = 'At least one command is required';
    return;
  }

  const input: JobConfigInput = {
    name: name.value.trim(),
    cron: manualOnly.value ? '' : cron.value.trim(),
    disable_cron: manualOnly.value,
    commands,
  };

  saving.value = true;
  const err = isEdit.value ? await updateJob(props.job!.slug, input) : await createJob(input);
  saving.value = false;

  if (err) {
    error.value = err;
    return;
  }
  emit('close', true);
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close', false)">
    <div class="card bg-base-100 w-full max-w-lg shadow-xl">
      <div class="card-body p-6 space-y-4">
        <h2 class="card-title">{{ title }}</h2>

        <label class="form-control">
          <span class="label-text mb-1">Name</span>
          <input v-model="name" type="text" class="input input-bordered input-sm w-full" :disabled="isEdit" placeholder="My backup job" />
        </label>

        <div class="form-control">
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="manualOnly" type="checkbox" class="checkbox checkbox-sm" />
            <span class="label-text">Manual only (no schedule)</span>
          </label>
        </div>

        <label v-if="!manualOnly" class="form-control">
          <span class="label-text mb-1">Cron expression</span>
          <input v-model="cron" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="0 3 * * *" />
          <div class="mt-2 flex flex-wrap gap-1.5">
            <button v-for="p in cronPresets" :key="p.value" type="button" class="btn btn-xs btn-ghost border border-base-300" @click="cron = p.value">
              {{ p.label }}
            </button>
          </div>
          <span class="label-text-alt text-base-content/50 mt-1">minute hour day-of-month month day-of-week</span>
        </label>

        <label class="form-control">
          <span class="label-text mb-1">Commands <span class="text-base-content/40">(one per line)</span></span>
          <textarea
            v-model="commandsText"
            class="textarea textarea-bordered textarea-sm w-full font-mono h-28"
            :placeholder="isEdit ? 'Leave empty to keep existing commands' : 'echo &quot;hello&quot;'"
          ></textarea>
        </label>

        <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>

        <div class="card-actions justify-end">
          <button class="btn btn-ghost btn-sm" @click="emit('close', false)">Cancel</button>
          <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">
            <span v-if="saving" class="loading loading-spinner loading-xs"></span>
            {{ isEdit ? 'Save changes' : 'Create job' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
