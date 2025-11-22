import { useState, useEffect, useMemo } from 'react';
import { Job } from '../types';
import { API_BASE_URL } from '../utils/constants';

interface TrendDataPoint {
  time: string;
  succeeded: number;
  failed: number;
  running: number;
  queued: number;
}

export function useJobTrends(jobs: Job[], windowSize: number = 30) {
  const [trendData, setTrendData] = useState<TrendDataPoint[]>([]);
  const [history, setHistory] = useState<Array<{ 
    timestamp: number; 
    stats: { succeeded: number; failed: number; running: number; queued: number };
    deltaStats?: { succeeded: number; failed: number; running: number; queued: number };
  }>>([]);

  // Update history with current job stats every 30 seconds for real-time updates
  useEffect(() => {
    const updateHistory = async () => {
      // Fetch real-time counts from API instead of using sample data
      try {
        const [queuedRes, succeededRes, runningRes, failedRes] = await Promise.all([
          fetch(`${API_BASE_URL}/jobs?status=QUEUED&limit=1`).then(r => r.json()),
          fetch(`${API_BASE_URL}/jobs?status=SUCCEEDED&limit=1`).then(r => r.json()),
          fetch(`${API_BASE_URL}/jobs?status=RUNNING&limit=1`).then(r => r.json()),
          fetch(`${API_BASE_URL}/jobs?status=FAILED&limit=1`).then(r => r.json()),
        ]);

        const currentStats = {
          succeeded: succeededRes.total || 0,
          failed: failedRes.total || 0,
          running: runningRes.total || 0,
          queued: queuedRes.total || 0,
        };

        // Debug log to verify data is being fetched correctly
        console.log('Job Trends - Fetched real-time stats:', currentStats);

        setHistory(prev => {
          const now = Date.now();
          
          // Calculate deltas (instantaneous changes) from previous entry
          let deltaStats = { succeeded: 0, failed: 0, running: 0, queued: 0 };
          if (prev.length > 0) {
            const lastEntry = prev[prev.length - 1];
            const timeDelta = (now - lastEntry.timestamp) / 1000; // seconds
            
            // Calculate rate of change per second, then scale to per minute for display
            deltaStats = {
              succeeded: Math.max(0, (currentStats.succeeded - lastEntry.stats.succeeded) / timeDelta * 60), // per minute
              failed: Math.max(0, (currentStats.failed - lastEntry.stats.failed) / timeDelta * 60),
              running: currentStats.running - lastEntry.stats.running, // instantaneous difference
              queued: currentStats.queued - lastEntry.stats.queued, // instantaneous difference
            };
          }
          
          const newEntry: { 
            timestamp: number; 
            stats: { succeeded: number; failed: number; running: number; queued: number };
            deltaStats: { succeeded: number; failed: number; running: number; queued: number };
          } = { 
            timestamp: now, 
            stats: currentStats,
            deltaStats: deltaStats
          };
          
          const newHistory = [...prev, newEntry];
          // Keep only last windowSize entries, sorted by timestamp
          const sorted = newHistory.sort((a, b) => a.timestamp - b.timestamp);
          return sorted.slice(-windowSize);
        });
      } catch (error) {
        console.error('Failed to fetch job stats:', error);
        // On error, use jobs array as fallback
        const fallbackStats = {
          succeeded: jobs.filter(j => j.status === 'SUCCEEDED').length,
          failed: jobs.filter(j => j.status === 'FAILED').length,
          running: jobs.filter(j => j.status === 'RUNNING').length,
          queued: jobs.filter(j => j.status === 'QUEUED').length,
        };
        setHistory(prev => {
          const now = Date.now();
          const newEntry = { 
            timestamp: now, 
            stats: fallbackStats,
            deltaStats: prev.length > 0 ? {
              succeeded: Math.max(0, (fallbackStats.succeeded - prev[prev.length - 1].stats.succeeded) / ((now - prev[prev.length - 1].timestamp) / 1000) * 60),
              failed: Math.max(0, (fallbackStats.failed - prev[prev.length - 1].stats.failed) / ((now - prev[prev.length - 1].timestamp) / 1000) * 60),
              running: fallbackStats.running - prev[prev.length - 1].stats.running,
              queued: fallbackStats.queued - prev[prev.length - 1].stats.queued,
            } : { succeeded: 0, failed: 0, running: 0, queued: 0 }
          };
          const newHistory = [...prev, newEntry];
          const sorted = newHistory.sort((a, b) => a.timestamp - b.timestamp);
          return sorted.slice(-windowSize);
        });
      }
    };

    // Initial update
    updateHistory();

    // Update every 30 seconds for real-time updates
    const interval = setInterval(updateHistory, 30000);
    return () => clearInterval(interval);
  }, [windowSize]);

  // Generate trend data from history (real-time snapshots)
  const generateTrendData = useMemo(() => {
    // Use history data which contains real-time snapshots
    if (history.length > 0) {
      // Sort by timestamp to ensure correct chronological order
      const sortedHistory = [...history].sort((a, b) => a.timestamp - b.timestamp);
      return sortedHistory.map((entry) => ({
        time: new Date(entry.timestamp).toLocaleTimeString('en-US', { 
          hour: '2-digit', 
          minute: '2-digit',
          second: '2-digit',
          hour12: false 
        }),
        // Use delta stats (instantaneous rate) instead of cumulative totals
        succeeded: entry.deltaStats?.succeeded || 0,
        failed: entry.deltaStats?.failed || 0,
        running: entry.deltaStats?.running || 0,
        queued: entry.deltaStats?.queued || 0,
      }));
    }
    
    // Fallback: if no history yet, use jobs array
    if (jobs.length === 0) {
      return [];
    }

    // Create a single data point from current jobs (temporary until history is populated)
    return [{
      time: new Date().toLocaleTimeString('en-US', { 
        hour: '2-digit', 
        minute: '2-digit',
        second: '2-digit',
        hour12: false 
      }),
      succeeded: jobs.filter(j => j.status === 'SUCCEEDED').length,
      failed: jobs.filter(j => j.status === 'FAILED').length,
      running: jobs.filter(j => j.status === 'RUNNING').length,
      queued: jobs.filter(j => j.status === 'QUEUED').length,
    }];
  }, [history, jobs]);

  useEffect(() => {
    setTrendData(generateTrendData);
  }, [generateTrendData]);

  // Calculate current stats from history (most recent snapshot)
  const currentStats = useMemo(() => {
    if (history.length > 0) {
      const latest = history[history.length - 1].stats;
      return {
        succeeded: latest.succeeded,
        failed: latest.failed,
        running: latest.running,
        queued: latest.queued,
        total: latest.succeeded + latest.failed + latest.running + latest.queued,
      };
    }
    // Fallback to jobs array if no history yet
    return {
      succeeded: jobs.filter(j => j.status === 'SUCCEEDED').length,
      failed: jobs.filter(j => j.status === 'FAILED').length,
      running: jobs.filter(j => j.status === 'RUNNING').length,
      queued: jobs.filter(j => j.status === 'QUEUED').length,
      total: jobs.length,
    };
  }, [history, jobs]);

  return { trendData, currentStats };
}

