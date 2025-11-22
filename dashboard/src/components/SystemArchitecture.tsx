import {
  Activity,
  AlertCircle,
  BarChart3,
  Box,
  Clock,
  Cpu,
  Database,
  Eye,
  GitBranch,
  Layers,
  Server,
  Zap,
} from "lucide-react";
import { useState } from "react";

type Tab = "overview" | "flow" | "components" | "monitoring";

type TabButtonProps = {
  id: Tab;
  label: string;
  icon: React.ComponentType<any>;
  active: boolean;
  onClick: (id: Tab) => void;
};

const TabButton = ({ id, label, icon: Icon, active, onClick }: TabButtonProps) => (
  <button
    onClick={() => onClick(id)}
    className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-all ${
      active ? "bg-blue-600 text-white shadow-lg" : "bg-gray-700 text-gray-300 hover:bg-gray-600"
    }`}
  >
    <Icon size={18} />
    {label}
  </button>
);

type ComponentBoxProps = {
  title: string;
  tech: string;
  color: string;
  icon: React.ComponentType<any>;
};

const ComponentBox = ({ title, tech, color, icon: Icon }: ComponentBoxProps) => (
  <div className={`p-4 rounded-lg border-2 ${color} bg-opacity-10`}>
    <div className="flex items-center gap-2 mb-2">
      <Icon size={20} />
      <h3 className="font-bold">{title}</h3>
    </div>
    <p className="text-sm text-gray-400">{tech}</p>
  </div>
);

const SystemArchitecture = () => {
  const [activeTab, setActiveTab] = useState<Tab>("overview");

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800 text-white p-8">
      <div className="max-w-7xl mx-auto">
        <header className="mb-8">
          <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
            Task Queue System Architecture
          </h1>
          <p className="text-gray-400">Production-Grade Distributed Job Processing Platform</p>
        </header>

        <div className="flex flex-wrap gap-3 mb-8">
          <TabButton id="overview" label="System Overview" icon={Layers} active={activeTab === "overview"} onClick={setActiveTab} />
          <TabButton id="flow" label="Data Flow" icon={GitBranch} active={activeTab === "flow"} onClick={setActiveTab} />
          <TabButton id="components" label="Components" icon={Box} active={activeTab === "components"} onClick={setActiveTab} />
          <TabButton id="monitoring" label="Observability" icon={Eye} active={activeTab === "monitoring"} onClick={setActiveTab} />
        </div>

        {activeTab === "overview" && (
          <div className="space-y-6">
            <div className="bg-gray-800 p-6 rounded-xl border border-gray-700">
              <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
                <Server size={24} />
                System Architecture Diagram
              </h2>
              <div className="bg-gray-900 p-8 rounded-lg border border-gray-600">
                <svg viewBox="0 0 1000 700" className="w-full">
                  <rect x="400" y="20" width="200" height="60" fill="#3b82f6" rx="8" />
                  <text x="500" y="55" textAnchor="middle" fill="white" fontSize="16" fontWeight="bold">
                    Client / CLI
                  </text>

                  <rect x="350" y="120" width="300" height="80" fill="#8b5cf6" rx="8" />
                  <text x="500" y="155" textAnchor="middle" fill="white" fontSize="18" fontWeight="bold">
                    API Gateway
                  </text>
                  <text x="500" y="180" textAnchor="middle" fill="white" fontSize="12">
                    Gin + gRPC
                  </text>

                  <line x1="500" y1="80" x2="500" y2="120" stroke="#60a5fa" strokeWidth="3" markerEnd="url(#arrowblue)" />

                  <rect x="100" y="260" width="200" height="100" fill="#10b981" rx="8" />
                  <text x="200" y="295" textAnchor="middle" fill="white" fontSize="16" fontWeight="bold">
                    PostgreSQL
                  </text>
                  <text x="200" y="320" textAnchor="middle" fill="white" fontSize="12">
                    Job Metadata
                  </text>
                  <text x="200" y="340" textAnchor="middle" fill="white" fontSize="12">
                    Logs & Results
                  </text>

                  <rect x="700" y="260" width="200" height="100" fill="#ef4444" rx="8" />
                  <text x="800" y="295" textAnchor="middle" fill="white" fontSize="16" fontWeight="bold">
                    Redis Streams
                  </text>
                  <text x="800" y="320" textAnchor="middle" fill="white" fontSize="12">
                    Message Broker
                  </text>
                  <text x="800" y="340" textAnchor="middle" fill="white" fontSize="12">
                    Job Queue
                  </text>

                  <line x1="400" y1="170" x2="250" y2="260" stroke="#60a5fa" strokeWidth="3" markerEnd="url(#arrowblue)" />
                  <line x1="600" y1="170" x2="750" y2="260" stroke="#60a5fa" strokeWidth="3" markerEnd="url(#arrowblue)" />

                  {[150, 410, 670].map((x, idx) => (
                    <g key={x}>
                      <rect x={x} y="450" width="180" height="120" fill="#f59e0b" rx="8" />
                      <text x={x + 90} y="485" textAnchor="middle" fill="white" fontSize="16" fontWeight="bold">
                        {`Worker Node ${idx === 2 ? "N" : idx + 1}`}
                      </text>
                      <text x={x + 90} y="510" textAnchor="middle" fill="white" fontSize="12">
                        Goroutine Pool
                      </text>
                      <text x={x + 90} y="530" textAnchor="middle" fill="white" fontSize="12">
                        16 workers
                      </text>
                      {[ -30, -10, 10 ].map((offset, i) => (
                        <circle key={i} cx={x + 90 + offset} cy="550" r="8" fill="#10b981" />
                      ))}
                    </g>
                  ))}

                  <line x1="750" y1="360" x2="280" y2="450" stroke="#fbbf24" strokeWidth="3" markerEnd="url(#arrowyellow)" />
                  <line x1="800" y1="360" x2="500" y2="450" stroke="#fbbf24" strokeWidth="3" markerEnd="url(#arrowyellow)" />
                  <line x1="850" y1="360" x2="760" y2="450" stroke="#fbbf24" strokeWidth="3" markerEnd="url(#arrowyellow)" />

                  <line x1="240" y1="450" x2="200" y2="360" stroke="#34d399" strokeWidth="2" strokeDasharray="5,5" />

                  <rect x="50" y="620" width="900" height="60" fill="#6366f1" rx="8" />
                  <text x="500" y="655" textAnchor="middle" fill="white" fontSize="16" fontWeight="bold">
                    Observability Layer: Prometheus + Grafana + OpenTelemetry + Jaeger
                  </text>

                  <defs>
                    <marker id="arrowblue" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto" markerUnits="strokeWidth">
                      <path d="M0,0 L0,6 L9,3 z" fill="#60a5fa" />
                    </marker>
                    <marker id="arrowyellow" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto" markerUnits="strokeWidth">
                      <path d="M0,0 L0,6 L9,3 z" fill="#fbbf24" />
                    </marker>
                  </defs>
                </svg>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <ComponentBox title="API Layer" tech="Gin, gRPC, JWT Auth" color="border-purple-500 text-purple-400" icon={Server} />
              <ComponentBox title="Storage" tech="PostgreSQL + Redis" color="border-green-500 text-green-400" icon={Database} />
              <ComponentBox title="Workers" tech="Go Goroutines, Channels" color="border-orange-500 text-orange-400" icon={Cpu} />
            </div>
          </div>
        )}

        {activeTab === "flow" && (
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700">
            <h2 className="text-2xl font-bold mb-6 flex items-center gap-2">
              <GitBranch size={24} />
              Job Lifecycle & Data Flow
            </h2>
            <div className="space-y-6">
              <div className="bg-gray-900 p-6 rounded-lg">
                <h3 className="text-xl font-bold mb-4 text-blue-400">1. Job Submission Flow</h3>
                <div className="space-y-3 text-sm">
                  {[
                    "Client → API: POST /v1/jobs with payload",
                    "API → PostgreSQL: Insert job record (status: QUEUED)",
                    "API → Redis: XADD job_stream * job_id={uuid} payload={json}",
                    "API → Client: Return 202 Accepted with job_id",
                  ].map((step, index) => (
                    <div key={step} className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center font-bold">{index + 1}</div>
                      <div className="flex-1 bg-gray-800 p-3 rounded">
                        <strong>Step {index + 1}:</strong> {step}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bg-gray-900 p-6 rounded-lg">
                <h3 className="text-xl font-bold mb-4 text-orange-400">2. Job Processing Flow</h3>
                <div className="space-y-3 text-sm">
                  {[
                    "Worker: XREADGROUP GROUP workers worker-1 STREAMS job_stream >",
                    "Worker → PostgreSQL: UPDATE jobs SET status='RUNNING', started_at=NOW()",
                    "Worker: Execute job via JobRegistry[job_type](ctx, payload)",
                    "Worker → PostgreSQL: UPDATE status='SUCCEEDED' or 'FAILED', result/error",
                    "Worker → Redis: XACK job_stream workers {message_id}",
                  ].map((step, index) => (
                    <div key={step} className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-orange-600 rounded-full flex items-center justify-center font-bold">{index + 1}</div>
                      <div className="flex-1 bg-gray-800 p-3 rounded">
                        {step}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bg-gray-900 p-6 rounded-lg">
                <h3 className="text-xl font-bold mb-4 text-red-400">3. Failure & Retry Flow</h3>
                <div className="space-y-3 text-sm">
                  {[
                    "On Failure: Check attempts < max_attempts",
                    "If retryable: status='RETRYING', increment attempts",
                    "Backoff: Wait exponential delay (2^attempts * 100ms)",
                    "Requeue: XADD job_stream with same job_id",
                    "If max attempts: Move to dead_letter_queue",
                  ].map((step, index) => (
                    <div key={step} className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-red-600 rounded-full flex items-center justify-center font-bold">{index + 1}</div>
                      <div className="flex-1 bg-gray-800 p-3 rounded">
                        {step}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === "components" && (
          <div className="space-y-6">
            <div className="bg-gray-800 p-6 rounded-xl border border-gray-700">
              <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
                <Box size={24} />
                Component Specifications
              </h2>

              <div className="space-y-4">
                <details className="bg-gray-900 p-4 rounded-lg cursor-pointer" open>
                  <summary className="font-bold text-lg text-purple-400 mb-3">API Server (Gin + gRPC)</summary>
                  <div className="space-y-3 text-sm pl-4">
                    <div><strong className="text-blue-400">Port:</strong> 8080 (HTTP), 9090 (gRPC)</div>
                    <div><strong className="text-blue-400">Middleware:</strong> CORS, Rate Limiting, JWT Auth, Request ID, Logging</div>
                    <div><strong className="text-blue-400">Rate Limit:</strong> 100 req/min per IP (go-chi/httprate)</div>
                    <div><strong className="text-blue-400">Health Check:</strong> GET /health → {"{status: \"ok\", uptime: 12345}"}</div>
                    <div><strong className="text-blue-400">Metrics:</strong> /metrics (Prometheus format)</div>
                  </div>
                </details>

                <details className="bg-gray-900 p-4 rounded-lg cursor-pointer">
                  <summary className="font-bold text-lg text-green-400 mb-3">PostgreSQL Schema</summary>
                  <pre className="bg-black p-3 rounded text-xs overflow-x-auto text-green-300">{`CREATE TABLE jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  type VARCHAR(100) NOT NULL,
  payload JSONB NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'QUEUED',
  priority INT DEFAULT 0,
  attempts INT DEFAULT 0,
  max_attempts INT DEFAULT 3,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  result JSONB,
  error TEXT,
  worker_id VARCHAR(100),
  INDEX idx_status (status),
  INDEX idx_type (type),
  INDEX idx_created_at (created_at)
);

