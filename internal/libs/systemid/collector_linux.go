//go:build linux

package systemid

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type linuxCollector struct{}

func newPlatformCollector() Collector {
	return &linuxCollector{}
}

func (c *linuxCollector) GetMACAddresses() ([]string, error) {
	var macs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	
	for _, iface := range interfaces {
		// Skip loopback and virtual interfaces
		if iface.Flags&net.FlagLoopback != 0 || strings.HasPrefix(iface.Name, "veth") || 
		   strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "br-") {
			continue
		}
		
		if iface.HardwareAddr.String() != "" && iface.HardwareAddr.String() != "00:00:00:00:00:00" {
			macs = append(macs, iface.HardwareAddr.String())
		}
	}
	
	return macs, nil
}

func (c *linuxCollector) GetCPUInfo() (string, error) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", err
	}
	
	var modelName, cpuFamily, model, stepping, hardware, cpuPart, cpuRevision string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				modelName = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "cpu family") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cpuFamily = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "model") && !strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				model = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "stepping") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				stepping = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Hardware") {
			// ARM-specific field
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				hardware = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "CPU part") {
			// ARM-specific field
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cpuPart = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "CPU revision") {
			// ARM-specific field
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cpuRevision = strings.TrimSpace(parts[1])
			}
		}
	}
	
	// For x86/x64 processors
	if modelName != "" {
		return fmt.Sprintf("%s|%s|%s|%s", modelName, cpuFamily, model, stepping), nil
	}
	
	// For ARM processors
	if hardware != "" || cpuPart != "" {
		return fmt.Sprintf("ARM|%s|%s|%s", hardware, cpuPart, cpuRevision), nil
	}
	
	// Fallback: try to get any processor info
	cmd := exec.Command("lscpu")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Model name:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("CPU info not found")
}

func (c *linuxCollector) GetOSInfo() (osType, osVersion, osArch string, err error) {
	osType = "linux"
	
	// Try to get OS version from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "VERSION=") {
				osVersion = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
				break
			}
		}
	}
	
	// Get architecture
	cmd := exec.Command("uname", "-m")
	if output, err := cmd.Output(); err == nil {
		osArch = strings.TrimSpace(string(output))
		// Normalize architecture names
		switch osArch {
		case "armv7l", "armv7", "armv6l", "armv6":
			osArch = "arm"
		case "aarch64", "arm64":
			osArch = "arm64"
		case "x86_64", "amd64":
			osArch = "amd64"
		case "i386", "i686":
			osArch = "386"
		case "mips64le":
			osArch = "mips64le"
		case "mips64":
			osArch = "mips64"
		case "ppc64le":
			osArch = "ppc64le"
		case "ppc64":
			osArch = "ppc64"
		case "riscv64":
			osArch = "riscv64"
		case "s390x":
			osArch = "s390x"
		}
	}
	
	return osType, osVersion, osArch, nil
}

func (c *linuxCollector) GetHostname() (string, error) {
	return os.Hostname()
}

func (c *linuxCollector) GetMachineID() (string, error) {
	// Try systemd machine-id first
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	
	// Try dbus machine-id
	if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	
	return "", fmt.Errorf("machine ID not found")
}

func (c *linuxCollector) GetSerialNumber() (string, error) {
	// Try DMI serial number
	if data, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
		serial := strings.TrimSpace(string(data))
		if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}
	
	// Try dmidecode command
	cmd := exec.Command("dmidecode", "-s", "system-serial-number")
	if output, err := cmd.Output(); err == nil {
		serial := strings.TrimSpace(string(output))
		if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}
	
	return "", fmt.Errorf("serial number not found")
}

func (c *linuxCollector) GetProductUUID() (string, error) {
	// Try DMI product UUID
	if data, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
		uuid := strings.TrimSpace(string(data))
		if uuid != "" && uuid != "None" {
			return uuid, nil
		}
	}
	
	// Try dmidecode command
	cmd := exec.Command("dmidecode", "-s", "system-uuid")
	if output, err := cmd.Output(); err == nil {
		uuid := strings.TrimSpace(string(output))
		if uuid != "" && uuid != "None" {
			return uuid, nil
		}
	}
	
	return "", fmt.Errorf("product UUID not found")
}

func (c *linuxCollector) GetBoardSerial() (string, error) {
	// Try DMI board serial
	if data, err := os.ReadFile("/sys/class/dmi/id/board_serial"); err == nil {
		serial := strings.TrimSpace(string(data))
		if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}
	
	// Try dmidecode command
	cmd := exec.Command("dmidecode", "-s", "baseboard-serial-number")
	if output, err := cmd.Output(); err == nil {
		serial := strings.TrimSpace(string(output))
		if serial != "" && serial != "None" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}
	
	return "", fmt.Errorf("board serial not found")
}

func (c *linuxCollector) GetDiskSerials() ([]string, error) {
	var serials []string
	
	// List all block devices
	blockDevs, err := filepath.Glob("/sys/block/sd*")
	if err != nil {
		return nil, err
	}
	
	blockDevs2, err := filepath.Glob("/sys/block/nvme*")
	if err == nil {
		blockDevs = append(blockDevs, blockDevs2...)
	}
	
	for _, dev := range blockDevs {
		// Skip partitions
		if strings.Contains(dev, "p") && len(dev) > 1 {
			continue
		}
		
		// Try to read serial from device
		serialPath := filepath.Join(dev, "device", "serial")
		if data, err := os.ReadFile(serialPath); err == nil {
			serial := strings.TrimSpace(string(data))
			if serial != "" {
				serials = append(serials, serial)
			}
		}
	}
	
	// If no serials found via sysfs, try lsblk
	if len(serials) == 0 {
		cmd := exec.Command("lsblk", "-ndo", "SERIAL")
		if output, err := cmd.Output(); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(output))
			for scanner.Scan() {
				serial := strings.TrimSpace(scanner.Text())
				if serial != "" {
					serials = append(serials, serial)
				}
			}
		}
	}
	
	return serials, nil
}

func (c *linuxCollector) GetUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

func (c *linuxCollector) GetHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}