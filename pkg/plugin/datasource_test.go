package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryDataReturnsErrorWithoutConfig(t *testing.T) {
	ds := Datasource{}

	_, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err == nil {
		t.Error("expected error when datasource is not configured, got nil")
	}
}
