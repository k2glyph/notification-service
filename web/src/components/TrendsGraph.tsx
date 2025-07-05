import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { API_ENDPOINTS } from '../config';
import { initializeSSE } from '../utils/sseHelper';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

// Data structures - Adjust based on actual API response
interface TimePoint {
  date?: string; // For daily/weekly data
  week?: string; // For weekly data
  timestamp?: string; // For queue size or more granular data
  count?: number; // For sent notifications
  failedCount?: number; // For failure trends
  successCount?: number; // For failure trends (optional)
  size?: number; // For queue size
}

interface TimeseriesData {
  dailySent: TimePoint[];
  weeklySent: TimePoint[];
  failureTrends: TimePoint[];
  queueSize: TimePoint[];
}

// Mock data for UI development if API fails or is not ready
const mockTimeSeriesData: TimeseriesData = {
  dailySent: [
    { date: '2023-10-01', count: 150 }, { date: '2023-10-02', count: 165 },
    { date: '2023-10-03', count: 130 }, { date: '2023-10-04', count: 180 },
    { date: '2023-10-05', count: 200 }, { date: '2023-10-06', count: 170 },
    { date: '2023-10-07', count: 190 },
  ],
  weeklySent: [
    { week: 'W39', count: 900 }, { week: 'W40', count: 1050 },
    { week: 'W41', count: 1100 }, { week: 'W42', count: 950 },
  ],
  failureTrends: [
    { date: '2023-10-01', failedCount: 10 }, { date: '2023-10-02', failedCount: 12 },
    { date: '2023-10-03', failedCount: 8 }, { date: '2023-10-04', failedCount: 15 },
    { date: '2023-10-05', failedCount: 5 }, { date: '2023-10-06', failedCount: 13 },
    { date: '2023-10-07', failedCount: 9 },
  ],
  queueSize: [
    { timestamp: '10:00', size: 15 }, { timestamp: '10:05', size: 20 },
    { timestamp: '10:10', size: 10 }, { timestamp: '10:15', size: 25 },
    { timestamp: '10:20', size: 18 },
  ],
};

const TrendsGraph: React.FC = () => {
  const [timeseriesData, setTimeseriesData] = useState<TimeseriesData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTimeseriesData = async () => {
      setLoading(true);
      try {
        const response = await axios.get<TimeseriesData>(API_ENDPOINTS.timeSeriesStats);
        setTimeseriesData(response.data);
        setError(null);
      } catch (err) {
        setError('Failed to fetch time series data. Displaying mock data.');
        console.error('Error fetching time series data:', err);
        setTimeseriesData(mockTimeSeriesData); // Use mock data on error
      } finally {
        setLoading(false);
      }
    };

    fetchTimeseriesData();
  }, []);

  // SSE Integration for Queue Size
  useEffect(() => {
    // Only establish SSE if timeseriesData is already loaded (to avoid issues with updating null state)
    // or if you want SSE to provide the initial data too (which would require more complex state handling).
    // For now, let's assume initial load via fetchTimeseriesData.
    if (!timeseriesData) return;

    const MAX_QUEUE_DATA_POINTS = 60; // Keep, for example, the last 60 data points (e.g., 1 hour if 1 point per minute)

    const eventHandlers = {
      'queue_size_update': (newPoint: TimePoint) => {
        console.log('SSE: Received queue_size_update', newPoint);
        setTimeseriesData(prevData => {
          if (!prevData) return null; // Should not happen if effect depends on timeseriesData

          const updatedQueueSize = [...(prevData.queueSize || []), newPoint];

          // Keep the array from growing indefinitely
          if (updatedQueueSize.length > MAX_QUEUE_DATA_POINTS) {
            updatedQueueSize.splice(0, updatedQueueSize.length - MAX_QUEUE_DATA_POINTS);
          }

          return {
            ...prevData,
            queueSize: updatedQueueSize
          };
        });
      }
    };

    const sseCleanup = initializeSSE(API_ENDPOINTS.sseStatsStream, eventHandlers, {
      onError: (event) => {
        console.error("SSE connection error in TrendsGraph:", event);
        // Optionally set a specific error for SSE connection issues related to trends
        // For now, the main 'error' state from initial fetch will cover general data issues.
        // setError(prevError => prevError || "Live queue size updates disconnected."); // Example
      }
    });

    return () => {
      sseCleanup();
    };
  }, [timeseriesData]); // Rerun if timeseriesData itself is replaced (e.g. by a full refresh)

  const renderChart = (data: TimePoint[] | undefined, title: string, dataKey: keyof TimePoint, xAxisKey: keyof TimePoint, lineColor: string) => {
    if (!data || data.length === 0) {
      return <p className="text-gray-500">No data available for {title}.</p>;
    }
    return (
      <div className="mb-8 p-4 border rounded-lg shadow">
        <h3 className="text-lg font-semibold mb-4 text-gray-700">{title}</h3>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
            <XAxis dataKey={xAxisKey} stroke="#666" tick={{ fontSize: 12 }} />
            <YAxis stroke="#666" tick={{ fontSize: 12 }} />
            <Tooltip
              contentStyle={{ backgroundColor: 'rgba(255, 255, 255, 0.8)', borderRadius: '0.5rem', padding: '0.5rem' }}
              itemStyle={{ color: lineColor }}
            />
            <Legend wrapperStyle={{ fontSize: 14 }} />
            <Line type="monotone" dataKey={dataKey} stroke={lineColor} strokeWidth={2} dot={{ r: 4 }} activeDot={{ r: 6 }} name={title.split(' ')[0]} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    );
  };

  if (loading && !timeseriesData) { // Show loading only if no data (even mock) is present yet
    return (
      <div className="bg-white shadow rounded-lg p-4 my-2">
        <h2 className="text-xl font-semibold mb-2 text-gray-700">Trends Over Time</h2>
        <p className="text-gray-500">Loading trend data...</p>
      </div>
    );
  }

  // Error message can be displayed alongside mock data
  // if (error && !timeseriesData) {
  //   return (
  //     <div className="bg-white shadow rounded-lg p-4 my-2">
  //       <h2 className="text-xl font-semibold mb-2 text-red-600">Error</h2>
  //       <p className="text-red-500">{error}</p>
  //     </div>
  //   );
  // }

  return (
    <div className="bg-white shadow rounded-lg p-6 my-2">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 border-b pb-2">Trends Over Time</h2>
      {error && <p className="text-red-500 mb-4">{error}</p>}

      {!timeseriesData ? (
         <p className="text-gray-500">{loading ? 'Loading trend data...' : 'No trend data available.'}</p>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {renderChart(timeseriesData.dailySent, 'Daily Notifications Sent', 'count', 'date', '#3b82f6')}
          {renderChart(timeseriesData.weeklySent, 'Weekly Notifications Sent', 'count', 'week', '#10b981')}
          {renderChart(timeseriesData.failureTrends, 'Failure Trends (Daily)', 'failedCount', 'date', '#ef4444')}
          {renderChart(timeseriesData.queueSize, 'Queue Size Over Time', 'size', 'timestamp', '#f97316')}
        </div>
      )}
    </div>
  );
};

export default TrendsGraph;
