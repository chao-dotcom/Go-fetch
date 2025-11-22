import { useEffect, useState } from "react";
import { Activity, CheckCircle, XCircle, Clock, RefreshCw } from "lucide-react";

interface Job {
  id: string;
  type: string;
  status: string;
  created_at: string;
  updated_at: string;
  attempts: number;
  max_attempts: number;
}

export default function JobMonitor() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    total: 0,
    queued: 0,
    running: 0,
    succeeded: 0,
    failed: 0,
  });

  const fetchJobs = async () => {
    try {
      const response = await fetch("http://localhost:8080/v1/jobs?limit=50");
      const data = await response.json();
      setJobs(data.jobs || []);
      
      // Calculate stats
      const newStats = {
        total: data.total || 0,
        queued: 0,
        running: 0,
        succeeded: 0,
        failed: 0,
      };
      
      (data.jobs || []).forEach((job: Job) => {
        if (job.status === "QUEUED") newStats.queued++;
        else if (job.status === "RUNNING") newStats.running++;
        else if (job.status === "SUCCEEDED") newStats.succeeded++;
        else if (job.status === "FAILED") newStats.failed++;
      });
      
      setStats(newStats);
    } catch (error) {
      console.error("Failed to fetch jobs:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();
    const interval = setInterval(fetchJobs, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status: string) => {
    switch (status) {
      case "QUEUED":
        return "bg-yellow-500";
      case "RUNNING":
        return "bg-blue-500";
      case "SUCCEEDED":
        return "bg-green-500";
      case "FAILED":
        return "bg-red-500";
      default:
        return "bg-gray-500";
    }
  };

  if (loading) {
    return (
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="animate-pulse text-gray-400">Loading jobs...</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white flex items-center gap-2">
          <Activity size={24} />
          Job Monitor
        </h2>
        <button
          onClick={fetchJobs}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
        >
          <RefreshCw size={18} />
          Refresh
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-5 gap-4 mb-6">
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="text-gray-400 text-sm">Total</div>
          <div className="text-2xl font-bold text-white">{stats.total}</div>
        </div>
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="text-gray-400 text-sm flex items-center gap-1">
            <Clock size={14} />
            Queued
          </div>
          <div className="text-2xl font-bold text-yellow-400">{stats.queued}</div>
        </div>
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="text-gray-400 text-sm flex items-center gap-1">
            <Activity size={14} />
            Running
          </div>
          <div className="text-2xl font-bold text-blue-400">{stats.running}</div>
        </div>
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="text-gray-400 text-sm flex items-center gap-1">
            <CheckCircle size={14} />
            Succeeded
          </div>
          <div className="text-2xl font-bold text-green-400">{stats.succeeded}</div>
        </div>
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="text-gray-400 text-sm flex items-center gap-1">
            <XCircle size={14} />
            Failed
          </div>
          <div className="text-2xl font-bold text-red-400">{stats.failed}</div>
        </div>
      </div>

      {/* Jobs Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead>
            <tr className="border-b border-gray-700">
              <th className="pb-3 text-gray-400 font-semibold">ID</th>
              <th className="pb-3 text-gray-400 font-semibold">Type</th>
              <th className="pb-3 text-gray-400 font-semibold">Status</th>
              <th className="pb-3 text-gray-400 font-semibold">Attempts</th>
              <th className="pb-3 text-gray-400 font-semibold">Created</th>
            </tr>
          </thead>
          <tbody>
            {jobs.length === 0 ? (
              <tr>
                <td colSpan={5} className="pt-4 text-center text-gray-400">
                  No jobs found
                </td>
              </tr>
            ) : (
              jobs.map((job) => (
                <tr key={job.id} className="border-b border-gray-700 hover:bg-gray-700">
                  <td className="py-3 text-white font-mono text-sm">{job.id.substring(0, 8)}...</td>
                  <td className="py-3 text-gray-300">{job.type}</td>
                  <td className="py-3">
                    <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-semibold ${getStatusColor(job.status)}`}>
                      {job.status}
                    </span>
                  </td>
                  <td className="py-3 text-gray-300">
                    {job.attempts}/{job.max_attempts}
                  </td>
                  <td className="py-3 text-gray-400 text-sm">
                    {new Date(job.created_at).toLocaleString()}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

