const API_BASE_URL = '/api'; // Assuming the API is served from the same domain

export const API_ENDPOINTS = {
  summaryStats: `${API_BASE_URL}/stats/summary`,
  channelStats: `${API_BASE_URL}/stats/channels`,
  notifications: `${API_BASE_URL}/notifications`, // Base for general notifications, params will be added
  failedNotifications: `${API_BASE_URL}/notifications/failed`,
  retryNotification: (id: string) => `${API_BASE_URL}/notifications/${id}/retry`,
  timeSeriesStats: `${API_BASE_URL}/stats/timeseries`,
  sseStatsStream: `${API_BASE_URL}/stats/stream`
};

// You can also add other configurations here, like default headers, timeouts, etc.
// For example:
// export const AXIOS_CONFIG = {
//   timeout: 10000, // 10 seconds
// };
