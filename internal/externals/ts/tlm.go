package ts

import "github.com/influxdata/influxdb-client-go/v2/api"

type TelemetryStore interface {
	WriteDatapoint(string)
	Errors() <-chan error
}

type influxTelemetryStore struct {
	writer api.WriteAPI
}

func NewInfluxTelemetryStore(writer api.WriteAPI) *influxTelemetryStore {
	return &influxTelemetryStore{
		writer: writer,
	}
}

func (s *influxTelemetryStore) WriteDatapoint(line string) {
	s.writer.WriteRecord(line)
}

func (s *influxTelemetryStore) Errors() <-chan error {
	return s.writer.Errors()
}
