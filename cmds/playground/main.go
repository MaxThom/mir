package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

type Device struct {
	Name      string     `json:"name,omitempty"`
	Namespace string     `json:"namespace"`
	Props     Properties `json:"properties"`
	Spec      Spec       `json:"spec"`
}

type Spec struct {
	Count  *int               `json:"count,omitempty"`
	Count2 *int               `json:"count2,omitempty"`
	Count3 *int               `json:"count3,omitempty"`
	Algo   string             `json:"algo"`
	Tar    []int              `json:"tar"`
	M      map[string]*string `json:"m"`
}

type Properties struct {
	Desired map[string]interface{} `json:"desired" yaml:"desired"`
}

func main() {
	main4()
	db, err := surreal.ConnectToDb("ws://127.0.0.1:8000/rpc", "global", "mir", "root", "root")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	store := mng.NewSurrealDeviceStore(db)

	dev := mir_models.NewDevice()
	dev.Meta = mir_models.Meta{
		Name:      "device1",
		Namespace: "namespace1",
		Labels: map[string]string{
			"test": "",
		},
	}

	bytes, err := json.Marshal(dev)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	structpb := structpb.Struct{}
	err = protojson.Unmarshal(bytes, &structpb)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	devs, err := store.UpdateDevice2(&core_apiv1.MergeDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{"upd2"},
		},
		Device: &structpb,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(devs)

}

func main4() {
	content :=
		`{"name": "device1",
"namespace": "namespace1",
"properties": {
 "desired": {
   "key1": null,
   "key2": null,
   "nested": null
 }
},
"spec": {
  "viagras": 1,
  "count": 0,
  "count3": null,
  "tar": [1,2,3],
  "m": {
  "r": null,
  "a": "test"
  }
}
}
`
	dev := Device{}
	d := json.NewDecoder(strings.NewReader(content))
	d.DisallowUnknownFields()
	err := d.Decode(&dev)
	fmt.Println(err)
	fmt.Println(dev)

	// How to validate if a json patch fit a model
	result := Device{} // make(map[string]interface{})
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		fmt.Println(err)
	}
	fmt.Println(result)

	result.Name = ""

	b, err := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
	fmt.Println(err)

}

func main3() {
	content :=
		`name: device1
namespace: namespace1
properties:
  desired:
    key1: nil
    key2: null
    nested:
`
	// data := &Device{}
	// if err := yaml.Unmarshal(content, data); err != nil {
	// 	fmt.Println(err)
	// }
	//fmt.Println(string(content))
	result := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(content), &result); err != nil {
		fmt.Println(err)
	}
	fmt.Println(result)

	content2 :=
		`name: device1
namespace: 5
`
	// data := &Device{}
	// if err := yaml.Unmarshal(content, data); err != nil {
	// 	fmt.Println(err)
	// }
	//fmt.Println(string(content))
	result2 := make(map[string]string)
	if err := yaml.Unmarshal([]byte(content2), &result2); err != nil {
		fmt.Println(err)
	}
	fmt.Println(result2)

	_ = map[string]interface{}{
		"name":      "device1",
		"namespace": "namespace1",
		"properties": map[string]interface{}{
			"desired": map[string]interface{}{
				"key1": "value1",
				"key2": nil,
				"nested": map[string]interface{}{
					"nestedKey1": "nestedValue1",
					"nestedKey2": true,
				},
			},
		},
	}
	st, e := structpb.NewStruct(result)
	fmt.Println(e)
	fmt.Println(st.AsMap())
	j, e := st.MarshalJSON()
	fmt.Println(e)
	fmt.Println(string(j))

}

func json2() {
	content :=
		`{"name": "device1",
"namespace": null,
"properties": {
  "desired": {
    "key1": null,
    "key2": null,
    "nested": null
  }
}
}
`
	// data := &Device{}
	// if err := yaml.Unmarshal(content, data); err != nil {
	// 	fmt.Println(err)
	// }
	//fmt.Println(string(content))
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		fmt.Println(err)
	}
	fmt.Println(result)

	_ = map[string]interface{}{
		"name":      "device1",
		"namespace": "namespace1",
		"properties": map[string]interface{}{
			"desired": map[string]interface{}{
				"key1": "value1",
				"key2": nil,
				"nested": map[string]interface{}{
					"nestedKey1": "nestedValue1",
					"nestedKey2": true,
				},
			},
		},
	}
	st, e := structpb.NewStruct(result)
	fmt.Println(e)
	fmt.Println(st.AsMap())
	j, e := st.MarshalJSON()
	fmt.Println(e)
	fmt.Println(string(j))

}
