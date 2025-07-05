import React from 'react';
import './App.css';
import SummaryStats from './components/SummaryStats';
import ChannelStats from './components/ChannelStats';
import RecentNotifications from './components/RecentNotifications';
import TrendsGraph from './components/TrendsGraph';
import FailedNotifications from './components/FailedNotifications';

function App() {
  return (
    <div className="min-h-screen bg-gray-100 text-gray-800">
      <header className="bg-blue-600 text-white shadow-md">
        <div className="container mx-auto p-4">
          <h1 className="text-2xl font-bold">Notification Service Dashboard</h1>
        </div>
      </header>

      <main className="container mx-auto p-4">
        {/* Summary Section */}
        <section aria-labelledby="summary-stats-heading">
          <h2 id="summary-stats-heading" className="sr-only">Summary Statistics</h2>
          <SummaryStats />
        </section>

        {/* Channel Statistics Section */}
        <section aria-labelledby="channel-stats-heading" className="mt-6">
          <h2 id="channel-stats-heading" className="sr-only">Channel Statistics</h2>
          <ChannelStats />
        </section>

        {/* Recent Notifications Table */}
        <section aria-labelledby="recent-notifications-heading" className="mt-6">
          <h2 id="recent-notifications-heading" className="sr-only">Recent Notifications</h2>
          <RecentNotifications />
        </section>

        {/* Trends Over Time Section */}
        <section aria-labelledby="trends-heading" className="mt-6">
          <h2 id="trends-heading" className="sr-only">Trends Over Time</h2>
          <TrendsGraph />
        </section>

        {/* Failed Notifications Panel */}
        <section aria-labelledby="failed-notifications-heading" className="mt-6">
          <h2 id="failed-notifications-heading" className="sr-only">Failed Notifications</h2>
          <FailedNotifications />
        </section>
      </main>

      <footer className="bg-gray-200 text-gray-600 p-4 text-center mt-8">
        <p>&copy; {new Date().getFullYear()} Notification Service. All rights reserved.</p>
      </footer>
    </div>
  );
}

export default App;
