package mir

import "github.com/maxthom/mir/internal/libs/systemid"

type IdPrefix struct {
	Prefix   string `json:"prefix,omitempty" yaml:"prefix"`
	Hostname bool   `json:"hostname,omitempty" yaml:"hostname"`
	Username bool   `json:"username,omitempty" yaml:"username"`
}

type IdGenerator struct {
	Salt      string `json:"salt,omitempty" yaml:"salt"`
	CPU       bool   `json:"cpu,omitempty" yaml:"cpu"`
	Disk      bool   `json:"disk,omitempty" yaml:"disk"`
	MAC       bool   `json:"mac,omitempty" yaml:"mac"`
	MachineID bool   `json:"machineId,omitempty" yaml:"machineId"`
	OS        bool   `json:"os,omitempty" yaml:"os"`
	Hostname  bool   `json:"hostname,omitempty" yaml:"hostname"`
	User      bool   `json:"user,omitempty" yaml:"user"`
	Plain     bool   `json:"plain,omitempty" yaml:"plain"`
}

func (i IdPrefix) ToSystemIdPrefixOpts() systemid.PrefixOptions {
	return systemid.PrefixOptions{
		Prefix:   i.Prefix,
		Hostname: i.Hostname,
		Username: i.Username,
	}
}

func (i IdPrefix) IsActive() bool {
	return i.Prefix != "" || i.Hostname || i.Username
}

func (i IdGenerator) ToSystemIdOpts() systemid.Options {
	return systemid.Options{
		IncludeCPU:       i.CPU,
		IncludeDisk:      i.Disk,
		IncludeMAC:       i.MAC,
		IncludeMachineID: i.MachineID,
		IncludeOS:        i.OS,
		IncludeHostname:  i.Hostname,
		IncludeUser:      i.User,
		Plain:            i.Plain,
	}
}

func (i IdGenerator) IsActive() bool {
	if i.CPU {
		return true
	}
	if i.Disk {
		return true
	}
	if i.MAC {
		return true
	}
	if i.MachineID {
		return true
	}
	if i.OS {
		return true
	}
	if i.Hostname {
		return true
	}
	if i.User {
		return true
	}
	if i.Salt != "" {
		return true
	}
	return false
}
