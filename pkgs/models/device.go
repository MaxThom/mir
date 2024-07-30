package models

import (
	"time"
)

type DeviceWithId struct {
	Id string `json:"id"`
	Device
}

type Device struct {
	ApiVersion string     `json:"apiVersion"`
	ApiName    string     `json:"apiName"`
	Meta       Meta       `json:"meta"`
	Spec       Spec       `json:"spec"`
	Properties Properties `json:"properties"`
	Status     Status     `json:"status"`
}

type Meta struct {
	Name        string             `json:"name"`
	Namespace   string             `json:"namespace"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type Spec struct {
	DeviceId string `json:"deviceId"`
	Disabled bool   `json:"disabled"`
}

type Properties struct {
}

type Status struct {
	Online         bool      `json:"online"`
	LastHearthbeat time.Time `json:"lastHearthbeat"`
}
