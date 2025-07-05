package store

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// postgresStore implements the Store interface for PostgreSQL.
type postgresStore struct {
	pool *pgxpool.Pool
}

// Ensure postgresStore implements Store (compile-time check)
var _ Store = (*postgresStore)(nil)

// PostgresStoreFactory implements the StoreFactory interface for PostgreSQL.
type PostgresStoreFactory struct{}

// NewPostgresStoreFactory creates a new PostgresStoreFactory.
// This function will be called from main.go
func NewPostgresStoreFactory() StoreFactory {
	return &PostgresStoreFactory{}
}

// NewStore creates and connects a new postgresStore instance.
func (f *PostgresStoreFactory) NewStore(connectionString string) (Store, error) {
	s := &postgresStore{} // Create an instance of the store

	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}

	// You might want to configure connection pool settings here
	// config.MaxConns = 10
	// config.MinConns = 2
	// config.MaxConnLifetime = time.Hour
	// config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	s.pool = pool

	// Ping the database to verify the connection.
	if err = s.pool.Ping(context.Background()); err != nil {
		s.pool.Close() // Close the pool if ping fails
		return nil, err
	}

	log.Println("Successfully connected to PostgreSQL.")
	return s, nil
}

// Ensure PostgresStoreFactory implements StoreFactory (compile-time check)
var _ StoreFactory = (*PostgresStoreFactory)(nil)

// Close closes the database connection pool. This is part of the Store interface.
func (s *postgresStore) Close() error {
	if s.pool != nil {
		s.pool.Close()
		log.Println("PostgreSQL connection pool closed.")
	}
	return nil
}

// RecordNotification inserts a new notification record into the database.
// It returns the UUID of the newly created record.
func (s *postgresStore) RecordNotification(ctx context.Context, serviceID string, payload []byte, recipientInfo string) (string, error) {
	var id string
	// Ensure payload is valid JSON before inserting into JSONB
	if !json.Valid(payload) {
		log.Printf("Warning: PostgreSQL store: payload for service %s is not valid JSON: %s", serviceID, string(payload))
	}

	query := `
		INSERT INTO notifications (service_id, payload, recipient_info, status, created_at, updated_at, attempts)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), 0)
		RETURNING id`

	var recipientSQL *string // Use *string to handle potential NULL for recipient_info
	if recipientInfo != "" {
		recipientSQL = &recipientInfo
	}

	err := s.pool.QueryRow(ctx, query, serviceID, payload, recipientSQL, "queued").Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateNotificationStatus updates the status, attempts, and error message of an existing notification.
func (s *postgresStore) UpdateNotificationStatus(ctx context.Context, notificationID string, status string, attempts int, errorMessage string) error {
	query := `
		UPDATE notifications
		SET status = $1, attempts = $2, error_message = $3, last_attempt_at = NOW(), updated_at = NOW()
		WHERE id = $4`

	var errMsgSQL *string // Use *string to handle potential NULL for error_message
	if errorMessage != "" {
		errMsgSQL = &errorMessage
	}

	_, err := s.pool.Exec(ctx, query, status, attempts, errMsgSQL, notificationID)
	return err
}

// GetNotification retrieves a notification by its ID.
func (s *postgresStore) GetNotification(ctx context.Context, notificationID string) (*Notification, error) {
	query := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, notificationID)
	n := &Notification{}
	var payloadBytes []byte

	err := row.Scan(
		&n.ID,
		&n.ServiceID,
		&n.RecipientInfo, // Directly scan into *string
		&payloadBytes,
		&n.Status,
		&n.Attempts,
		&n.LastAttemptAt, // Directly scan into *time.Time
		&n.CreatedAt,
		&n.UpdatedAt,
		&n.ErrorMessage, // Directly scan into *string
	)
	if err != nil {
		// pgx.ErrNoRows is handled correctly by returning err
		return nil, err
	}
	n.Payload = payloadBytes
	return n, nil
}

