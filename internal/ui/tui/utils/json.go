package utils

import "encoding/json"

// FormatJSON pretty-prints a JSON string with the specified indentation.
// Returns the formatted JSON or an error if the input is invalid.
func FormatJSON(jsonStr string, prefix, indent string) (string, error) {
	if jsonStr == "" {
		return "", nil
	}

	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}

	prettyJSON, err := json.MarshalIndent(obj, prefix, indent)
	if err != nil {
		return "", err
	}

	return string(prettyJSON), nil
}
