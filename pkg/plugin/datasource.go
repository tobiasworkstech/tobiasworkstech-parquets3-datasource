package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/grafana/grafana-aws-sdk/pkg/awsauth"
	"github.com/grafana/grafana-aws-sdk/pkg/awsds"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/tobiasworkstech/parquets3-datasource/pkg/duckdb"
	"github.com/tobiasworkstech/parquets3-datasource/pkg/models"
	"github.com/tobiasworkstech/parquets3-datasource/pkg/parquet"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	pluginCtx := backend.PluginContext{
		DataSourceInstanceSettings: &settings,
	}
	s, err := models.LoadPluginSettings(pluginCtx, settings)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Info("Initializing S3 datasource", "region", s.Region, "endpoint", s.Endpoint, "authType", s.AuthType)

	return &Datasource{
		awsConfigProvider: awsauth.NewConfigProvider(),
		settings:          s,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	awsConfigProvider awsauth.ConfigProvider
	settings          *models.PluginSettings
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// getAWSConfig creates an AWS config using the grafana-aws-sdk auth provider
func (d *Datasource) getAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	httpClient, err := httpclient.New()
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	authSettings := awsauth.Settings{
		LegacyAuthType:     d.settings.AuthType,
		AssumeRoleARN:      d.settings.AssumeRoleARN,
		ExternalID:         d.settings.ExternalID,
		Endpoint:           d.settings.Endpoint,
		Region:             region,
		AccessKey:          d.settings.AccessKey,
		SecretKey:          d.settings.SecretKey,
		CredentialsProfile: d.settings.Profile,
		HTTPClient:         httpClient,
	}

	cfg, err := d.awsConfigProvider.GetConfig(ctx, authSettings)
	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}

// getS3Client creates an S3 client with the proper configuration
func (d *Datasource) getS3Client(ctx context.Context) (*s3.Client, error) {
	if d.settings == nil {
		return nil, fmt.Errorf("datasource settings not configured")
	}
	region := d.settings.Region
	if region == "" {
		region = d.settings.DefaultRegion
	}

	cfg, err := d.getAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if d.settings.Endpoint != "" {
			o.UsePathStyle = true
			log.DefaultLogger.Info("Applying Path-Style routing", "endpoint", d.settings.Endpoint)
		}
	}), nil
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {

	// create response struct
	response := backend.NewQueryDataResponse()

	s3Client, err := d.getS3Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, q, s3Client)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *Datasource) query(ctx context.Context, query backend.DataQuery, s3Client *s3.Client) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm models.QueryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	log.DefaultLogger.Debug("Query received", "refID", query.RefID, "queryType", qm.QueryType, "variableQueryType", qm.VariableQueryType, "path", qm.Path, "filePattern", qm.FilePattern)

	// Handle variable queries - check queryType, variableQueryType, or empty path with refId="variable"
	if qm.QueryType == "variable" || qm.VariableQueryType != "" || (qm.Path == "" && query.RefID == "variable") {
		return d.handleVariableQuery(ctx, s3Client, qm)
	}

	// If SQL query is provided, use DuckDB
	if qm.SQLQuery != "" {
		executor := duckdb.NewExecutor(s3Client, d.settings.Bucket)
		frames, err := executor.ExecuteSQL(ctx, qm.Path, qm.SQLQuery)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("execute SQL: %v", err.Error()))
		}
		response.Frames = append(response.Frames, frames...)
		return response
	}

	// Handle regular parquet queries (read entire file)
	frames, err := parquet.ReadParquetFromS3(ctx, s3Client, d.settings.Bucket, qm.Path)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("read parquet: %v", err.Error()))
	}

	response.Frames = append(response.Frames, frames...)

	return response
}

