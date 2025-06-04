//go:build darwin

package systemid

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type darwinCollector struct{}

func newPlatformCollector() Collector {
	return &darwinCollector{}
}

func (c *darwinCollector) GetMACAddresses() ([]string, error) {
	var macs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	
	for _, iface := range interfaces {
		// Skip loopback and virtual interfaces
		if iface.Flags&net.FlagLoopback != 0 || 
		   strings.HasPrefix(iface.Name, "utun") ||
		   strings.HasPrefix(iface.Name, "awdl") ||
		   strings.HasPrefix(iface.Name, "llw") ||
		   strings.HasPrefix(iface.Name, "bridge") {
			continue
		}
		
		if iface.HardwareAddr.String() != "" && iface.HardwareAddr.String() != "00:00:00:00:00:00" {
			macs = append(macs, iface.HardwareAddr.String())
		}
	}
	
	return macs, nil
}

func (c *darwinCollector) GetCPUInfo() (string, error) {
	// Check if this is Apple Silicon first
	cmd := exec.Command("sysctl", "-n", "hw.optional.arm64")
	if output, err := cmd.Output(); err == nil && strings.TrimSpace(string(output)) == "1" {
		// This is Apple Silicon, get chip info differently
		var chipBrand string
		
		// Try to get the chip brand
		if cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				chipBrand = strings.TrimSpace(string(output))
			}
		}
		
		// If brand_string is not available (common on Apple Silicon), construct it
		if chipBrand == "" {
			// Get the chip model
			cmd = exec.Command("sysctl", "-n", "hw.model")
			if output, err := cmd.Output(); err == nil {
				model := strings.TrimSpace(string(output))
				chipBrand = fmt.Sprintf("Apple %s", model)
			}
		}
		
		// Get additional ARM-specific info
		var coreCount, perfCoreCount, effCoreCount string
		
		if cmd := exec.Command("sysctl", "-n", "hw.ncpu"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				coreCount = strings.TrimSpace(string(output))
			}
		}
		
		if cmd := exec.Command("sysctl", "-n", "hw.perflevel0.logicalcpu"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				perfCoreCount = strings.TrimSpace(string(output))
			}
		}
		
		if cmd := exec.Command("sysctl", "-n", "hw.perflevel1.logicalcpu"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				effCoreCount = strings.TrimSpace(string(output))
			}
		}
		
		return fmt.Sprintf("%s|cores:%s|perf:%s|eff:%s", chipBrand, coreCount, perfCoreCount, effCoreCount), nil
	}
	
	// Intel Mac - use the original method
	// Get CPU brand string
	cmd = exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
	brandOutput, err := cmd.Output()
	if err != nil {
		return "", err
	}
	brand := strings.TrimSpace(string(brandOutput))
	
	// Get CPU family
	cmd = exec.Command("sysctl", "-n", "machdep.cpu.family")
	familyOutput, err := cmd.Output()
	if err != nil {
		return brand, nil
	}
	family := strings.TrimSpace(string(familyOutput))
	
	// Get CPU model
	cmd = exec.Command("sysctl", "-n", "machdep.cpu.model")
	modelOutput, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("%s|%s", brand, family), nil
	}
	model := strings.TrimSpace(string(modelOutput))
	
	return fmt.Sprintf("%s|%s|%s", brand, family, model), nil
}

func (c *darwinCollector) GetOSInfo() (osType, osVersion, osArch string, err error) {
	osType = "darwin"
	
	// Get OS version
	cmd := exec.Command("sw_vers", "-productVersion")
	if output, err := cmd.Output(); err == nil {
		osVersion = strings.TrimSpace(string(output))
	}
	
	// Get architecture
	cmd = exec.Command("uname", "-m")
	if output, err := cmd.Output(); err == nil {
		osArch = strings.TrimSpace(string(output))
		// Normalize architecture names to match Go's runtime.GOARCH
		switch osArch {
		case "x86_64":
			osArch = "amd64"
		case "arm64", "aarch64":
			osArch = "arm64"
		case "i386", "i686":
			osArch = "386"
		// Apple Silicon Macs running x86_64 code via Rosetta 2
		// will still report arm64 from uname -m
		}
		
		// Double-check for Rosetta 2 translation
		// sysctl will show the actual hardware architecture
		if sysCtlCmd := exec.Command("sysctl", "-n", "hw.optional.arm64"); sysCtlCmd != nil {
			if sysCtlOutput, err := sysCtlCmd.Output(); err == nil {
				if strings.TrimSpace(string(sysCtlOutput)) == "1" {
					// This is Apple Silicon hardware
					osArch = "arm64"
				}
			}
		}
	}
	
	return osType, osVersion, osArch, nil
}

func (c *darwinCollector) GetHostname() (string, error) {
	return os.Hostname()
}

func (c *darwinCollector) GetMachineID() (string, error) {
	// Try to get hardware UUID
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[3], nil
			}
		}
	}
	
	return "", fmt.Errorf("machine ID not found")
}

func (c *darwinCollector) GetSerialNumber() (string, error) {
	cmd := exec.Command("ioreg", "-c", "IOPlatformExpertDevice", "-d", "2")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "IOPlatformSerialNumber") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[3], nil
			}
		}
	}
	
	return "", fmt.Errorf("serial number not found")
}

func (c *darwinCollector) GetProductUUID() (string, error) {
	// On macOS, this is the same as machine ID
	return c.GetMachineID()
}

func (c *darwinCollector) GetBoardSerial() (string, error) {
	// Try to get logic board serial
	cmd := exec.Command("ioreg", "-c", "IOPlatformExpertDevice", "-d", "2")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "board-id") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[3], nil
			}
		}
	}
	
	// Fall back to main serial number
	return c.GetSerialNumber()
}

func (c *darwinCollector) GetDiskSerials() ([]string, error) {
	var serials []string
	
	// Get disk list
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Parse disk identifiers from plist output
	disks := []string{}
	lines := strings.Split(string(output), "\n")
	inDisks := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "<key>AllDisks</key>") {
			inDisks = true
			continue
		}
		if inDisks && strings.Contains(line, "</array>") {
			break
		}
		if inDisks && strings.Contains(line, "<string>") {
			disk := strings.TrimPrefix(line, "<string>")
			disk = strings.TrimSuffix(disk, "</string>")
			if strings.HasPrefix(disk, "disk") && !strings.Contains(disk, "s") {
				disks = append(disks, disk)
			}
		}
	}
	
	// Get serial for each disk
	for _, disk := range disks {
		cmd := exec.Command("diskutil", "info", disk)
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		
		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Media Name:") || strings.Contains(line, "Device / Media Name:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					serial := strings.TrimSpace(parts[1])
					if serial != "" && !strings.Contains(serial, "Media") {
						serials = append(serials, serial)
					}
				}
			}
		}
	}
	
	return serials, nil
}

func (c *darwinCollector) GetUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

func (c *darwinCollector) GetHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}