// ListNotifications retrieves a paginated list of notifications, ordered by creation date descending.
// It also returns the total count of notifications matching the criteria (ignoring pagination).
func (s *postgresStore) ListNotifications(ctx context.Context, limit int, offset int) ([]Notification, int, error) {
	countQuery := `SELECT COUNT(*) FROM notifications`
	var totalCount int
	err := s.pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	if totalCount == 0 || offset >= totalCount {
		return []Notification{}, totalCount, nil // No notifications or offset is out of bounds
	}

	listQuery := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0, limit) // Pre-allocate slice capacity
	for rows.Next() {
		n := Notification{}
		var payloadBytes []byte
		err := rows.Scan(
			&n.ID,
			&n.ServiceID,
			&n.RecipientInfo,
			&payloadBytes,
			&n.Status,
			&n.Attempts,
			&n.LastAttemptAt,
			&n.CreatedAt,
			&n.UpdatedAt,
			&n.ErrorMessage,
		)
		if err != nil {
			// Log individual row scan error, but try to continue if possible,
			// or return immediately depending on desired error handling.
			log.Printf("Error scanning notification row in ListNotifications (PostgreSQL): %v", err)
			// For simplicity, returning error here. A more robust app might collect valid rows.
			return nil, totalCount, err
		}
		n.Payload = payloadBytes
		notifications = append(notifications, n)
	}

	if err = rows.Err(); err != nil {
		return nil, totalCount, err
	}

	return notifications, totalCount, nil
}

// --- Dashboard Specific Methods ---

