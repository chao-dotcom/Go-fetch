import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface ChartData {
  time: string;
  succeeded: number;
  failed: number;
  running: number;
  queued?: number;
}

interface MetricsChartProps {
  data: ChartData[];
  title?: string;
}

export default function MetricsChart({ data, title }: MetricsChartProps) {
  // Calculate the maximum value in the data to set appropriate Y-axis domain
  const maxValue = data.length > 0
    ? Math.max(
        ...data.flatMap(d => [
          d.succeeded || 0,
          d.failed || 0,
          d.running || 0,
          d.queued || 0
        ])
      )
    : 0;
  
  // Set Y-axis domain: [0, max(500, maxValue * 1.1)] to ensure at least 500 and add 10% padding
  const yAxisDomain: [number, number] = [0, Math.max(500, Math.ceil(maxValue * 1.1))];

  return (
    <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
      {title && <h3 className="text-lg font-semibold text-white mb-4">{title}</h3>}
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
          <XAxis 
            dataKey="time" 
            stroke="#9ca3af"
            angle={-45}
            textAnchor="end"
            height={80}
            interval="preserveStartEnd"
          />
          <YAxis stroke="#9ca3af" domain={yAxisDomain} />
          <Tooltip
            contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px' }}
            labelStyle={{ color: '#f3f4f6' }}
          />
          <Legend />
          <Line
            type="monotone"
            dataKey="succeeded"
            stroke="#10b981"
            strokeWidth={2}
            name="Succeeded"
            dot={{ fill: '#10b981', r: 3 }}
          />
          <Line
            type="monotone"
            dataKey="failed"
            stroke="#ef4444"
            strokeWidth={2}
            name="Failed"
            dot={{ fill: '#ef4444', r: 3 }}
          />
          <Line
            type="monotone"
            dataKey="running"
            stroke="#3b82f6"
            strokeWidth={2}
            name="Running"
            dot={{ fill: '#3b82f6', r: 3 }}
          />
          {data.length > 0 && data[0].queued !== undefined && (
            <Line
              type="monotone"
              dataKey="queued"
              stroke="#f59e0b"
              strokeWidth={2}
              name="Queued"
              dot={{ fill: '#f59e0b', r: 3 }}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

