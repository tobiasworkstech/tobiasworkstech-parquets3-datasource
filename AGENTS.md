# AGENTS.md — Parquet S3 Datasource Plugin

**Plugin ID**: `tobiasworkstech-parquets3-datasource`

## Build & Dev

```bash
# Frontend
npm install
npm run build            # Production build
npm run dev              # Watch mode
npm run lint             # ESLint
npm run test:ci          # Jest tests

# Backend
go mod download
mage -v                  # Build all platforms (linux/darwin/windows, amd64/arm64)
go build -o dist/gpx_parquet_s3_datasource_darwin_arm64 ./pkg  # Dev build (macOS ARM)

# Go tests (NOTE: pkg/plugin/datasource_test.go is a scaffolding stub that panics on nil S3 client — pre-existing, not a real failure)
go test -v ./...

# Docker dev environment (Grafana + MinIO with sample data)
cd docker && docker-compose up -d
# Grafana: http://localhost:3000 (admin/admin123)
# MinIO:   http://localhost:9001 (minioadmin/minioadmin123)

# Playwright E2E tests (requires Docker running)
GRAFANA_ADMIN_PASSWORD=admin123 npx playwright test tests/dashboards.spec.ts
```

## Architecture

### Frontend (`src/`)
- `components/ConfigEditor.tsx` — S3 connection settings (endpoint, bucket, auth type, access/secret keys)
- `components/QueryEditor.tsx` — Query builder (bucket/path selection, SQL query, column/filter selection)
- `datasource.ts` — Query execution, template variable support, resource API calls
- `types/index.ts` — TypeScript interfaces (`ParquetS3Query`, `ParquetS3DataSourceOptions`)

### Backend (`pkg/`)
- `plugin/datasource.go` — Main plugin entry: query handler, health checks, resource HTTP endpoints (`/health`, `/buckets`, `/files`, `/schema`, `/query`)
- `parquet/reader.go` — Reads Parquet files from S3 via Arrow, converts to Grafana data frames
- `duckdb/executor.go` — Custom SQL executor that runs queries on in-memory Grafana data frames (NOT actual DuckDB). Supports SELECT, WHERE, GROUP BY, ORDER BY, LIMIT, aggregates (COUNT, SUM, AVG, MIN, MAX)
- `s3/client.go` — AWS SDK S3 client factory
- `models/` — Shared data models

### Resource Endpoints
`/health`, `/buckets`, `/files`, `/schema`, `/query`

