package clients

import (
	"fmt"
	"strings"
)

// Server subject are liscened server side
type ServerSubject string

// Device subject are liscened device side
type DeviceSubject string

func (s ServerSubject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s DeviceSubject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s ServerSubject) GetSource() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (s ServerSubject) GetId() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (s ServerSubject) GetModule() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[2]
}

func (s ServerSubject) GetVersion() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[3]
}

func (s ServerSubject) GetFunction() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[4]
}

func (s DeviceSubject) GetId() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[0]
}

func (s DeviceSubject) GetVersion() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}

func (s DeviceSubject) GetFunction() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

func (s DeviceSubject) GetVersionAndFunction() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 {
		return ""
	}
	return strings.Join(parts[1:], ".")
}
