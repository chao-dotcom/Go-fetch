export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/v1';

export const POLLING_INTERVAL = 5000; // 5 seconds

export const JOB_STATUS_COLORS = {
  QUEUED: 'bg-gray-500',
  RUNNING: 'bg-blue-500',
  SUCCEEDED: 'bg-green-500',
  FAILED: 'bg-red-500',
  RETRYING: 'bg-yellow-500',
  CANCELLED: 'bg-gray-400',
} as const;

export const WORKER_STATUS_COLORS = {
  active: 'bg-green-500',
  idle: 'bg-yellow-500',
  dead: 'bg-red-500',
  draining: 'bg-orange-500',
  starting: 'bg-blue-500',
  busy: 'bg-purple-500',
} as const;

