import { createRouter, createWebHistory } from 'vue-router';

import DashboardView from './pages/DashboardView.vue';
import JobDetailView from './pages/JobDetailView.vue';
import ActivityView from './pages/ActivityView.vue';
import CommandView from './pages/CommandView.vue';

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView, meta: { title: 'GoCron' } },
  { path: '/jobs/:id', name: 'jobDetail', component: JobDetailView, meta: { title: 'Job' } },
  { path: '/activity', name: 'activity', component: ActivityView, meta: { title: 'Activity' } },
  { path: '/commands', name: 'commandView', component: CommandView, meta: { title: 'Command' } },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, _, next) => {
  document.title = `${to.meta.title}`;
  next();
});

export default router;
