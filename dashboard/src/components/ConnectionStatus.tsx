import { useState, useEffect } from 'react';
import { apiClient } from '../api/client';
import { CheckCircle, XCircle, AlertCircle } from 'lucide-react';

export default function ConnectionStatus() {
  const [status, setStatus] = useState<'connected' | 'disconnected' | 'checking'>('checking');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const checkConnection = async () => {
      try {
        setStatus('checking');
        await apiClient.getHealth();
        setStatus('connected');
        setError(null);
      } catch (err: any) {
        setStatus('disconnected');
        setError(err.message || 'Failed to connect to API');
      }
    };

    checkConnection();
    const interval = setInterval(checkConnection, 10000); // Check every 10 seconds
    return () => clearInterval(interval);
  }, []);

  if (status === 'checking') {
    return (
      <div className="flex items-center gap-2 text-gray-400 text-sm">
        <AlertCircle className="w-4 h-4 animate-pulse" />
        <span>Checking connection...</span>
      </div>
    );
  }

  if (status === 'connected') {
    return (
      <div className="flex items-center gap-2 text-green-400 text-sm">
        <CheckCircle className="w-4 h-4" />
        <span>Connected to API</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 text-red-400 text-sm">
      <XCircle className="w-4 h-4" />
      <span>Disconnected: {error}</span>
    </div>
  );
}

