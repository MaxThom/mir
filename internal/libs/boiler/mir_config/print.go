package mir_config

import (
	"fmt"
	"reflect"
	"strings"

	"encoding/json"
)

// Marshal the struct to json replacing the value of the fields tagged with
// `cfg:"secret"` with '****'. Useful to print configuration or other structs
// with secret fields that we want to hide such as passwords or keys.
// If the secret field is set to default value, it will be left empty
// so we know a secret might not have been loaded. Use pointers to distinguish
// between default values and not present.
//
// Supports json tag options:
// - omitempty: omits the field if it's an empty value (empty string, zero value, nil pointer, empty slice/map)
// - omitzero: omits the field if it's the zero value for its type
func JsonMarshalWithoutSecrets(v any) ([]byte, error) {
	return jsonMarshalWithoutSecretsValue(reflect.ValueOf(v), false)
}

func jsonMarshalWithoutSecretsValue(val reflect.Value, isSecret bool) ([]byte, error) {
	// Handle nil interfaces and pointers
	if !val.IsValid() || (val.Kind() == reflect.Ptr && val.IsNil()) {
		return json.Marshal(nil)
	}

	// Dereference pointers
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		data := make(map[string]any)
		for i := 0; i < val.Type().NumField(); i++ {
			// If field is private, it can't interface
			if !val.Field(i).CanInterface() {
				continue
			}
			fieldType := val.Type().Field(i)
			tag := fieldType.Tag.Get("cfg")

			jsonTag := fieldType.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = fieldType.Name
			} else if jsonTag == "-" {
				continue
			}
			// Parse json tag options
			var omitempty, omitzero bool
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				options := jsonTag[idx+1:]
				jsonTag = jsonTag[:idx]
				omitempty = strings.Contains(options, "omitempty")
				omitzero = strings.Contains(options, "omitzero")
			}

			fieldVal := val.Field(i)
			isFieldSecret := tag == "secret"
			
			// Check if field should be omitted
			if shouldOmitField(fieldVal, omitempty, omitzero) {
				continue
			}
			
			// Special handling for collections with secret tag
			if isFieldSecret && (fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array || fieldVal.Kind() == reflect.Map) {
				processed, err := processSecretCollection(fieldVal)
				if err != nil {
					return nil, err
				}
				data[jsonTag] = processed
			} else if isFieldSecret && !isDefaultValue(fieldVal) {
				// Non-collection secret fields
				data[jsonTag] = "****"
			} else {
				// Regular fields
				processed, err := processFieldValue(fieldVal, false)
				if err != nil {
					return nil, err
				}
				data[jsonTag] = processed
			}
		}
		return json.Marshal(data)

	case reflect.Map:
		if isSecret {
			return processSecretMap(val)
		}
		data := make(map[string]any)
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			mapVal := val.MapIndex(key)
			processed, err := processFieldValue(mapVal, false)
			if err != nil {
				return nil, err
			}
			data[keyStr] = processed
		}
		return json.Marshal(data)

	case reflect.Slice, reflect.Array:
		if isSecret {
			return processSecretSlice(val)
		}
		var data []any
		for i := 0; i < val.Len(); i++ {
			processed, err := processFieldValue(val.Index(i), false)
			if err != nil {
				return nil, err
			}
			data = append(data, processed)
		}
		return json.Marshal(data)

	case reflect.Interface:
		if val.IsNil() {
			return json.Marshal(nil)
		}
		return jsonMarshalWithoutSecretsValue(val.Elem(), isSecret)

	default:
		// For primitive types
		if isSecret && !isDefaultValue(val) {
			return json.Marshal("****")
		}
		return json.Marshal(val.Interface())
	}
}

func processSecretCollection(val reflect.Value) (any, error) {
	switch val.Kind() {
	case reflect.Map:
		data := make(map[string]any)
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			mapVal := val.MapIndex(key)
			if !isDefaultValue(mapVal) {
				data[keyStr] = "****"
			} else {
				data[keyStr] = mapVal.Interface()
			}
		}
		return data, nil
		
	case reflect.Slice, reflect.Array:
		var data []any
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if !isDefaultValue(elem) {
				data = append(data, "****")
			} else {
				data = append(data, elem.Interface())
			}
		}
		return data, nil
		
	default:
		return nil, fmt.Errorf("unexpected collection type: %v", val.Kind())
	}
}

func processSecretMap(val reflect.Value) ([]byte, error) {
	data := make(map[string]any)
	for _, key := range val.MapKeys() {
		keyStr := fmt.Sprintf("%v", key.Interface())
		mapVal := val.MapIndex(key)
		if !isDefaultValue(mapVal) {
			data[keyStr] = "****"
		} else {
			data[keyStr] = mapVal.Interface()
		}
	}
	return json.Marshal(data)
}

func processSecretSlice(val reflect.Value) ([]byte, error) {
	var data []any
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		if !isDefaultValue(elem) {
			data = append(data, "****")
		} else {
			data = append(data, elem.Interface())
		}
	}
	return json.Marshal(data)
}

func processFieldValue(val reflect.Value, isSecret bool) (any, error) {
	if !val.IsValid() {
		return nil, nil
	}

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array, reflect.Interface:
		nestedData, err := jsonMarshalWithoutSecretsValue(val, isSecret)
		if err != nil {
			return nil, err
		}
		var result any
		if err := json.Unmarshal(nestedData, &result); err != nil {
			return nil, err
		}
		return result, nil
	default:
		if isSecret && !isDefaultValue(val) {
			return "****", nil
		}
		return val.Interface(), nil
	}
}

func isDefaultValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return true
	}
	zeroValue := reflect.Zero(v.Type()).Interface()
	return reflect.DeepEqual(v.Interface(), zeroValue)
}

func shouldOmitField(v reflect.Value, omitempty, omitzero bool) bool {
	if !v.IsValid() {
		return omitempty || omitzero
	}
	
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return omitempty || omitzero
		}
		v = v.Elem()
	}
	
	// Check omitzero
	if omitzero && isDefaultValue(v) {
		return true
	}
	
	// Check omitempty
	if omitempty {
		switch v.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			return v.Len() == 0
		case reflect.String:
			return v.String() == ""
		case reflect.Bool:
			return !v.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint() == 0
		case reflect.Float32, reflect.Float64:
			return v.Float() == 0
		case reflect.Interface, reflect.Ptr:
			return v.IsNil()
		}
	}
	
	return false
}