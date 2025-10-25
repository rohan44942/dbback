# dbback — A Database Backup Utility

This repository contains a comprehensive database backup utility designed to be flexible, efficient, and easy to manage. It has evolved from a simple CLI tool into a powerful, schedulable, and API-driven backup controller.

## Current Features (Phase 2)

The project has moved beyond its initial scaffolding and now includes a robust set of features:

- **Advanced Storage Options**:
  - Seamlessly back up to the **local filesystem** or any **S3-compatible object storage** (e.g., AWS S3, MinIO).
  - Storage backend is configurable via a central `config.yaml` file.

- **Streaming Architecture**:
  - Backups are **streamed directly** from the database dump command, compressed on-the-fly, and uploaded to storage.
  - This avoids creating temporary files, saving disk space and improving performance.

- **Expanded Database Support**:
  - **SQLite**: Backs up the database file directly.
  - **PostgreSQL**: Uses `pg_dump`.
  - **MySQL**: Uses `mysqldump`.
  - **MongoDB**: Uses `mongodump`.

- **Controller with HTTP API**:
  - A new `controller` service (`cmd/controller`) runs as a persistent daemon.
  - Exposes a RESTful API to manage the backup system.
  - API endpoints for listing backups and creating, listing, and deleting backup schedules.

- **Automated Job Scheduling**:
  - A built-in cron-based scheduler automatically runs backup jobs based on schedules defined via the CLI or API.
  - Schedules are persisted in the metadata database and loaded on controller startup.

- **Enhanced CLI (`dbback`)**:
  - **`backup`**: Manually trigger backups to S3 or stream them locally to `stdout`.
  - **`restore`**: Restore a specific backup from storage to a target file.
  - **`list`**: View a history of completed backups.
  - **`schedule`**: Add, list, or remove automated backup schedules.

- **Robust Metadata & Configuration**:
  - Backup and schedule metadata is stored in a durable **SQLite database**.
  - Application behavior (storage, server address, etc.) is managed through a `config.yaml` file.

- **Logging and Notifications**:
  - Implemented structured logging (`slog`) throughout the application for clear, machine-readable log output.
  - Automatic **Slack notifications** for backup failures. This is configured by setting the `SLACK_WEBHOOK_URL` environment variable.

- **Containerized Deployment**:
  - Includes a `docker-compose.dev.yml` for a one-command development setup, complete with a MinIO instance for S3 testing.

## Getting Started

### Run with Docker (Recommended)

The easiest way to get the controller and its dependencies running is with Docker Compose.

```sh
# This will build the dbback image and start the controller and MinIO.
docker-compose -f docker-compose.dev.yml up --build

# The controller API will be available at http://localhost:8080
# The MinIO console will be available at http://localhost:9001
```

### Run Manually

1.  **Build the binaries:**
    ```sh
    # Build the CLI tool
    go build -o bin/dbback ./cmd/dbback

    # Build the controller
    go build -o bin/controller ./cmd/controller
    ```

2.  **Configure the application:**
    - Copy `configs/config.yaml.example` to `configs/config.yaml` and edit it to match your environment (e.g., S3 credentials, database path).

3.  **Run the controller:**
    To enable Slack notifications, first set the environment variable:
    ```sh
    # Get this URL from your Slack App's "Incoming Webhooks" configuration.
    export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
    ```

    Then run the controller:
    ```sh
    ./bin/controller
    ```

4.  **Use the CLI:**
    ```sh
    # Schedule a daily backup of a SQLite database
    ./bin/dbback schedule add --type sqlite --source /path/to/your/app.db --cron "0 2 * * *"

    # Manually back up a PostgreSQL database and stream it to a local file
    ./bin/dbback backup --type postgres --source "postgresql://user:pass@host:5432/dbname" --destination local > my-postgres-backup.gz
    ```

## Future Work (Phase 3)

The roadmap for `dbback` includes several key enhancements to make it a production-grade backup solution:

- **End-to-End Encryption**:
  - Integrate the existing `crypto` package to encrypt backups before they are sent to storage.
  - The encryption key will be managed securely, and the CLI/API will be updated to handle encrypted backups and restores.

- **Automated Backup Retention**:
  - Fully implement and integrate the retention policy logic from the `retention` package.
  - The controller will periodically run a cleanup job to prune old backups from storage and metadata according to the retention policy defined in each schedule.

- **Comprehensive Logging and Monitoring**:
  - Add a metrics endpoint (e.g., `/metrics`) for Prometheus scraping to monitor backup successes, failures, durations, and sizes.

- **API and System Testing**:
  - Develop a suite of integration tests for the API to ensure reliability.
  - Write end-to-end tests for the backup and restore flows for each supported database.

- **Improved Error Handling and Notifications**:
  - Enhance error reporting in the API and logs to make debugging easier.

- **Web UI**:
  - Build a simple web-based user interface that interacts with the controller's API to provide a dashboard for viewing backup history, managing schedules, and monitoring system health.
