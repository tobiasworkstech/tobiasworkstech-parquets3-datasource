package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type PluginSettings struct {
	Region   string                `json:"region"`
	Bucket   string                `json:"bucket"`
	Endpoint string                `json:"endpoint"`
	UseSSL   bool                  `json:"useSSL"`
	Secrets  *SecretPluginSettings `json:"-"`
}

type SecretPluginSettings struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	backend.Logger.Info("Loading plugin settings", "json", string(source.JSONData))
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	cutset := " \t\n\r`\""
	settings.Region = strings.Trim(settings.Region, cutset)
	settings.Bucket = strings.Trim(settings.Bucket, cutset)
	settings.Endpoint = strings.Trim(settings.Endpoint, cutset)

	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)

	return &settings, nil
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	return &SecretPluginSettings{
		AccessKey: source["accessKey"],
		SecretKey: source["secretKey"],
	}
}
