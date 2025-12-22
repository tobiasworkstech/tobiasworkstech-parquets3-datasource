# Parquet-S3-Datasource for Grafana

Query and visualize Apache Parquet files stored in Amazon S3 or S3-compatible storage directly in Grafana.

## Overview

The Parquet-S3-Datasource plugin enables you to connect Grafana to your data lake stored in Parquet format on Amazon S3, MinIO, Wasabi, DigitalOcean Spaces, or any S3-compatible storage. Leverage the efficiency of columnar Parquet files for fast analytics and visualization without needing to load data into a traditional database.

## Demo

Watch the plugin in action - from datasource configuration to querying Parquet files:

![Plugin Demo](img/demo-full.webp)

### Quick Start Example

![Querying Parquet Data](img/screenshot-query-data.png)

## Features

- **Direct Parquet File Access**: Query Parquet files directly from S3 without intermediate databases
- **S3-Compatible Storage Support**: Works with Amazon S3, MinIO, Wasabi, DigitalOcean Spaces, and more
- **Apache Arrow Integration**: Efficient data processing using Apache Arrow for fast query execution
- **Configurable Endpoints**: Support for custom S3 endpoints for private cloud deployments
- **Path-Style Routing**: Automatic configuration for S3-compatible storage that requires path-style URLs

## Requirements

- Grafana >= 11.6.0
- S3 or S3-compatible storage with read access
- Parquet files in your S3 bucket

## Getting Started

### Installation

Install the plugin using the Grafana CLI:

```bash
grafana-cli plugins install tobiasworkstech-parquets3-datasource
```

Or via Docker:

```bash
docker run -d -p 3000:3000 \
  -e "GF_INSTALL_PLUGINS=tobiasworkstech-parquets3-datasource" \
  grafana/grafana
```

### Configuration

1. Navigate to **Configuration** > **Data Sources** in your Grafana instance
2. Click **Add data source**
3. Search for and select **Parquet-S3-Datasource**
4. Configure the following settings:
   - **Region**: Your S3 region (e.g., `us-east-1`)
   - **Bucket**: The name of your S3 bucket containing Parquet files
   - **Endpoint** (optional): Custom S3 endpoint URL (e.g., `http://minio:9000` for MinIO)
   - **Access Key**: Your S3 access key ID
   - **Secret Key**: Your S3 secret access key
5. Click **Save & test** to verify the connection

### Usage

1. Create a new dashboard or open an existing one
2. Add a new panel
3. Select your Parquet-S3-Datasource as the data source
4. In the **Parquet File Path** field, enter the path to your Parquet file (e.g., `data/metrics.parquet`)
5. Click **Run query** to visualize your data

## Configuration Examples

### Amazon S3

```
Region: us-east-1
Bucket: my-data-lake
Endpoint: (leave empty)
Access Key: AKIA...
Secret Key: ***
```

### MinIO (Local Development)

```
Region: us-east-1
Bucket: parquet-data
Endpoint: http://minio:9000
Access Key: minioadmin
Secret Key: minioadmin
```

### Wasabi

```
Region: us-east-1
Bucket: my-bucket
Endpoint: https://s3.wasabisys.com
Access Key: YOUR_WASABI_KEY
Secret Key: ***
```

## Supported Parquet Features

- All primitive data types (INT32, INT64, FLOAT, DOUBLE, BOOLEAN, BINARY, STRING)
- Nested structures (STRUCT, LIST, MAP)
- Compression codecs (SNAPPY, GZIP, LZ4, ZSTD)
- Column pruning for efficient data retrieval

## Troubleshooting

### Connection Failed

- Verify your S3 credentials are correct
- Ensure the bucket exists and is accessible
- Check network connectivity to your S3 endpoint
- For custom endpoints, verify the endpoint URL format

### No Data Returned

- Confirm the Parquet file path is correct
- Ensure the file exists in the specified bucket
- Check that your access key has read permissions

### Invalid Plugin Signature (Development)

For development environments, add this to your Grafana configuration:

```ini
[plugins]
allow_loading_unsigned_plugins = tobiasworkstech-parquets3-datasource
```

## Development

### Prerequisites

- Go >= 1.21
- Node.js >= 22
- Docker and Docker Compose

### Building the Plugin

```bash
# Build frontend
cd tobiasworkstech-parquets3-datasource
npm install
npm run build

# Build backend for Linux (for Docker)
GOOS=linux GOARCH=arm64 go build -o dist/gpx_parquet_s3_datasource_linux_arm64 ./pkg
```

### Running Locally

```bash
docker compose up -d
```

Access Grafana at `http://localhost:3001`.

## License

Apache 2.0 License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on [GitHub](https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource).

## Support

For issues, questions, or feature requests, please visit the [GitHub repository](https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/issues).