func (s *postgresStore) GetDashboardGlobalStats(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time) (*DashboardGlobalStats, error) {
	stats := DashboardGlobalStats{}
	args := make([]interface{}, 0)
	argIdx := 1

	baseQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) AS total_sent,
			COALESCE(SUM(CASE WHEN status LIKE 'failed%' THEN 1 ELSE 0 END), 0) AS total_failed,
			COALESCE(SUM(CASE WHEN status = 'queued' OR status = 'processing' THEN 1 ELSE 0 END), 0) AS notifications_in_queue,
			COALESCE(AVG(CASE WHEN status = 'sent' AND last_attempt_at IS NOT NULL THEN EXTRACT(EPOCH FROM (last_attempt_at - created_at)) ELSE NULL END), 0) AS avg_delivery_time_sec
		FROM notifications
	`
	whereClauses := []string{}
	if dateRangeStart != nil {
		whereClauses = append(whereClauses, "created_at >= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeStart)
		argIdx++
	}
	if dateRangeEnd != nil {
		whereClauses = append(whereClauses, "created_at <= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeEnd)
		argIdx++
	}

	if len(whereClauses) > 0 {
		baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	row := s.pool.QueryRow(ctx, baseQuery, args...)
	err := row.Scan(
		&stats.TotalSent,
		&stats.TotalFailed,
		&stats.NotificationsInQueue,
		&stats.AvgDeliveryTimeSec,
	)
	if err != nil {
		log.Printf("Error scanning global stats: %v. Query: %s, Args: %v", err, baseQuery, args)
		return nil, err
	}

	// Calculate SuccessRate separately to avoid division by zero in SQL if no relevant notifications
	if stats.TotalSent+stats.TotalFailed > 0 {
		stats.SuccessRate = (float64(stats.TotalSent) * 100.0) / float64(stats.TotalSent+stats.TotalFailed)
	} else {
		stats.SuccessRate = 0
	}

	return &stats, nil
}

func (s *postgresStore) GetDashboardChannelStats(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time) ([]DashboardChannelStat, error) {
	// TODO: Implement actual SQL query
	// Query should:
	// - GROUP BY service_id
	// - For each service_id, calculate Total, Failed, SuccessRate similar to GlobalStats
	// - Respect dateRangeStart and dateRangeEnd on created_at if provided.
	args := make([]interface{}, 0)
	argIdx := 1
	query := `
		SELECT
			service_id,
			COUNT(*) AS total_notifications,
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) AS total_sent,
			COALESCE(SUM(CASE WHEN status LIKE 'failed%' THEN 1 ELSE 0 END), 0) AS total_failed
		FROM notifications
	`
	whereClauses := []string{}
	if dateRangeStart != nil {
		whereClauses = append(whereClauses, "created_at >= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeStart)
		argIdx++
	}
	if dateRangeEnd != nil {
		whereClauses = append(whereClauses, "created_at <= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeEnd)
		argIdx++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " GROUP BY service_id ORDER BY service_id"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("Error querying channel stats: %v. Query: %s, Args: %v", err, query, args)
		return nil, err
	}
	defer rows.Close()

	var results []DashboardChannelStat
	for rows.Next() {
		var stat DashboardChannelStat
		var totalSent, totalNotifications int64 // total_notifications from COUNT(*)
		err := rows.Scan(&stat.Channel, &totalNotifications, &totalSent, &stat.Failed)
		if err != nil {
			log.Printf("Error scanning channel stat row: %v", err)
			return nil, err
		}
		stat.Total = totalNotifications // Total for the channel based on filters
		if totalSent+stat.Failed > 0 {
			stat.SuccessRate = (float64(totalSent) * 100.0) / float64(totalSent+stat.Failed)
		} else {
			stat.SuccessRate = 0
		}
		results = append(results, stat)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating channel stat rows: %v", err)
		return nil, err
	}
	return results, nil
}

func (s *postgresStore) ListDashboardActivity(ctx context.Context, params DashboardActivityParams) ([]Notification, int, error) {
	// TODO: Implement actual SQL query. This will be an enhanced version of ListNotifications.
	// It needs to:
	// - Add WHERE clauses for:
	//   - created_at >= params.StartDate AND created_at <= params.EndDate (if provided)
	//   - service_id = params.ChannelFilter (if provided)
	//   - status = params.StatusFilter (if provided, may need to handle 'failed%' wildcard)
	//   - (id ILIKE '%' || params.SearchTerm || '%' OR recipient_info ILIKE '%' || params.SearchTerm || '%' OR error_message ILIKE '%' || params.SearchTerm || '%') (if provided)
	// - Continue to use LIMIT and OFFSET for pagination.
	// - Return totalCount respecting the filters (this will require a separate COUNT query with the same WHERE clauses).

	var args []interface{}
	var whereClauses []string
	argIdx := 1

	if params.StartDate != nil {
		whereClauses = append(whereClauses, "created_at >= $"+strconv.Itoa(argIdx))
		args = append(args, *params.StartDate)
		argIdx++
	}
	if params.EndDate != nil {
		whereClauses = append(whereClauses, "created_at <= $"+strconv.Itoa(argIdx))
		args = append(args, *params.EndDate)
		argIdx++
	}
	if params.ChannelFilter != "" {
		whereClauses = append(whereClauses, "service_id = $"+strconv.Itoa(argIdx))
		args = append(args, params.ChannelFilter)
		argIdx++
	}
	if params.StatusFilter != "" {
		if strings.HasSuffix(params.StatusFilter, "%") { // e.g. "failed%"
			whereClauses = append(whereClauses, "status LIKE $"+strconv.Itoa(argIdx))
		} else {
			whereClauses = append(whereClauses, "status = $"+strconv.Itoa(argIdx))
		}
		args = append(args, params.StatusFilter)
		argIdx++
	}
	if params.SearchTerm != "" {
		searchTermPattern := "%" + params.SearchTerm + "%"
		searchClause := "(id::text ILIKE $" + strconv.Itoa(argIdx) +
			" OR recipient_info ILIKE $" + strconv.Itoa(argIdx+1) +
			" OR error_message ILIKE $" + strconv.Itoa(argIdx+2) + ")"
		whereClauses = append(whereClauses, searchClause)
		args = append(args, searchTermPattern, searchTermPattern, searchTermPattern) // Add pattern for each ILIKE
		argIdx += 3
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM notifications " + whereSQL
	var totalCount int
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		log.Printf("Error counting dashboard activity: %v. Query: %s, Args: %v", err, countQuery, args)
		return nil, 0, err
	}

	if totalCount == 0 || params.Offset >= totalCount {
		return []Notification{}, totalCount, nil
	}

	// Add limit and offset to args for the list query
	// These are not part of the WHERE clause args, so they need separate placeholders
	listQueryArgs := make([]interface{}, len(args))
	copy(listQueryArgs, args) // Copy existing args

	listQuery := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications ` + whereSQL + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)

	listQueryArgs = append(listQueryArgs, params.Limit, params.Offset)

	rows, err := s.pool.Query(ctx, listQuery, listQueryArgs...)
	if err != nil {
		log.Printf("Error listing dashboard activity: %v. Query: %s, Args: %v", err, listQuery, listQueryArgs)
		return nil, totalCount, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0, params.Limit)
	for rows.Next() {
		n := Notification{}
		var payloadBytes []byte
		err := rows.Scan(
			&n.ID, &n.ServiceID, &n.RecipientInfo, &payloadBytes, &n.Status,
			&n.Attempts, &n.LastAttemptAt, &n.CreatedAt, &n.UpdatedAt, &n.ErrorMessage,
		)
		if err != nil {
			log.Printf("Error scanning dashboard activity row: %v", err)
			return nil, totalCount, err
		}
		n.Payload = payloadBytes
		notifications = append(notifications, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating dashboard activity rows: %v", err)
		return nil, totalCount, err
	}
	return notifications, totalCount, nil
}

