import { Server, Activity, Clock, CheckCircle, XCircle } from 'lucide-react';
import { Worker } from '../types';
import StatusBadge from './StatusBadge';
import { formatRelativeTime, formatNumber } from '../utils/formatters';

interface WorkerListProps {
  workers: Worker[];
  loading: boolean;
}

export default function WorkerList({ workers, loading }: WorkerListProps) {
  if (loading && workers.length === 0) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading workers...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700">
      <div className="p-6 border-b border-gray-700">
        <h2 className="text-2xl font-bold text-white flex items-center gap-2">
          <Server className="w-6 h-6" />
          Workers
        </h2>
      </div>

      {workers.length === 0 ? (
        <div className="p-8 text-center text-gray-400">No workers registered</div>
      ) : (
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {workers.map((worker) => (
              <div key={worker.id} className="bg-gray-700 rounded-lg p-4 border border-gray-600">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Server className="w-5 h-5 text-blue-400" />
                    <h3 className="font-bold text-white">{worker.hostname || worker.id.substring(0, 8)}</h3>
                  </div>
                  <StatusBadge status={worker.status} />
                </div>

                <div className="space-y-2 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1">
                      <Activity className="w-4 h-4" />
                      Active Jobs
                    </span>
                    <span className="text-white font-semibold">{worker.active_jobs}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1">
                      <CheckCircle className="w-4 h-4" />
                      Total Jobs
                    </span>
                    <span className="text-white font-semibold">{formatNumber(worker.total_jobs || worker.total_processed || 0)}</span>
                  </div>
                  {worker.total_failed !== undefined && (
                    <div className="flex items-center justify-between">
                      <span className="text-gray-400 flex items-center gap-1">
                        <XCircle className="w-4 h-4" />
                        Total Failed
                      </span>
                      <span className="text-white font-semibold">{formatNumber(worker.total_failed)}</span>
                    </div>
                  )}
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1">
                      <Server className="w-4 h-4" />
                      Pool Size
                    </span>
                    <span className="text-white font-semibold">{worker.pool_size}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1">
                      <Clock className="w-4 h-4" />
                      Last Heartbeat
                    </span>
                    <span className="text-white font-semibold text-xs">{formatRelativeTime(worker.last_heartbeat)}</span>
                  </div>
                </div>

                <div className="mt-3 pt-3 border-t border-gray-600">
                  <div className="text-xs text-gray-400">
                    Started: {formatRelativeTime(worker.started_at)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

