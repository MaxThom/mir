package store

import (
	"sort"
	"strings"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
)

var Bus *bus.BusConn
var Devices []*core.Device
var ScreenHeight, ScreenWidth int

func GetAnnotationsSuggestions(devices []*core.Device) []string {
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
func GetLabelsSuggestions(devices []*core.Device) []string {
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

func GetDeviceIdSuggestions(devices []*core.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string

	for _, d := range devices {
		s = append(s, d.Spec.DeviceId)
	}
	return s
}

func GetNamespaceSuggestions(devices []*core.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string
	for _, d := range devices {
		s = append(s, d.Meta.Namespace)
	}
	return s
}
