import { useEffect, useState } from "react";
import { Server, Activity, Clock, CheckCircle } from "lucide-react";

interface Worker {
  id: string;
  hostname: string;
  pool_size: number;
  active_jobs: number;
  total_jobs: number;
  last_heartbeat: string;
  started_at: string;
  status: string;
}

export default function WorkerStatus() {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchWorkers = async () => {
    try {
      const response = await fetch("http://localhost:8080/v1/workers");
      const data = await response.json();
      setWorkers(data || []);
    } catch (error) {
      console.error("Failed to fetch workers:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkers();
    const interval = setInterval(fetchWorkers, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const getTimeAgo = (timestamp: string) => {
    const seconds = Math.floor((Date.now() - new Date(timestamp).getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  };

  if (loading) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading workers...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <h2 className="text-2xl font-bold text-white flex items-center gap-2 mb-6">
        <Server size={24} />
        Worker Status
      </h2>

      {workers.length === 0 ? (
        <div className="text-center text-gray-400 py-8">No workers registered</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workers.map((worker) => (
            <div key={worker.id} className="bg-gray-700 rounded-lg p-4 border border-gray-600">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Server size={20} className="text-blue-400" />
                  <h3 className="font-bold text-white">{worker.hostname || worker.id.substring(0, 8)}</h3>
                </div>
                <span className={`px-2 py-1 rounded-full text-xs font-semibold ${
                  worker.status === "busy" ? "bg-blue-500" : "bg-green-500"
                }`}>
                  {worker.status}
                </span>
              </div>

              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-gray-400 flex items-center gap-1">
                    <Activity size={14} />
                    Active Jobs
                  </span>
                  <span className="text-white font-semibold">{worker.active_jobs}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-gray-400 flex items-center gap-1">
                    <CheckCircle size={14} />
                    Total Processed
                  </span>
                  <span className="text-white font-semibold">{worker.total_jobs}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-gray-400 flex items-center gap-1">
                    <Server size={14} />
                    Pool Size
                  </span>
                  <span className="text-white font-semibold">{worker.pool_size}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-gray-400 flex items-center gap-1">
                    <Clock size={14} />
                    Last Heartbeat
                  </span>
                  <span className="text-white font-semibold">{getTimeAgo(worker.last_heartbeat)}</span>
                </div>
              </div>

              <div className="mt-3 pt-3 border-t border-gray-600">
                <div className="text-xs text-gray-400">
                  Started: {new Date(worker.started_at).toLocaleString()}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

