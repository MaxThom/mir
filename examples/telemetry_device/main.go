package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"syscall"
	"time"

	"github.com/maxthom/mir/examples/telemetry_device/gen"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Builder().
		DeviceId("0x238n9").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		TelemetrySchema(
			gen.File_telemetry_proto,
		).
		TelemetrySchemaProto(
			protodesc.ToFileDescriptorProto(gen.File_command_proto),
			protodesc.ToFileDescriptorProto(gen.File_utils_proto),
		).
		Build()
	if err != nil {
		panic(err)
	}
	l := m.Logger()

	l.Info().Msg("Mir is ready for launch... Launching!")
	mirWg, err := m.Launch(ctx)
	if err != nil {
		l.Error().Err(err).Msg("Abort launch error")
		os.Exit(1)
	}
	l.Info().Msg("Mir is at maxq and nominal")

	go func() {
		for {
			time.Sleep(3 * time.Second)
			data := gen.Telemetry{
				Temperature: rand.Int32N(101),
				Pressure:    rand.Int32N(101),
				Humidity:    rand.Int32N(101),
			}
			m.SendTelemetry(&data)

			l.Debug().Str("module", "telemetry").Any("data", data).Msg("send tlm")
		}
	}()

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}

func PlayWithProtoSchema() {
	// Test data
	data := gen.Telemetry{
		Temperature: 25,
		Pressure:    58,
		Humidity:    80,
	}
	dataBytes, err := proto.Marshal(&data)
	if err != nil {
		log.Fatalf("Can't marshal data to byte: %v", err)
	}

	fmt.Println(gen.File_telemetry_proto.FullName())
	fmt.Println(gen.File_telemetry_proto.Package())
	fmt.Println(gen.File_telemetry_proto.Name())

	// Add all proto definitions to a file set
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(gen.File_telemetry_proto),
			protodesc.ToFileDescriptorProto(gen.File_command_proto),
			protodesc.ToFileDescriptorProto(gen.File_utils_proto),
		},
	}

	// Marshal the FileDescriptorSet to bytes
	bytes, err := proto.Marshal(fileDescriptorSet)
	if err != nil {
		log.Fatalf("Failed to marshal descriptor: %v", err)
	}

	// Print the bytes
	// We will send this via network
	fmt.Printf("Schema in bytes: %v\n", bytes)

	// Unmarshal the bytes back to a FileDescriptorSet
	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(bytes, pbSet); err != nil {
		log.Fatalf("Failed to unmarshal descriptor: %v", err)
	}

	// Create registry from the FileDescriptorSet
	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		log.Fatalf("Failed to create registry: %v", err)
	}

	// Print all the files and messages in the registry
	reg.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fmt.Println(fd.FullName())
		for i := 0; i < fd.Messages().Len(); i++ {
			fmt.Println(fd.Messages().Get(i).FullName())
		}
		return true
	})

	// Find the descriptor by name
	desc, err := reg.FindDescriptorByName("v1alpha.telemetry_example.Telemetry")
	if err != nil {
		log.Fatalf("Failed to find descriptor: %v", err)
	}

	// Unmarshal data with the descriptor
	msgType := desc.(protoreflect.MessageDescriptor)
	dynMsg := dynamicpb.NewMessage(msgType)
	if err := proto.Unmarshal(dataBytes, dynMsg); err != nil {
		log.Fatalf("Failed to deserialize message: %v", err)
	}
	fmt.Println(dynMsg)

	// Unmarshal data with the specific struct
	dataBack := gen.Telemetry{}
	if err := proto.Unmarshal(dataBytes, &dataBack); err != nil {
		log.Fatalf("Failed to deserialize message: %v", err)
	}
	fmt.Println(dynMsg)

}
