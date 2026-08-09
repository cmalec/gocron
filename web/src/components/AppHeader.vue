<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useJobs } from '../stores/useJobs';

const router = useRouter();
const { search, busy, loading, runAll } = useJobs();
const emit = defineEmits<{ newJob: [] }>();
</script>

<template>
  <header class="sticky top-0 z-30 border-b border-base-300 bg-base-100/90 backdrop-blur">
    <div class="container flex h-14 items-center gap-3">
      <button v-if="$route.name !== 'dashboard'" @click="router.push('/')" class="btn btn-ghost btn-sm btn-circle" title="Back">
        <span class="icon-[fa7-solid--chevron-left] size-4"></span>
      </button>
      <router-link to="/" class="flex items-center gap-2 font-bold text-lg tracking-tight">
        <img src="/static/logo.webp" class="h-7 w-7" alt="GoCron" />
        <span>GoCron</span>
      </router-link>

      <div v-if="$route.name === 'dashboard'" class="relative ml-4 hidden sm:block">
        <span class="icon-[fa7-solid--magnifying-glass] absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-base-content/40"></span>
        <input v-model="search" type="text" placeholder="Filter jobs…" class="input input-sm input-bordered w-56 pl-8" />
      </div>

      <div class="ml-auto flex items-center gap-1.5">
        <button v-if="$route.name === 'dashboard'" @click="emit('newJob')" class="btn btn-sm btn-soft" title="Add a new job">
          <span class="icon-[fa7-solid--plus] size-3.5"></span>
          <span class="hidden md:inline">New job</span>
        </button>
        <button v-if="$route.name === 'dashboard'" @click="runAll" :disabled="busy" class="btn btn-primary btn-sm" title="Run all enabled jobs">
          <span v-if="!busy || loading" class="icon-[fa7-solid--play] size-3.5"></span>
          <span v-else class="loading loading-spinner loading-xs"></span>
          <span class="hidden md:inline">Run all</span>
        </button>
        <button @click="router.push('/activity')" class="btn btn-ghost btn-sm btn-circle" title="Activity">
          <span class="icon-[fa7-solid--clock-rotate-left] size-4"></span>
        </button>
        <button @click="router.push('/commands')" class="btn btn-ghost btn-sm btn-circle" title="Terminal">
          <span class="icon-[fa7-solid--terminal] size-4"></span>
        </button>
        <a href="/api/docs" class="btn btn-ghost btn-sm btn-circle" title="API docs">
          <span class="icon-[simple-icons--openapiinitiative] size-4"></span>
        </a>
      </div>
    </div>
  </header>
</template>
