package ts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/libs/api/health"
	"github.com/maxthom/mir/internal/libs/external/influx"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

type TelemetryStore interface {
	RetrieveMeasurementsFields(ctx context.Context, measurement string) ([]string, error)
	Query(ctx context.Context, ids []string, measurement string, fields []string, start time.Time, end time.Time, aggregationWindow string) (*mir_apiv1.QueryTelemetry, error)
	GetExploreQuery(ids []string, measurement string) string
	WriteDatapoint(string)
	Errors() <-chan error
}

type influxTelemetryStore struct {
	ctx     context.Context
	org     string
	bucket  string
	client  influxdb2.Client
	writer  api.WriteAPI
	querier api.QueryAPI
	errors  chan error
}

func NewInfluxTelemetryStore(ctx context.Context, org, bucket string, client influxdb2.Client) *influxTelemetryStore {
	s := &influxTelemetryStore{
		ctx:     ctx,
		org:     org,
		bucket:  bucket,
		client:  client,
		writer:  client.WriteAPI(org, bucket),
		querier: client.QueryAPI(org),
		errors:  make(chan error),
	}

	// Start monitoring writer errors
	go s.monitorErrors()
	go s.monitorConnection()

	return s
}

func (s *influxTelemetryStore) WriteDatapoint(line string) {
	s.writer.WriteRecord(line)
}

func (s *influxTelemetryStore) Errors() <-chan error {
	return s.errors
}

func (s *influxTelemetryStore) monitorConnection() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(5 * time.Second):
			_, err := s.client.Health(s.ctx)
			if err != nil {
				health.SetComponentUnready(health.ComponentInflux)
			} else {
				health.SetComponentReady(health.ComponentInflux)
			}
		}
	}
}

func (s *influxTelemetryStore) monitorErrors() {
	for err := range s.writer.Errors() {
		if err != nil && strings.Contains(err.Error(), "not found: organization") {
			if err := influx.CreateOrgAndBucket(context.TODO(), s.client, s.org, s.bucket); err != nil {
				s.errors <- err
			} else {
				s.errors <- fmt.Errorf("influx org '%s' and bucket '%s' not found, creating...", s.org, s.bucket)
			}
		} else if err != nil && strings.Contains(err.Error(), "not found: bucket") {
			if err := influx.CreateOrgAndBucket(context.TODO(), s.client, s.org, s.bucket); err != nil {
				s.errors <- err
			} else {
				s.errors <- fmt.Errorf("influx bucket '%s' not found, creating...", s.bucket)
			}
		} else if err != nil && strings.Contains(err.Error(), "connection refused") {
			health.SetComponentUnready(health.ComponentInflux)
		} else {
			s.errors <- err
		}
	}
	close(s.errors)
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

func (s *influxTelemetryStore) GetExploreQuery(ids []string, measurement string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`from(bucket: "%s")`, s.bucket))
	sb.WriteString("\n")
	sb.WriteString(`  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)`)
	sb.WriteString("\n")

	conds := []string{}
	for _, id := range ids {
		conds = append(conds, fmt.Sprintf(`r["__id"] == "%s"`, id))
	}
	sb.WriteString(fmt.Sprintf(`  |> filter(fn: (r) => %s)`, strings.Join(conds, " or ")))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`  |> filter(fn: (r) => r["_measurement"] == "%s")`, measurement))
	return sb.String()
}

// TODO
// [ ] integration tests
func (s *influxTelemetryStore) Query(ctx context.Context, ids []string, measurement string, fields []string, start time.Time, end time.Time, aggregationWindow string) (*mir_apiv1.QueryTelemetry, error) {
	values := mir_apiv1.QueryTelemetry{}

	// Build and Execute Flux query
	qry := generateInfluxQuery(s.bucket, ids, measurement, fields, start, end, aggregationWindow)
	result, err := s.querier.Query(context.Background(), qry)
	if err != nil {
		return nil, err
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("query parsing error: %w\n", result.Err())
	}

	// Parse Results
	// We always get _time and __id as the first 2 columns,
	// then the rest of the fields that do not start with _
	// __ are Mir system fields, we could add more
	// _ are Influx system fields, we ignore all of them except _time
	setFns := []func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint{}
	for result.Next() {
		// Influx can return multiple table, but the query we built
		// with pivot should always return only one
		// We parse the headers and generate the conversion functions
		// once then we use it for every row
		if result.TableChanged() {
			// Timestamp
			dt, setFn, err := generateDataConvFn("time")
			if err != nil {
				return nil, fmt.Errorf("unsupported data type '%s' for column '%s'", "time", "_time")
			}
			values.Headers = append(values.Headers, "_time")
			values.Datatypes = append(values.Datatypes, dt)
			setFns = append(setFns, setFn)

			// DeviceIds
			dt, setFn, err = generateDataConvFn("string")
			if err != nil {
				return nil, fmt.Errorf("unsupported data type '%s' for column '%s'", "string", "__id")
			}
			values.Headers = append(values.Headers, "__id")
			values.Datatypes = append(values.Datatypes, dt)
			setFns = append(setFns, setFn)

			// Other fields
			for _, col := range result.TableMetadata().Columns() {
				if strings.HasPrefix(col.Name(), "_") || col.Name() == "result" || col.Name() == "table" {
					continue
				}

				dt, setFn, err := generateDataConvFn(col.DataType())
				if err != nil {
					return nil, fmt.Errorf("unsupported data type '%s' for column '%s'", col.DataType(), col.Name())
				}
				values.Headers = append(values.Headers, col.Name())
				values.Datatypes = append(values.Datatypes, dt)
				setFns = append(setFns, setFn)
			}
		}
		// Each rows
		row := mir_apiv1.QueryTelemetry_Row{
			Datapoints: make([]*mir_apiv1.QueryTelemetry_Row_DataPoint, len(values.Headers)),
		}
		for i, col := range values.Headers {
			val := result.Record().ValueByKey(col)
			dp := setFns[i](val)
			row.Datapoints[i] = dp
		}
		values.Rows = append(values.Rows, &row)
	}

	return &values, nil
}

