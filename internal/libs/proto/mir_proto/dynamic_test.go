package mir_proto

import (
	"slices"
	"testing"

	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"gotest.tools/assert"
)

func TestSchemaTlm(t *testing.T) {
	// Arrange
	file := NewFileDescriptor("swarmv2", "telemetry")
	env := NewTelemetryDescriptor("env", nil, devicev1.TimestampType_TIMESTAMP_TYPE_NANO)
	pwr := NewTelemetryDescriptor("pwr", nil, devicev1.TimestampType_TIMESTAMP_TYPE_NANO)
	file.MessageType = append(file.MessageType,
		env,
		pwr,
	)
	sch, err := NewMirProtoSchemaWithMir(file)
	if err != nil {
		t.Error(err)
	}

	// Act
	tlmList, err := sch.GetTelemetryList(nil, nil)
	if err != nil {
		t.Error(err)
	}
	pkgList := sch.GetPackageList()

	// Assert
	assert.Equal(t, 3, len(pkgList))
	assert.Equal(t, true, slices.Contains(pkgList, "swarmv2"))
	assert.Equal(t, true, slices.Contains(pkgList, "mir.device.v1"))
	assert.Equal(t, true, slices.Contains(pkgList, "google.protobuf"))

	assert.Equal(t, 2, len(tlmList))
	assert.Equal(t, "swarmv2.env", tlmList[0].GetName())
	assert.Equal(t, "swarmv2.pwr", tlmList[1].GetName())
}

func TestSchemaCmd(t *testing.T) {
	// Arrange
	file := NewFileDescriptor("swarmv2", "telemetry")
	env := NewCommandDescriptor("env", nil)
	pwr := NewCommandDescriptor("pwr", nil)
	file.MessageType = append(file.MessageType,
		env,
		pwr,
	)
	sch, err := NewMirProtoSchemaWithMir(file)
	if err != nil {
		t.Error(err)
	}

	// Act
	tlmList, err := sch.GetCommandsList(nil)
	if err != nil {
		t.Error(err)
	}
	pkgList := sch.GetPackageList()

	// Assert
	assert.Equal(t, 3, len(pkgList))
	assert.Equal(t, true, slices.Contains(pkgList, "swarmv2"))
	assert.Equal(t, true, slices.Contains(pkgList, "mir.device.v1"))
	assert.Equal(t, true, slices.Contains(pkgList, "google.protobuf"))

	assert.Equal(t, 2, len(tlmList))
	assert.Equal(t, "swarmv2.env", tlmList[0].GetName())
	assert.Equal(t, "swarmv2.pwr", tlmList[1].GetName())
}

func TestSchemaCfg(t *testing.T) {
	// Arrange
	file := NewFileDescriptor("swarmv2", "telemetry")
	env := NewConfigDescriptor("env", nil)
	pwr := NewConfigDescriptor("pwr", nil)
	file.MessageType = append(file.MessageType,
		env,
		pwr,
	)
	sch, err := NewMirProtoSchemaWithMir(file)
	if err != nil {
		t.Error(err)
	}

	// Act
	tlmList, err := sch.GetConfigList(nil)
	if err != nil {
		t.Error(err)
	}
	pkgList := sch.GetPackageList()

	// Assert
	assert.Equal(t, 3, len(pkgList))
	assert.Equal(t, true, slices.Contains(pkgList, "swarmv2"))
	assert.Equal(t, true, slices.Contains(pkgList, "mir.device.v1"))
	assert.Equal(t, true, slices.Contains(pkgList, "google.protobuf"))

	assert.Equal(t, 2, len(tlmList))
	assert.Equal(t, "swarmv2.env", tlmList[0].GetName())
	assert.Equal(t, "swarmv2.pwr", tlmList[1].GetName())
}

func TestDynamic(t *testing.T) {
	// Arrange
	file := NewFileDescriptor("swarmv2", "telemetry")
	env := NewTelemetryDescriptor("env", nil, devicev1.TimestampType_TIMESTAMP_TYPE_NANO)
	pwr := NewTelemetryDescriptor("pwr", nil, devicev1.TimestampType_TIMESTAMP_TYPE_NANO)
	file.MessageType = append(file.MessageType,
		env,
		pwr,
	)
	sch, err := NewMirProtoSchemaWithMir(file)
	if err != nil {
		t.Error(err)
	}

	// Act
	envDesc, err := sch.FindDescriptorByName("swarmv2.env")
	if err != nil {
		t.Error(err)
	}
	envDescMsg := envDesc.(protoreflect.MessageDescriptor)

	envMsg := dynamicpb.NewMessage(envDescMsg)
	envMsgR := envMsg.ProtoReflect()
	envMsgR.Set(envDescMsg.Fields().ByName("ts"), protoreflect.ValueOfInt64(23123))

	bytes, _ := protojson.Marshal(envMsg)
	assert.Equal(t, "{\"ts\":\"23123\"}", string(bytes))
}
