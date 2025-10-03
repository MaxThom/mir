package clients

import (
	"fmt"
	"strings"
)

// Server subject are liscened server side
type ClientSubject string

// Device subject are liscened device side
type DeviceSubject string

func (s ClientSubject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s DeviceSubject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s ClientSubject) GetSource() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (s ClientSubject) GetId() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (s ClientSubject) GetModule() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[2]
}

func (s ClientSubject) GetVersion() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[3]
}

func (s ClientSubject) GetFunction() string {
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