func (s *postgresStore) GetDashboardVolumeTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error) {
	// TODO: Implement actual SQL query
	// Query should:
	// - Use DATE_TRUNC(period, created_at) to group notifications (period can be 'day', 'week', 'hour').
	// - COUNT(*) for volume.
	// - Filter by startDate and endDate on created_at.
	// - Return a list of {Date: (truncated date string), Volume: count}.

	dateFormat := "YYYY-MM-DD"
	if period == "hourly" {
		dateFormat = "YYYY-MM-DD HH24:MI"
	} else if period == "weekly" {
		// For weekly, DATE_TRUNC returns the Monday of the week.
		// TO_CHAR can format it, e.g., to show week number or start date.
		dateFormat = "YYYY-MM-DD" // Or "YYYY-WW" for week number
	}

	query := `
		SELECT
			TO_CHAR(date_trunc($1, created_at), $2) AS trend_date,
			COUNT(*) AS volume
		FROM notifications
		WHERE created_at >= $3 AND created_at < $4
		GROUP BY trend_date
		ORDER BY trend_date
	`
	// Note: endDate for query should be exclusive if using date_trunc 'day' and comparing against full timestamps.
	// Or adjust to include the full end day: created_at <= $4 (if $4 is end of day)
	// For simplicity, using < endDate, assuming endDate might be start of next day, or adjust API to pass end of day.
	// Let's adjust endDate to be exclusive for the query:
	exclusiveEndDate := endDate.Add(24 * time.Hour) // If period is daily, this makes sense.
	if period == "hourly" {
		exclusiveEndDate = endDate.Add(1 * time.Hour)
	}

	rows, err := s.pool.Query(ctx, query, period, dateFormat, startDate, exclusiveEndDate)
	if err != nil {
		log.Printf("Error querying volume trend: %v", err)
		return nil, err
	}
	defer rows.Close()

	var points []DashboardHistoricalPoint
	for rows.Next() {
		var p DashboardHistoricalPoint
		err := rows.Scan(&p.Date, &p.Volume)
		if err != nil {
			log.Printf("Error scanning volume trend row: %v", err)
			return nil, err
		}
		points = append(points, p)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating volume trend rows: %v", err)
		return nil, err
	}
	return points, nil
}

func (s *postgresStore) GetDashboardFailureRateTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error) {
	// TODO: Implement actual SQL query
	// Query should:
	// - Use DATE_TRUNC(period, created_at).
	// - For each period, calculate total sent and total failed.
	// - Compute failure rate: (total_failed * 100.0) / (total_sent + total_failed) (if denominator > 0)
	// - Filter by startDate and endDate.
	// - Return {Date: ..., Volume: (total_sent+total_failed), Failures: total_failed } (client can calc rate or API can provide)
	//   The current DashboardHistoricalPoint struct has `Failures` (count) and `Volume` (total count for period).
	//   The frontend will calculate the rate: Failures / Volume * 100.
	//   Or, this function can return the rate directly in one of the fields if struct is adapted.
	//   Let's return Failures (count) and Volume (total attempted: sent + failed for that period).

	dateFormat := "YYYY-MM-DD"
	if period == "hourly" {
		dateFormat = "YYYY-MM-DD HH24:MI"
	} else if period == "weekly" {
		dateFormat = "YYYY-MM-DD"
	}

	query := `
		SELECT
			TO_CHAR(date_trunc($1, created_at), $2) AS trend_date,
			COALESCE(SUM(CASE WHEN status LIKE 'failed%' THEN 1 ELSE 0 END), 0) AS failed_count,
			COALESCE(SUM(CASE WHEN status = 'sent' OR status LIKE 'failed%' THEN 1 ELSE 0 END), 0) AS total_attempted_count
			-- total_attempted_count here is specific to sent/failed for rate calculation, not all notifications
		FROM notifications
		WHERE created_at >= $3 AND created_at < $4
			AND (status = 'sent' OR status LIKE 'failed%') -- Only consider finalized states for rate calculation
		GROUP BY trend_date
		ORDER BY trend_date
	`
	exclusiveEndDate := endDate.Add(24 * time.Hour)
	if period == "hourly" {
		exclusiveEndDate = endDate.Add(1 * time.Hour)
	}

	rows, err := s.pool.Query(ctx, query, period, dateFormat, startDate, exclusiveEndDate)
	if err != nil {
		log.Printf("Error querying failure rate trend: %v", err)
		return nil, err
	}
	defer rows.Close()

	var points []DashboardHistoricalPoint
	for rows.Next() {
		var p DashboardHistoricalPoint
		// In DashboardHistoricalPoint, 'Volume' can represent total_attempted_count for this context
		err := rows.Scan(&p.Date, &p.Failures, &p.Volume)
		if err != nil {
			log.Printf("Error scanning failure rate trend row: %v", err)
			return nil, err
		}
		// Frontend will calculate rate: (p.Failures / p.Volume) * 100 if p.Volume > 0
		points = append(points, p)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating failure rate trend rows: %v", err)
		return nil, err
	}
	return points, nil
}

