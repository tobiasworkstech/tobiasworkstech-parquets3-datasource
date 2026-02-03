# Grafana Parquet S3 Plugin

A Grafana datasource plugin for querying Apache Parquet files from Amazon S3 or S3-compatible storage (MinIO, Wasabi, DigitalOcean Spaces) with full SQL support powered by DuckDB.

## Features

- **SQL Queries**: Full SQL support with SELECT, WHERE, GROUP BY, ORDER BY, LIMIT
- **Visual Query Builder**: PostgreSQL-style interface with point-and-click query building
- **Template Variables**: Dynamic dashboards with file listing and SQL-based variables
- **Multiple Storage Providers**: Amazon S3, MinIO, Wasabi, DigitalOcean Spaces
- **Sample Dashboards**: Pre-built dashboards for Iris, Titanic, and Time Series data

## Project Structure

```
├── tobiasworkstech-parquets3-datasource/   # Plugin source code
│   ├── src/                                 # React frontend (TypeScript)
│   ├── pkg/                                 # Go backend
│   │   ├── plugin/                          # Datasource implementation
│   │   ├── duckdb/                          # SQL query executor
│   │   ├── parquet/                         # Parquet file reader
│   │   └── models/                          # Data models
│   └── provisioning/                        # Grafana provisioning configs
│       ├── datasources/                     # Auto-configured datasource
│       └── dashboards/                      # Sample dashboards
├── samples/                                 # Sample parquet files
├── cmd/                                     # CLI tools
│   ├── generate_parquet/                    # Generate test data
│   └── upload_to_minio/                     # Upload to MinIO
├── docker-compose.yml                       # Development environment
└── Dockerfile.grafana                       # Custom Grafana image
```

## Quick Start

### 1. Start Development Environment

```bash
docker compose up -d
```

- **Grafana**: http://localhost:3001
- **MinIO Console**: http://localhost:9001 (minioadmin/minioadmin)

### 2. Explore Sample Dashboards

The plugin comes with pre-configured dashboards:
- Iris Flower Dataset
- Titanic Survival Dataset
- Time Series Metrics

### 3. Create Your Own Queries

1. Go to **Explore** in Grafana
2. Select **parquet-s3-datasource**
3. Choose a parquet file from the dropdown
4. Use the visual builder or write SQL directly

## Building the Plugin

### Frontend

```bash
cd tobiasworkstech-parquets3-datasource
npm install
npm run build
```

### Backend

```bash
cd tobiasworkstech-parquets3-datasource

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dist/gpx_parquet_s3_datasource_linux_amd64 ./pkg

# Linux ARM64 (Apple Silicon Docker)
GOOS=linux GOARCH=arm64 go build -o dist/gpx_parquet_s3_datasource_linux_arm64 ./pkg

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o dist/gpx_parquet_s3_datasource_darwin_arm64 ./pkg

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/gpx_parquet_s3_datasource_windows_amd64.exe ./pkg
```

## SQL Query Examples

```sql
-- Basic query
SELECT * FROM parquet LIMIT 100

-- Filtering
SELECT name, value FROM parquet
WHERE value > 50
ORDER BY value DESC

-- Aggregations
SELECT category, COUNT(*) as count, AVG(price) as avg_price
FROM parquet
GROUP BY category
ORDER BY count DESC

-- Column with special characters
SELECT "sepal.length", "petal.width" FROM parquet
```

## Template Variables

### List Files
Query Type: `List Files`
- Lists all parquet files in the bucket
- Supports prefix filtering and regex patterns

### SQL Query
Query Type: `SQL Query`
- Path: `data.parquet`
- SQL: `SELECT DISTINCT category FROM parquet`

## Configuration

### MinIO (Development)
```
Region: us-east-1
Bucket: parquet-data
Endpoint: http://minio:9000
Access Key: minioadmin
Secret Key: minioadmin
```

### Amazon S3
```
Region: us-east-1
Bucket: my-data-lake
Endpoint: (leave empty)
Access Key: AKIA...
Secret Key: ***
```

## Technical Details

- **Frontend**: React + TypeScript with Grafana UI components
- **Backend**: Go with AWS SDK v2 and Apache Arrow
- **SQL Engine**: DuckDB for query execution
- **Data Format**: Apache Parquet with Arrow integration

## Release

**Latest Version**: v1.1.0

Download: [tobiasworkstech-parquets3-datasource-1.1.0.zip](https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/releases/download/v1.1.0/tobiasworkstech-parquets3-datasource-1.1.0.zip)

MD5: `2fb5a4acb1961984002225b19d7a1ac2`

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
