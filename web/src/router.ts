import { createRouter, createWebHistory, isNavigationFailure, NavigationFailureType } from 'vue-router';

import DashboardView from './pages/DashboardView.vue';
import JobDetailView from './pages/JobDetailView.vue';
import ActivityView from './pages/ActivityView.vue';
import CommandView from './pages/CommandView.vue';
import LoginView from './pages/LoginView.vue';
import { useAuth } from './stores/useAuth';

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView, meta: { title: 'GoCron' } },
  { path: '/jobs/:id', name: 'jobDetail', component: JobDetailView, meta: { title: 'Job' } },
  { path: '/activity', name: 'activity', component: ActivityView, meta: { title: 'Activity' } },
  { path: '/commands', name: 'commandView', component: CommandView, meta: { title: 'Command' } },
  { path: '/login', name: 'loginView', component: LoginView, meta: { title: 'GoCron - Sign in', public: true } },
];

declare module 'vue-router' {
  interface RouteMeta {
    title: string;
    public?: boolean;
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  document.title = `${to.meta.title}`;

  if (to.meta.public) {
    return true;
  }

  const auth = useAuth();
  if (!auth.ready.value) await auth.fetchCurrentUser();

  if (auth.authEnabled.value && !auth.authenticated.value) {
    return { name: 'loginView', query: { redirect: to.fullPath } };
  }
});

router.onError((error) => {
  if (!isNavigationFailure(error, NavigationFailureType.aborted | NavigationFailureType.cancelled)) {
    throw error;
  }
});

export default router;
