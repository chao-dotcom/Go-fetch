import { clsx } from 'clsx';
import { JobStatus } from '../types';
import { JOB_STATUS_COLORS, WORKER_STATUS_COLORS } from '../utils/constants';

interface StatusBadgeProps {
  status: JobStatus | 'active' | 'idle' | 'dead' | 'draining' | 'starting' | 'busy';
  className?: string;
}

export default function StatusBadge({ status, className }: StatusBadgeProps) {
  const isJobStatus = ['QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED', 'RETRYING', 'CANCELLED'].includes(status);
  const colorClass = isJobStatus
    ? JOB_STATUS_COLORS[status as JobStatus]
    : (WORKER_STATUS_COLORS[status as keyof typeof WORKER_STATUS_COLORS] || 'bg-gray-500');

  return (
    <span
      className={clsx(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        colorClass,
        'text-white',
        className
      )}
    >
      {status}
    </span>
  );
}

