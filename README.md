# Grafana Parquet S3 Plugin

This project implements a Grafana Datasource plugin that loads and queries Parquet files stored in Amazon S3 or S3-compatible storage like Minio.

## Project Structure

- `tobiasworkstech-parquets3-datasource`: The Grafana plugin source code (React frontend & Go backend).
- `docker-compose.yml`: Orchestration for Grafana and Minio.
- `Dockerfile.grafana`: Custom Grafana image with the plugin embedded.
- `generate_parquet.go`: Script to generate sample `.parquet` data for testing.
- `upload_to_minio.go`: Script to upload files to the local Minio instance.

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose.
- [Go](https://golang.org/) (for building the backend).
- [Node.js & npm](https://nodejs.org/) (for building the frontend).

## Installation & Build

### 1. Build the Plugin Frontend
```bash
cd tobiasworkstech-parquets3-datasource
npm install
npm run build
```

### 2. Build the Plugin Backend
You need to build the backend for the target architecture of the Docker container (Linux).
```bash
# For ARM64 (Apple Silicon / Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o dist/gpx_parquet_s3_datasource_linux_arm64 ./pkg

# For AMD64 (Standard Intel/AMD)
GOOS=linux GOARCH=amd64 go build -o dist/gpx_parquet_s3_datasource_linux_amd64 ./pkg
```

## Running the Environment

### 1. Start Grafana and Minio
```bash
docker compose up -d
```
- **Grafana**: [http://localhost:3001](http://localhost:3001)
- **Minio Console**: [http://localhost:9001](http://localhost:9001) (User/Pass: `minioadmin` / `minioadmin`)

### 2. Prepare Test Data
```bash
# Generate sample Parquet file
go run generate_parquet.go

# Upload to Minio bucket 'parquet-data'
go run upload_to_minio.go
```

## Configuring Grafana

1. Log in to Grafana at [http://localhost:3001](http://localhost:3001).
2. Go to **Connections > Data Sources > Add data source**.
3. Search for **Parquet-S3-Datasource**.
4. Enter the following settings:
   - **Region**: `us-east-1`
   - **Bucket**: `parquet-data`
   - **Endpoint**: `http://minio:9000`
   - **Access Key**: `minioadmin`
   - **Secret Key**: `minioadmin`
5. Click **Save & test**.

## Querying Data

1. Create a new **Dashboard** or go to **Explore**.
2. Select your **Parquet-S3-Datasource**.
3. In the **Parquet File Path** field, enter: `test.parquet`.
4. Run the query to see the data loaded from S3!

## Backend Implementation Details

The plugin uses:
- `github.com/aws/aws-sdk-go-v2`: For S3 interactions.
- `github.com/apache/arrow-go/v18/parquet/pqarrow`: To read Parquet files efficiently into Arrow Tables, which are then converted to Grafana Data Frames.
- Custom `S3ReaderAt` with `Seek` support to handle Parquet's random-access requirements over HTTP range requests.


