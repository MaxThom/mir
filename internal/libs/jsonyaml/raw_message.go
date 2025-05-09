package jsonyaml

import (
	"encoding/json"
	"fmt"
)

// RawMessage is a wrapper around json.RawMessage that implements custom YAML marshaling
type RawMessage json.RawMessage

// MarshalYAML implements the yaml.Marshaler interface.
// It unmarshals the JSON data into a generic interface{} which will then be
// marshaled to YAML by the YAML encoder, effectively expanding the JSON structure.
func (y RawMessage) MarshalYAML() (interface{}, error) {
	if len(y) == 0 {
		return nil, nil
	}

	var result interface{}
	err := json.Unmarshal(y, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON for YAML conversion: %w", err)
	}

	return result, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It marshals the YAML data into JSON format.
func (y *RawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data interface{}
	if err := unmarshal(&data); err != nil {
		return err
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling YAML data to JSON: %w", err)
	}

	*y = bytes
	return nil
}

// MarshalJSON simply returns the raw JSON message.
func (y RawMessage) MarshalJSON() ([]byte, error) {
	return []byte(y), nil
}

// UnmarshalJSON sets the raw JSON message.
func (y *RawMessage) UnmarshalJSON(data []byte) error {
	*y = data
	return nil
}
