import { BarChart3, TrendingUp, Activity, Clock } from 'lucide-react';
import { QueueStats as QueueStatsType } from '../types';
import { formatNumber } from '../utils/formatters';

interface QueueStatsProps {
  stats: QueueStatsType[];
  loading: boolean;
}

export default function QueueStats({ stats, loading }: QueueStatsProps) {
  if (loading && stats.length === 0) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading queue metrics...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700">
      <div className="p-6 border-b border-gray-700">
        <h2 className="text-2xl font-bold text-white flex items-center gap-2">
          <BarChart3 className="w-6 h-6" />
          Queue Statistics
        </h2>
      </div>

      {stats.length === 0 ? (
        <div className="p-8 text-center text-gray-400">No queue data available</div>
      ) : (
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {stats.map((queue) => (
              <div key={queue.name} className="bg-gray-700 rounded-lg p-6 border border-gray-600">
                <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
                  <Activity className="w-5 h-5" />
                  {queue.name}
                </h3>

                <div className="space-y-4">
                  <div>
                    <div className="text-gray-400 text-sm mb-1">Queue Depth</div>
                    <div className="text-3xl font-bold text-yellow-400">
                      {queue.depth >= 0 ? formatNumber(queue.depth) : queue.depth === -1 ? 'Unknown' : 'N/A'}
                    </div>
                  </div>

                  <div>
                    <div className="text-gray-400 text-sm mb-1 flex items-center gap-1">
                      <TrendingUp className="w-4 h-4" />
                      Processing Rate
                    </div>
                    <div className="text-2xl font-bold text-blue-400">
                      {queue.processing_rate > 0 ? `${formatNumber(queue.processing_rate)}/s` : '0/s'}
                    </div>
                  </div>

                  <div>
                    <div className="text-gray-400 text-sm mb-1 flex items-center gap-1">
                      <Clock className="w-4 h-4" />
                      Avg Latency
                    </div>
                    <div className="text-2xl font-bold text-green-400">
                      {queue.avg_latency_ms > 0 ? `${formatNumber(queue.avg_latency_ms)}ms` : '0ms'}
                    </div>
                  </div>

                  {queue.pending_count !== undefined && (
                    <div>
                      <div className="text-gray-400 text-sm mb-1">Pending</div>
                      <div className="text-xl font-bold text-orange-400">
                        {formatNumber(queue.pending_count)}
                      </div>
                    </div>
                  )}

                  {queue.broker_depth !== undefined && (
                    <div>
                      <div className="text-gray-400 text-sm mb-1">Broker Depth</div>
                      <div className="text-xl font-bold text-purple-400">
                        {formatNumber(queue.broker_depth)}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

