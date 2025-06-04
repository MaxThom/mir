package systemid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"strings"
)

// SystemInfo contains various system identifiers
type SystemInfo struct {
	// Hardware identifiers
	MACAddresses []string `json:"mac_addresses"`
	CPUInfo      string   `json:"cpu_info"`

	// OS identifiers
	OSType         string `json:"os_type"`
	OSVersion      string `json:"os_version"`
	OSArchitecture string `json:"os_architecture"`
	Hostname       string `json:"hostname"`

	// System identifiers
	MachineID    string `json:"machine_id"`
	SerialNumber string `json:"serial_number"`
	ProductUUID  string `json:"product_uuid"`
	BoardSerial  string `json:"board_serial"`

	// Disk identifiers
	DiskSerials []string `json:"disk_serials"`

	// Additional identifiers
	Username string `json:"username"`
	HomeDir  string `json:"home_dir"`
}

// Options for device ID generation
type Options struct {
	// Include network interfaces (IncludeMAC addresses)
	IncludeMAC bool
	// Include CPU information
	IncludeCPU bool
	// Include OS information
	IncludeOS bool
	// Include disk serials
	IncludeDisk bool
	// Include user information
	IncludeUser bool
	// Include machine-specific IDs
	IncludeMachineID bool
	// Include hostname
	IncludeHostname bool
	// Custom salt for ID generation
	Salt string
	// Leave the ID in plain text
	Plain bool
}

type PrefixOptions struct {
	Prefix   string
	Hostname bool
	Username bool
}

// DefaultOptions returns options with all identifiers enabled
func DefaultOptions() Options {
	return Options{
		IncludeMAC:       true,
		IncludeCPU:       true,
		IncludeOS:        true,
		IncludeDisk:      true,
		IncludeUser:      false, // Disabled by default for privacy
		IncludeMachineID: true,
		IncludeHostname:  true,
		Salt:             "",
	}
}

// DefaultUniqueOptions returns options with MAC, MachineID, and Hostname enabled
func DefaultUniqueOptions() Options {
	return Options{
		IncludeMAC:       true,
		IncludeCPU:       false,
		IncludeOS:        false,
		IncludeDisk:      false,
		IncludeUser:      false, // Disabled by default for privacy
		IncludeMachineID: true,
		IncludeHostname:  true,
		Salt:             "",
	}
}

// Collector interface for platform-specific implementations
type Collector interface {
	GetMACAddresses() ([]string, error)
	GetCPUInfo() (string, error)
	GetOSInfo() (osType, osVersion, osArch string, err error)
	GetHostname() (string, error)
	GetMachineID() (string, error)
	GetSerialNumber() (string, error)
	GetProductUUID() (string, error)
	GetBoardSerial() (string, error)
	GetDiskSerials() ([]string, error)
	GetUsername() (string, error)
	GetHomeDir() (string, error)
}

// GetSystemInfo collects all available system information
func GetSystemInfo() (*SystemInfo, error) {
	collector := newPlatformCollector()
	info := &SystemInfo{}

	// Collect MAC addresses
	if macs, err := collector.GetMACAddresses(); err == nil {
		info.MACAddresses = macs
	}

	// Collect CPU info
	if cpu, err := collector.GetCPUInfo(); err == nil {
		info.CPUInfo = cpu
	}

	// Collect OS info
	if osType, osVersion, osArch, err := collector.GetOSInfo(); err == nil {
		info.OSType = osType
		info.OSVersion = osVersion
		info.OSArchitecture = osArch
	} else {
		// Fallback to runtime info
		info.OSType = runtime.GOOS
		info.OSArchitecture = runtime.GOARCH
	}

	// Collect hostname
	if hostname, err := collector.GetHostname(); err == nil {
		info.Hostname = hostname
	}

	// Collect machine ID
	if machineID, err := collector.GetMachineID(); err == nil {
		info.MachineID = machineID
	}

	// Collect serial number
	if serial, err := collector.GetSerialNumber(); err == nil {
		info.SerialNumber = serial
	}

	// Collect product UUID
	if uuid, err := collector.GetProductUUID(); err == nil {
		info.ProductUUID = uuid
	}

	// Collect board serial
	if boardSerial, err := collector.GetBoardSerial(); err == nil {
		info.BoardSerial = boardSerial
	}

	// Collect disk serials
	if diskSerials, err := collector.GetDiskSerials(); err == nil {
		info.DiskSerials = diskSerials
	}

	// Collect user info
	if username, err := collector.GetUsername(); err == nil {
		info.Username = username
	}

	if homeDir, err := collector.GetHomeDir(); err == nil {
		info.HomeDir = homeDir
	}

	return info, nil
}

