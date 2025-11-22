import { useState, useEffect } from 'react';
import { X, RefreshCw, Trash2 } from 'lucide-react';
import { Job, JobLog } from '../types';
import { apiClient } from '../api/client';
import StatusBadge from './StatusBadge';
import { formatDate, formatRelativeTime } from '../utils/formatters';

interface JobDetailProps {
  job: Job;
  onClose: () => void;
  onRetry?: (jobId: string) => void;
  onCancel?: (jobId: string) => void;
}

export default function JobDetail({ job, onClose, onRetry, onCancel }: JobDetailProps) {
  const [logs, setLogs] = useState<JobLog[]>([]);
  const [loadingLogs, setLoadingLogs] = useState(false);
  const [activeTab, setActiveTab] = useState<'details' | 'logs'>('details');

  useEffect(() => {
    if (activeTab === 'logs') {
      loadLogs();
    }
  }, [activeTab, job.id]);

  const loadLogs = async () => {
    setLoadingLogs(true);
    try {
      const data = await apiClient.getJobLogs(job.id);
      setLogs(data);
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLoadingLogs(false);
    }
  };

  const handleRetry = () => {
    if (onRetry) {
      onRetry(job.id);
    }
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel(job.id);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-gray-800 rounded-lg border border-gray-700 w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-700 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold text-white">Job Details</h2>
            <p className="text-gray-400 text-sm font-mono">{job.id}</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Tabs */}
        <div className="px-6 border-b border-gray-700 flex gap-4">
          <button
            onClick={() => setActiveTab('details')}
            className={`py-3 px-2 border-b-2 transition-colors ${
              activeTab === 'details'
                ? 'border-blue-500 text-blue-400'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            Details
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`py-3 px-2 border-b-2 transition-colors ${
              activeTab === 'logs'
                ? 'border-blue-500 text-blue-400'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            Logs
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-gray-400 text-sm">Status</label>
                  <div className="mt-1">
                    <StatusBadge status={job.status} />
                  </div>
                </div>
                <div>
                  <label className="text-gray-400 text-sm">Type</label>
                  <div className="mt-1 text-white">{job.type}</div>
                </div>
                <div>
                  <label className="text-gray-400 text-sm">Priority</label>
                  <div className="mt-1 text-white">{job.priority}</div>
                </div>
                <div>
                  <label className="text-gray-400 text-sm">Attempts</label>
                  <div className="mt-1 text-white">
                    {job.attempts} / {job.max_attempts}
                  </div>
                </div>
                <div>
                  <label className="text-gray-400 text-sm">Created</label>
                  <div className="mt-1 text-white">{formatDate(job.created_at)}</div>
                  <div className="text-gray-400 text-xs">{formatRelativeTime(job.created_at)}</div>
                </div>
                <div>
                  <label className="text-gray-400 text-sm">Updated</label>
                  <div className="mt-1 text-white">{formatDate(job.updated_at)}</div>
                  <div className="text-gray-400 text-xs">{formatRelativeTime(job.updated_at)}</div>
                </div>
                {job.started_at && (
                  <div>
                    <label className="text-gray-400 text-sm">Started</label>
                    <div className="mt-1 text-white">{formatDate(job.started_at)}</div>
                  </div>
                )}
                {job.finished_at && (
                  <div>
                    <label className="text-gray-400 text-sm">Finished</label>
                    <div className="mt-1 text-white">{formatDate(job.finished_at)}</div>
                  </div>
                )}
                {job.worker_id && (
                  <div>
                    <label className="text-gray-400 text-sm">Worker ID</label>
                    <div className="mt-1 text-white font-mono text-sm">{job.worker_id}</div>
                  </div>
                )}
              </div>

              {job.payload && (
                <div>
                  <label className="text-gray-400 text-sm">Payload</label>
                  <pre className="mt-1 p-4 bg-gray-900 rounded-lg text-white text-sm overflow-x-auto">
                    {JSON.stringify(job.payload, null, 2)}
                  </pre>
                </div>
              )}

              {job.result && (
                <div>
                  <label className="text-gray-400 text-sm">Result</label>
                  <pre className="mt-1 p-4 bg-gray-900 rounded-lg text-white text-sm overflow-x-auto">
                    {job.result}
                  </pre>
                </div>
              )}

              {job.error && (
                <div>
                  <label className="text-gray-400 text-sm">Error</label>
                  <pre className="mt-1 p-4 bg-red-900 bg-opacity-20 rounded-lg text-red-400 text-sm overflow-x-auto">
                    {job.error}
                  </pre>
                </div>
              )}
            </div>
          ) : (
            <div>
              {loadingLogs ? (
                <div className="text-gray-400">Loading logs...</div>
              ) : logs.length === 0 ? (
                <div className="text-gray-400">No logs available</div>
              ) : (
                <div className="space-y-2">
                  {logs.map((log) => (
                    <div
                      key={log.id}
                      className="p-3 bg-gray-900 rounded-lg border border-gray-700"
                    >
                      <div className="flex items-center justify-between mb-1">
                        <span className={`text-xs font-semibold ${
                          log.level === 'ERROR' ? 'text-red-400' :
                          log.level === 'WARN' ? 'text-yellow-400' :
                          log.level === 'INFO' ? 'text-blue-400' :
                          'text-gray-400'
                        }`}>
                          {log.level}
                        </span>
                        <span className="text-gray-500 text-xs">
                          {formatRelativeTime(log.created_at)}
                        </span>
                      </div>
                      <div className="text-white text-sm">{log.message}</div>
                      {log.metadata && (
                        <pre className="mt-2 text-xs text-gray-400 overflow-x-auto">
                          {JSON.stringify(log.metadata, null, 2)}
                        </pre>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="px-6 py-4 border-t border-gray-700 flex gap-2">
          {job.status === 'FAILED' && onRetry && (
            <button
              onClick={handleRetry}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors flex items-center gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Retry
            </button>
          )}
          {(job.status === 'QUEUED' || job.status === 'RUNNING') && onCancel && (
            <button
              onClick={handleCancel}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg transition-colors flex items-center gap-2"
            >
              <Trash2 className="w-4 h-4" />
              Cancel
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

