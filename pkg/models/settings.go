package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-aws-sdk/pkg/awsds"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type PluginSettings struct {
	awsds.AWSDatasourceSettings
	Bucket string `json:"bucket"`
}

func LoadPluginSettings(ctx backend.PluginContext, source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	backend.Logger.Info("Loading plugin settings", "json", string(source.JSONData))
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	// Load AWS settings from the datasource
	if err := settings.Load(source); err != nil {
		return nil, fmt.Errorf("could not load AWS settings: %w", err)
	}

	cutset := " \t\n\r`\""
	settings.Bucket = strings.Trim(settings.Bucket, cutset)

	return &settings, nil
}
