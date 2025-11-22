import axios, { AxiosInstance } from 'axios';
import { Job, Worker, QueueStats, JobLog } from '../types';
import { API_BASE_URL } from '../utils/constants';

class APIClient {
  private client: AxiosInstance;

  constructor(baseURL: string = API_BASE_URL) {
    this.client = axios.create({
      baseURL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor for auth token
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });
  }

  // Jobs
  async getJobs(params?: {
    status?: string;
    type?: string;
    limit?: number;
    offset?: number;
  }) {
    const { data } = await this.client.get<{
      jobs: Job[];
      total: number;
      limit: number;
      offset: number;
    }>('/jobs', { params });
    return data;
  }

  async getJob(id: string) {
    const { data } = await this.client.get<Job>(`/jobs/${id}`);
    return data;
  }

  async createJob(job: {
    type: string;
    payload: Record<string, any>;
    priority?: number;
    max_attempts?: number;
  }) {
    const { data } = await this.client.post('/jobs', job);
    return data;
  }

  async cancelJob(id: string) {
    const { data } = await this.client.delete(`/jobs/${id}`);
    return data;
  }

  async retryJob(id: string) {
    const { data } = await this.client.post(`/jobs/${id}/retry`);
    return data;
  }

  async getJobLogs(id: string) {
    const { data } = await this.client.get<JobLog[]>(`/jobs/${id}/logs`);
    return data;
  }

  // Workers
  async getWorkers(status?: string) {
    const { data } = await this.client.get<{ value?: Worker[]; Count?: number } | Worker[]>('/workers', {
      params: { status },
    });
    // Handle both response formats: { value: Worker[], Count: number } or Worker[]
    if (Array.isArray(data)) {
      return data;
    }
    return (data as { value?: Worker[] }).value || [];
  }

  // Queues
  async getQueueStats() {
    const { data } = await this.client.get<{ value?: QueueStats[]; Count?: number } | QueueStats[]>('/queues');
    // Handle both response formats: { value: QueueStats[], Count: number } or QueueStats[]
    if (Array.isArray(data)) {
      return data;
    }
    return (data as { value?: QueueStats[] }).value || [];
  }

  // Health (health endpoint is at root, not /v1)
  async getHealth() {
    const baseURL = this.client.defaults.baseURL || API_BASE_URL;
    const healthURL = baseURL.replace('/v1', '') + '/health';
    const { data } = await axios.get(healthURL, { timeout: 5000 });
    return data;
  }
}

export const apiClient = new APIClient();
export default APIClient;

