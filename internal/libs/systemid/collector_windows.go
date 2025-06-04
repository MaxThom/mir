//go:build windows

package systemid

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"unsafe"
)

type windowsCollector struct{}

func newPlatformCollector() Collector {
	return &windowsCollector{}
}

func (c *windowsCollector) GetMACAddresses() ([]string, error) {
	var macs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and virtual interfaces
		if iface.Flags&net.FlagLoopback != 0 ||
			strings.Contains(iface.Name, "Virtual") ||
			strings.Contains(iface.Name, "VirtualBox") ||
			strings.Contains(iface.Name, "VMware") {
			continue
		}

		if iface.HardwareAddr.String() != "" && iface.HardwareAddr.String() != "00:00:00:00:00:00" {
			macs = append(macs, iface.HardwareAddr.String())
		}
	}

	return macs, nil
}

func (c *windowsCollector) GetCPUInfo() (string, error) {
	cmd := exec.Command("wmic", "cpu", "get", "Name,Manufacturer,ProcessorId", "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	var name, manufacturer, processorID string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			name = strings.TrimPrefix(line, "Name=")
		} else if strings.HasPrefix(line, "Manufacturer=") {
			manufacturer = strings.TrimPrefix(line, "Manufacturer=")
		} else if strings.HasPrefix(line, "ProcessorId=") {
			processorID = strings.TrimPrefix(line, "ProcessorId=")
		}
	}

	return fmt.Sprintf("%s|%s|%s", name, manufacturer, processorID), nil
}

func (c *windowsCollector) GetOSInfo() (osType, osVersion, osArch string, err error) {
	osType = "windows"

	// Get version using registry
	cmd := exec.Command("cmd", "/c", "ver")
	if output, err := cmd.Output(); err == nil {
		osVersion = strings.TrimSpace(string(output))
	}

	// Get architecture
	osArch = getWindowsArch()

	return osType, osVersion, osArch, nil
}

func getWindowsArch() string {
	// Check PROCESSOR_ARCHITECTURE environment variable first
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")

	// Normalize architecture names to match Go's runtime.GOARCH
	switch arch {
	case "AMD64":
		return "amd64"
	case "x86", "X86":
		return "386"
	case "ARM64":
		return "arm64"
	case "ARM":
		return "arm"
	}

	// Check if we're a 32-bit process on 64-bit Windows
	var isWow64 bool
	err := windows_IsWow64Process(syscall.Handle(1), &isWow64)
	if err == nil && isWow64 {
		// We're running under WOW64, check native architecture
		nativeArch := os.Getenv("PROCESSOR_ARCHITEW6432")
		switch nativeArch {
		case "AMD64":
			return "amd64"
		case "ARM64":
			return "arm64"
		}
	}

	// For Windows on ARM, check if we're running x86 emulation
	if arch == "x86" || arch == "X86" {
		// Additional check for ARM-based Windows
		cmd := exec.Command("wmic", "cpu", "get", "Architecture", "/value")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Architecture=") {
					archCode := strings.TrimPrefix(line, "Architecture=")
					// Architecture codes from WMI:
					// 0 = x86, 5 = ARM, 9 = x64, 12 = ARM64
					switch archCode {
					case "0":
						return "386"
					case "5":
						return "arm"
					case "9":
						return "amd64"
					case "12":
						return "arm64"
					}
				}
			}
		}
	}

	// Default fallback based on what we detected
	if arch == "x86" || arch == "X86" {
		return "386"
	}

	return "unknown"
}

func windows_IsWow64Process(handle syscall.Handle, isWow64 *bool) error {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer dll.Release()

	proc, err := dll.FindProc("IsWow64Process")
	if err != nil {
		// Function doesn't exist on older Windows versions
		return err
	}

	r1, _, err := proc.Call(uintptr(handle), uintptr(unsafe.Pointer(isWow64)))
	if r1 == 0 {
		return err
	}
	return nil
}

func (c *windowsCollector) GetHostname() (string, error) {
	return os.Hostname()
}

func (c *windowsCollector) GetMachineID() (string, error) {
	// Get Windows Machine GUID from registry
	cmd := exec.Command("reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MachineGuid") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[len(parts)-1], nil
			}
		}
	}

	return "", fmt.Errorf("machine ID not found")
}

func (c *windowsCollector) GetSerialNumber() (string, error) {
	cmd := exec.Command("wmic", "bios", "get", "SerialNumber", "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SerialNumber=") {
			serial := strings.TrimPrefix(line, "SerialNumber=")
			if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
				return serial, nil
			}
		}
	}

	return "", fmt.Errorf("serial number not found")
}

func (c *windowsCollector) GetProductUUID() (string, error) {
	cmd := exec.Command("wmic", "csproduct", "get", "UUID", "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "UUID=") {
			uuid := strings.TrimPrefix(line, "UUID=")
			if uuid != "" && uuid != "None" {
				return uuid, nil
			}
		}
	}

	return "", fmt.Errorf("product UUID not found")
}

func (c *windowsCollector) GetBoardSerial() (string, error) {
	cmd := exec.Command("wmic", "baseboard", "get", "SerialNumber", "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SerialNumber=") {
			serial := strings.TrimPrefix(line, "SerialNumber=")
			if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
				return serial, nil
			}
		}
	}

	return "", fmt.Errorf("board serial not found")
}

func (c *windowsCollector) GetDiskSerials() ([]string, error) {
	var serials []string

	cmd := exec.Command("wmic", "diskdrive", "get", "SerialNumber", "/value")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SerialNumber=") {
			serial := strings.TrimPrefix(line, "SerialNumber=")
			if serial != "" {
				serials = append(serials, serial)
			}
		}
	}

	return serials, nil
}

func (c *windowsCollector) GetUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

func (c *windowsCollector) GetHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}
