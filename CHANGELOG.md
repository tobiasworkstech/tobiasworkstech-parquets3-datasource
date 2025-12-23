# Changelog

All notable changes to this project will be documented in this file.

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