### Multi-File & Date-Partitioned Queries
- **Explicit multi-file**: `qm.Paths` (array) or a comma-separated `qm.Path` selects several files to concatenate (schemas must match) — handled by `resolveKeys()` / `parquet.ReadParquetFilesFromS3()` in `pkg/plugin/datasource.go`.
- **Date-range pattern** (`qm.PathPattern`, e.g. `"events/dt={date}/events.parquet"`): expanded server-side by `expandDatePattern()` into one S3 key per day covered by the query's `backend.DataQuery.TimeRange` (inclusive), so a dashboard's top time-picker automatically selects the matching daily-partitioned files — no `$file` variable, no manual selection, no S3 globbing (which the plugin doesn't support). `qm.DateFormat` is a Go reference-time layout for the `{date}` token (default `"2006-01-02"`). Capped at `maxPatternDays` (370) to avoid unbounded S3 fan-out from an oversized time range. `pathPattern` takes priority over `path`/`paths` when set, and is honored by both regular queries and SQL variable queries (`handleVariableQuery`).

### Template Variables
`buckets()`, `files(bucket, prefix)`, `columns(bucket, key)`

### Alerting Support
The plugin has `"alerting": true` in `plugin.json`, enabling Grafana alert rules to use this datasource. Alert rules can query parquet files via SQL and evaluate thresholds. See `provisioning/alerting/alerts.yml` for the provisioned example alert.

### Error Handling & Logging
- **Error responses to users are generic** — never expose internal details (connection strings, IPs, ports) in error messages returned to the frontend. Always log the detailed error server-side at `Error` level and return a message like `"... see Grafana server log for details"`.
- **Operational logging uses `Debug` level** — initialization, path-style routing, variable queries, file listing, and health check logs are all `Debug`, not `Info`, to avoid excessive noise in production.
- **Nil safety** — AWS SDK pointer fields (e.g., `*head.ContentLength`) must be nil-checked before dereferencing.

## Docker Setup

### Services (`docker/docker-compose.yml`)
- **minio** — S3-compatible storage (port 9000/9001)
- **minio-setup** — Python container that generates 5 sample parquet files and uploads to MinIO
- **grafana** — Grafana with plugin mounted from `../dist/`, provisioned with datasource, dashboards, and alert rules

### Sample Parquet Files (generated inline by the `minio-setup` service in `docker/docker-compose.yml`)
- `iris.parquet` — 150 rows, Iris flower dataset (sepal/petal measurements, species)
- `titanic.parquet` — 891 rows, Titanic survival dataset (uses pandas which produces `LARGE_STRING` Arrow type)
- `metrics_timeseries.parquet` — Time-series server metrics (CPU, memory, disk, network)
- `sample_metrics.parquet` — Simple metrics data
- `test.parquet` — Basic test file
- `sales_north.parquet`, `sales_south.parquet`, `sales_east.parquet`, `sales_west.parquet` — Regional sales orders (order_id, order_date, region, product, category, quantity, unit_price, revenue, customer_id), ~150-250 rows each. All four share an identical schema and exist specifically to demonstrate **multi-file search**: select several of them at once in the query editor (or pass `paths`/comma-separated `path`) to query them as one combined table.
- `events/dt=YYYY-MM-DD/events.parquet` — 14 daily files (event_time, dt, event, user_id, session_id, duration_ms), ~50-150 rows/day, one file per of the last 14 days. Exists to demonstrate the **`pathPattern` date-range feature**: a query with `pathPattern="events/dt={date}/events.parquet"` auto-expands to the files matching the dashboard's time-picker range.

### Docker Build for Apple Silicon
Docker on Apple Silicon runs `aarch64`/`arm64` containers, NOT `amd64`. Always build:
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/gpx_parquet_s3_datasource_linux_arm64 ./pkg
```
NOT `GOARCH=amd64`.

## Provisioned Dashboards

6 dashboards in `provisioning/dashboards/`:
- `iris-dataset.json` (uid: `iris-dataset`) — Table, stat panels (Total Samples, Species Distribution, Average Measurements), bar chart by species
- `titanic-dataset.json` (uid: `titanic-dataset`) — Table, stat panels (Total Passengers, Survivors, Average Fare), table panels with gauge cells (Passengers by Class, by Gender), Survival by Class, Embarkation Ports
- `metrics-timeseries.json` (uid: `metrics-timeseries`) — 6 time-series panels (CPU, Memory, Disk, Network, Requests, Errors), 3 gauge panels (Average CPU/Memory/Disk), Host Summary Statistics table
- `parquet-s3-example.json` (uid: `parquet-s3-example`) — All Data table, Filtered table, Aggregations table, stat panels, bar chart
- `multi-file-sales.json` (uid: `multi-file-sales`) — Demonstrates querying multiple Parquet files at once. Targets set both `paths` (array) and comma-joined `path` against `sales_north.parquet`, `sales_south.parquet`, `sales_east.parquet`, `sales_west.parquet`. Panels: stat panels (Total Revenue, Total Orders, Regions Covered, Average Order Value), combined table of all regions, and grouped tables (Revenue by Region, Revenue by Category, Top Products)
- `daily-events-time-range.json` (uid: `daily-events-time-range`) — Demonstrates the `pathPattern` date-range feature. Every target sets `pathPattern="events/dt={date}/events.parquet"` (no fixed `path`); changing the dashboard's time-picker (default "Last 7 days", data spans 14 days) changes which daily files are queried automatically. Panels: stat panels (Total Events, Unique Users, Average Duration, Daily Files Included), Events per Day bar chart (`GROUP BY dt`), Events by Type table, Recent Events table

All dashboards use `${datasource}` template variable of type `datasource` filtered to `tobiasworkstech-parquets3-datasource`.

## Provisioned Alerting

1 alert rule in `provisioning/alerting/alerts.yml`:
- **High CPU Alert** (uid: `parquet-s3-high-cpu`) — Queries `metrics_timeseries.parquet` for `AVG(cpu_percent)`, fires when > 50%. Evaluates every 1 minute. Labels: `severity: warning`, `source: parquet-s3`.

The datasource has a stable provisioned UID (`parquet-s3`) in `provisioning/datasources/datasources.yml` so alert rules can reference it reliably.

## Playwright E2E Tests

File: `tests/dashboards.spec.ts` — 7 tests verifying all 4 dashboards + API provisioning + datasource health.

### Running
```bash
GRAFANA_ADMIN_PASSWORD=admin123 npx playwright test tests/dashboards.spec.ts
```

### Key Details
- Uses `@grafana/plugin-e2e` auth framework (configured in `playwright.config.ts`)
- Auth project runs first, stores session in `playwright/.auth/admin.json`
- `GRAFANA_ADMIN_PASSWORD=admin123` required (Docker Grafana uses admin123, not default admin)
- Tests use panel title text selectors (`getByText('Panel Title')`) — stable across Grafana versions
- Time-series charts render in canvas (no extractable text) — verify via legend field names (e.g., `cpu_percent`)
- Gauge panels below viewport fold — must scroll down before asserting visibility
- Screenshots saved to `playwright-results/`

## Known Bugs & Fixes (Historical)

### 1. LARGE_STRING Panic (`pkg/parquet/reader.go`)
**Symptom**: `panic: interface conversion: arrow.Array is *array.LargeString, not *array.String`
**Cause**: Pandas generates parquet files with `arrow.LARGE_STRING` type. The original code combined `STRING` and `LARGE_STRING` in one switch case but only cast to `*array.String`.
**Fix**: Separate `arrow.STRING` and `arrow.LARGE_STRING` into distinct switch cases with proper type assertions.

### 2. GROUP BY Not Aggregating (`pkg/duckdb/executor.go`)
**Symptom**: GROUP BY queries return one row per record instead of grouped rows (e.g., 891 rows instead of 3 for `GROUP BY Pclass`).
**Cause**: `fmt.Sprintf("%v", field.At(i))` on nullable pointer fields (`*int64`, `*string`, etc.) prints memory addresses (e.g., `0x14000123456`), making every row a unique group key.
**Fix**: Added `formatValue()` helper that dereferences pointers before formatting. Used in group key construction.

### 3. Grafana Barchart with Numeric X-Axis
**Symptom**: Grafana barchart panel shows error "Bar charts require a string or time field".
**Cause**: `Pclass` column is int64 but Grafana's barchart visualization requires a string/time x-axis.
**Fix**: Changed panel type from `barchart` to `table` with gauge cell overrides for the count column.

## SQL Executor Details (`pkg/duckdb/executor.go`)

Despite the package name `duckdb`, this is a **custom in-memory SQL executor** — it does NOT use actual DuckDB. It:
1. Reads the full parquet file from S3 into a Grafana data frame
2. Parses the SQL query with a simple regex-based parser
3. Executes SELECT, WHERE, GROUP BY, ORDER BY, LIMIT on the in-memory frame
4. All field values are nullable pointers (`*string`, `*int64`, `*float64`, etc.)
5. The `formatValue()` function must be used when comparing/grouping field values to avoid pointer address issues

### Aggregate Functions
`COUNT(*)`, `COUNT(col)`, `SUM(col)`, `AVG(col)`, `MIN(col)`, `MAX(col)`

### WHERE Operators
`=`, `!=`, `>`, `<`, `>=`, `<=`, `LIKE`, `IN`, `IS NULL`, `IS NOT NULL`

## Adding a New Query Option
1. Add field to `QueryModel` struct in `pkg/plugin/datasource.go`
2. Add field to `ParquetS3Query` interface in `src/types/index.ts`
3. Add UI control in `src/components/QueryEditor.tsx`
4. Handle in query execution logic in `pkg/duckdb/executor.go`

## Release Process

### Automated (CI)
```bash
npm version patch        # Bumps version, commits, creates tag
git push origin main --tags  # Triggers release workflow
```
The release workflow (`grafana/plugin-actions/build-plugin@main`) builds, signs, and publishes the zip automatically. Build provenance attestation is enabled (`attestation: true`) — generates verifiable SLSA provenance records for the release artifact.

### Build Provenance Attestation
Both CI and release workflows generate build provenance attestations:
- **Release** (`release.yml`): Uses `grafana/plugin-actions/build-plugin` with `attestation: true`
- **CI** (`ci.yml`): Uses `actions/attest-build-provenance@v2` on the packaged zip (push to main only, skipped on PRs)
- Required permissions: `id-token: write`, `contents: write`, `attestations: write`
- Attestations are viewable on the GitHub repo's attestations page

### Manual Release (when CI signing fails)
Use this when the `GRAFANA_ACCESS_POLICY_TOKEN` is expired or misconfigured:

```bash
# 1. Bump version
npm version patch
git push origin main --tags

# 2. Update dist/plugin.json version and updated date manually

# 3. Package with correct archive structure (root dir must be the plugin ID, NOT dist/)
PLUGIN_ID="tobiasworkstech-parquets3-datasource"
PLUGIN_VERSION="1.2.4"   # match dist/plugin.json
ARCHIVE="${PLUGIN_ID}-${PLUGIN_VERSION}.zip"
rm -f "${ARCHIVE}"
cp -r dist "${PLUGIN_ID}"
zip "${ARCHIVE}" "${PLUGIN_ID}" -r
rm -rf "${PLUGIN_ID}"

# 4. Create GitHub release and attach zip
gh release create v${PLUGIN_VERSION} "${ARCHIVE}" \
  --repo tobiasworkstech/tobiasworkstech-parquets3-datasource \
  --title "Parquet-S3-Datasource v${PLUGIN_VERSION}" \
  --notes "Release notes here." \
  --latest
```

### Known CI Signing Issue
**Symptom**: Release workflow fails at signing step with:
```
sign-plugin@3.2.1  Error signing manifest.
Server responded with status code 409 along with:
 • code: InvalidArgument
 • message: Field is required: rootUrls
```
**Cause**: `GRAFANA_ACCESS_POLICY_TOKEN` secret is expired or was created as a private plugin token instead of a community plugin token.
**Fix**: Regenerate the token on Grafana's portal (grafana.com → My Account → Access Policies) with `plugins:write` scope for a **community plugin** (no rootUrls required), then update the `GRAFANA_ACCESS_POLICY_TOKEN` secret in GitHub repo Settings → Secrets and variables → Actions.

### Archive Structure (Critical)
The zip must contain the plugin ID as the root directory — **not** `dist/`:
```
tobiasworkstech-parquets3-datasource/   ← correct
  plugin.json
  module.js
  gpx_parquet_s3_datasource_*
  ...
```
If submitted with `dist/` as root, Grafana validator returns: `❌ no-ident-root-dir`

### Grafana Marketplace Submission
- Submit the zip manually at Grafana's plugin submission portal
- MD5 checksum: `md5 <archive>.zip`
- Re-run a failed release workflow: `gh run rerun <run-id> --repo tobiasworkstech/tobiasworkstech-parquets3-datasource`

## Rules
- **`.config/` directory is managed by Grafana plugin tools — DO NOT MODIFY**
- **Never commit files containing `valdemarpavesi`** — use `tobiasworkstech` in all committed code
- No `console.log` in production frontend code
- Avoid `os.Getenv` and direct filesystem access in plugin Go code
