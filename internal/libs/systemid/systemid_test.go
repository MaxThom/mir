package systemid

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get system info: %v", err)
	}

	// Print system info as JSON for debugging
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal system info: %v", err)
	}
	t.Logf("System Info:\n%s", string(data))

	// Verify we have at least some basic info
	if info.OSType == "" {
		t.Error("OS type should not be empty")
	}
	if info.OSArchitecture == "" {
		t.Error("OS architecture should not be empty")
	}
	if info.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
}

func TestGenerateDeviceID(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "Default options",
			opts: DefaultOptions(),
		},
		{
			name: "Only MAC addresses",
			opts: Options{
				IncludeMAC: true,
			},
		},
		{
			name: "Only OS info",
			opts: Options{
				IncludeOS: true,
			},
		},
		{
			name: "With custom salt",
			opts: Options{
				IncludeMAC:       true,
				IncludeOS:        true,
				IncludeMachineID: true,
				Salt:             "my-custom-salt",
			},
		},
		{
			name: "All identifiers",
			opts: Options{
				IncludeMAC:       true,
				IncludeCPU:       true,
				IncludeOS:        true,
				IncludeDisk:      true,
				IncludeUser:      true,
				IncludeMachineID: true,
				IncludeHostname:  true,
			},
		},
		{
			name: "With hostname only",
			opts: Options{
				IncludeHostname: true,
			},
		},
		{
			name: "Without hostname",
			opts: Options{
				IncludeMAC:      true,
				IncludeCPU:      true,
				IncludeHostname: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenerateDeviceID(tt.opts)
			if err != nil {
				t.Errorf("Failed to generate device ID: %v", err)
				return
			}

			if id == "" {
				t.Error("Device ID should not be empty")
			}

			// Device ID should be a 64-character hex string (SHA256)
			if len(id) != 64 {
				t.Errorf("Device ID should be 64 characters, got %d", len(id))
			}

			t.Logf("Generated device ID: %s", id)

			// Generate again with same options - should be identical
			id2, err := GenerateDeviceID(tt.opts)
			if err != nil {
				t.Errorf("Failed to generate device ID second time: %v", err)
				return
			}

			if id != id2 {
				t.Error("Device ID should be consistent across multiple generations")
			}
		})
	}
}

func TestGetShortDeviceID(t *testing.T) {
	opts := DefaultOptions()
	shortID, err := GetShortDeviceID(opts)
	if err != nil {
		t.Fatalf("Failed to get short device ID: %v", err)
	}

	if len(shortID) != 16 {
		t.Errorf("Short device ID should be 16 characters, got %d", len(shortID))
	}

	t.Logf("Short device ID: %s", shortID)
}

func TestDeviceIDUniqueness(t *testing.T) {
	// Test that different salt values produce different IDs
	opts1 := Options{
		IncludeMAC: true,
		Salt:       "salt1",
	}
	opts2 := Options{
		IncludeMAC: true,
		Salt:       "salt2",
	}

	id1, err := GenerateDeviceID(opts1)
	if err != nil {
		t.Fatalf("Failed to generate device ID with salt1: %v", err)
	}

	id2, err := GenerateDeviceID(opts2)
	if err != nil {
		t.Fatalf("Failed to generate device ID with salt2: %v", err)
	}

	if id1 == id2 {
		t.Error("Device IDs with different salts should be different")
	}
}

