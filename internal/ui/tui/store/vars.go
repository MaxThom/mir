package store

import (
	"sort"
	"strings"

	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
)

var Bus *bus.BusConn
var Devices []*core_api.Device
var ScreenHeight, ScreenWidth int

func GetAnnotationsSuggestions(devices []*core_api.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string

	for _, d := range devices {
		lbls := []string{}
		keys := []string{}
		for k := range d.Meta.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lbls = append(lbls, k+"="+d.Meta.Annotations[k])
		}
		s = append(s, strings.Join(lbls, ";"))
	}
	return s
}

// TODO suggestion could be in any order, not just sorted
func GetLabelsSuggestions(devices []*core_api.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string

	for _, d := range devices {
		lbls := []string{}
		keys := []string{}
		for k := range d.Meta.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lbls = append(lbls, k+"="+d.Meta.Labels[k])
		}
		s = append(s, strings.Join(lbls, ";"))
	}
	return s
}

func GetDeviceIdSuggestions(devices []*core_api.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string

	for _, d := range devices {
		s = append(s, d.Spec.DeviceId)
	}
	return s
}

func GetNamespaceSuggestions(devices []*core_api.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string
	for _, d := range devices {
		s = append(s, d.Meta.Namespace)
	}
	return s
}
