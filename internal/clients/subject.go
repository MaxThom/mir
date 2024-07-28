package routes

import (
	"fmt"
	"strings"
)

type Subject string
type DeviceSubject string

func (s Subject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s DeviceSubject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s Subject) GetSource() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (s Subject) GetId() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (s Subject) GetModule() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[2]
}

func (s Subject) GetVersion() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[3]
}

func (s Subject) GetFunction() string {
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
