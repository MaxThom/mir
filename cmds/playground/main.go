package main

import (
	"context"
	"fmt"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	core_client "github.com/maxthom/mir/services/core"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	ctx := context.Background()
	b, _, cons, err := createPublisherForStream(ctx, "nats://127.0.0.1:4222", jetstream.StreamConfig{
		Name:     "device",
		Subjects: []string{"device.*"},
		NoAck:    true,
	}, jetstream.ConsumerConfig{
		Durable:        "registration_test",
		FilterSubjects: []string{},
		AckPolicy:      jetstream.AckExplicitPolicy,
	})
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			msgs, err := cons.Fetch(10, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				fmt.Println(err)
			}

			for msg := range msgs.Messages() {
				fmt.Println("imbe " + string(msg.Data()) + " bape")

				fmt.Println(msg.Reply())
				err = b.Publish(msg.Reply(), []byte("hello"))
				if err != nil {
					fmt.Println(err)
				}
				msg.Ack()
			}
			if msgs.Error() != nil {
				fmt.Println(err)
			}
		}
	}()

	time.Sleep(2 * time.Second)
	devReq := &core.CreateDeviceRequest{
		DeviceId: "0x994b",
		Labels: map[string]string{
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility": "air_quality",
		},
	}
	_, err = core_client.PublishDeviceCreateRequest(b, devReq)
	if err != nil {
		fmt.Println(err)
	}

}

func createPublisherForStream(ctx context.Context, busUrl string, jsCfg jetstream.StreamConfig, consCfg jetstream.ConsumerConfig) (*bus.BusConn, jetstream.Stream, jetstream.Consumer, error) {
	b, err := bus.New(busUrl)
	if err != nil {
		return nil, nil, nil, err
	}

	js, err := jetstream.New(b.Conn)
	if err != nil {
		return b, nil, nil, err
	}

	stream, err := js.CreateOrUpdateStream(ctx, jsCfg)
	if err != nil {
		return b, stream, nil, err
	}

	// retrieve consumer handle from a stream
	cons, err := stream.CreateOrUpdateConsumer(ctx, consCfg)

	return b, stream, cons, err
}
