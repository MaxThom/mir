package main

import (
	"flag"
	"log"
	"math"
	"time"

	"github.com/maxthom/mir/cmds/swarm/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// protoc --go_out=. --descriptor_set_out=./gen/device.pb --include_imports ./device.proto

type SinWave struct {
	Amplitude float64 // Peak value of the sine wave
	Frequency float64 // How many cycles per second
	Phase     float64 // How many samples per second
}

func main() {
	// Some flags
	seconds := flag.Int("sec", -1, "number of seconds")
	rate := flag.Int("rate", 1, "how many telemetry per second")
	flag.Parse()

	// Nats
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	subject := "v1alpha.XFEA12.FactoA.tlm.proto"

	// Define some random sensors with sin waves
	waves := map[string]SinWave{
		"temperature": {
			Amplitude: 1.0,
			Frequency: 0.5,
			Phase:     0.0,
		},
		"pressure": {
			Amplitude: 2.0,
			Frequency: 0.2,
			Phase:     1.0,
		},
		"humidity": {
			Amplitude: -1.0,
			Frequency: 0.25,
			Phase:     -0.5,
		},
	}

	// Create a ticker on sameple rate
	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()
	startTime := time.Now()

	// Generate telemetry and send to nats
	for range ticker.C {
		// Calculate elapsed time in seconds
		elapsed := time.Since(startTime).Seconds()

		// Generate sine value
		tlm := generateSinWavesPayload(elapsed, waves)
		// fmt.Println(tlm)

		serializedData, err := proto.Marshal(tlm)
		if err != nil {
			log.Fatal("Failed to serialize data:", err)
		}

		if err := nc.PublishMsg(&nats.Msg{
			Subject: subject,
			Data:    serializedData,
			Header: nats.Header(map[string][]string{
				"__pb":     {"swarm.Telemetry"},
				"deviceId": {"XFEA12"},
			}),
		}); err != nil {
			log.Fatal("Failed to publish data:", err)
		}

		// Stop after 10 seconds
		if *seconds != -1 && elapsed >= float64(*seconds) {
			break
		}
	}
}

func generateSinWavesPayload(elapsed float64, waves map[string]SinWave) *gen.Telemetry {
	now := time.Now().UTC()
	i := gen.Telemetry{
		Time: &timestamppb.Timestamp{
			Seconds: now.Unix(),
			Nanos:   int32(now.UnixNano() % 1e9),
		},
		Sensors: map[string]float64{},
	}

	for k, v := range waves {
		// Generate sine value
		x := 2.0*math.Pi*v.Frequency*elapsed + v.Phase
		y := v.Amplitude * math.Sin(x)

		i.Sensors[k] = y
	}

	return &i
}
