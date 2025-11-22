import { useState } from 'react';
import { Search, RefreshCw, ChevronDown, ChevronUp } from 'lucide-react';
import { Job } from '../types';
import StatusBadge from './StatusBadge';
import { formatRelativeTime } from '../utils/formatters';

interface JobListProps {
  jobs: Job[];
  loading: boolean;
  onRefresh?: () => void;
  onJobClick?: (job: Job) => void;
}

export default function JobList({ jobs, loading, onRefresh, onJobClick }: JobListProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [isTableCollapsed, setIsTableCollapsed] = useState(false);

  const filteredJobs = jobs.filter((job) => {
    const matchesSearch = job.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
      job.type.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'all' || job.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  if (loading && jobs.length === 0) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading jobs...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700">
      {/* Header with collapse button */}
      <div className="p-4 border-b border-gray-700 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Jobs ({filteredJobs.length})</h2>
        <button
          onClick={() => setIsTableCollapsed(!isTableCollapsed)}
          className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors flex items-center gap-2 text-sm text-gray-300"
        >
          {isTableCollapsed ? (
            <>
              <ChevronDown className="w-4 h-4" />
              Expand
            </>
          ) : (
            <>
              <ChevronUp className="w-4 h-4" />
              Collapse
            </>
          )}
        </button>
      </div>

      {/* Filters */}
      <div className="p-4 border-b border-gray-700 flex gap-4 items-center">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
          <input
            type="text"
            placeholder="Search by ID or type..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Status</option>
          <option value="QUEUED">Queued</option>
          <option value="RUNNING">Running</option>
          <option value="SUCCEEDED">Succeeded</option>
          <option value="FAILED">Failed</option>
          <option value="RETRYING">Retrying</option>
          <option value="CANCELLED">Cancelled</option>
        </select>
        {onRefresh && (
          <button
            onClick={onRefresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors flex items-center gap-2"
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
        )}
      </div>

      {/* Table */}
      {!isTableCollapsed && (
        <div className="overflow-x-auto">
          <table className="w-full">
          <thead>
            <tr className="border-b border-gray-700">
              <th className="px-4 py-3 text-left text-gray-400 font-semibold text-sm">ID</th>
              <th className="px-4 py-3 text-left text-gray-400 font-semibold text-sm">Type</th>
              <th className="px-4 py-3 text-left text-gray-400 font-semibold text-sm">Status</th>
              <th className="px-4 py-3 text-left text-gray-400 font-semibold text-sm">Attempts</th>
              <th className="px-4 py-3 text-left text-gray-400 font-semibold text-sm">Created</th>
            </tr>
          </thead>
          <tbody>
            {filteredJobs.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-400">
                  No jobs found
                </td>
              </tr>
            ) : (
              filteredJobs.map((job) => (
                <tr
                  key={job.id}
                  className="border-b border-gray-700 hover:bg-gray-700 cursor-pointer transition-colors"
                  onClick={() => onJobClick?.(job)}
                >
                  <td className="px-4 py-3 text-white font-mono text-sm">{job.id.substring(0, 8)}...</td>
                  <td className="px-4 py-3 text-gray-300">{job.type}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={job.status} />
                  </td>
                  <td className="px-4 py-3 text-gray-300">
                    {job.attempts}/{job.max_attempts}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-sm">
                    {formatRelativeTime(job.created_at)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
        </div>
      )}
      {isTableCollapsed && (
        <div className="p-4 text-center text-gray-400 text-sm">
          Table collapsed. Click "Expand" to view jobs.
        </div>
      )}
    </div>
  );
}

