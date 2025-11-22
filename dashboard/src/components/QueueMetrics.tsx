import { useEffect, useState } from "react";
import { BarChart3, TrendingUp, Activity } from "lucide-react";

interface QueueStat {
  name: string;
  depth: number;
  processing_rate: number;
  avg_latency_ms: number;
}

export default function QueueMetrics() {
  const [queues, setQueues] = useState<QueueStat[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchQueues = async () => {
    try {
      const response = await fetch("http://localhost:8080/v1/queues");
      const data = await response.json();
      setQueues(data || []);
    } catch (error) {
      console.error("Failed to fetch queues:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchQueues();
    const interval = setInterval(fetchQueues, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading queue metrics...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <h2 className="text-2xl font-bold text-white flex items-center gap-2 mb-6">
        <BarChart3 size={24} />
        Queue Metrics
      </h2>

      {queues.length === 0 ? (
        <div className="text-center text-gray-400 py-8">No queue data available</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {queues.map((queue) => (
            <div key={queue.name} className="bg-gray-700 rounded-lg p-6">
              <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
                <Activity size={20} />
                {queue.name}
              </h3>
              
              <div className="space-y-4">
                <div>
                  <div className="text-gray-400 text-sm mb-1">Queue Depth</div>
                  <div className="text-3xl font-bold text-yellow-400">{queue.depth >= 0 ? queue.depth : "N/A"}</div>
                </div>
                
                <div>
                  <div className="text-gray-400 text-sm mb-1 flex items-center gap-1">
                    <TrendingUp size={14} />
                    Processing Rate
                  </div>
                  <div className="text-2xl font-bold text-blue-400">
                    {queue.processing_rate > 0 ? `${queue.processing_rate}/s` : "N/A"}
                  </div>
                </div>
                
                <div>
                  <div className="text-gray-400 text-sm mb-1">Avg Latency</div>
                  <div className="text-2xl font-bold text-green-400">
                    {queue.avg_latency_ms > 0 ? `${queue.avg_latency_ms}ms` : "N/A"}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

