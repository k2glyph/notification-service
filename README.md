# Notification Service

A simple notification service that can send notifications via Slack and Email. It now features PostgreSQL integration to record every notification request and its state.

## Features
* In-memory queue (default)
* Redis queue (optional)
* Slack notifications
* Email notifications
* Database integration for tracking notification states (Supports PostgreSQL and MySQL).

## Environment Variables
* `DATABASE_URL` - (Required) Connection string for the database.
    * Example for PostgreSQL: `postgres://user:password@localhost:5432/notifications_db?sslmode=disable`
    * Example for MySQL: `user_mysql:password_mysql@tcp(localhost:3306)/notifications_db_mysql?parseTime=true`
* `DATABASE_TYPE` - (Optional) Specifies the type of database. Supported values: `postgres`, `mysql`. Defaults to `postgres` if not set.
* `redisURL` - (Optional) Connection string for Redis. If not specified, an in-memory queue will be used. Example: `redis://localhost:6379/0`
* `slackWebhookURL` - (Optional) Slack Webhook URL to enable Slack notifications.
* `smtpHost` - (Optional) SMTP server host to enable Email notifications.
* `smtpPort` - (Optional) SMTP server port.
* `smtpUsername` - (Optional) SMTP username.
* `smtpPassword` - (Optional) SMTP password.
* `smtpFrom` - (Optional) SMTP From address.

## Running with Docker Compose (Recommended for Development)

This service now includes a `docker-compose.yml` file to easily run the application along with PostgreSQL and MySQL databases.

1.  **Ensure Docker and Docker Compose are installed.**
2.  **Setup Database Schema:**
    Before running the application for the first time, you need to apply the database migrations for your chosen database.
    *   Start the target database container:
        *   For PostgreSQL: `docker-compose up -d postgres`
        *   For MySQL: `docker-compose up -d mysql`
    *   Wait a few seconds for the database to initialize.
    *   Apply the appropriate migration script:
        *   For PostgreSQL:
            ```bash
            docker-compose exec -T postgres psql -U user -d notifications_db < migrations/postgres/001_create_notifications_table.up.sql
            ```
        *   For MySQL:
            ```bash
            docker-compose exec -T mysql mysql -u root -prootpassword notifications_db_mysql < migrations/mysql/001_create_notifications_table.up.sql
            ```
        (Adjust credentials and database names if you changed them in `docker-compose.yml`).
3.  **Set Environment Variables (Update `.env` or Docker Compose `app` service):**
    The `docker-compose.yml` `app` service is pre-configured to use MySQL. To use PostgreSQL, you'll need to comment out/change the `DATABASE_URL` and `DATABASE_TYPE` for the `app` service in `docker-compose.yml` or preferably use a `.env` file.

    Example `.env` file for PostgreSQL:
    ```env
    DATABASE_URL="postgres://user:password@postgres:5432/notifications_db?sslmode=disable"
    DATABASE_TYPE="postgres"
    # Optional:
    # SLACK_WEBHOOK_URL="your_slack_webhook_url"
    # SMTP_HOST="your_smtp_host"
    ```
    Example `.env` file for MySQL (matches current docker-compose default for app):
    ```env
    DATABASE_URL="user_mysql:password_mysql@tcp(mysql:3306)/notifications_db_mysql?parseTime=true"
    DATABASE_TYPE="mysql"
    # Optional:
    # SLACK_WEBHOOK_URL="your_slack_webhook_url"
    # SMTP_HOST="your_smtp_host"
    # SMTP_PORT="587"
    # SMTP_USERNAME="your_username"
    # SMTP_PASSWORD="your_password"
    # SMTP_FROM="notifications@example.com"
    # REDIS_URL="redis://redis:6379/0" # If you add a redis service to docker-compose
    ```
    Note: In `docker-compose.yml`, the application service (`app`) refers to database services by their service names (e.g., `postgres`, `mysql`). So, the hostname in `DATABASE_URL` should match these service names when running `app` via Docker Compose. If running the Go app manually and connecting to a database exposed on `localhost` by Docker, then `localhost` is correct in the `DATABASE_URL`.

