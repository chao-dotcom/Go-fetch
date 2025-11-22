import { useState, useEffect } from 'react';
import { apiClient } from '../api/client';
import { QueueStats } from '../types';

export function useQueueStats(autoRefresh = true, interval = 5000) {
  const [stats, setStats] = useState<QueueStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchStats = async () => {
    try {
      setLoading(true);
      const data = await apiClient.getQueueStats();
      setStats(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();

    if (autoRefresh) {
      const timer = setInterval(fetchStats, interval);
      return () => clearInterval(timer);
    }
  }, [autoRefresh, interval]);

  return { stats, loading, error, refetch: fetchStats };
}

