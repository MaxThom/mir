package mir_v1

import (
	"time"
)

type ValueType string
type GeneratorType string

const (
	Sin      GeneratorType = "sin"
	Cos      GeneratorType = "cos"
	Tan      GeneratorType = "tan"
	Random   GeneratorType = "random"
	Linear   GeneratorType = "linear"
	Constant GeneratorType = "constant"
	Sawtooth GeneratorType = "sawtooth"
	Square   GeneratorType = "square"
)

// e

const (
	Int8    ValueType = "int8"
	Int16   ValueType = "int16"
	Int32   ValueType = "int32"
	Int64   ValueType = "int64"
	Float32 ValueType = "float32"
	Float64 ValueType = "float64"
	Message ValueType = "message"
)

const ()

// SwarmConfig is the root configuration structure
type Swarm struct {
	Object `json:",inline" yaml:",inline"`
	Spec   SwarmSpec `yaml:"swarm"`
}

func NewSwarm() Swarm {
	return Swarm{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "swarm",
		},
	}
}

// SwarmSpec contains the main swarm configuration
type SwarmSpec struct {
	LogLevel        string                `yaml:"logLevel"`
	Devices         []SwarmDevice         `yaml:"devices"`
	TelemetryFields []SwarmTelemetryField `yaml:"telemetryFields"`
}

// SwarmDevice defines a device template with count and telemetry groups
type SwarmDevice struct {
	Count     int                   `yaml:"count"`
	Meta      SwarmDeviceMeta       `yaml:"meta"`
	Telemetry []SwarmTelemetryGroup `yaml:"telemetry"`
}

// SwarmDeviceMeta contains device metadata
type SwarmDeviceMeta struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	Namespace   string            `yaml:"namespace"`
}

// SwarmTelemetryGroup represents a proto message with update interval
type SwarmTelemetryGroup struct {
	Name     string            `yaml:"name"`
	Interval time.Duration     `yaml:"interval"`
	Tags     map[string]string `yaml:"tags,omitempty"`
	Fields   []string          `yaml:"fields"`
}

// SwarmTelemetryField defines a telemetry field with type and generator
type SwarmTelemetryField struct {
	Name      string                   `yaml:"name"`
	Type      ValueType                `yaml:"type"` // int8|int16|int32|int64|float32|float64|message
	Tags      map[string]string        `yaml:"tags,omitempty"`
	Generator *SwarmTelemetryGenerator `yaml:"generator,omitempty"` // nil for message types
	Fields    []string                 `yaml:"fields,omitempty"`    // for message types
}

// SwarmTelemetryGenerator defines value generation configuration
type SwarmTelemetryGenerator struct {
	Expr string
}
