package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/maxthom/mir/internal/libs/jsonyaml"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Event struct {
	Name       string
	FamilyName string
	Age        int
	Json       jsonyaml.RawMessage `yaml:"-"`
}

func main() {
	nc, _ := nats.Connect(nats.DefaultURL)
	defer nc.Drain()

	js, _ := jetstream.New(nc)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List bucket names
	lister := js.KeyValueStoreNames(ctx)
	for name := range lister.Name() {
		fmt.Printf("KV Bucket: %s\n", name)
	}
	fmt.Println("done")

	bucket, err := js.KeyValue(ctx, "txasest")
	if err != nil {
		if err == jetstream.ErrBucketNotFound {
			fmt.Println("Bucket not found")
		} else {
			fmt.Println(err)
		}
	}
	keys, err := bucket.Keys(ctx, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(keys)

	key, err := bucket.Get(ctx, "pipi")
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(5 * time.Second)
	rev, err := bucket.Update(ctx, "pipi", []byte("bouet"), key.Revision())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(rev)

	pattern := "*"
	matches := []string{}
	for _, key := range keys {
		if matched, _ := filepath.Match(pattern, key); matched {
			val, _ := bucket.Get(ctx, key)
			matches = append(matches, fmt.Sprintf("%s-%s", key, val.Value()))
		}
	}
	fmt.Println(matches)

	// e := Event{
	// 	Name:       "Max",
	// 	FamilyName: "Thom",
	// 	Age:        24,
	// }

	// // Create a random JSON object
	// randomJsonObj := map[string]any{
	// 	"id":        123456,
	// 	"active":    true,
	// 	"score":     98.76,
	// 	"tags":      []string{"golang", "json", "random"},
	// 	"timestamp": "2023-08-15T14:22:31Z",
	// 	"address": map[string]any{
	// 		"street":  "123 Main St",
	// 		"city":    "Techville",
	// 		"zipcode": 10101,
	// 	},
	// 	"metadata": map[string]any{
	// 		"version": 2.1,
	// 		"debug":   false,
	// 		"counts":  []int{1, 2, 3, 4, 5},
	// 	},
	// 	"nullable": nil,
	// }

	// // Convert map to JSON and assign to Event.Json
	// jsonData, _ := json.Marshal(randomJsonObj)
	// e.Json = jsonyaml.RawMessage(jsonData)

	// b, err := json.MarshalIndent(e, "", "  ")
	// fmt.Println(err)
	// fmt.Println(string(b))

	// y, err := yaml.Marshal(e)
	// fmt.Println(err)
	// fmt.Println(string(y))

}
