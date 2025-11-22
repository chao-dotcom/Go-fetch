import { useState, useEffect } from 'react';
import { apiClient } from '../api/client';
import { Worker } from '../types';

export function useWorkers(autoRefresh = true, interval = 5000) {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchWorkers = async () => {
    try {
      setLoading(true);
      const data = await apiClient.getWorkers();
      setWorkers(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkers();

    if (autoRefresh) {
      const timer = setInterval(fetchWorkers, interval);
      return () => clearInterval(timer);
    }
  }, [autoRefresh, interval]);

  return { workers, loading, error, refetch: fetchWorkers };
}

