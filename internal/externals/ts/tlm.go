package ts

import (
	"context"
	"fmt"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
)

type TelemetryStore interface {
	RetrieveMeasurementsFields(ctx context.Context, measurement string) ([]string, error)
	WriteDatapoint(string)
	Errors() <-chan error
}

type influxTelemetryStore struct {
	org     string
	bucket  string
	client  influxdb2.Client
	writer  api.WriteAPI
	querier api.QueryAPI
}

func NewInfluxTelemetryStore(org, bucket string, client influxdb2.Client) *influxTelemetryStore {
	return &influxTelemetryStore{
		org:     org,
		bucket:  bucket,
		client:  client,
		writer:  client.WriteAPI(org, bucket),
		querier: client.QueryAPI(org),
	}
}

func (s *influxTelemetryStore) WriteDatapoint(line string) {
	s.writer.WriteRecord(line)
}

func (s *influxTelemetryStore) Errors() <-chan error {
	return s.writer.Errors()
}

type TagInfo struct {
	Name   string
	Values []string
}

func (s *influxTelemetryStore) RetrieveMeasurementsFields(ctx context.Context, measurement string) ([]string, error) {
	fields := []string{}
	query := fmt.Sprintf(`
			import "influxdata/influxdb/schema"
		    schema.measurementFieldKeys(
		        bucket: "%s",
		        measurement: "%s"
		    )
        `, s.bucket, measurement)
	result, err := s.querier.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	if result.Err() != nil {
		return nil, result.Err()
	}

	for result.Next() {
		record := result.Record()
		value := record.Value()
		fields = append(fields, value.(string))
	}

	return fields, nil
}

func (s *influxTelemetryStore) GenerateTelemetryQuery(target *core_apiv1.Targets, measurement string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`from(bucket: "%s")
  			|> range(start: v.timeRangeStart, stop: v.timeRangeStop)
			|> filter(fn: (r) => r["_measurement"] == "%s")
			|> filter(fn: (r) => r["__id"] == "0xf86tlm")
	`, s.bucket, measurement))

	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => )`))
	conds := []string{}
	for _, id := range target.Ids {
		conds = append(conds, fmt.Sprintf(`r["__id"] == "%s"`, id))
	}
	for _, name := range target.Names {
		conds = append(conds, fmt.Sprintf(`r["__name"] == "%s"`, name))
	}
	for _, ns := range target.Namespaces {
		conds = append(conds, fmt.Sprintf(`r["__namespace"] == "%s"`, ns))
	}
	for k, v := range target.Labels {
		sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r["%s"] == "%s")`, k, v))
	}
	return ""
}
