# Grafana Parquet S3 Plugin

A Grafana datasource plugin for querying Apache Parquet files from Amazon S3 or S3-compatible storage (MinIO, Wasabi, DigitalOcean Spaces) with full SQL support powered by DuckDB.

[![Release](https://img.shields.io/github/v/release/tobiasworkstech/tobiasworkstech-parquets3-datasource)](https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Skills & Capabilities

### Data Querying
| Skill | Description |
|-------|-------------|
| **SQL Queries** | Full SQL support with SELECT, WHERE, GROUP BY, ORDER BY, LIMIT |
| **Aggregations** | COUNT, SUM, AVG, MIN, MAX functions |
| **Filtering** | Complex WHERE conditions with =, !=, >, <, >=, <=, LIKE, IN, IS NULL |
| **Sorting** | ORDER BY with ASC/DESC on multiple columns |
| **Pagination** | LIMIT clause for result set control |
| **Column Selection** | Select specific columns or use wildcards |

### Visual Query Builder
| Skill | Description |
|-------|-------------|
| **Point-and-Click** | Build queries without writing SQL |
| **Column Picker** | Select columns from dropdown with type info |
| **Aggregation Selector** | Add COUNT, SUM, AVG, MIN, MAX to columns |
| **Filter Builder** | Visual WHERE clause construction |
| **Group By Builder** | Multi-select GROUP BY columns |
| **Order By Builder** | Sort configuration with direction |
| **SQL Preview** | Real-time generated SQL display |
| **Mode Switching** | Toggle between Builder and Code modes |

### Template Variables
| Skill | Description |
|-------|-------------|
| **File Listing** | List parquet files from S3 bucket |
| **Prefix Listing** | List folders/prefixes for navigation |
| **SQL Variables** | Generate variable values from SQL queries |
| **Regex Filtering** | Filter file lists with regex patterns |
| **Dynamic Dashboards** | Use variables in file paths and queries |

### Storage Support
| Skill | Description |
|-------|-------------|
| **Amazon S3** | Native AWS S3 support |
| **MinIO** | S3-compatible local/cloud storage |
| **Wasabi** | Cloud storage with S3 API |
| **DigitalOcean Spaces** | DO's object storage |
| **Custom Endpoints** | Any S3-compatible storage |
| **Path-Style URLs** | Support for legacy S3 URL format |

### Data Format Support
| Skill | Description |
|-------|-------------|
| **Parquet Files** | Apache Parquet columnar format |
| **All Data Types** | INT, FLOAT, DOUBLE, STRING, BOOLEAN, BINARY |
| **Nested Structures** | STRUCT, LIST, MAP types |
| **Compression** | SNAPPY, GZIP, LZ4, ZSTD codecs |
| **Large Files** | Efficient streaming with Arrow |

### Grafana Integration
| Skill | Description |
|-------|-------------|
| **Explore View** | Full support in Grafana Explore |
| **Dashboard Panels** | Table, Stat, Bar Chart, Time Series |
| **Provisioning** | Auto-configure via YAML |
| **Health Check** | Connection testing on save |

## Project Structure

```
├── tobiasworkstech-parquets3-datasource/   # Plugin source code
│   ├── src/                                 # React frontend (TypeScript)
│   │   ├── components/
│   │   │   ├── QueryEditor.tsx              # Visual query builder
│   │   │   ├── ConfigEditor.tsx             # Datasource configuration
│   │   │   └── VariableQueryEditor.tsx      # Template variable editor
│   │   ├── datasource.ts                    # Datasource implementation
│   │   └── types.ts                         # TypeScript interfaces
│   ├── pkg/                                 # Go backend
│   │   ├── plugin/datasource.go             # Main datasource logic
│   │   ├── duckdb/executor.go               # SQL query executor
│   │   ├── parquet/reader.go                # Parquet file reader
│   │   └── models/                          # Data models
│   └── provisioning/                        # Grafana provisioning
│       ├── datasources/                     # Auto-configured datasource
│       └── dashboards/                      # Sample dashboards
├── samples/                                 # Sample parquet files
│   ├── iris.parquet                         # Iris flower dataset
│   ├── titanic.parquet                      # Titanic survival dataset
│   └── metrics_timeseries.parquet           # Time series metrics
├── cmd/                                     # CLI tools
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

Pre-configured dashboards available:
- **Iris Flower Dataset** - Classic ML dataset with measurements
- **Titanic Survival Dataset** - Survival analysis with aggregations
- **Time Series Metrics** - Server performance visualization
- **Example Dashboard** - SQL query demonstrations

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

-- Multiple conditions
SELECT * FROM parquet
WHERE status = 'active' AND value > 100
ORDER BY created_at DESC
LIMIT 50
```

## Template Variable Examples

### List Files Variable
```
Query Type: List Files
Prefix: data/2024/
File Pattern: *.parquet
```

### SQL-Based Variable
```
Query Type: SQL Query
Path: categories.parquet
SQL: SELECT DISTINCT category FROM parquet ORDER BY category
```

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

### Wasabi
```
Region: us-east-1
Bucket: my-bucket
Endpoint: https://s3.wasabisys.com
Access Key: YOUR_KEY
Secret Key: ***
```

## Technical Stack

| Component | Technology |
|-----------|------------|
| Frontend | React + TypeScript |
| Backend | Go 1.21+ |
| SQL Engine | DuckDB |
| Data Format | Apache Parquet + Arrow |
| AWS SDK | aws-sdk-go-v2 |
| UI Components | Grafana UI |

## Release

**Latest Version**: v1.1.0

**Download**: [tobiasworkstech-parquets3-datasource-1.1.0.zip](https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/releases/download/v1.1.0/tobiasworkstech-parquets3-datasource-1.1.0.zip)

**MD5**: `2fb5a4acb1961984002225b19d7a1ac2`

**Supported Platforms**:
- Linux AMD64 / ARM64
- macOS ARM64
- Windows AMD64

## Requirements

- Grafana >= 11.0.0
- S3 or S3-compatible storage
- Parquet files in bucket

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! Please open an issue or PR on [GitHub](https://github.com/tobiasworkstech/parquets3-datasource).

## Support

For issues or questions, visit the [GitHub Issues](https://github.com/tobiasworkstech/parquets3-datasource/issues).
