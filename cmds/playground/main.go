package main

import (
	"encoding/json"
	"fmt"

	"github.com/maxthom/mir/internal/libs/jsonyaml"
	"gopkg.in/yaml.v3"
)

type Event struct {
	Name       string
	FamilyName string
	Age        int
	Json       jsonyaml.RawMessage `yaml:"-"`
}

func main() {

	e := Event{
		Name:       "Max",
		FamilyName: "Thom",
		Age:        24,
	}

	// Create a random JSON object
	randomJsonObj := map[string]any{
		"id":        123456,
		"active":    true,
		"score":     98.76,
		"tags":      []string{"golang", "json", "random"},
		"timestamp": "2023-08-15T14:22:31Z",
		"address": map[string]any{
			"street":  "123 Main St",
			"city":    "Techville",
			"zipcode": 10101,
		},
		"metadata": map[string]any{
			"version": 2.1,
			"debug":   false,
			"counts":  []int{1, 2, 3, 4, 5},
		},
		"nullable": nil,
	}

	// Convert map to JSON and assign to Event.Json
	jsonData, _ := json.Marshal(randomJsonObj)
	e.Json = jsonyaml.RawMessage(jsonData)

	b, err := json.MarshalIndent(e, "", "  ")
	fmt.Println(err)
	fmt.Println(string(b))

	y, err := yaml.Marshal(e)
	fmt.Println(err)
	fmt.Println(string(y))

}