4.  **Build and Run the Application:**
    ```bash
    docker-compose up --build app
    ```
    This will build the `app` image (if changed) and start the `app` service along with its declared dependencies (PostgreSQL and MySQL services will also start if not already running).
    To run in detached mode: `docker-compose up --build -d app`.

## Running Manually (Go application, with Docker for DB)

1.  **Start Your Chosen Database using Docker Compose:**
    *   For PostgreSQL: `docker-compose up -d postgres`
    *   For MySQL: `docker-compose up -d mysql`
2.  **Apply Migrations** (as described in step 2 of "Running with Docker Compose", choosing the correct script for your database).
3.  **Set Environment Variables** for your shell session.
    Example for PostgreSQL:
    ```bash
    export DATABASE_URL="postgres://user:password@localhost:5432/notifications_db?sslmode=disable"
    export DATABASE_TYPE="postgres"
    ```
    Example for MySQL:
    ```bash
    export DATABASE_URL="user_mysql:password_mysql@tcp(localhost:3306)/notifications_db_mysql?parseTime=true"
    export DATABASE_TYPE="mysql"
    ```
    ```bash
    # Common for both:
    # export SLACK_WEBHOOK_URL="..."
    # ... etc.
    ```
4.  **Run the Go application:**
    ```bash
    go run cmd/server/main.go
    ```

## API

### Push Notification

* **POST** `/api/push/{service}`
  * `{service}` can be `slack` or `email`.
  * Request body should be a JSON payload specific to the service.

#### Slack Example
```json
{
    "text": "Hello from notification service!",
    "channel": "#your-channel"
}
```
The `channel` can also be passed as `recipientInfo` if not in the payload.

#### Email Example
```json
{
    "to": "recipient@example.com",
    "subject": "Test Email",
    "body": "This is a test email from the notification service."
}
```
The `to` field can also be passed as `recipientInfo` if not in the payload.

The API will respond with `202 Accepted` if the message is successfully recorded and queued.
The actual sending of the notification happens asynchronously.
The state of each notification (e.g., "queued", "processing", "sent", "failed") is tracked in the `notifications` table in the chosen database. You can query this table to see the status of each request.
The `recipient_info` field in the database will be populated based on fields like "to" (for email) or "channel" (for Slack) if found in the request payload.

## Web UI for Notifications

A simple web UI is available to view the status and details of processed notifications.

*   **URL:** `http://localhost:8080/ui/notifications`
    *   The root path `/` will also redirect here.
*   **Features:**
    *   Displays a paginated list of notifications from the database, ordered by creation time (newest first).
    *   Shows details such as ID, Service, Recipient, Status, Attempts, Payload summary, Timestamps, and Error messages.
    *   Includes basic pagination controls ("First", "Previous", "Next", "Last").
    *   A "Refresh Page" button to manually update the list.

## Load Testing with k6

A basic load test script (`loadtest.js`) is included to simulate traffic to the notification service.

### Prerequisites
*   **Install k6:** Follow the instructions on the [official k6 website](https://k6.io/docs/getting-started/installation/).

### Running the Load Test
1.  **Ensure the Notification Service is running.** You can run it using Docker Compose or manually as described in the sections above. By default, the k6 script targets `http://localhost:8080`.
2.  **Open your terminal** in the root directory of this project.
3.  **Run the k6 script:**
    ```bash
    k6 run loadtest.js
    ```
4.  **Customize the load (Optional):**
    You can override the default virtual users (VUs) and duration using environment variables:
    ```bash
    # Run with 20 VUs for 60 seconds
    k6 run --env VUS=20 --env DURATION=60s loadtest.js
    ```
    You can also change the `BASE_URL` if your service is running elsewhere:
    ```bash
    k6 run --env BASE_URL=http://your-service-host:port loadtest.js
    ```

### Observing Results
K6 will output metrics to the console during and after the test run, including:
*   `http_reqs`: Total HTTP requests made.
*   `http_req_duration`: Response time metrics (avg, p95, p99, etc.).
*   `http_req_failed`: Percentage of failed requests (e.g., non-2xx/3xx responses).
*   `vus`: Number of active virtual users.
*   `checks`: Success rate of your defined checks (e.g., status code is 202).

Observe these metrics and the application logs to understand how the service performs under load. The script includes basic thresholds for request failure rate, duration, and check success.
```
