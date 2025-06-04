package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/maxthom/mir/internal/libs/systemid"
)

func main() {
	var (
		showInfo     = flag.Bool("info", false, "Show detailed system information")
		showShort    = flag.Bool("short", false, "Show short device ID (16 chars)")
		jsonOutput   = flag.Bool("json", false, "Output system info as JSON")
		includeMAC   = flag.Bool("mac", true, "Include MAC addresses")
		includeCPU   = flag.Bool("cpu", true, "Include CPU information")
		includeOS    = flag.Bool("os", true, "Include OS information")
		includeDisks = flag.Bool("disks", true, "Include disk serials")
		includeUser  = flag.Bool("user", false, "Include user information")
		includeMID   = flag.Bool("machine-id", true, "Include machine ID")
		includeHost  = flag.Bool("hostname", true, "Include hostname")
		salt         = flag.String("salt", "", "Custom salt for ID generation")
		structured   = flag.Bool("structured", false, "Generate structured ID with visible OS/arch")
		reversible   = flag.Bool("reversible", false, "Generate reversible ID (WARNING: exposes system info)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate a unique device ID based on system identifiers.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Generate device ID with default options\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -short             # Generate short device ID\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -info              # Show system information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -info -json        # Show system info as JSON\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -salt myapp-v1     # Generate ID with custom salt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -mac -no-cpu       # Only use MAC addresses\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -structured        # Generate structured ID (hostname-hash)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -reversible        # Generate reversible ID (WARNING: exposes info)\n", os.Args[0])
	}

	flag.Parse()

	if *showInfo {
		showSystemInfo(*jsonOutput)
		return
	}

	// Configure options
	opts := systemid.Options{
		IncludeMAC:       *includeMAC,
		IncludeCPU:       *includeCPU,
		IncludeOS:        *includeOS,
		IncludeDisk:      *includeDisks,
		IncludeUser:      *includeUser,
		IncludeMachineID: *includeMID,
		IncludeHostname:  *includeHost,
		Salt:             *salt,
	}

	// Generate device ID based on selected type
	var err error

	switch {
	case *structured:
		// Generate structured ID with visible OS/arch info
		structID, err := systemid.GenerateStructuredID(systemid.PrefixOptions{Hostname: true}, opts)
		if err != nil {
			log.Fatalf("Error generating structured ID: %v", err)
		}
		fmt.Println(structID.ToString())

	case *reversible:
		// Generate reversible ID (WARNING: exposes system information)
		revID, err := systemid.GenerateReversibleID(opts)
		if err != nil {
			log.Fatalf("Error generating reversible ID: %v", err)
		}
		fmt.Println(revID)

		// Demonstrate decoding
		if decoded, err := systemid.DecodeReversibleID(revID); err == nil {
			fmt.Fprintf(os.Stderr, "\nDecoded info: OS=%s/%s, Host=%s\n",
				decoded.OSInfo.Type, decoded.OSInfo.Arch, decoded.Hostname)
		}

	default:
		// Generate standard irreversible hash ID
		var deviceID string
		if *showShort {
			deviceID, err = systemid.GetShortDeviceID(opts)
		} else {
			deviceID, err = systemid.GenerateDeviceID(opts)
		}
		if err != nil {
			log.Fatalf("Error generating device ID: %v", err)
		}
		fmt.Println(deviceID)
	}
}

