package mir_config

import (
	"reflect"

	"encoding/json"
)

// Marshal the struct to json replacing the value of the fields tagged with
// `cfg:"secret"` with '****'. Useful to print configuration or other structs
// with secret fields that we want to hide such as passwords or keys.
// If the secret field is set to default value, it will be left empty
// so we know a secret might not have been loaded. Use pointers to distinguish
// between default values and not present.
func JsonMarshalWithoutSecrets(v any) ([]byte, error) {
	data := make(map[string]any)
	val := reflect.ValueOf(v)

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
		}

		if tag == "secret" && !isDefaultValue(val.Field(i)) {
			data[jsonTag] = "****"
		} else if fieldType.Type.Kind() == reflect.Struct {
			nestedData, err := JsonMarshalWithoutSecrets(val.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			var nestedMap map[string]any
			if err := json.Unmarshal(nestedData, &nestedMap); err != nil {
				return nil, err
			}
			data[jsonTag] = nestedMap
		} else {
			data[jsonTag] = val.Field(i).Interface()
		}
	}
	return json.Marshal(data)
}

func isDefaultValue(v reflect.Value) bool {
	zeroValue := reflect.Zero(v.Type()).Interface()
	return reflect.DeepEqual(v.Interface(), zeroValue)
}