func generateInfluxQuery(bucket string, ids []string, measurement string, fields []string, start time.Time, end time.Time, aggregationWindow string) string {
	// Time
	if start.IsZero() {
		// 1h default
		start = time.Now().UTC().Add(-1 * time.Hour)
	}
	timeFilter := fmt.Sprintf(`|> range(start: %s)`, start.Format(time.RFC3339))
	if !end.IsZero() {
		timeFilter = fmt.Sprintf(`|> range(start: %s, stop: %s)`, start.Format(time.RFC3339), end.Format(time.RFC3339))
	}

	// Ids
	idsFilter := ""
	if len(ids) > 0 {
		filterIds := []string{}
		for _, id := range ids {
			filterIds = append(filterIds, fmt.Sprintf(`r["__id"] == "%s"`, id))
		}
		idsFilter = fmt.Sprintf("|> filter(fn: (r) => %s)", strings.Join(filterIds, " or "))
	}

	// Fields
	fieldsFilter := ""
	if len(fields) > 0 {
		fieldConds := []string{}
		for _, field := range fields {
			fieldConds = append(fieldConds, fmt.Sprintf(`r["_field"] == "%s"`, field))
		}
		fieldsFilter = fmt.Sprintf("|> filter(fn: (r) => %s)", strings.Join(fieldConds, " or "))
	}

	// Aggregation window
	aggregateStep := ""
	if aggregationWindow != "" {
		aggregateStep = fmt.Sprintf("|> aggregateWindow(every: %s, fn: mean, createEmpty: false)", aggregationWindow)
	}

	return fmt.Sprintf(`from(bucket:"%s")
		%s
		|> filter(fn: (r) => r["_measurement"] == "%s")
		%s
		%s
		%s
		|> pivot(
		    rowKey: ["_time", "__id"],
		    columnKey: ["_field"],
		    valueColumn: "_value"
		)
		|> group()
		|> sort(columns: ["_time"], desc: false)
		`, bucket, timeFilter, measurement, idsFilter, fieldsFilter, aggregateStep)
}

func generateDataConvFn(datatype string) (mir_apiv1.DataType, func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint, error) {
	switch datatype {
	case "string":
		return mir_apiv1.DataType_DATA_TYPE_STRING,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(string)
				if ok {
					r.ValueString = &v
				}
				return &r
			}, nil
	case "long":
		return mir_apiv1.DataType_DATA_TYPE_INT64,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(int64)
				if ok {
					r.ValueInt64 = &v
				}
				return &r
			}, nil
	case "double":
		return mir_apiv1.DataType_DATA_TYPE_DOUBLE,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(float64)
				if ok {
					r.ValueDouble = &v
				}
				return &r
			}, nil
	case "unsignedLong":
		return mir_apiv1.DataType_DATA_TYPE_UINT64,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(uint64)
				if ok {
					r.ValueUint64 = &v
				}
				return &r
			}, nil
	case "boolean":
		return mir_apiv1.DataType_DATA_TYPE_BOOL,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(bool)
				if ok {
					r.ValueBool = &v
				}
				return &r
			}, nil
	case "time":
		return mir_apiv1.DataType_DATA_TYPE_TIMESTAMP,
			func(val any) *mir_apiv1.QueryTelemetry_Row_DataPoint {
				r := mir_apiv1.QueryTelemetry_Row_DataPoint{}
				v, ok := val.(time.Time)
				if ok {
					r.ValueTimestamp = mir_v1.AsProtoTimestamp(v)
				}
				return &r
			}, nil
	default:
		return mir_apiv1.DataType_DATA_TYPE_UNSPECIFIED, nil, errors.New("invalid data type: " + datatype)
	}
}