CREATE TABLE job_logs (
  id BIGSERIAL PRIMARY KEY,
  job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
  level VARCHAR(20),
  message TEXT,
  timestamp TIMESTAMPTZ DEFAULT NOW(),
  INDEX idx_job_id (job_id)
);`}</pre>
                </details>

                <details className="bg-gray-900 p-4 rounded-lg cursor-pointer">
                  <summary className="font-bold text-lg text-red-400 mb-3">Redis Configuration</summary>
                  <div className="space-y-3 text-sm pl-4">
                    <div><strong className="text-blue-400">Stream:</strong> job_stream</div>
                    <div><strong className="text-blue-400">Consumer Group:</strong> workers</div>
                    <div><strong className="text-blue-400">Max Length:</strong> ~10000 (MAXLEN ~ 10000)</div>
                    <div><strong className="text-blue-400">Dead Letter:</strong> dead_letter_queue (separate stream)</div>
                    <div><strong className="text-blue-400">Connection Pool:</strong> 10 connections</div>
                    <div><strong className="text-blue-400">Timeout:</strong> 5s read, 3s write</div>
                  </div>
                </details>

                <details className="bg-gray-900 p-4 rounded-lg cursor-pointer">
                  <summary className="font-bold text-lg text-orange-400 mb-3">Worker Configuration</summary>
                  <pre className="bg-black p-3 rounded text-xs overflow-x-auto text-orange-300">{`worker:
  id: "worker-\${HOSTNAME}-\${PID}"
  pool_size: 16
  max_retries: 5
  backoff_multiplier: 2
  heartbeat_interval: 5s
  shutdown_timeout: 30s
  
