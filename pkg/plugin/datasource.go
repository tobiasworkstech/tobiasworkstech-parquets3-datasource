package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/pkg/models"
	"github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/pkg/parquet"
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
	s, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if s.Endpoint != "" && service == s3.ServiceID {
			return aws.Endpoint{
				URL:               s.Endpoint,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.Secrets.AccessKey, s.Secrets.SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Info("Initializing S3 datasource", "region", s.Region, "endpoint", s.Endpoint)

	return &Datasource{
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.Region = s.Region
			if s.Endpoint != "" {
				o.UsePathStyle = true
				log.DefaultLogger.Info("Applying Path-Style routing", "endpoint", s.Endpoint)
			}
		}),
		settings: s,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	s3Client *s3.Client
	settings *models.PluginSettings
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm models.QueryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	frames, err := parquet.ReadParquetFromS3(ctx, d.s3Client, d.settings.Bucket, qm.Path)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("read parquet: %v", err.Error()))
	}

	response.Frames = append(response.Frames, frames...)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "region", d.settings.Region, "bucket", d.settings.Bucket, "endpoint", d.settings.Endpoint)

	if d.settings.Region == "" {
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
	if d.settings.Secrets.AccessKey == "" || d.settings.Secrets.SecretKey == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "S3 credentials are missing",
		}, nil
	}

	// Attempt to list objects to verify connectivity
	_, err := d.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
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

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working and connected to S3",
	}, nil
}