// GenerateDeviceID generates a unique device ID based on system information
func GenerateDeviceID(opts Options) (string, error) {
	info, err := GetSystemInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get system info: %w", err)
	}

	return GenerateDeviceIDFromInfo(info, opts)
}

// GenerateDeviceIDFromInfo generates a device ID from provided system info
func GenerateDeviceIDFromInfo(info *SystemInfo, opts Options) (string, error) {
	components := []string{}

	// Add components based on options
	if opts.IncludeMAC && len(info.MACAddresses) > 0 {
		// Sort MACs for consistency
		macs := make([]string, len(info.MACAddresses))
		copy(macs, info.MACAddresses)
		sort.Strings(macs)
		components = append(components, strings.Join(macs, ","))
	}

	if opts.IncludeCPU && info.CPUInfo != "" {
		components = append(components, info.CPUInfo)
	}

	if opts.IncludeOS {
		if info.OSType != "" {
			components = append(components, info.OSType)
		}
		if info.OSVersion != "" {
			components = append(components, info.OSVersion)
		}
		if info.OSArchitecture != "" {
			components = append(components, info.OSArchitecture)
		}
	}

	if opts.IncludeMachineID {
		// Try different machine identifiers in order of preference
		if info.MachineID != "" {
			components = append(components, info.MachineID)
		} else if info.ProductUUID != "" {
			components = append(components, info.ProductUUID)
		} else if info.SerialNumber != "" {
			components = append(components, info.SerialNumber)
		} else if info.BoardSerial != "" {
			components = append(components, info.BoardSerial)
		}
	}

	if opts.IncludeDisk && len(info.DiskSerials) > 0 {
		// Sort disk serials for consistency
		disks := make([]string, len(info.DiskSerials))
		copy(disks, info.DiskSerials)
		sort.Strings(disks)
		components = append(components, strings.Join(disks, ","))
	}

	if opts.IncludeUser {
		if info.Username != "" {
			components = append(components, info.Username)
		}
	}

	if opts.IncludeHostname && info.Hostname != "" {
		components = append(components, info.Hostname)
	}

	// Add hostname as a fallback if we have no other components
	if len(components) == 0 && info.Hostname != "" {
		components = append(components, info.Hostname)
	}

	if len(components) == 0 {
		return "", fmt.Errorf("no system identifiers found")
	}

	// Add salt if provided
	if opts.Salt != "" {
		components = append(components, opts.Salt)
	}

	// Generate hash
	data := normalizeString(strings.Join(components, "__"))
	if opts.Plain {
		return data, nil
	}

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// GetShortDeviceID returns a shorter version of the device ID (first 16 characters)
func GetShortDeviceID(opts Options) (string, error) {
	fullID, err := GenerateDeviceID(opts)
	if err != nil {
		return "", err
	}

	if len(fullID) > 16 && !opts.Plain {
		return fullID[:16], nil
	}

	return fullID, nil
}

func GetPrefix(opts PrefixOptions) (string, error) {
	info, err := GetSystemInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get system info: %w", err)
	}

	prefix := []string{}
	if opts.Prefix != "" {
		prefix = append(prefix, opts.Prefix)
	}
	if opts.Hostname {
		prefix = append(prefix, info.Hostname)
	}
	if opts.Username {
		prefix = append(prefix, info.Username)
	}

	return strings.Join(prefix, "-"), nil
}

// GetPrefixedDeviceID generates a device ID with an optional prefix
// This is a convenience function that combines GetPrefix and device ID generation
func GetPrefixedDeviceID(prefixOpts PrefixOptions, deviceOpts Options) (string, error) {
	prefix, err := GetPrefix(prefixOpts)
	if err != nil {
		return "", fmt.Errorf("failed to get prefix: %w", err)
	}

	deviceID, err := GenerateDeviceID(deviceOpts)
	if err != nil {
		return "", fmt.Errorf("failed to generate device ID: %w", err)
	}

	// For short device IDs
	shortID := deviceID
	if len(deviceID) > 16 {
		shortID = deviceID[:16]
	}

	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, shortID), nil
	}

	return shortID, nil
}

// normalizeString removes spaces and special characters, converting to lowercase
func normalizeString(s string) string {
	// Remove or replace common special characters
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")

	// Remove other special characters
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Clean up multiple consecutive hyphens
	cleaned := result.String()
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}

	// Trim hyphens from start and end
	cleaned = strings.Trim(cleaned, "-")

	return cleaned
}