func (s *postgresStore) GetDashboardQueueSizeTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error) {
	// TODO: Implement actual SQL query
	// As discussed, this is tricky. A possible interpretation:
	// - Count notifications that *entered* 'queued' or 'processing' status within each period.
	// - DATE_TRUNC(period, created_at) WHERE status IN ('queued', 'processing')
	// - This is NOT concurrent queue size.
	// Alternative: Count items that were *still* in 'queued'/'processing' at the END of each period (more complex query).
	// For now, count items created_at in period with status 'queued' or 'processing'.
	dateFormat := "YYYY-MM-DD"
	if period == "hourly" { // Most relevant for queue size
		dateFormat = "YYYY-MM-DD HH24:MI"
	} else if period == "weekly" {
		dateFormat = "YYYY-MM-DD"
	}

	query := `
		SELECT
			TO_CHAR(date_trunc($1, created_at), $2) AS trend_date,
			COALESCE(SUM(CASE WHEN status = 'queued' OR status = 'processing' THEN 1 ELSE 0 END), 0) AS queue_volume
		FROM notifications
		WHERE created_at >= $3 AND created_at < $4
		GROUP BY trend_date
		ORDER BY trend_date
	`
	exclusiveEndDate := endDate.Add(24 * time.Hour) // Default for daily/weekly
	if period == "hourly" {
		exclusiveEndDate = endDate.Add(1 * time.Hour)
	}

	rows, err := s.pool.Query(ctx, query, period, dateFormat, startDate, exclusiveEndDate)
	if err != nil {
		log.Printf("Error querying queue size trend: %v", err)
		return nil, err
	}
	defer rows.Close()

	var points []DashboardHistoricalPoint
	for rows.Next() {
		var p DashboardHistoricalPoint
		err := rows.Scan(&p.Date, &p.QueueSize)
		if err != nil {
			log.Printf("Error scanning queue size trend row: %v", err)
			return nil, err
		}
		points = append(points, p)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating queue size trend rows: %v", err)
		return nil, err
	}
	return points, nil
}

func (s *postgresStore) GetDashboardFailureReasons(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time, limit int) ([]DashboardFailureReason, error) {
	// TODO: Implement actual SQL query
	// Query should:
	// - Filter by status LIKE 'failed%'
	// - GROUP BY error_message (non-NULL error messages)
	// - COUNT(*) as Count
	// - ORDER BY Count DESC
	// - LIMIT to the specified limit.
	// - Respect dateRangeStart and dateRangeEnd on created_at if provided.

	args := make([]interface{}, 0)
	whereClauses := []string{"status LIKE 'failed%'", "error_message IS NOT NULL"} // Base conditions
	argIdx := 1

	if dateRangeStart != nil {
		whereClauses = append(whereClauses, "created_at >= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeStart)
		argIdx++
	}
	if dateRangeEnd != nil {
		whereClauses = append(whereClauses, "created_at <= $"+strconv.Itoa(argIdx))
		args = append(args, *dateRangeEnd)
		argIdx++
	}

	queryBase := "SELECT COALESCE(error_message, 'Unknown Error') AS reason, COUNT(*) AS count FROM notifications "
	queryWhere := ""
	if len(whereClauses) > 0 {
		queryWhere = "WHERE " + strings.Join(whereClauses, " AND ")
	}
	queryGroupByOrderByLimit := " GROUP BY reason ORDER BY count DESC LIMIT $" + strconv.Itoa(argIdx)

	finalQuery := queryBase + queryWhere + queryGroupByOrderByLimit
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, finalQuery, args...)
	if err != nil {
		log.Printf("Error querying failure reasons: %v. Query: %s, Args: %v", err, finalQuery, args)
		return nil, err
	}
	defer rows.Close()

	var reasons []DashboardFailureReason
	for rows.Next() {
		var r DashboardFailureReason
		err := rows.Scan(&r.Reason, &r.Count)
		if err != nil {
			log.Printf("Error scanning failure reason row: %v", err)
			return nil, err
		}
		reasons = append(reasons, r)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating failure reason rows: %v", err)
		return nil, err
	}
	return reasons, nil
}
