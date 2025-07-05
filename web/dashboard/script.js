document.addEventListener('DOMContentLoaded', () => {
    const themeToggle = document.getElementById('theme-toggle');
    const currentTheme = localStorage.getItem('theme') || 'light';

    if (currentTheme === 'dark') {
        document.body.classList.add('dark-mode');
        document.body.classList.remove('light-mode');
    } else {
        document.body.classList.add('light-mode');
        document.body.classList.remove('dark-mode');
    }

    themeToggle.addEventListener('click', () => {
        document.body.classList.toggle('dark-mode');
        document.body.classList.toggle('light-mode');

        let theme = 'light';
        if (document.body.classList.contains('dark-mode')) {
            theme = 'dark';
        }
        localStorage.setItem('theme', theme);
    });

    // Placeholder data for charts is now removed or will be replaced by API calls.
    // const channelData = { ... };

    // Chart instances to be updated
    let totalNotificationsChartInstance, failedNotificationsChartInstance, successRateChartInstance;
    let volumeTrendChartInstance, failureRateTrendChartInstance, queueSizeTrendChartInstance;
    // failureReasonsChartInstance is already declared globally in its section

    // Chart.js configurations
    const defaultChartOptions = (isDarkMode) => ({
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            y: {
                beginAtZero: true,
                ticks: {
                    color: isDarkMode ? '#e0e0e0' : '#333'
                },
                grid: {
                    color: isDarkMode ? 'rgba(224, 224, 224, 0.2)' : 'rgba(0, 0, 0, 0.1)'
                }
            },
            x: {
                ticks: {
                    color: isDarkMode ? '#e0e0e0' : '#333'
                },
                grid: {
                    color: isDarkMode ? 'rgba(224, 224, 224, 0.2)' : 'rgba(0, 0, 0, 0.1)'
                }
            }
        },
        plugins: {
            legend: {
                labels: {
                    color: isDarkMode ? '#e0e0e0' : '#333'
                }
            }
        }
    });

    const barChartColors = ['#36A2EB', '#FF6384', '#FFCE56']; // Blue, Red, Yellow

    async function updateGlobalStatsDisplay() {
        let url = '/api/dashboard/global-stats';
        const dateParams = getDateRangeQueryString();
        if (dateParams) url += `?${dateParams}`;

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            const stats = await response.json();

            document.getElementById('total-sent').textContent = stats.totalSent?.toLocaleString() || '0';
            document.getElementById('total-failed').textContent = stats.totalFailed?.toLocaleString() || '0';
            document.getElementById('in-queue').textContent = stats.notificationsInQueue?.toLocaleString() || '0';
            document.getElementById('avg-delivery-time').textContent = `${stats.avgDeliveryTimeSec?.toFixed(1) || '0'}s`;
            document.getElementById('success-rate').textContent = `${stats.successRate?.toFixed(1) || '0'}%`;
        } catch (error) {
            console.error("Error fetching global stats:", error);
            // Display error or placeholder text in UI
            document.getElementById('total-sent').textContent = 'Error';
            // ... and for other stats elements
        }
    }

    async function updateChannelStatsCharts() {
        let url = '/api/dashboard/channel-stats';
        const dateParams = getDateRangeQueryString();
        if (dateParams) url += `?${dateParams}`;

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            const stats = await response.json(); // Expected: Array of DashboardChannelStat

            const labels = stats.map(s => s.channel);
            const totalData = stats.map(s => s.total);
            const failedData = stats.map(s => s.failed);
            const successRateData = stats.map(s => s.successRate);
            const isDarkMode = document.body.classList.contains('dark-mode');

            // Total Notifications Chart
            const totalCtx = document.getElementById('totalNotificationsChart')?.getContext('2d');
            if (totalCtx) {
                if (totalNotificationsChartInstance) totalNotificationsChartInstance.destroy();
                totalNotificationsChartInstance = new Chart(totalCtx, {
                    type: 'bar',
                    data: { labels, datasets: [{ label: 'Total Notifications', data: totalData, backgroundColor: barChartColors }] },
                    options: defaultChartOptions(isDarkMode)
                });
            }

            // Failed Notifications Chart
            const failedCtx = document.getElementById('failedNotificationsChart')?.getContext('2d');
            if (failedCtx) {
                if (failedNotificationsChartInstance) failedNotificationsChartInstance.destroy();
                failedNotificationsChartInstance = new Chart(failedCtx, {
                    type: 'bar',
                    data: { labels, datasets: [{ label: 'Failed Notifications', data: failedData, backgroundColor: barChartColors }] },
                    options: defaultChartOptions(isDarkMode)
                });
            }

            // Success Rate Chart
            const successRateCtx = document.getElementById('successRateChart')?.getContext('2d');
            if (successRateCtx) {
                if (successRateChartInstance) successRateChartInstance.destroy();
                successRateChartInstance = new Chart(successRateCtx, {
                    type: 'bar',
                    data: { labels, datasets: [{ label: 'Success Rate (%)', data: successRateData, backgroundColor: barChartColors }] },
                    options: { ...defaultChartOptions(isDarkMode), scales: { ...defaultChartOptions(isDarkMode).scales, y: { ...defaultChartOptions(isDarkMode).scales.y, min: 0, max: 100 } } } // Min 0, Max 100 for percentage
                });
            }
        } catch (error) {
            console.error("Error fetching channel stats:", error);
        }
    }

    function getDateRangeQueryString() {
        if (dateRangePicker && dateRangePicker._flatpickr && dateRangePicker._flatpickr.selectedDates.length > 0) {
            const from = dateRangePicker._flatpickr.formatDate(dateRangePicker._flatpickr.selectedDates[0], "Y-m-d");
            let params = `from=${from}`;
            if (dateRangePicker._flatpickr.selectedDates.length === 2) {
                const to = dateRangePicker._flatpickr.formatDate(dateRangePicker._flatpickr.selectedDates[1], "Y-m-d");
                params += `&to=${to}`;
            }
            return params;
        }
        return "";
    }


    // --- Recent Activity Timeline ---
    const activityTimeline = document.getElementById('activity-timeline');
    const activityChannelFilter = document.getElementById('activity-channel-filter');
    const activityStatusFilter = document.getElementById('activity-status-filter');
    const searchInput = document.getElementById('search-recipient-message');
    const dateRangePicker = document.getElementById('date-range-picker');

    // Initialize Flatpickr for date range selection
    if (dateRangePicker) {
        flatpickr(dateRangePicker, {
            mode: "range",
            dateFormat: "Y-m-d",
            onChange: function(selectedDates, dateStr, instance) {
                // This function will be called when a date range is selected
                // Trigger filtering based on selectedDates
                console.log("Date range selected:", selectedDates);
                renderActivityTimeline(recentActivityData); // Re-render with new date filter
            },
            // For dark mode, Flatpickr has its own dark theme
            // theme: document.body.classList.contains('dark-mode') ? "dark" : "light" // or use their specific dark theme class
        });
    }
     // Update Flatpickr theme on body class change (e.g., when toggling dark/light mode)
    const observer = new MutationObserver(mutations => {
        mutations.forEach(mutation => {
            if (mutation.attributeName === 'class' && dateRangePicker && dateRangePicker._flatpickr) {
                const isDarkMode = document.body.classList.contains('dark-mode');
                // Flatpickr's dark theme is often applied by adding a class to the calendar instance or its container
                // This might require more specific handling depending on how flatpickr themes are applied/toggled
                // For now, we'll log it. A robust solution might involve re-initializing or directly manipulating theme classes.
                console.log(`Theme changed. Dark mode: ${isDarkMode}. Flatpickr might need theme update.`);
                // Example: dateRangePicker._flatpickr.set('theme', isDarkMode ? 'dark' : 'light'); // This specific method may not exist.
            }
        });
    });
    observer.observe(document.body, { attributes: true });


    // Placeholder recent activity data is now removed, will be fetched from API.
    // const recentActivityData = [ ... ];

    let timelineCurrentPage = 1;
    const timelineItemsPerPage = 20; // Or whatever is desired

    function getChannelIcon(channel) {
        // Ensure channel is not null or undefined before calling toLowerCase
        const lowerChannel = channel ? channel.toLowerCase() : 'unknown';
        if (lowerChannel === 'slack') return '<i class="fab fa-slack channel-icon-slack"></i>';
        if (lowerChannel === 'email') return '<i class="fas fa-envelope channel-icon-email"></i>';
        if (lowerChannel === 'telegram') return '<i class="fab fa-telegram-plane channel-icon-telegram"></i>';
        return '<i class="fas fa-question-circle"></i>';
    }

    async function renderActivityTimeline(page = 1) {
        if (!activityTimeline) return;
        activityTimeline.innerHTML = '<li>Loading activity...</li>'; // Show loading state
        timelineCurrentPage = page;

        // Fetch data using the new function
        const apiResponse = await fetchActivityDataForTimeline(timelineCurrentPage, timelineItemsPerPage);

        activityTimeline.innerHTML = ''; // Clear loading or previous items

        if (!apiResponse.notifications || apiResponse.notifications.length === 0) {
            activityTimeline.innerHTML = '<li>No recent activity matching your filters.</li>';
            renderTimelinePagination(0, timelineCurrentPage, timelineItemsPerPage);
            return;
        }

        apiResponse.notifications.forEach(item => {
            const li = document.createElement('li');
            const displayTimestamp = item.last_attempt_at || item.created_at;
            const formattedTimestamp = displayTimestamp ? new Date(displayTimestamp).toLocaleString() : 'N/A';
            const statusClass = item.status ? item.status.toLowerCase() : 'unknown';

            li.innerHTML = `
                <span class="timestamp">${formattedTimestamp}</span>
                <span class="channel">
                    <span class="channel-icon">${getChannelIcon(item.service_id)}</span>
                    ${item.service_id || 'N/A'}
                </span>
                <span class="destination">${item.recipient_info || 'N/A'}</span>
                <span class="status ${statusClass}">${item.status || 'Unknown'}</span>
            `;
            if (item.error_message) {
                 li.title = `Error: ${item.error_message}`; // Show error on hover
            }
            activityTimeline.appendChild(li);
        });
        renderTimelinePagination(apiResponse.totalCount, timelineCurrentPage, timelineItemsPerPage);
    }

    function renderTimelinePagination(totalItems, currentPage, itemsPerPage) {
        const paginationContainer = document.getElementById('timeline-pagination') || createTimelinePaginationContainer();
        paginationContainer.innerHTML = ''; // Clear previous pagination

        if (totalItems === 0) return;
        const totalPages = Math.ceil(totalItems / itemsPerPage);
        if (totalPages <= 1) return;

        const createButton = (text, pageNum, isDisabled = false) => {
            const button = document.createElement('button');
            button.textContent = text;
            button.disabled = isDisabled;
            button.addEventListener('click', () => renderActivityTimeline(pageNum));
            return button;
        };

        paginationContainer.appendChild(createButton('Previous', currentPage - 1, currentPage === 1));

        const pageInfo = document.createElement('span');
        pageInfo.textContent = ` Page ${currentPage} of ${totalPages} `;
        pageInfo.style.margin = "0 10px"; // Add some spacing
        paginationContainer.appendChild(pageInfo);

        paginationContainer.appendChild(createButton('Next', currentPage + 1, currentPage === totalPages));
    }

    function createTimelinePaginationContainer() {
        let container = document.getElementById('timeline-pagination');
        if (!container) {
            container = document.createElement('div');
            container.id = 'timeline-pagination';
            container.className = 'pagination-controls';
            const recentActivitySection = document.getElementById('recent-activity');
            // Insert after the timeline-container but before any subsequent sections if possible
            const timelineCont = recentActivitySection?.querySelector('.timeline-container');
            if (timelineCont && timelineCont.parentNode) {
                 timelineCont.parentNode.insertBefore(container, timelineCont.nextSibling);
            } else {
                recentActivitySection?.appendChild(container);
            }
        }
        return container;
    }

    // Initial render call - this will now fetch data
    renderActivityTimeline(1);

    // Event listeners for filters - now they trigger a re-render from page 1
    activityChannelFilter?.addEventListener('change', () => renderActivityTimeline(1));
    activityStatusFilter?.addEventListener('change', () => renderActivityTimeline(1));

    let searchDebounceTimeout;
    searchInput?.addEventListener('input', () => {
        clearTimeout(searchDebounceTimeout);
        searchDebounceTimeout = setTimeout(() => renderActivityTimeline(1), 300);
    });

    // Adjust Flatpickr's onChange to call the new renderActivityTimeline and update other data
    if (dateRangePicker && dateRangePicker._flatpickr) {
        const existingOnChange = dateRangePicker._flatpickr.config.onChange;
        dateRangePicker._flatpickr.config.onChange = [function(selectedDates, dateStr, instance) {
            // Call existing handlers if any (like the console.log from initial setup)
            existingOnChange.forEach(fn => fn(selectedDates, dateStr, instance));

            console.log("Global Date range selected:", selectedDates);
            // Re-render timeline and update all data that depends on the global date range
            renderActivityTimeline(1);
            updateDashboardData(); // New function to refresh all date-dependent data
        }];
    }

    // Function to update all data sections that depend on the global date range
    function updateDashboardData() {
        updateGlobalStatsDisplay();
        updateChannelStatsCharts();
        updateAllHistoricalTrendCharts();
        // Data for open tabs might also need refresh if they are date-sensitive
        // For instance, if 'Recent Failures' tab is open, re-populate it.
        const activeTab = document.querySelector('.tab-link.active');
        if (activeTab) {
            const activeTabName = activeTab.getAttribute('onclick').match(/'([^']+)'/)[1];
            if (activeTabName === 'failed-notifications') {
                populateRecentFailedNotifications();
            } else if (activeTabName === 'failure-reasons') {
                renderFailureReasonsChart();
            }
            // Retry queue is less likely to be date-range filtered, but could be if desired.
        }
    }

    // Initial data load
    function initialLoad() {
        renderActivityTimeline(1); // Fetches initial timeline page
        updateDashboardData(); // Fetches data for all other sections

        // Populate the initially active tab's content if it requires async data
        // By default, 'failed-notifications' is active.
        const activeTab = document.querySelector('.tab-link.active');
        if (activeTab) {
            const activeTabName = activeTab.getAttribute('onclick').match(/'([^']+)'/)[1];
            if (activeTabName === 'failed-notifications') {
                populateRecentFailedNotifications();
            }
            // renderFailureReasonsChart is called when tab is clicked, so not strictly needed here unless it's the default active
        }
    }

    initialLoad();


    // --- Historical Trends Charts ---
    const volumeTrendCtx = document.getElementById('volumeTrendChart')?.getContext('2d');
    const failureRateTrendCtx = document.getElementById('failureRateTrendChart')?.getContext('2d');
    const queueSizeTrendCtx = document.getElementById('queueSizeTrendChart')?.getContext('2d');

    // Placeholder data for historical trends is removed.

    const lineChartOptions = (isDarkMode, titleText = '') => ({ // Removed title param, use titleText for specific needs
        ...defaultChartOptions(isDarkMode),
        plugins: {
            ...defaultChartOptions(isDarkMode).plugins,
            title: {
                display: false, // Title is in H3 above canvas
                text: title,
                color: isDarkMode ? '#e0e0e0' : '#333'
            }
        },
        tension: 0.1 // Makes line charts slightly curved
    });

    async function updateHistoricalTrendChart(ctx, instanceVariable, apiUrlEndpoint, label, dataField, borderColor, period = 'daily') {
        if (!ctx) return;

        let url = `${apiUrlEndpoint}?period=${period}`;
        const dateParams = getDateRangeQueryString(); // Uses global date picker
        if (dateParams) {
            url += `&${dateParams}`; // Assumes API handles 'from' and 'to' for trends
        } else {
            // Default to last 7 days if no global date range is set
            const endDate = new Date();
            const startDate = new Date();
            startDate.setDate(endDate.getDate() - 7);
            url += `&startDate=${startDate.toISOString().split('T')[0]}&endDate=${endDate.toISOString().split('T')[0]}`;
        }

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status} for ${apiUrlEndpoint}`);
            const trendData = await response.json(); // Expected: Array of DashboardHistoricalPoint

            const labels = trendData.map(p => p.date);
            const data = trendData.map(p => p[dataField]);
            const isDarkMode = document.body.classList.contains('dark-mode');

            if (instanceVariable) instanceVariable.destroy();

            const chartConfig = {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: label,
                        data: data,
                        borderColor: borderColor,
                        backgroundColor: `${borderColor}33`, // Add some transparency
                        fill: true,
                        tension: 0.1
                    }]
                },
                options: lineChartOptions(isDarkMode)
            };

            // Specific Y-axis for failure rate (0-100)
            if (dataField === 'failures' && label.includes('%')) { // Assuming 'failures' here means rate for the 'failureRate' endpoint
                 chartConfig.options.scales.y = { ...chartConfig.options.scales.y, min: 0, max: 100 };
            }


            return new Chart(ctx, chartConfig);

        } catch (error) {
            console.error(`Error fetching ${label} trend:`, error);
            ctx.fillText("Error loading chart.", 10, 50); // Basic error display on canvas
            return null;
        }
    }

    async function updateAllHistoricalTrendCharts() {
        volumeTrendChartInstance = await updateHistoricalTrendChart(volumeTrendCtx, volumeTrendChartInstance, '/api/dashboard/historical-trends/volume', 'Notifications Sent', 'volume', '#36A2EB');
        // For failure rate, the API might return 'failures' count and 'volume' (total attempts) per point, or directly a rate.
        // Assuming API for failure-rate returns points with a 'value' field representing the rate.
        // Or if it returns 'failures' and 'volume', the chart label/data extraction needs adjustment.
        // Let's assume for now the API '/api/dashboard/historical-trends/failure-rate' returns points with a 'failures' field that represents the rate itself for simplicity, or that the Y-axis should just show failure counts.
        // If it's a rate, the Y-axis should be 0-100.
        failureRateTrendChartInstance = await updateHistoricalTrendChart(failureRateTrendCtx, failureRateTrendChartInstance, '/api/dashboard/historical-trends/failure-rate', 'Failure Rate (%)', 'failures', '#FF6384');
        queueSizeTrendChartInstance = await updateHistoricalTrendChart(queueSizeTrendCtx, queueSizeTrendChartInstance, '/api/dashboard/historical-trends/queue-size', 'Queue Size', 'queueSize', '#FFCE56', 'hourly'); // Default to hourly for queue
    }


    // --- Optional Panels - Tabbed Content ---
    window.openTab = function(event, tabName) {
        let i, tabcontent, tablinks;
        tabcontent = document.getElementsByClassName("tab-content");
        for (i = 0; i < tabcontent.length; i++) {
            tabcontent[i].style.display = "none";
        }
        tablinks = document.getElementsByClassName("tab-link");
        for (i = 0; i < tablinks.length; i++) {
            tablinks[i].className = tablinks[i].className.replace(" active", "");
        }
        document.getElementById(tabName).style.display = "block";
        event.currentTarget.className += " active";

        // If the tab contains a chart that needs to be rendered or updated, do it here
        if (tabName === 'failure-reasons') { // Render/update chart each time tab is opened
            renderFailureReasonsChart();
        } else if (tabName === 'failed-notifications') {
            populateRecentFailedNotifications();
        } else if (tabName === 'retry-queue') {
            populateRetryQueueInfo();
        }
        // Templates tab remains placeholder for now
    }

    async function populateRecentFailedNotifications() {
        const failedNotificationsList = document.getElementById('failed-notifications-list');
        if (!failedNotificationsList) return;
        failedNotificationsList.innerHTML = '<li>Loading recent failures...</li>';

        let url = '/api/dashboard/recent-activity?status=failed&limit=10'; // Fetch 10 most recent failed
        const dateParams = getDateRangeQueryString(); // Apply global date filter if set
        if (dateParams) url += `&${dateParams}`;

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            const data = await response.json(); // Expects { notifications: [], ... }

            failedNotificationsList.innerHTML = ''; // Clear loading message
            const recentFailures = data.notifications;

            if (recentFailures && recentFailures.length > 0) {
                recentFailures.forEach(item => {
                    const li = document.createElement('li');
                    const displayTimestamp = item.last_attempt_at || item.created_at;
                    const formattedTimestamp = displayTimestamp ? new Date(displayTimestamp).toLocaleString() : 'N/A';
                    li.innerHTML = `
                        <span>${formattedTimestamp} - <b>${item.service_id}</b> to ${item.recipient_info || 'N/A'}</span>
                        <span class="error-message" title="${item.error_message || ''}">${item.error_message || 'Unknown error'}</span>
                    `;
                    failedNotificationsList.appendChild(li);
                });
            } else {
                failedNotificationsList.innerHTML = '<li>No recent failed notifications found.</li>';
            }
        } catch (error) {
            console.error("Error fetching recent failed notifications:", error);
            failedNotificationsList.innerHTML = '<li>Error loading recent failures.</li>';
        }
    }


    // Populate Notification Templates (Placeholder - as data source is unclear from backend)
    const templatesList = document.getElementById('templates-list');
    if (templatesList) {
        const templates = [
            { name: 'Welcome Email', usage: 1200, channel: 'Email' },
            { name: 'Password Reset', usage: 300, channel: 'Email' },
            { name: 'Slack Daily Digest', usage: 50, channel: 'Slack' },
            { name: 'Order Confirmation', usage: 800, channel: 'Telegram' }
        ];
        templates.forEach(template => {
            const li = document.createElement('li');
            li.innerHTML = `<span>${template.name} (${template.channel})</span> <span>Used ${template.usage} times</span>`;
            templatesList.appendChild(li);
        });
    }

    async function populateRetryQueueInfo() {
        const retryQueueSizeEl = document.getElementById('retry-queue-size');
        const retryQueueList = document.getElementById('retry-queue-list');
        if (!retryQueueSizeEl || !retryQueueList) return;

        retryQueueList.innerHTML = '<li>Loading retry queue...</li>';

        // Option 1: Use global stats for size, then fetch items if size > 0
        // For simplicity, let's assume global stats endpoint provides queue size.
        // And a separate call to recent-activity for items in 'queued' or 'processing'

        try {
            // Get queue size from global stats (already fetched or fetch again if needed)
            // This part assumes global stats are fresh or we re-fetch.
            // For this example, let's assume it's reflected in the displayed global stat.
            // A more robust way: fetch `/api/dashboard/global-stats` if not recently fetched.
            const queueSizeFromGlobal = parseInt(document.getElementById('in-queue').textContent) || 0;
            retryQueueSizeEl.textContent = queueSizeFromGlobal.toLocaleString();

            if (queueSizeFromGlobal > 0) {
                 // Fetch actual items in queue
                let url = `/api/dashboard/recent-activity?status=queued&limit=20`; // Also fetch 'processing' if that's part of retry
                // Consider if date range should apply to "items in queue" view
                // const dateParams = getDateRangeQueryString();
                // if (dateParams) url += `&${dateParams}`;

                const response = await fetch(url);
                if (!response.ok) throw new Error (`HTTP error! status: ${response.status} for queued items`);
                const data = await response.json();
                const queuedItems = data.notifications;

                retryQueueList.innerHTML = '';
                if (queuedItems && queuedItems.length > 0) {
                    queuedItems.forEach(item => {
                        const li = document.createElement('li');
                        const createdAt = new Date(item.created_at).toLocaleString();
                        li.innerHTML = `<span>${item.id} (${item.service_id} to ${item.recipient_info || 'N/A'}) - Queued: ${createdAt}</span> <span>Attempts: ${item.attempts}</span>`;
                        retryQueueList.appendChild(li);
                    });
                } else if (queueSizeFromGlobal > 0) { // Size from global stats but no items fetched (e.g. only 'processing')
                    retryQueueList.innerHTML = '<li>Details for items in queue are not fully displayed here, but items exist.</li>';
                } else {
                     retryQueueList.innerHTML = '<li>Retry queue is empty.</li>';
                }
            } else {
                retryQueueList.innerHTML = '<li>Retry queue is empty.</li>';
            }

        } catch (error) {
            console.error("Error fetching retry queue info:", error);
            retryQueueList.innerHTML = '<li>Error loading retry queue information.</li>';
            retryQueueSizeEl.textContent = "Error";
        }
    }

    // Common Failure Reasons Chart
    let failureReasonsChartInstance = null; // Keep this global for the chart instance
    async function renderFailureReasonsChart() {
        const failureReasonsCtx = document.getElementById('failureReasonsChart')?.getContext('2d');
        if (!failureReasonsCtx) return;

        let url = '/api/dashboard/failure-reasons?limit=10'; // Fetch top 10 reasons
        const dateParams = getDateRangeQueryString();
        if (dateParams) url += `&${dateParams}`;

        try {
            const response = await fetch(url);
            if(!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            const reasonDataApi = await response.json(); // Expected: Array of DashboardFailureReason {reason: string, count: number}

            const isDarkMode = document.body.classList.contains('dark-mode');
            const labels = reasonDataApi.map(r => r.reason);
            const counts = reasonDataApi.map(r => r.count);

            if (failureReasonsChartInstance) failureReasonsChartInstance.destroy();
            failureReasonsChartInstance = new Chart(failureReasonsCtx, {
                type: 'doughnut',
                data: {
                    labels: labels, // Use fetched labels
                    datasets: [{
                        label: 'Failure Reasons',
                        data: counts, // Use fetched counts
                        backgroundColor: [
                            '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF', '#FF9F40'
                        ],
                        hoverOffset: 4
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'top',
                            labels: {
                                color: isDarkMode ? '#e0e0e0' : '#333'
                            }
                        },
                        title: {
                            display: false, // Title is in H3
                            text: 'Common Failure Reasons',
                            color: isDarkMode ? '#e0e0e0' : '#333'
                        }
                    }
                }
            });
        }
    }
    // Initial call if the tab is active by default (it's not, but good practice if it were)
    // if (document.getElementById('failure-reasons')?.style.display === 'block') {
    // renderFailureReasonsChart();
    // }
    // Call for the first time when the tab is actually opened by the user, handled in openTab()

    // --- Data Export Functionality ---
    const exportCsvButton = document.getElementById('export-data-csv');
    const exportJsonButton = document.getElementById('export-data-json');

    // Store current filtered data for export - This is updated by fetchActivityDataForTimeline
    let currentFilteredActivityForExport = [];

    // This function is now primarily for fetching data for the timeline.
    // The data for export is a side-effect (currentFilteredActivityForExport).
    async function fetchActivityDataForTimeline(currentPage = 1, limit = 10) {
        const searchTerm = searchInput.value.toLowerCase(); // searchInput is global
        const selectedChannel = activityChannelFilter.value; // activityChannelFilter is global
        const selectedStatus = activityStatusFilter.value;

        let queryParams = new URLSearchParams({
            page: currentPage, // Assuming API will handle page or offset. Let's use offset for now.
            limit: limit,
            offset: (currentPage - 1) * limit
        });

        if (dateRangePicker && dateRangePicker._flatpickr && dateRangePicker._flatpickr.selectedDates.length > 0) {
            queryParams.append('from', dateRangePicker._flatpickr.formatDate(dateRangePicker._flatpickr.selectedDates[0], "Y-m-d"));
            if (dateRangePicker._flatpickr.selectedDates.length === 2) {
                queryParams.append('to', dateRangePicker._flatpickr.formatDate(dateRangePicker._flatpickr.selectedDates[1], "Y-m-d"));
            }
        }
        if (selectedChannel) queryParams.append('channel', selectedChannel);
        if (selectedStatus) queryParams.append('status', selectedStatus);
        if (searchTerm) queryParams.append('search', searchTerm);

        try {
            const response = await fetch(`/api/dashboard/recent-activity?${queryParams.toString()}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const data = await response.json();
            currentFilteredActivityForExport = data.notifications || []; // Store for export
            return data; // Expected: { notifications: [], totalCount: 0 }
        } catch (error) {
            console.error("Error fetching recent activity:", error);
            activityTimeline.innerHTML = '<li>Error loading activity. Please try again.</li>';
            currentFilteredActivityForExport = [];
            return { notifications: [], totalCount: 0 }; // Return empty on error
        }
    }


    function downloadFile(filename, content, mimeType) {
        const blob = new Blob([content], { type: mimeType });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(link.href);
    }

    // Simplified getFilteredActivityData for export, uses the cached data from last timeline fetch.
    function getFilteredActivityDataForExport() {
        return currentFilteredActivityForExport;
    }

    function exportToCsv() {
        const dataToExport = getFilteredActivityDataForExport(); // Use the new function
        if (dataToExport.length === 0) {
            alert('No data to export. Current timeline view is empty or filters resulted in no data.');
            return;
        }
        // API response fields: id, service_id, recipient_info, status, attempts, last_attempt_at, created_at, error_message
        const headers = ['ID', 'Timestamp (Created)', 'Timestamp (Last Attempt)', 'Channel', 'Recipient', 'Status', 'Attempts', 'Error'];
        const csvRows = [headers.join(',')];

        dataToExport.forEach(item => {
            const createdAt = item.created_at ? new Date(item.created_at).toLocaleString() : 'N/A';
            const lastAttemptAt = item.last_attempt_at ? new Date(item.last_attempt_at).toLocaleString() : 'N/A';
            const row = [
                item.id,
                createdAt,
                lastAttemptAt,
                item.service_id || '',
                item.recipient_info || '',
                item.status || '',
                item.attempts,
                item.error_message || ''
            ];
            const escapedRow = row.map(value => `"${String(value).replace(/"/g, '""')}"`);
            csvRows.push(escapedRow.join(','));
        });

        downloadFile('notification_activity.csv', csvRows.join('\n'), 'text/csv;charset=utf-8;');
    }

    function exportToJson() {
        const dataToExport = getFilteredActivityDataForExport(); // Use the new function
        if (dataToExport.length === 0) {
            alert('No data to export. Current timeline view is empty or filters resulted in no data.');
            return;
        }
        const jsonContent = JSON.stringify(dataToExport, null, 2);
        downloadFile('notification_activity.json', jsonContent, 'application/json;charset=utf-8;');
    }

    exportCsvButton?.addEventListener('click', exportToCsv);
    exportJsonButton?.addEventListener('click', exportToJson);

});
