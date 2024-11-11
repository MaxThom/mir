package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func unmarshalTypeFromStdInOrFile[T any](path string) ([]*T, error) {
	var empty []*T
	content, ok := ReadFromPipedStdIn()
	if !ok {
		contentB, err := os.ReadFile(path)
		content = string(contentB)
		if err != nil {
			e := MirDeserializationError{e: err}
			return empty, e
		}
	}
	var devs []*T
	if isJsonString(content) {
		if isJsonArray(content) {
			err := json.Unmarshal([]byte(content), &devs)
			if err != nil {
				e := MirDeserializationError{e: err}
				return empty, e
			}
		} else {
			dev := new(T)
			err := json.Unmarshal([]byte(content), dev)
			if err != nil {
				e := MirDeserializationError{e: err}
				return empty, e
			}
			devs = append(devs, dev)
		}
	} else {
		if isYamlArray(content) {
			err := yaml.Unmarshal([]byte(content), &devs)
			if err != nil {
				e := MirDeserializationError{e: err}
				return empty, e
			}
		} else {
			dev := new(T)
			err := yaml.Unmarshal([]byte(content), dev)
			if err != nil {
				e := MirDeserializationError{e: err}
				return empty, e
			}
			devs = append(devs, dev)
		}
	}

	return devs, nil
}

func mapToSortedString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted string
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}

func isPipedStdIn() bool {
	fi, e := os.Stdin.Stat()
	if e != nil {
		return false
	}
	return fi.Mode()&os.ModeNamedPipe != 0
}

func ReadFromPipedStdIn() (string, bool) {
	fi, e := os.Stdin.Stat()
	if e != nil {
		return "", false
	}
	// 0 equal no pipe
	if fi.Mode()&os.ModeNamedPipe != 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			return "", false
		}
		if len(data) > 0 {
			return string(data), true
		}
	}
	return "", false
}

func isJsonString(s string) bool {
	trimmed := strings.TrimSpace(s)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

func isJsonArray(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func isYamlArray(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "-")
}

func labelToPointerLabel(lbl map[string]string) map[string]*string {
	res := make(map[string]*string)
	for k, v := range lbl {
		res[k] = &v
	}
	return res
}

func getTargetFromNameNs(n string) Target {
	nameNs := strings.Split(n, "/")
	if nameNs[0] == "" || nameNs[0] == "*" {
		return Target{
			Namespaces: []string{nameNs[1]},
		}
	} else if len(nameNs) == 1 || (len(nameNs) > 1 && nameNs[1] == "") {
		return Target{
			Names: []string{nameNs[0]},
		}
	} else if len(nameNs) > 1 {
		return Target{
			Names:      []string{nameNs[0]},
			Namespaces: []string{nameNs[1]},
		}
	}
	return Target{}
}