queues:
  - name: "default"
    priority: 0
  - name: "high_priority"
    priority: 10
  - name: "low_priority"
    priority: -10
    
job_timeout: 300s
max_job_execution_time: 3600s`}</pre>
                </details>
              </div>
            </div>
          </div>
        )}

        {activeTab === "monitoring" && (
          <div className="space-y-6">
            <div className="bg-gray-800 p-6 rounded-xl border border-gray-700">
              <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
                <Activity size={24} />
                Observability Stack
              </h2>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                <div className="bg-gradient-to-br from-orange-900 to-orange-800 p-4 rounded-lg">
                  <h3 className="font-bold text-lg mb-2 flex items-center gap-2">
                    <BarChart3 size={20} />
                    Prometheus Metrics
                  </h3>
                  <ul className="space-y-1 text-sm">
                    <li>• job_processing_duration_seconds</li>
                    <li>• job_failures_total</li>
                    <li>• job_retries_total</li>
                    <li>• worker_active_jobs</li>
                    <li>• queue_depth</li>
                    <li>• worker_heartbeat_timestamp</li>
                    <li>• api_request_duration_seconds</li>
                  </ul>
                </div>

                <div className="bg-gradient-to-br from-blue-900 to-blue-800 p-4 rounded-lg">
                  <h3 className="font-bold text-lg mb-2 flex items-center gap-2">
                    <Zap size={20} />
                    OpenTelemetry Tracing
                  </h3>
                  <ul className="space-y-1 text-sm">
                    <li>• Trace ID propagation</li>
                    <li>• Span: API → Broker → Worker</li>
                    <li>• Jaeger exporter (port 14268)</li>
                    <li>• Baggage: user_id, job_type</li>
                    <li>• Sampling: 100% dev, 10% prod</li>
                  </ul>
                </div>

                <div className="bg-gradient-to-br from-purple-900 to-purple-800 p-4 rounded-lg">
                  <h3 className="font-bold text-lg mb-2 flex items-center gap-2">
                    <Clock size={20} />
                    Structured Logging
                  </h3>
                  <ul className="space-y-1 text-sm">
                    <li>• Zap logger (JSON format)</li>
                    <li>• Fields: job_id, worker_id, trace_id</li>
                    <li>• Levels: debug, info, warn, error</li>
                    <li>• Output: stdout (Docker logs)</li>
                    <li>• Rotation: daily, 7 day retention</li>
                  </ul>
                </div>

                <div className="bg-gradient-to-br from-red-900 to-red-800 p-4 rounded-lg">
                  <h3 className="font-bold text-lg mb-2 flex items-center gap-2">
                    <AlertCircle size={20} />
                    Alerting Rules
                  </h3>
                  <ul className="space-y-1 text-sm">
                    <li>• High job failure rate (&gt; 10%)</li>
                    <li>• Queue depth (&gt; 10000)</li>
                    <li>• Worker down (no heartbeat 30s)</li>
                    <li>• API latency p99 (&gt; 1s)</li>
                    <li>• Dead letter queue growth</li>
                  </ul>
                </div>
              </div>

              <div className="bg-gray-900 p-4 rounded-lg">
                <h3 className="font-bold text-lg mb-3 text-yellow-400">Dashboard Metrics</h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-center">
                  {[
                    { label: "Jobs Processed", value: "1,234", color: "text-green-400" },
                    { label: "In Queue", value: "156", color: "text-blue-400" },
                    { label: "Active Workers", value: "12", color: "text-orange-400" },
                    { label: "Avg Latency", value: "450ms", color: "text-purple-400" },
                  ].map(({ label, value, color }) => (
                    <div key={label} className="bg-gray-800 p-3 rounded">
                      <div className={`text-2xl font-bold ${color}`}>{value}</div>
                      <div className="text-xs text-gray-400">{label}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default SystemArchitecture;

