import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { API_ENDPOINTS } from '../config';
import { initializeSSE } from '../utils/sseHelper';

interface SummaryData {
  totalNotifications: number;
  failedNotifications: number;
  notificationsInQueue: number;
  averageDeliveryTime: number; // Assuming in seconds
  successRate: number; // Assuming percentage
}

const SummaryStats: React.FC = () => {
  const [summaryData, setSummaryData] = useState<SummaryData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSummaryData = async () => {
      try {
        setLoading(true);
        const response = await axios.get<SummaryData>(API_ENDPOINTS.summaryStats);
        setSummaryData(response.data);
        setError(null);
      } catch (err) {
        setError('Failed to fetch summary data. Please try again later.');
        console.error('Error fetching summary data:', err);
        // Keep placeholder data or show error state
        // For now, let's set some mock data on error for UI development
        setSummaryData({
            totalNotifications: 0,
            failedNotifications: 0,
            notificationsInQueue: 0,
            averageDeliveryTime: 0,
            successRate: 0
        });
      } finally {
        setLoading(false);
      }
    };

    fetchSummaryData();
  }, []);

  // SSE Integration
  useEffect(() => {
    const eventHandlers = {
      'summary_update': (data: SummaryData) => {
        console.log('SSE: Received summary_update', data);
        setSummaryData(data);
        // If live data comes in, clear any previous fetch error
        setError(null);
        // Potentially set loading to false if initial load was waiting only for SSE
        // However, with initial fetch, loading is handled by that.
      }
    };

    const sseCleanup = initializeSSE(API_ENDPOINTS.sseStatsStream, eventHandlers, {
      onError: (event) => {
        console.error("SSE connection error in SummaryStats:", event);
        // Optionally, set an error state specific to SSE if needed,
        // or rely on the initial fetch's error handling for stale data.
        // If the initial fetch succeeded, the dashboard will show stale data.
        // If it failed, it will show mock data + initial fetch error.
        // We could add a specific "Live updates disconnected" message.
        if (!error) { // Only set SSE-specific error if no general fetch error exists
            setError("Live updates disconnected. Displaying last known data.");
        }
      }
    });

    return () => {
      sseCleanup();
    };
    // Rerun effect if API_ENDPOINTS.sseStatsStream changes (though unlikely for a const)
    // Or if we wanted to dynamically change event handlers, but not the case here.
  }, [error]); // Added error to dependency array to potentially re-establish connection or clear error message

  const StatCard: React.FC<{ title: string; value: string | number; unit?: string }> = ({ title, value, unit }) => (
    <div className="bg-gray-50 p-4 rounded-lg shadow">
      <h3 className="text-sm font-medium text-gray-500 truncate">{title}</h3>
      <p className="mt-1 text-3xl font-semibold text-gray-900">
        {value}
        {unit && <span className="text-sm font-medium text-gray-500 ml-1">{unit}</span>}
      </p>
    </div>
  );

  if (loading) {
    return (
      <div className="bg-white shadow rounded-lg p-4 my-2">
        <h2 className="text-xl font-semibold mb-2 text-gray-700">Summary Statistics</h2>
        <p className="text-gray-500">Loading summary data...</p>
      </div>
    );
  }

  if (error && !summaryData) { // Only show full error if no data (even mock) is available
    return (
      <div className="bg-white shadow rounded-lg p-4 my-2">
        <h2 className="text-xl font-semibold mb-2 text-red-600">Error</h2>
        <p className="text-red-500">{error}</p>
      </div>
    );
  }

  // If there's an error but we have mock data (from the catch block), we can still render the cards
  // A more sophisticated approach might involve specific UI for "stale" or "error" data

  return (
    <div className="bg-white shadow rounded-lg p-6 my-2">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 border-b pb-2">Summary Statistics</h2>
      {error && <p className="text-red-500 mb-4">Note: Could not load live data. Displaying zeros or last known values. {error}</p>}
      {summaryData ? (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          <StatCard title="Total Notifications" value={summaryData.totalNotifications.toLocaleString()} />
          <StatCard title="Failed Notifications" value={summaryData.failedNotifications.toLocaleString()} />
          <StatCard title="In Queue" value={summaryData.notificationsInQueue.toLocaleString()} />
          <StatCard title="Avg. Delivery Time" value={summaryData.averageDeliveryTime.toFixed(1)} unit="s" />
          <StatCard title="Success Rate" value={summaryData.successRate.toFixed(1)} unit="%" />
        </div>
      ) : (
        // This case should ideally not be reached if loading is false and error handling sets mock data
        <p className="text-gray-500">No summary data available.</p>
      )}
    </div>
  );
};

export default SummaryStats;
