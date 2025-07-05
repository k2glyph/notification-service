import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { API_ENDPOINTS } from '../config';
import { initializeSSE } from '../utils/sseHelper';

interface ChannelStatData {
  channelName: string;
  totalNotifications: number;
  failedNotifications: number;
  successRate: number; // Assuming percentage
}

const ChannelStats: React.FC = () => {
  const [channelData, setChannelData] = useState<ChannelStatData[] | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchChannelData = async () => {
      try {
        setLoading(true);
        const response = await axios.get<ChannelStatData[]>(API_ENDPOINTS.channelStats);
        setChannelData(response.data);
        setError(null);
      } catch (err) {
        setError('Failed to fetch channel statistics. Please try again later.');
        console.error('Error fetching channel statistics:', err);
        // Set mock data on error for UI development
        setChannelData([
          { channelName: 'Email', totalNotifications: 0, failedNotifications: 0, successRate: 0 },
          { channelName: 'Slack', totalNotifications: 0, failedNotifications: 0, successRate: 0 },
        ]);
      } finally {
        setLoading(false);
      }
    };

    fetchChannelData();
  }, []);

  // SSE Integration for Channel Stats
  useEffect(() => {
    const eventHandlers = {
      'channel_stats_update': (data: ChannelStatData[]) => {
        console.log('SSE: Received channel_stats_update', data);
        setChannelData(data);
        setError(null); // Clear previous fetch error if live data arrives
      }
    };

    const sseCleanup = initializeSSE(API_ENDPOINTS.sseStatsStream, eventHandlers, {
      onError: (event) => {
        console.error("SSE connection error in ChannelStats:", event);
        if (!error) { // Only set SSE-specific error if no general fetch error exists
            setError("Live channel updates disconnected. Displaying last known data.");
        }
      }
    });

    return () => {
      sseCleanup();
    };
  }, [error]); // Dependency on error to potentially clear SSE specific error message

  if (loading) {
    return (
      <div className="bg-white shadow rounded-lg p-4 my-2">
        <h2 className="text-xl font-semibold mb-2 text-gray-700">Channel Statistics</h2>
        <p className="text-gray-500">Loading channel statistics...</p>
      </div>
    );
  }

  // Show error message, but still attempt to render table if mock data is present
  // This allows the UI structure to be visible even if live data fails

  return (
    <div className="bg-white shadow rounded-lg p-6 my-2">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 border-b pb-2">Channel Statistics</h2>
      {error && <p className="text-red-500 mb-4">Note: Could not load live data. Displaying zeros or last known values. {error}</p>}

      {channelData && channelData.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Channel
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Total Sent
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Failed
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Success Rate
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {channelData.map((channel) => (
                <tr key={channel.channelName}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{channel.channelName}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{channel.totalNotifications.toLocaleString()}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{channel.failedNotifications.toLocaleString()}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{channel.successRate.toFixed(2)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-gray-500">No channel statistics available.</p>
      )}
    </div>
  );
};

export default ChannelStats;
