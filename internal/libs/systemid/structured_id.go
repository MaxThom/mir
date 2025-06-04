package systemid

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// StructuredDeviceID represents a device ID that contains visible, parseable information
type StructuredDeviceID struct {
	// Visible components
	Prefix string `json:"prefix"`

	// Hashed unique identifier (irreversible)
	UniqueHash string `json:"hash"`
}

// GenerateStructuredID creates a device ID with both visible and hashed components
func GenerateStructuredID(prefixOpts PrefixOptions, opts Options) (*StructuredDeviceID, error) {
	info, err := GetSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	// Generate the irreversible hash
	hash, err := GenerateDeviceIDFromInfo(info, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hash: %w", err)
	}

	prefix := []string{}
	if prefixOpts.Prefix != "" {
		prefix = append(prefix, prefixOpts.Prefix)
	}
	if prefixOpts.Hostname {
		prefix = append(prefix, info.Hostname)
	}
	if prefixOpts.Username {
		prefix = append(prefix, info.Username)
	}

	return &StructuredDeviceID{
		Prefix:     strings.Join(prefix, "-"),
		UniqueHash: hash[:16], // Use first 16 chars for brevity
	}, nil
}

// ToString converts the structured ID to a string format
func (s *StructuredDeviceID) ToString() string {
	if s.Prefix == "" {
		return s.UniqueHash
	}
	return fmt.Sprintf("%s_%s", s.Prefix, s.UniqueHash)
}

// ParseStructuredID parses a structured ID string
func ParseStructuredID(id string) (*StructuredDeviceID, error) {
	lastDashIndex := strings.LastIndex(id, "_")
	if lastDashIndex == -1 {
		return nil, fmt.Errorf("invalid structured ID format")
	}

	return &StructuredDeviceID{
		Prefix:     id[:lastDashIndex],
		UniqueHash: id[lastDashIndex+1:],
	}, nil
}

// ReversibleDeviceInfo contains system information that can be encoded/decoded
type ReversibleDeviceInfo struct {
	Version      int      `json:"v"`
	MACAddresses []string `json:"mac,omitempty"`
	CPUInfo      string   `json:"cpu,omitempty"`
	OSInfo       struct {
		Type    string `json:"type"`
		Version string `json:"ver"`
		Arch    string `json:"arch"`
	} `json:"os"`
	Hostname string `json:"host,omitempty"`
}

// GenerateReversibleID creates an encoded device ID that can be decoded
// WARNING: This reveals system information and should only be used when necessary
func GenerateReversibleID(opts Options) (string, error) {
	info, err := GetSystemInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get system info: %w", err)
	}

	reversible := &ReversibleDeviceInfo{
		Version:  1,
		Hostname: info.Hostname,
	}

	if opts.IncludeMAC {
		reversible.MACAddresses = info.MACAddresses
	}

	if opts.IncludeCPU {
		reversible.CPUInfo = info.CPUInfo
	}

	if opts.IncludeOS {
		reversible.OSInfo.Type = info.OSType
		reversible.OSInfo.Version = info.OSVersion
		reversible.OSInfo.Arch = info.OSArchitecture
	}

	// Encode to JSON then base64
	data, err := json.Marshal(reversible)
	if err != nil {
		return "", fmt.Errorf("failed to marshal info: %w", err)
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// DecodeReversibleID decodes a reversible device ID back to system information
func DecodeReversibleID(encodedID string) (*ReversibleDeviceInfo, error) {
	data, err := base64.URLEncoding.DecodeString(encodedID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var info ReversibleDeviceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &info, nil
}

// CompareDeviceIDs compares two device IDs and returns what can be determined
type IDComparison struct {
	// What we can determine from hashed IDs
	Identical bool `json:"identical"`

	// What we can determine from structured IDs
	SameHostname *bool `json:"same_hostname,omitempty"`
}

// CompareIDs compares two device IDs and returns what can be determined
func CompareIDs(id1, id2 string) IDComparison {
	comp := IDComparison{
		Identical: id1 == id2,
	}

	// Try to parse as structured IDs
	struct1, err1 := ParseStructuredID(id1)
	struct2, err2 := ParseStructuredID(id2)

	if err1 == nil && err2 == nil {
		sameHostname := struct1.Prefix == struct2.Prefix
		comp.SameHostname = &sameHostname
	}

	return comp
}