func showSystemInfo(asJSON bool) {
	info, err := systemid.GetSystemInfo()
	if err != nil {
		log.Fatalf("Error getting system info: %v", err)
	}

	if asJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		}
		fmt.Println(string(data))
		return
	}

	// Pretty print system information
	fmt.Println("System Information")
	fmt.Println("==================")
	fmt.Printf("OS Type:         %s\n", info.OSType)
	fmt.Printf("OS Version:      %s\n", info.OSVersion)
	fmt.Printf("OS Architecture: %s", info.OSArchitecture)

	// Add architecture-specific notes
	switch info.OSArchitecture {
	case "arm":
		fmt.Println(" (32-bit ARM - ARMv6/v7)")
	case "arm64":
		fmt.Println(" (64-bit ARM - ARMv8/Apple Silicon)")
	case "amd64":
		fmt.Println(" (64-bit x86)")
	case "386":
		fmt.Println(" (32-bit x86)")
	default:
		fmt.Println()
	}

	fmt.Printf("Hostname:        %s\n", info.Hostname)
	fmt.Printf("Username:        %s\n", info.Username)
	fmt.Printf("Home Directory:  %s\n", info.HomeDir)
	fmt.Println()

	fmt.Println("Hardware Information")
	fmt.Println("===================")
	fmt.Printf("CPU Info:        %s\n", info.CPUInfo)
	fmt.Printf("Machine ID:      %s\n", info.MachineID)
	fmt.Printf("Serial Number:   %s\n", info.SerialNumber)
	fmt.Printf("Product UUID:    %s\n", info.ProductUUID)
	fmt.Printf("Board Serial:    %s\n", info.BoardSerial)
	fmt.Println()

	fmt.Println("Network Interfaces")
	fmt.Println("==================")
	if len(info.MACAddresses) > 0 {
		for i, mac := range info.MACAddresses {
			fmt.Printf("MAC Address %d:   %s\n", i+1, mac)
		}
	} else {
		fmt.Println("No MAC addresses found")
	}
	fmt.Println()

	fmt.Println("Storage Devices")
	fmt.Println("===============")
	if len(info.DiskSerials) > 0 {
		for i, serial := range info.DiskSerials {
			fmt.Printf("Disk Serial %d:   %s\n", i+1, serial)
		}
	} else {
		fmt.Println("No disk serials found")
	}
	fmt.Println()

	// Generate and show device IDs with different options
	fmt.Println("Device ID Examples")
	fmt.Println("==================")

	// Default ID
	if id, err := systemid.GenerateDeviceID(systemid.DefaultOptions()); err == nil {
		fmt.Printf("Default ID:      %s\n", id)
	}

	// Short ID
	if id, err := systemid.GetShortDeviceID(systemid.DefaultOptions()); err == nil {
		fmt.Printf("Short ID:        %s\n", id)
	}

	// MAC only
	if id, err := systemid.GenerateDeviceID(systemid.Options{IncludeMAC: true}); err == nil {
		fmt.Printf("MAC-only ID:     %s\n", id)
	}

	// With custom salt
	if id, err := systemid.GenerateDeviceID(systemid.Options{
		IncludeMAC:       true,
		IncludeMachineID: true,
		Salt:             "example-app",
	}); err == nil {
		fmt.Printf("With salt:       %s\n", id)
	}

	// Architecture-specific device ID (useful for multi-arch deployments)
	if id, err := systemid.GenerateDeviceID(systemid.Options{
		IncludeMAC:       true,
		IncludeOS:        true,
		IncludeMachineID: true,
		Salt:             fmt.Sprintf("app-%s", info.OSArchitecture),
	}); err == nil {
		fmt.Printf("Arch-specific:   %s (salt: app-%s)\n", id, info.OSArchitecture)
	}

	fmt.Println()
	fmt.Println("ID Type Comparison")
	fmt.Println("==================")

	// Structured ID (partially visible)
	if structID, err := systemid.GenerateStructuredID(systemid.PrefixOptions{Hostname: true}, systemid.DefaultOptions()); err == nil {
		fmt.Printf("Structured:      %s\n", structID.ToString())
		fmt.Printf("                 └─ Hostname is visible, hash is irreversible\n")
	}

	// Reversible ID (fully decodable)
	if revID, err := systemid.GenerateReversibleID(systemid.Options{
		IncludeMAC: true,
		IncludeOS:  true,
	}); err == nil {
		fmt.Printf("Reversible:      %s...\n", revID[:32])
		if decoded, err := systemid.DecodeReversibleID(revID); err == nil {
			fmt.Printf("                 └─ Can decode: %d MACs, OS=%s/%s\n",
				len(decoded.MACAddresses), decoded.OSInfo.Type, decoded.OSInfo.Arch)
		}
	}

	// Standard hash (irreversible)
	if hashID, err := systemid.GenerateDeviceID(systemid.DefaultOptions()); err == nil {
		fmt.Printf("SHA256 Hash:     %s\n", hashID)
		fmt.Printf("                 └─ Cannot reverse - one-way cryptographic hash\n")
	}
}
