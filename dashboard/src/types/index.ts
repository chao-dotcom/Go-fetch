export type JobStatus = 'QUEUED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'RETRYING' | 'CANCELLED';

export interface Job {
  id: string;
  type: string;
  payload: Record<string, any>;
  status: JobStatus;
  priority: number;
  attempts: number;
  max_attempts: number;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
  result?: string;
  error?: string;
  worker_id?: string;
}

export interface Worker {
  id: string;
  hostname: string;
  status: 'active' | 'idle' | 'dead' | 'draining' | 'starting' | 'busy';
  pool_size: number;
  active_jobs: number;
  total_jobs?: number;  // API returns total_jobs
  total_processed?: number;  // May not be available
  total_failed?: number;  // May not be available
  started_at: string;
  last_heartbeat: string;
}

export interface QueueStats {
  name: string;
  depth: number;
  broker_depth: number;
  pending_count: number;
  processing_rate: number;
  avg_latency_ms: number;
}

export interface JobLog {
  id: string;
  job_id: string;
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  created_at: string;
  metadata?: Record<string, any>;
}

