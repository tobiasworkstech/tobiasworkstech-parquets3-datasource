# Changelog

All notable changes to this project will be documented in this file.

## [1.2.0] - 2026-02-14

### Bug Fixes

- **Fix LARGE_STRING Parquet panic**: Pandas-generated Parquet files use `LARGE_STRING` Arrow type, which caused a panic in the Parquet reader. Separated `arrow.LARGE_STRING` and `arrow.STRING` handling with proper type assertions.
- **Fix GROUP BY not aggregating**: GROUP BY queries returned one row per record instead of grouped results. The SQL executor was using pointer addresses as group keys (`*int64`, `*string`). Added `formatValue()` helper to dereference pointers before grouping.
- **Fix Titanic dashboard bar charts**: Grafana barchart panels require a string x-axis but `Pclass` is int64. Changed to table panels with gauge cell rendering.

### Improvements

- Added Playwright E2E tests verifying all 4 provisioned dashboards load with correct data
- Added Docker Compose dev environment (`docker/docker-compose.yml`) with MinIO and sample data generation
- Updated all provisioned dashboard JSON files for Grafana compatibility
- Added `CLAUDE.md` project documentation

## [1.1.0] - 2026-02-02

### Features

- **SQL Query Support**: Built-in, in-memory SQL engine (a custom lightweight executor, not DuckDB)
  - SELECT, WHERE, GROUP BY, ORDER BY, LIMIT clauses
  - Aggregation functions (COUNT, SUM, AVG, MIN, MAX)
  - Filtering and sorting
- **Visual Query Builder**: PostgreSQL-style query builder interface
  - Column selection with aggregations
  - Filter toggle with condition builder
  - Group By toggle with multi-column selection
  - Order By toggle with ASC/DESC
  - SQL Preview panel showing generated query
- **Template Variables**: Support for dashboard template variables
  - List files in bucket (with regex filtering)
  - List prefixes/folders
  - SQL-based variable queries
- **Explore View**: Enhanced query editor for Grafana Explore
  - File selector with regex filtering
  - Refresh button to reload file list
  - Builder and Code modes
- **Sample Dashboards**: Pre-built dashboards demonstrating plugin capabilities
  - Iris Dataset dashboard
  - Titanic Survival Dataset dashboard
  - Time Series Metrics dashboard

### Improvements

- Reduced Grafana version requirement to 11.0.0+
- Better error handling and logging
- Improved path-style routing for S3-compatible storage
- ARM64 binary support for Apple Silicon

## [1.0.0] - 2025-12-22

### Features

- **Initial Release**: Parquet-S3-Datasource plugin for Grafana
- **S3 Connectivity**: Support for Amazon S3 and S3-compatible storage providers:
  - Amazon S3
  - MinIO
  - Wasabi
  - DigitalOcean Spaces
  - Any S3-compatible storage
- **Direct Parquet Querying**: Read Apache Parquet files directly from S3 without intermediate databases
- **Apache Arrow Integration**: Efficient columnar data processing using Apache Arrow
- **Custom Endpoints**: Configurable S3 endpoints for private cloud deployments
- **Path-Style Routing**: Automatic configuration for storage systems requiring path-style URLs
- **Data Types Support**: All Parquet primitive types, nested structures, and compression codecs
- **Grafana 11.6+**: Fully compatible with Grafana 11.6.0 and above

### Components

- React-based frontend with TypeScript
- Go backend utilizing AWS SDK v2
- Docker Compose setup for local development with MinIO
- Comprehensive documentation and configuration examples
- Demo video and screenshots

### Configuration

- Region selection
- Bucket specification
- Custom endpoint URLs
- Secure credential management (Access Key/Secret Key)

### Development

- Provisioned datasource for quick Docker setup
- Sample Parquet file generation script
- Minio upload utilities
- End-to-end testing capabilities
