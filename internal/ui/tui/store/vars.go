package store

import (
	"sort"
	"strings"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

var Bus *mir.Mir
var Devices []mir_v1.Device
var ScreenHeight, ScreenWidth int

func GetAnnotationsSuggestions(devices []mir_v1.Device) []string {
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
func GetLabelsSuggestions(devices []mir_v1.Device) []string {
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

func GetDeviceIdSuggestions(devices []mir_v1.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string

	for _, d := range devices {
		s = append(s, d.Spec.DeviceId)
	}
	return s
}

func GetNamespaceSuggestions(devices []mir_v1.Device) []string {
	if devices == nil {
		return []string{}
	}

	var s []string
	for _, d := range devices {
		s = append(s, d.Meta.Namespace)
	}
	return s
}
