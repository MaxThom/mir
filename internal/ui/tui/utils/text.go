package utils

import (
	"fmt"
	"sort"
	"strings"
)

// WrapTextOptions configures text wrapping behavior
type WrapTextOptions struct {
	MaxLineWidth   int
	BreakWidth     int
	LinePrefix     string
	PreserveSpaces bool
}

// DefaultWrapOptions returns sensible defaults for text wrapping
func DefaultWrapOptions() WrapTextOptions {
	return WrapTextOptions{
		MaxLineWidth:   DefaultWrapWidth,
		BreakWidth:     DefaultBreakPoint,
		LinePrefix:     DefaultLinePrefix,
		PreserveSpaces: true,
	}
}

// WrapText wraps long text into multiple lines based on options.
// It attempts to break at word boundaries when possible.
func WrapText(text string, opts WrapTextOptions) string {
	if len(text) <= opts.MaxLineWidth {
		return text
	}

	lines := []string{}
	start := 0

	for start < len(text) {
		end := start + opts.MaxLineWidth
		if end > len(text) {
			end = len(text)
		} else {
			// Find the next space after the break point
			searchStart := start + opts.BreakWidth
			if searchStart < end && searchStart < len(text) {
				spaceIdx := strings.IndexByte(text[searchStart:end], ' ')
				if spaceIdx != -1 {
					end = searchStart + spaceIdx
				}
			}
		}

		lines = append(lines, text[start:end])
		start = end

		// Skip the space if we broke at one
		if start < len(text) && text[start] == ' ' {
			start++
		}
	}

	return strings.Join(lines, "\n"+opts.LinePrefix)
}

// MapToSortedString converts a map to a sorted key=value string.
// Keys are sorted alphabetically and joined with the specified separator.
func MapToSortedString(m map[string]string, separator string) string {
	if len(m) == 0 {
		return ""
	}

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build result
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(separator)
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}

// BuildDeviceTitle creates a device title with smart truncation.
// If there are more devices than maxDisplay, it shows the first few and indicates how many more.
func BuildDeviceTitle(ids []string, maxDisplay int) string {
	if len(ids) <= maxDisplay {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[0:maxDisplay], ", ") + " & " +
		fmt.Sprintf("%d more", len(ids)-maxDisplay)
}
