import { Activity, Clock, CheckCircle, XCircle, Server } from 'lucide-react';
import { Job, Worker } from '../types';
import { formatNumber } from '../utils/formatters';

interface JobStatsProps {
  jobs: Job[];
  workers: Worker[];
}

export default function JobStats({ jobs, workers }: JobStatsProps) {
  const totalJobs = jobs.length;
  const runningJobs = jobs.filter((j) => j.status === 'RUNNING').length;
  const succeededJobs = jobs.filter((j) => j.status === 'SUCCEEDED').length;
  const failedJobs = jobs.filter((j) => j.status === 'FAILED').length;
  const activeWorkers = workers.filter((w) => w.status === 'active').length;

  const stats = [
    {
      title: 'Total Jobs',
      value: totalJobs,
      icon: Activity,
      color: 'blue',
    },
    {
      title: 'Running',
      value: runningJobs,
      icon: Clock,
      color: 'yellow',
    },
    {
      title: 'Succeeded',
      value: succeededJobs,
      icon: CheckCircle,
      color: 'green',
    },
    {
      title: 'Failed',
      value: failedJobs,
      icon: XCircle,
      color: 'red',
    },
    {
      title: 'Active Workers',
      value: activeWorkers,
      icon: Server,
      color: 'purple',
    },
  ];

  const colorClasses = {
    blue: 'bg-blue-900 text-blue-400',
    yellow: 'bg-yellow-900 text-yellow-400',
    green: 'bg-green-900 text-green-400',
    red: 'bg-red-900 text-red-400',
    purple: 'bg-purple-900 text-purple-400',
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6">
      {stats.map((stat) => {
        const Icon = stat.icon;
        return (
          <div key={stat.title} className="bg-gray-800 rounded-lg p-4 border border-gray-700">
            <div className="flex items-center justify-between mb-2">
              <span className="text-gray-400 text-sm">{stat.title}</span>
              <div className={`p-2 rounded ${colorClasses[stat.color as keyof typeof colorClasses]}`}>
                <Icon className="w-5 h-5" />
              </div>
            </div>
            <div className="text-2xl font-bold">{formatNumber(stat.value)}</div>
          </div>
        );
      })}
    </div>
  );
}