// handleVariableQuery handles template variable queries
func (d *Datasource) handleVariableQuery(ctx context.Context, s3Client *s3.Client, qm models.QueryModel) backend.DataResponse {
	var response backend.DataResponse

	// Determine the actual variable query type
	vqt := strings.ToLower(strings.TrimSpace(qm.VariableQueryType))

	log.DefaultLogger.Info("Variable query received", "rawVariableQueryType", qm.VariableQueryType, "normalizedVqt", vqt, "queryType", qm.QueryType)

	// Handle SQL queries
	if vqt == "sql" || qm.SQLQuery != "" {
		if qm.SQLQuery == "" {
			return backend.ErrDataResponse(backend.StatusBadRequest, "SQL query is required for sql variable type")
		}
		if qm.Path == "" {
			return backend.ErrDataResponse(backend.StatusBadRequest, "Path is required for sql variable type")
		}
		executor := duckdb.NewExecutor(s3Client, d.settings.Bucket)
		frames, err := executor.ExecuteSQL(ctx, qm.Path, qm.SQLQuery)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("execute SQL: %v", err))
		}
		// For variable queries, extract first column values and deduplicate
		if len(frames) > 0 && len(frames[0].Fields) > 0 {
			field := frames[0].Fields[0]
			seen := make(map[string]bool)
			var uniqueValues []string
			for i := 0; i < field.Len(); i++ {
				// Get the concrete value, handling nullable fields
				val := field.CopyAt(i)
				var strVal string
				if val == nil {
					strVal = ""
				} else if s, ok := val.(*string); ok && s != nil {
					strVal = *s
				} else {
					strVal = fmt.Sprintf("%v", val)
				}
				if !seen[strVal] {
					seen[strVal] = true
					uniqueValues = append(uniqueValues, strVal)
				}
			}
			response.Frames = append(response.Frames, createStringFrame("sql_results", uniqueValues))
		}
		return response
	}

	// Handle prefixes queries
	if vqt == "prefixes" {
		values, err := d.listPrefixes(ctx, s3Client, qm.Prefix)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("list prefixes: %v", err))
		}
		response.Frames = append(response.Frames, createStringFrame("prefixes", values))
		return response
	}

	// Default: list files (handles "files", "variable", "", or any other value)
	log.DefaultLogger.Info("Listing files", "prefix", qm.Prefix, "filePattern", qm.FilePattern)
	values, err := d.listFiles(ctx, s3Client, qm.Prefix, qm.FilePattern)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("list files: %v", err))
	}
	response.Frames = append(response.Frames, createStringFrame("files", values))
	return response
}

// listPrefixes lists prefixes (folders) in the S3 bucket
func (d *Datasource) listPrefixes(ctx context.Context, s3Client *s3.Client, prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.settings.Bucket),
		Delimiter: aws.String("/"),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	result, err := s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	var prefixes []string
	for _, cp := range result.CommonPrefixes {
		if cp.Prefix != nil {
			// Remove trailing slash and get the folder name
			p := strings.TrimSuffix(*cp.Prefix, "/")
			if prefix != "" {
				// Remove the parent prefix to get just the folder name
				p = strings.TrimPrefix(p, prefix)
			}
			prefixes = append(prefixes, p)
		}
	}

	return prefixes, nil
}

// listFiles lists parquet files in the S3 bucket matching the pattern
func (d *Datasource) listFiles(ctx context.Context, s3Client *s3.Client, prefix, pattern string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(d.settings.Bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	result, err := s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	// Default pattern to *.parquet if not specified
	if pattern == "" {
		pattern = "*.parquet"
	}

	var files []string
	for _, obj := range result.Contents {
		if obj.Key != nil {
			key := *obj.Key
			// Get just the filename
			filename := filepath.Base(key)
			// Match against pattern
			matched, _ := filepath.Match(pattern, filename)
			if matched {
				files = append(files, key)
			}
		}
	}

	return files, nil
}

// createStringFrame creates a data frame with a single string column
func createStringFrame(name string, values []string) *data.Frame {
	frame := data.NewFrame(name)
	frame.Fields = append(frame.Fields, data.NewField("value", nil, values))
	return frame
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	region := d.settings.Region
	if region == "" {
		region = d.settings.DefaultRegion
	}

	log.DefaultLogger.Info("CheckHealth called", "region", region, "bucket", d.settings.Bucket, "endpoint", d.settings.Endpoint, "authType", d.settings.AuthType)

	if region == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Region is missing",
		}, nil
	}
	if d.settings.Bucket == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Bucket is missing",
		}, nil
	}

	// Create S3 client for health check
	s3Client, err := d.getS3Client(ctx)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Failed to create S3 client: %v", err),
		}, nil
	}

	// Attempt to list objects to verify connectivity
	_, err = s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(d.settings.Bucket),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		log.DefaultLogger.Error("CheckHealth S3 connection failed", "error", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("S3 connection failed: %v", err),
		}, nil
	}

	// Determine auth type for success message
	at := d.settings.AuthType
	authType := at.String()
	if authType == "" {
		def := awsds.AuthTypeDefault
		authType = def.String()
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: fmt.Sprintf("Data source is working and connected to S3 (auth: %s)", authType),
	}, nil
}
