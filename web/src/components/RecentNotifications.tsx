import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { API_ENDPOINTS } from '../config';

// Data structures
interface Notification {
  id: string;
  timestamp: string; // ISO date string
  channel: 'Email' | 'Slack' | 'SMS' | string; // Allow other channels
  status: 'success' | 'fail' | string; // Allow other statuses
  destination: string;
  retryStatus?: 'not_attempted' | 'pending' | 'failed' | 'succeeded' | string; // Optional
}

interface NotificationsApiResponse {
  notifications: Notification[];
  totalItems: number;
  totalPages: number;
  currentPage: number;
}

interface Filters {
  status: string;
  channel: string;
  search: string;
  fromDate: string;
  toDate: string;
}

const ITEMS_PER_PAGE = 10;

const RecentNotifications: React.FC = () => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [filters, setFilters] = useState<Filters>({
    status: '',
    channel: '',
    search: '',
    fromDate: '',
    toDate: '',
  });

  const [currentPage, setCurrentPage] = useState<number>(1);
  const [totalPages, setTotalPages] = useState<number>(0);

  const fetchNotifications = useCallback(async (page: number, currentFilters: Filters) => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (currentFilters.status) params.append('status', currentFilters.status);
      if (currentFilters.channel) params.append('channel', currentFilters.channel);
      if (currentFilters.search) params.append('search', currentFilters.search);
      if (currentFilters.fromDate) params.append('fromDate', currentFilters.fromDate);
      if (currentFilters.toDate) params.append('toDate', currentFilters.toDate);
      params.append('page', page.toString());
      params.append('limit', ITEMS_PER_PAGE.toString());

      const url = `${API_ENDPOINTS.notifications}?${params.toString()}`;
      const response = await axios.get<NotificationsApiResponse>(url);

      setNotifications(response.data.notifications);
      setTotalPages(response.data.totalPages);
      setCurrentPage(response.data.currentPage);

    } catch (err) {
      setError('Failed to fetch notifications. Please try again later.');
      console.error('Error fetching notifications:', err);
      setNotifications([]); // Clear notifications on error
       // Mock data for UI development in case of error
       setNotifications([
        { id: 'mock1', timestamp: new Date().toISOString(), channel: 'Email', status: 'success', destination: 'mock@example.com', retryStatus: 'not_attempted' },
        { id: 'mock2', timestamp: new Date().toISOString(), channel: 'Slack', status: 'fail', destination: '#mockchannel', retryStatus: 'failed' },
      ]);
      setTotalPages(1);
      setCurrentPage(1);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNotifications(currentPage, filters);
  }, [fetchNotifications, currentPage, filters]);

  const handleFilterChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFilters(prev => ({ ...prev, [name]: value }));
    setCurrentPage(1); // Reset to first page on filter change
  };

  const handleDateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    // Basic validation or formatting for date can be added here if needed
    setFilters(prev => ({ ...prev, [name]: value }));
    setCurrentPage(1);
  };

  const handleSearch = () => {
    // This function is implicitly called when filters state changes due to search input
    // Alternatively, a dedicated search button could call:
    fetchNotifications(1, filters);
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch (e) {
      return 'Invalid Date';
    }
  };

  return (
    <div className="bg-white shadow rounded-lg p-6 my-2">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 border-b pb-2">Recent Notifications</h2>

      {/* Filters Section */}
      <div className="mb-6 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 items-end">
        <div>
          <label htmlFor="channel-filter" className="block text-sm font-medium text-gray-700">Channel</label>
          <select id="channel-filter" name="channel" value={filters.channel} onChange={handleFilterChange} className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
            <option value="">All</option>
            <option value="Email">Email</option>
            <option value="Slack">Slack</option>
            {/* Add other channels as needed */}
          </select>
        </div>
        <div>
          <label htmlFor="status-filter" className="block text-sm font-medium text-gray-700">Status</label>
          <select id="status-filter" name="status" value={filters.status} onChange={handleFilterChange} className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
            <option value="">All</option>
            <option value="success">Success</option>
            <option value="fail">Fail</option>
          </select>
        </div>
        <div>
          <label htmlFor="fromDate-filter" className="block text-sm font-medium text-gray-700">From Date</label>
          <input type="date" id="fromDate-filter" name="fromDate" value={filters.fromDate} onChange={handleDateChange} className="mt-1 focus:ring-indigo-500 focus:border-indigo-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md p-2"/>
        </div>
        <div>
          <label htmlFor="toDate-filter" className="block text-sm font-medium text-gray-700">To Date</label>
          <input type="date" id="toDate-filter" name="toDate" value={filters.toDate} onChange={handleDateChange} className="mt-1 focus:ring-indigo-500 focus:border-indigo-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md p-2"/>
        </div>
        <div className="lg:col-span-1"> {/* Search input can take full width on smaller screens if needed or adjust layout */}
          <label htmlFor="search-filter" className="block text-sm font-medium text-gray-700">Search</label>
          <input
            type="text"
            id="search-filter"
            name="search"
            value={filters.search}
            onChange={handleFilterChange} // Triggers refetch on input change
            placeholder="Destination or ID..."
            className="mt-1 focus:ring-indigo-500 focus:border-indigo-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md p-2"
          />
           {/* A search button could be added here if debouncing or explicit search trigger is preferred */}
        </div>
      </div>

      {error && <p className="text-red-500 mb-4">Note: Could not load live data. Displaying mock data. {error}</p>}

      {/* Table Section */}
      {loading ? (
        <p className="text-gray-500">Loading notifications...</p>
      ) : (
        <>
          <div className="overflow-x-auto mb-4">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Channel</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Destination</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Retry Status</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {notifications.length > 0 ? notifications.map((n) => (
                  <tr key={n.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{formatDate(n.timestamp)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{n.channel}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        n.status === 'success' ? 'bg-green-100 text-green-800' :
                        n.status === 'fail' ? 'bg-red-100 text-red-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {n.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 truncate" title={n.destination}>{n.destination}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{n.retryStatus || 'N/A'}</td>
                  </tr>
                )) : (
                  <tr>
                    <td colSpan={5} className="px-6 py-4 text-center text-sm text-gray-500">No notifications found matching your criteria.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination Controls */}
          {totalPages > 0 && (
            <div className="flex justify-between items-center mt-4">
              <button
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage === 1 || loading}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-gray-700">
                Page {currentPage} of {totalPages}
              </span>
              <button
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages || loading}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default RecentNotifications;
