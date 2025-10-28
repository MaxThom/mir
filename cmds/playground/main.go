package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

func main() {

	m, err := mir.Connect(*zerolog.DefaultContextLogger, "playground", "nats://localhost:4222")
	if err != nil {
		panic(err)
	}

	for i := range 120 {
		time.Sleep(time.Millisecond * 100)
		fmt.Println(i)
		if err := m.Event().Publish(m.Event().NewSubject(strconv.Itoa(i), "playground", "v1", "test"),
			mir_v1.EventSpec{
				Type:    mir_v1.EventTypeNormal,
				Reason:  "test",
				Message: "testing buffer",
				// Payload:       jsonyaml.RawMessage{},
				// RelatedObject: mir_v1.Object{},
			}, nil); err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("done")
}