func ExampleGenerateDeviceID() {
	// Generate a device ID with default options
	opts := DefaultOptions()
	deviceID, err := GenerateDeviceID(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Device ID: %s\n", deviceID[:16]+"...")

	// Generate with custom options
	customOpts := Options{
		IncludeMAC:       true,
		IncludeMachineID: true,
		Salt:             "my-app-v1.0",
	}
	customID, err := GenerateDeviceID(customOpts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Custom Device ID: %s\n", customID[:16]+"...")
}

func ExampleGetSystemInfo() {
	info, err := GetSystemInfo()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("OS: %s %s (%s)\n", info.OSType, info.OSVersion, info.OSArchitecture)
	fmt.Printf("Hostname: %s\n", info.Hostname)
	fmt.Printf("MAC Addresses: %d found\n", len(info.MACAddresses))
}

// Benchmark device ID generation
func BenchmarkGenerateDeviceID(b *testing.B) {
	opts := DefaultOptions()
	for i := 0; i < b.N; i++ {
		_, err := GenerateDeviceID(opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetSystemInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetSystemInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestArchitectureNormalization(t *testing.T) {
	// This test demonstrates that the architecture is normalized
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get system info: %v", err)
	}

	// Check that architecture is normalized to Go conventions
	validArchitectures := map[string]bool{
		"amd64":    true, // 64-bit x86
		"386":      true, // 32-bit x86
		"arm64":    true, // 64-bit ARM (including Apple Silicon)
		"arm":      true, // 32-bit ARM (armv6, armv7)
		"mips64":   true,
		"mips64le": true,
		"ppc64":    true,
		"ppc64le":  true,
		"riscv64":  true,
		"s390x":    true,
		"unknown":  true, // Windows fallback
	}

	if !validArchitectures[info.OSArchitecture] {
		t.Errorf("Unexpected normalized architecture: %s", info.OSArchitecture)
	}

	t.Logf("Detected architecture: %s", info.OSArchitecture)
	t.Logf("CPU Info: %s", info.CPUInfo)
}

func TestHashIrreversibility(t *testing.T) {
	// Generate a device ID
	deviceID, err := GenerateDeviceID(DefaultOptions())
	if err != nil {
		t.Fatalf("Failed to generate device ID: %v", err)
	}

	// The device ID should be a 64-character hex string (SHA256)
	if len(deviceID) != 64 {
		t.Errorf("Device ID should be 64 characters, got %d", len(deviceID))
	}

	// Verify it's valid hex
	for _, c := range deviceID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Device ID contains non-hex character: %c", c)
		}
	}

	// Demonstrate that we cannot reverse the hash
	// Even with the same options, we can only verify by regenerating
	deviceID2, err := GenerateDeviceID(DefaultOptions())
	if err != nil {
		t.Fatalf("Failed to regenerate device ID: %v", err)
	}

	if deviceID != deviceID2 {
		t.Error("Device ID should be consistent for same system")
	}

	t.Logf("Irreversible device ID (SHA256): %s", deviceID)
}

func TestStructuredDeviceID(t *testing.T) {
	// Test structured ID that contains visible information
	structured, err := GenerateStructuredID(PrefixOptions{Hostname: true}, DefaultOptions())
	if err != nil {
		t.Fatalf("Failed to generate structured ID: %v", err)
	}

	// Convert to string
	idString := structured.ToString()
	t.Logf("Structured ID: %s", idString)

	// Parse it back
	parsed, err := ParseStructuredID(idString)
	if err != nil {
		t.Fatalf("Failed to parse structured ID: %v", err)
	}

	// Verify fields
	if parsed.Prefix != structured.Prefix {
		t.Errorf("Hostname mismatch: %s != %s", parsed.Prefix, structured.Prefix)
	}
	if parsed.UniqueHash != structured.UniqueHash {
		t.Errorf("Hash mismatch: %s != %s", parsed.UniqueHash, structured.UniqueHash)
	}

	// The hash portion is still irreversible
	t.Logf("Visible hostname: %s", parsed.Prefix)
	t.Logf("Irreversible hash portion: %s", parsed.UniqueHash)
}

func TestReversibleDeviceID(t *testing.T) {
	// WARNING: This test demonstrates reversible IDs which expose system info
	opts := Options{
		IncludeMAC: true,
		IncludeOS:  true,
		IncludeCPU: true,
	}

	// Generate reversible ID
	reversibleID, err := GenerateReversibleID(opts)
	if err != nil {
		t.Fatalf("Failed to generate reversible ID: %v", err)
	}

	t.Logf("Reversible ID (base64): %s", reversibleID)

	// Decode it back
	decoded, err := DecodeReversibleID(reversibleID)
	if err != nil {
		t.Fatalf("Failed to decode reversible ID: %v", err)
	}

	// Log what we recovered
	t.Logf("Decoded information:")
	t.Logf("  OS: %s %s (%s)", decoded.OSInfo.Type, decoded.OSInfo.Version, decoded.OSInfo.Arch)
	t.Logf("  Hostname: %s", decoded.Hostname)
	t.Logf("  CPU: %s", decoded.CPUInfo)
	if len(decoded.MACAddresses) > 0 {
		t.Logf("  MACs: %v", decoded.MACAddresses)
	}

	// Verify we can recover the information
	info, _ := GetSystemInfo()
	if decoded.OSInfo.Type != info.OSType {
		t.Errorf("OS type not recovered correctly: %s != %s", decoded.OSInfo.Type, info.OSType)
	}
}

func TestIDComparison(t *testing.T) {
	// Generate different types of IDs
	hashID1, _ := GenerateDeviceID(DefaultOptions())
	hashID2, _ := GenerateDeviceID(Options{IncludeMAC: true}) // Different options

	// Compare hash IDs - can only tell if identical
	comp1 := CompareIDs(hashID1, hashID1)
	if !comp1.Identical {
		t.Error("Same hash IDs should be identical")
	}

	comp2 := CompareIDs(hashID1, hashID2)
	if comp2.Identical {
		t.Error("Different hash IDs should not be identical")
	}

	// Compare structured IDs - can extract more info
	struct1, _ := GenerateStructuredID(PrefixOptions{Hostname: true}, DefaultOptions())
	struct2, _ := GenerateStructuredID(PrefixOptions{Hostname: true}, DefaultOptions())

	comp3 := CompareIDs(struct1.ToString(), struct2.ToString())
	if comp3.SameHostname == nil || !*comp3.SameHostname {
		t.Error("Structured IDs from same system should have same hostname")
	}

	t.Logf("Hash comparison - can only check identity: %v", comp2)
	t.Logf("Structured comparison - can check hostname: Hostname match=%v",
		*comp3.SameHostname)
}

func ExampleGenerateDeviceID_irreversible() {
	// Generate an irreversible device ID
	deviceID, _ := GenerateDeviceID(DefaultOptions())
	fmt.Printf("SHA256 Device ID: %s\n", deviceID[:16]+"...")

	// This ID cannot be reversed to get system info
	// You can only verify by regenerating with same system

	// For visible info, use structured ID
	structured, _ := GenerateStructuredID(PrefixOptions{Hostname: true}, DefaultOptions())
	fmt.Printf("Structured ID: %s\n", structured.ToString())
	// Output might be: mir-laptop-f282b622a980dca4
}
