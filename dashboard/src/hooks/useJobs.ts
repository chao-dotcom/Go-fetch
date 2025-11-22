import { useState, useEffect } from 'react';
import { apiClient } from '../api/client';
import { Job } from '../types';

export function useJobs(autoRefresh = true, interval = 5000) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchJobs = async () => {
    try {
      setLoading(true);
      // Fetch jobs from different statuses to get a better mix
      // This ensures we see both queued and completed jobs
      const [allJobs, queuedJobs, succeededJobs, runningJobs] = await Promise.all([
        apiClient.getJobs({ limit: 400 }), // Recent jobs
        apiClient.getJobs({ status: 'QUEUED', limit: 300 }),
        apiClient.getJobs({ status: 'SUCCEEDED', limit: 200 }),
        apiClient.getJobs({ status: 'RUNNING', limit: 100 }),
      ]);

      // Combine and deduplicate by job ID
      const jobMap = new Map<string, Job>();
      
      // Add jobs in priority order: running > queued > succeeded > all
      [...runningJobs.jobs, ...queuedJobs.jobs, ...succeededJobs.jobs, ...allJobs.jobs].forEach(job => {
        if (!jobMap.has(job.id)) {
          jobMap.set(job.id, job);
        }
      });

      // Convert map to array and sort by updated_at (most recent first)
      const combinedJobs = Array.from(jobMap.values()).sort((a, b) => {
        const aTime = new Date(a.updated_at || a.created_at).getTime();
        const bTime = new Date(b.updated_at || b.created_at).getTime();
        return bTime - aTime;
      });

      setJobs(combinedJobs);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();

    if (autoRefresh) {
      const timer = setInterval(fetchJobs, interval);
      return () => clearInterval(timer);
    }
  }, [autoRefresh, interval]);

  return { jobs, loading, error, refetch: fetchJobs };
}

