export const RunStatus = {
  Running: 1,
  Stopped: 2,
  Finished: 3,
  Canceled: 4,
} as const;

export function statusLabel(statusId: number): string {
  switch (statusId) {
    case RunStatus.Running:
      return 'Running';
    case RunStatus.Stopped:
      return 'Failed';
    case RunStatus.Finished:
      return 'Success';
    case RunStatus.Canceled:
      return 'Canceled';
    default:
      return 'Unknown';
  }
}

export function statusBadgeClass(statusId: number): string {
  switch (statusId) {
    case RunStatus.Running:
      return 'badge-info';
    case RunStatus.Stopped:
      return 'badge-error';
    case RunStatus.Finished:
      return 'badge-success';
    case RunStatus.Canceled:
      return 'badge-warning';
    default:
      return 'badge-ghost';
  }
}

export function statusDotClass(statusId: number): string {
  switch (statusId) {
    case RunStatus.Running:
      return 'bg-info animate-pulse';
    case RunStatus.Stopped:
      return 'bg-error';
    case RunStatus.Finished:
      return 'bg-success';
    case RunStatus.Canceled:
      return 'bg-warning';
    default:
      return 'bg-base-300';
  }
}

export function timeAgo(unixMs: number): string {
  if (!unixMs) return '—';
  const seconds = Math.floor((Date.now() - unixMs) / 1000);
  if (seconds < 0) return timeUntil(unixMs);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

export function timeUntil(unixMs: number): string {
  if (!unixMs) return '—';
  const seconds = Math.floor((unixMs - Date.now()) / 1000);
  if (seconds < 0) return timeAgo(unixMs);
  if (seconds < 60) return `in ${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `in ${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `in ${hours}h`;
  const days = Math.floor(hours / 24);
  return `in ${days}d`;
}
