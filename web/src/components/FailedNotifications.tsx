import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { API_ENDPOINTS } from '../config';

interface FailedNotification {
  id: string;
  timestamp: string;
  channel: string;
  status: 'fail'; // Should always be 'fail'
  destination: string;
  errorMessage?: string; // Optional, but expected for failed notifications
  retryStatus?: string; // e.g., 'pending', 'max_attempts_reached'
  // Add any other relevant fields from your API
}

// Mock data for UI development
const mockFailedNotifications: FailedNotification[] = [
  { id: 'fail_mock_1', timestamp: new Date(Date.now() - 3600000).toISOString(), channel: 'Email', status: 'fail', destination: 'test1@example.com', errorMessage: 'SMTP Connection Timeout', retryStatus: 'max_attempts_reached' },
  { id: 'fail_mock_2', timestamp: new Date(Date.now() - 7200000).toISOString(), channel: 'Slack', status: 'fail', destination: '#general', errorMessage: 'Invalid token', retryStatus: 'pending' },
  { id: 'fail_mock_3', timestamp: new Date(Date.now() - 10800000).toISOString(), channel: 'SMS', status: 'fail', destination: '+1234567890', errorMessage: 'Insufficient balance' },
];


const FailedNotifications: React.FC = () => {
  const [failedNotifications, setFailedNotifications] = useState<FailedNotification[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [retryingIds, setRetryingIds] = useState<Set<string>>(new Set()); // Track IDs being retried

  const fetchFailedNotifications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get<FailedNotification[]>(API_ENDPOINTS.failedNotifications);
      setFailedNotifications(response.data);
    } catch (err) {
      setError('Failed to fetch failed notifications. Displaying mock data.');
      console.error('Error fetching failed notifications:', err);
      setFailedNotifications(mockFailedNotifications); // Use mock data on error
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFailedNotifications();
  }, [fetchFailedNotifications]);

  const handleRetry = async (notificationId: string) => {
    if (retryingIds.has(notificationId)) return; // Already retrying

    setRetryingIds(prev => new Set(prev).add(notificationId));
    try {
      await axios.post(API_ENDPOINTS.retryNotification(notificationId));
      // If retry is successful, you might want to:
      // 1. Show a success message (e.g., using a toast notification library)
      // 2. Refetch the list of failed notifications to update its status or remove it if it succeeded.
      //    Or, optimistically update the UI for this item.
      alert(`Retry initiated for notification ${notificationId}.`); // Simple feedback
      fetchFailedNotifications(); // Refetch to update the list
    } catch (err) {
      console.error(`Error retrying notification ${notificationId}:`, err);
      alert(`Failed to retry notification ${notificationId}.`); // Simple error feedback
      // Potentially update the notification's retryStatus in the local state if the API provides it
    } finally {
      setRetryingIds(prev => {
        const next = new Set(prev);
        next.delete(notificationId);
        return next;
      });
    }
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch (e) {
      return 'Invalid Date';
    }
  };

  if (loading && failedNotifications.length === 0) {
    return (
      <div className="bg-white shadow rounded-lg p-4 my-2">
        <h2 className="text-xl font-semibold mb-2 text-gray-700">Failed Notifications</h2>
        <p className="text-gray-500">Loading failed notifications...</p>
      </div>
    );
  }

  return (
    <div className="bg-white shadow rounded-lg p-6 my-2">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 border-b pb-2">Failed Notifications Panel</h2>
      {error && <p className="text-red-500 mb-4">{error}</p>}

      {failedNotifications.length === 0 && !loading ? (
        <p className="text-gray-500">No failed notifications found.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Channel</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Destination</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Error Message</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Retry Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {failedNotifications.map((n) => (
                <tr key={n.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{formatDate(n.timestamp)}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{n.channel}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 truncate" title={n.destination}>{n.destination}</td>
                  <td className="px-6 py-4 text-sm text-red-600" title={n.errorMessage}>{n.errorMessage || 'No error message provided.'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{n.retryStatus || 'N/A'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button
                      onClick={() => handleRetry(n.id)}
                      disabled={retryingIds.has(n.id) || n.retryStatus === 'max_attempts_reached'}
                      className="text-indigo-600 hover:text-indigo-900 disabled:text-gray-400 disabled:cursor-not-allowed"
                    >
                      {retryingIds.has(n.id) ? 'Retrying...' : 'Retry'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default FailedNotifications;
