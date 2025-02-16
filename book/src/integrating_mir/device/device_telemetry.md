# Device Telemetry

Device telemetry is the most common way to send data from the device to the server.
This is the hot path and is used to send data that does not require a reply.
This type of data is of timeseries as each datapoint sent is attached to a timestamp of different precision (you choose on your needs).
The Mir telemetry module will ingest and store it in [InfluxDB](https://www.influxdata.com):

> InfluxDB is a time series database designed to handle high write and query loads.
> InfluxDB is meant to be used as a backing store for any use case involving large amounts of timestamped data, including DevOps monitoring, application metrics, IoT sensor data, and real-time analytics.

First, lets define a telemetry message in your schema:
```proto
syntax = "proto3";

package schemav1;
option go_package = "github.com/maxthom/mir-device/schemav1";

import "mir/device/v1/mir.proto";

message Env {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELEMETRY;

	int64 ts = 1 [(mir.device.v1.timestamp) = TIMESTAMP_TYPE_NANO];
	int32 temperature = 2;
	int32 pressure = 3;
	int32 humidity = 4;
}
```

Here we define a message `Env` that will be used. The options are used to annotate the message with metadata:

- `mir.device.v1.message_type`: This tell the server that this message is of telemetry type.
- `mir.device.v1.timestamp`: This tell the server that the field `ts` is the main timestamp and the precision is NANOSECONDS. Second, Microsecond and Millisecond are also available.

Lets regenerate the schema:

```bash
protoc --proto_path=schemav1/ \
       --go_out=schemav1 \
       --go_opt=paths=source_relative \
       schemav1/schema.proto
```

Let's create a function that send telemetry data to the server every 3 seconds.
To do so, we use the `m.SendTelemetry` function that take any proto message:

```go
package main

import (
	"context"
	"math/rand/v2"
	"mir-device/schemav1"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maxthom/mir/pkgs/device/mir"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		Schema(schemav1.File_schema_proto).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

  // Start go routine for not to block main thread
	go func() {
		for {
			select {
			case <-ctx.Done():
			  // If context get cancelled, stop sending telemetry and
				// decrease the wait group for graceful shutdown
				wg.Done()
				return
			case <-time.After(3 * time.Second):
				if err := m.SendTelemetry(&schemav1.Env{
					Ts:          time.Now().UTC().UnixNano(),
					Temperature: rand.Int32N(101),
					Pressure:    rand.Int32N(101),
					Humidity:    rand.Int32N(101),
				}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending telemetry")
				}
			}
		}
	}()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal

	cancel()
	wg.Wait()
}
```

And just like that, we now have telemetry that his stored server side
```bash
mir tlm list weather

1. weather/default
schemav1.Env{} localhost:3000/explore
```

Click on the link to open Grafana and visualize the data.

Voila! You have successfully sent telemetry data to the server. Add more message to the schema and send more data!
Use the CLI to quickly get link to the telemetry data in Grafana and use the generated query to create powerful dashboard.

*! Note: All [protobuf definition](https://protobuf.dev/programming-guides/proto3/) are supported except OneOf*
