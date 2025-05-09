package jsonyaml

import (
	"encoding/json"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAMLMessageMarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected interface{}
	}{
		{
			name:     "empty",
			json:     ``,
			expected: nil,
		},
		{
			name:     "null",
			json:     `null`,
			expected: nil,
		},
		{
			name:     "string",
			json:     `"test"`,
			expected: "test",
		},
		{
			name:     "number",
			json:     `123`,
			expected: float64(123),
		},
		{
			name:     "boolean",
			json:     `true`,
			expected: true,
		},
		{
			name: "object",
			json: `{"key": "value", "nested": {"inner": 42}}`,
			expected: map[string]interface{}{
				"key": "value",
				"nested": map[string]interface{}{
					"inner": float64(42),
				},
			},
		},
		{
			name:     "array",
			json:     `[1, "two", true]`,
			expected: []interface{}{float64(1), "two", true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := RawMessage([]byte(tt.json))

			// Test MarshalYAML
			result, err := msg.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MarshalYAML() = %v, want %v", result, tt.expected)
			}

			// Test full YAML marshaling
			_, err = yaml.Marshal(struct {
				Data RawMessage `yaml:"data"`
			}{Data: msg})
			if err != nil {
				t.Fatalf("yaml.Marshal failed: %v", err)
			}

		})
	}
}

func TestYAMLMessageRoundTrip(t *testing.T) {
	// Create complex test data
	testData := map[string]interface{}{
		"string":  "hello",
		"number":  42,
		"boolean": true,
		"nested": map[string]interface{}{
			"array": []interface{}{1, 2, 3},
			"null":  nil,
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Create YAMLMessage
	msg := RawMessage(jsonBytes)

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(msg)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}

	// Unmarshal from YAML to new YAMLMessage
	var newMsg RawMessage
	if err := yaml.Unmarshal(yamlBytes, &newMsg); err != nil {
		t.Fatalf("yaml.Unmarshal failed: %v", err)
	}

	// Unmarshal both to interface{} for comparison
	var originalData interface{}
	var newData interface{}

	if err := json.Unmarshal(msg, &originalData); err != nil {
		t.Fatalf("json.Unmarshal of original failed: %v", err)
	}

	if err := json.Unmarshal(newMsg, &newData); err != nil {
		t.Fatalf("json.Unmarshal of new failed: %v", err)
	}

	// Compare the data
	if !reflect.DeepEqual(originalData, newData) {
		t.Errorf("Round-trip data doesn't match\nOriginal: %v\nNew: %v", originalData, newData)
	}
}
