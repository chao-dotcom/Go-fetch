import { useState } from 'react';
import { useJobs, useWorkers, useQueueStats } from '../hooks';
import { useJobTrends } from '../hooks/useJobTrends';
import JobList from './JobList';
import JobDetail from './JobDetail';
import JobStats from './JobStats';
import WorkerList from './WorkerList';
import QueueStatsComponent from './QueueStats';
import MetricsChart from './MetricsChart';
import Navbar from './Navbar';
import { Job } from '../types';
import { apiClient } from '../api/client';

type Tab = 'jobs' | 'workers' | 'queues';

export default function Dashboard() {
  const [activeTab, setActiveTab] = useState<Tab>('jobs');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);

  const { jobs, loading: jobsLoading, refetch: refetchJobs } = useJobs();
  const { workers, loading: workersLoading } = useWorkers();
  const { stats: queueStatsData, loading: statsLoading } = useQueueStats();
  const { trendData } = useJobTrends(jobs, 60); // 60 data points = 60 minutes of history

  const handleJobClick = (job: Job) => {
    setSelectedJob(job);
  };

  const handleRetry = async (jobId: string) => {
    try {
      await apiClient.retryJob(jobId);
      refetchJobs();
      setSelectedJob(null);
    } catch (error) {
      console.error('Failed to retry job:', error);
    }
  };

  const handleCancel = async (jobId: string) => {
    try {
      await apiClient.cancelJob(jobId);
      refetchJobs();
      setSelectedJob(null);
    } catch (error) {
      console.error('Failed to cancel job:', error);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <Navbar />

      <div className="px-6 py-6">
        {/* Stats Cards */}
        <JobStats jobs={jobs} workers={workers} />

        {/* Job Trends - Below stats, above tabs */}
        <div className="mb-6">
          <MetricsChart data={trendData} title="Job Trends" />
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <TabButton
            active={activeTab === 'jobs'}
            onClick={() => setActiveTab('jobs')}
          >
            Jobs
          </TabButton>
          <TabButton
            active={activeTab === 'workers'}
            onClick={() => setActiveTab('workers')}
          >
            Workers
          </TabButton>
          <TabButton
            active={activeTab === 'queues'}
            onClick={() => setActiveTab('queues')}
          >
            Queues
          </TabButton>
        </div>

        {/* Content */}
        {activeTab === 'jobs' && (
          <div className="space-y-6">
            <JobList
              jobs={jobs}
              loading={jobsLoading}
              onRefresh={refetchJobs}
              onJobClick={handleJobClick}
            />
          </div>
        )}
        {activeTab === 'workers' && (
          <WorkerList workers={workers} loading={workersLoading} />
        )}
        {activeTab === 'queues' && (
          <QueueStatsComponent stats={queueStatsData} loading={statsLoading} />
        )}
      </div>

      {/* Job Detail Modal */}
      {selectedJob && (
        <JobDetail
          job={selectedJob}
          onClose={() => setSelectedJob(null)}
          onRetry={handleRetry}
          onCancel={handleCancel}
        />
      )}
    </div>
  );
}

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2 rounded-lg transition-colors ${
        active
          ? 'bg-blue-600 text-white'
          : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
      }`}
    >
      {children}
    </button>
  );
}

