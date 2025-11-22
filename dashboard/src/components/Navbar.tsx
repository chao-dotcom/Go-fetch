import { Activity } from 'lucide-react';
import ConnectionStatus from './ConnectionStatus';

export default function Navbar() {
  return (
    <header className="bg-gray-800 border-b border-gray-700 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Activity className="w-6 h-6 text-blue-400" />
          <h1 className="text-2xl font-bold text-white">Task Queue Dashboard</h1>
        </div>
        <div className="flex items-center gap-4">
          <ConnectionStatus />
        </div>
      </div>
    </header>
  );
}

