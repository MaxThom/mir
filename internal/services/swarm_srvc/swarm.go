package swarm_srvc

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/ui"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	ErrFieldNotFound        = fmt.Errorf("could not find field")
	ErrUnsupportedValueType = fmt.Errorf("unsupported value type")
	ErrCreatingSchema       = fmt.Errorf("error creating schema")
	ErrIncubatingSwarm      = fmt.Errorf("error incubating device swarm")
	ErrIncubatingDevice     = fmt.Errorf("error incubating a device")
	ErrDeployingSwarm       = fmt.Errorf("error deploying swarm")
	ErrFindingMessage       = fmt.Errorf("error finding message in schema")
	ErrCreatingGenerator    = fmt.Errorf("error creating telemetry field generator")

	//go:embed swarm.example.yaml
	SwarmExampleFile []byte

	l zerolog.Logger
)

type SwarmService struct {
	cfg         mir_v1.Swarm
	swarm       swarm.Swarm
	schemaFiles map[string][]*descriptorpb.FileDescriptorProto
	tlmFunc     map[string][]func(context.Context, *sync.WaitGroup, *mir.Mir)
	cmdFunc     map[string][]func(*mir.Mir)
	cfgFunc     map[string][]func(*mir.Mir)
}

func NewSwarmService(logger zerolog.Logger, mirCtx ui.Context, swarmCfg mir_v1.Swarm, bus *nats.Conn) (*SwarmService, error) {
	l = logger.With().Str("sub", "swarm").Logger()
	protoFiles, err := createSchemasForDeviceGroup(swarmCfg)
	if err != nil {
		return nil, err
	}

	tlmFunc, err := createTlmFnForDeviceGroup(swarmCfg, protoFiles)
	if err != nil {
		return nil, err
	}

	cmdFunc, err := createCmdFnForDeviceGroup(swarmCfg, protoFiles)
	if err != nil {
		return nil, err
	}

	cfgFunc, err := createCfgFnForDeviceGroup(swarmCfg, protoFiles)
	if err != nil {
		return nil, err
	}

	s, err := createSwarmForDeviceGroup(swarmCfg, bus, mirCtx, protoFiles)
	if err != nil {
		return nil, err
	}

	l.Info().Int("device_count", len(s.Devices)).Msg("Swarm created !")

	return &SwarmService{
		cfg:         swarmCfg,
		swarm:       s,
		schemaFiles: protoFiles,
		tlmFunc:     tlmFunc,
		cmdFunc:     cmdFunc,
		cfgFunc:     cfgFunc,
	}, nil
}

func (s *SwarmService) Deploy(ctx context.Context) ([]*sync.WaitGroup, error) {
	batchSize := 10
	if s.cfg.Spec.DeployBatchSize != 0 {
		batchSize = s.cfg.Spec.DeployBatchSize
	}
	wgs, err := s.swarm.BatchDeploy(ctx, batchSize)
	if err != nil {
		return wgs, fmt.Errorf("%w: %w", ErrDeployingSwarm, err)
	}

	l.Info().Int("device_count", len(s.swarm.Devices)).Msg("Swarm deployed !")

	wg := &sync.WaitGroup{}
	for _, d := range s.swarm.Devices {
		devGroupName := strings.Split(d.GetDeviceId(), "__")[0]
		// Telemetry
		for _, f := range s.tlmFunc[devGroupName] {
			wg.Add(1)
			go f(ctx, wg, d)
		}
		// Command
		for _, h := range s.cmdFunc[devGroupName] {
			h(d)
		}
		// Properties
		for _, h := range s.cfgFunc[devGroupName] {
			h(d)
		}
	}
	l.Info().Int("device_count", len(s.swarm.Devices)).Msg("Swarm deployed !")
	return append(wgs, wg), nil
}

func createSwarmForDeviceGroup(swarmCfg mir_v1.Swarm, bus *nats.Conn, mirCtx ui.Context, protoFiles map[string][]*descriptorpb.FileDescriptorProto) (swarm.Swarm, error) {
	s := swarm.NewSwarm(bus)
	for _, devGroup := range swarmCfg.Spec.Devices {
		createReqs := make([]*mir_apiv1.NewDevice, devGroup.Count)
		if devGroup.Count == 1 {
			createReqs[0] = &mir_apiv1.NewDevice{
				Meta: &mir_apiv1.Meta{
					Name:        devGroup.Meta.Name,
					Namespace:   devGroup.Meta.Namespace,
					Labels:      devGroup.Meta.Labels,
					Annotations: devGroup.Meta.Annotations,
				},
				Spec: &mir_apiv1.DeviceSpec{
					DeviceId: devGroup.Meta.Name,
				},
			}
		} else {
			for i := range devGroup.Count {
				createReqs[i] = &mir_apiv1.NewDevice{
					Meta: &mir_apiv1.Meta{
						Name:        devGroup.Meta.Name + "__" + strconv.Itoa(i),
						Namespace:   devGroup.Meta.Namespace,
						Labels:      devGroup.Meta.Labels,
						Annotations: devGroup.Meta.Annotations,
					},
					Spec: &mir_apiv1.DeviceSpec{
						DeviceId: devGroup.Meta.Name + "__" + strconv.Itoa(i),
					},
				}
			}
		}
		_, err := s.AddDevices(createReqs...).
			WithLogLevel(mir.LogLevel(swarmCfg.Spec.LogLevel)).
			WithPrettyLogger(true).
			WithCredentials(mirCtx.Credentials).
			WithCerticate(mirCtx.TlsCert, mirCtx.TlsKey).
			WithCA(mirCtx.RootCA).
			WithSchemaProto(protoFiles[devGroup.Meta.Name]...).
			Incubate()
		if err != nil {
			return s, fmt.Errorf("%w: %w", ErrIncubatingSwarm, err)
		}
	}
	return s, nil
}

func createSchemasForDeviceGroup(swarmCfg mir_v1.Swarm) (map[string][]*descriptorpb.FileDescriptorProto, error) {
	protoFiles := map[string][]*descriptorpb.FileDescriptorProto{}
	packageName := "swarm." + swarmCfg.Meta.Namespace + "." + swarmCfg.Meta.Name
	fieldsMaps := make(map[string]mir_v1.SwarmField)
	fieldsMessages := []mir_v1.SwarmField{}
	for _, field := range swarmCfg.Spec.Fields {
		fieldsMaps[field.Name] = field
		if field.Type == mir_v1.Message {
			fieldsMessages = append(fieldsMessages, field)
		}
	}

	// Create messages
	tlmFile := mir_proto.NewFileDescriptor(packageName, "fields")
	for _, msg := range fieldsMessages {
		msgDesc := &descriptorpb.DescriptorProto{
			Name:    proto.String(msg.Name),
			Options: &descriptorpb.MessageOptions{},
		}
		if msg.Tags != nil {
			proto.SetExtension(msgDesc.Options, devicev1.E_Meta, &devicev1.Meta{Tags: msg.Tags})
		}

		for i, fieldName := range msg.Fields {
			field, ok := fieldsMaps[fieldName]
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrFieldNotFound, fieldName)
			}
			vt, err := valueTypeToProtoType(field.Type)
			if err != nil {
				return nil, err
			}
			fieldDesc := &descriptorpb.FieldDescriptorProto{
				Name:    proto.String(field.Name),
				Number:  proto.Int32(int32(i + 1)),
				Type:    &vt,
				Options: &descriptorpb.FieldOptions{},
			}
			if field.Type == mir_v1.Message {
				fieldDesc.TypeName = proto.String(packageName + "." + field.Name)
			}
			proto.SetExtension(fieldDesc.Options, devicev1.E_FieldMeta, &devicev1.FieldMeta{Tags: field.Tags})
			msgDesc.Field = append(msgDesc.Field, fieldDesc)
		}
		tlmFile.MessageType = append(tlmFile.MessageType, msgDesc)
	}

	// Create root file for each device group
	for _, swarmDevs := range swarmCfg.Spec.Devices {
		files := []*descriptorpb.FileDescriptorProto{tlmFile}
		file := mir_proto.NewFileDescriptor(packageName, swarmDevs.Meta.Name)
		file.Dependency = append(file.Dependency, packageName+"/fields.proto")

		// Telemetry
		for _, tlmGroup := range swarmDevs.Telemetry {
			tlmDesc := mir_proto.NewTelemetryDescriptor(tlmGroup.Name, &devicev1.Meta{Tags: tlmGroup.Tags}, devicev1.TimestampType_TIMESTAMP_TYPE_NANO)

			for i, tlmName := range tlmGroup.Fields {
				tlmField, ok := fieldsMaps[tlmName]
				if !ok {
					return nil, fmt.Errorf("%w: %s", ErrFieldNotFound, tlmName)
				}
				vt, err := valueTypeToProtoType(tlmField.Type)
				if err != nil {
					return nil, err
				}
				fieldDesc := &descriptorpb.FieldDescriptorProto{
					Name:    proto.String(tlmField.Name),
					Number:  proto.Int32(int32(i + 2)), // +2 for the ts field and for index start at 1
					Type:    &vt,
					Options: &descriptorpb.FieldOptions{},
				}
				if tlmField.Type == mir_v1.Message {
					fieldDesc.TypeName = proto.String(packageName + "." + tlmField.Name)
				}
				proto.SetExtension(fieldDesc.Options, devicev1.E_FieldMeta, &devicev1.FieldMeta{Tags: tlmField.Tags})
				tlmDesc.Field = append(tlmDesc.Field, fieldDesc)
			}
			file.MessageType = append(file.MessageType, tlmDesc)
		}

		// Commands
		for _, cmdGroup := range swarmDevs.Commands {
			desc := mir_proto.NewCommandDescriptor(cmdGroup.Name, &devicev1.Meta{Tags: cmdGroup.Tags})

			for i, name := range cmdGroup.Fields {
				field, ok := fieldsMaps[name]
				if !ok {
					return nil, fmt.Errorf("%w: %s", ErrFieldNotFound, name)
				}
				vt, err := valueTypeToProtoType(field.Type)
				if err != nil {
					return nil, err
				}
				fieldDesc := &descriptorpb.FieldDescriptorProto{
					Name:    proto.String(field.Name),
					Number:  proto.Int32(int32(i + 1)),
					Type:    &vt,
					Options: &descriptorpb.FieldOptions{},
				}
				if field.Type == mir_v1.Message {
					fieldDesc.TypeName = proto.String(packageName + "." + field.Name)
				}
				proto.SetExtension(fieldDesc.Options, devicev1.E_FieldMeta, &devicev1.FieldMeta{Tags: field.Tags})
				desc.Field = append(desc.Field, fieldDesc)
			}
			file.MessageType = append(file.MessageType, desc)
		}

		// Properties
		for _, cfgGroup := range swarmDevs.Properties {
			desc := mir_proto.NewConfigDescriptor(cfgGroup.Name, &devicev1.Meta{Tags: cfgGroup.Tags})

			for i, name := range cfgGroup.Fields {
				field, ok := fieldsMaps[name]
				if !ok {
					return nil, fmt.Errorf("%w: %s", ErrFieldNotFound, name)
				}
				vt, err := valueTypeToProtoType(field.Type)
				if err != nil {
					return nil, err
				}
				fieldDesc := &descriptorpb.FieldDescriptorProto{
					Name:    proto.String(field.Name),
					Number:  proto.Int32(int32(i + 1)),
					Type:    &vt,
					Options: &descriptorpb.FieldOptions{},
				}
				if field.Type == mir_v1.Message {
					fieldDesc.TypeName = proto.String(packageName + "." + field.Name)
				}
				proto.SetExtension(fieldDesc.Options, devicev1.E_FieldMeta, &devicev1.FieldMeta{Tags: field.Tags})
				desc.Field = append(desc.Field, fieldDesc)
			}
			file.MessageType = append(file.MessageType, desc)
		}
		files = append(files, file)
		protoFiles[swarmDevs.Meta.Name] = files
	}

	return protoFiles, nil
}

func createTlmFnForDeviceGroup(swarmCfg mir_v1.Swarm, protoFiles map[string][]*descriptorpb.FileDescriptorProto) (map[string][]func(context.Context, *sync.WaitGroup, *mir.Mir), error) {
	telemetryFieldsMap := make(map[string]mir_v1.SwarmField)
	for _, field := range swarmCfg.Spec.Fields {
		telemetryFieldsMap[field.Name] = field
	}
	tlmFunc := map[string][]func(context.Context, *sync.WaitGroup, *mir.Mir){}
	packageName := "swarm." + swarmCfg.Meta.Namespace + "." + swarmCfg.Meta.Name
	for _, devGroup := range swarmCfg.Spec.Devices {
		for _, tlmGroup := range devGroup.Telemetry {
			sch, err := mir_proto.NewMirProtoSchemaWithMir(protoFiles[devGroup.Meta.Name]...)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCreatingSchema, err)
			}
			desc, err := sch.FindDescriptorByName(protoreflect.FullName(packageName + "." + tlmGroup.Name))
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrFindingMessage, err)
			}
			descMsg := desc.(protoreflect.MessageDescriptor)
			tlmMsgSetter, err := SetTelemetryMsg(descMsg, telemetryFieldsMap)
			if err != nil {
				return nil, err
			}

			fn := func(ctx context.Context, wg *sync.WaitGroup, d *mir.Mir) {
				for {
					select {
					case <-ctx.Done():
						wg.Done()
						return
					case <-time.After(tlmGroup.Interval):
						now := time.Now().UTC()
						msg, err := tlmMsgSetter(now)
						if err != nil {
							fmt.Println(err)
						}
						if err := d.SendTelemetry(msg); err != nil {
							fmt.Println(err)
						}
					}
				}
			}
			tlmFunc[devGroup.Meta.Name] = append(tlmFunc[devGroup.Meta.Name], fn)
		}
	}
	return tlmFunc, nil
}

func createCmdFnForDeviceGroup(swarmCfg mir_v1.Swarm, protoFiles map[string][]*descriptorpb.FileDescriptorProto) (map[string][]func(*mir.Mir), error) {
	fieldsMap := make(map[string]mir_v1.SwarmField)
	for _, field := range swarmCfg.Spec.Fields {
		fieldsMap[field.Name] = field
	}
	cmdFunc := map[string][]func(*mir.Mir){}
	packageName := "swarm." + swarmCfg.Meta.Namespace + "." + swarmCfg.Meta.Name
	for _, devGroup := range swarmCfg.Spec.Devices {
		for _, cmdGroup := range devGroup.Commands {
			sch, err := mir_proto.NewMirProtoSchemaWithMir(protoFiles[devGroup.Meta.Name]...)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCreatingSchema, err)
			}
			desc, err := sch.FindDescriptorByName(protoreflect.FullName(packageName + "." + cmdGroup.Name))
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrFindingMessage, err)
			}
			descMsg := desc.(protoreflect.MessageDescriptor)
			msg := dynamicpb.NewMessage(descMsg)

			fn := func(d *mir.Mir) {
				d.HandleCommand(msg, func(m proto.Message) (proto.Message, error) {
					time.Sleep(cmdGroup.Delay)
					return m, nil
				})
			}

			cmdFunc[devGroup.Meta.Name] = append(cmdFunc[devGroup.Meta.Name], fn)
		}
	}
	return cmdFunc, nil
}

func createCfgFnForDeviceGroup(swarmCfg mir_v1.Swarm, protoFiles map[string][]*descriptorpb.FileDescriptorProto) (map[string][]func(*mir.Mir), error) {
	fieldsMap := make(map[string]mir_v1.SwarmField)
	for _, field := range swarmCfg.Spec.Fields {
		fieldsMap[field.Name] = field
	}
	cfgFunc := map[string][]func(*mir.Mir){}
	packageName := "swarm." + swarmCfg.Meta.Namespace + "." + swarmCfg.Meta.Name
	for _, devGroup := range swarmCfg.Spec.Devices {
		for _, cmdGroup := range devGroup.Properties {
			sch, err := mir_proto.NewMirProtoSchemaWithMir(protoFiles[devGroup.Meta.Name]...)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCreatingSchema, err)
			}
			desc, err := sch.FindDescriptorByName(protoreflect.FullName(packageName + "." + cmdGroup.Name))
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrFindingMessage, err)
			}
			descMsg := desc.(protoreflect.MessageDescriptor)
			msg := dynamicpb.NewMessage(descMsg)

			fn := func(d *mir.Mir) {
				d.HandleProperties(msg, func(m proto.Message) {
					time.Sleep(cmdGroup.Delay)
					d.SendProperties(m)
				})
			}

			cfgFunc[devGroup.Meta.Name] = append(cfgFunc[devGroup.Meta.Name], fn)
		}
	}
	return cfgFunc, nil
}

func valueTypeToProtoType(vt mir_v1.ValueType) (descriptorpb.FieldDescriptorProto_Type, error) {
	switch vt {
	case mir_v1.Int8:
		// Note: Protobuf doesn't have native int8, using int32 as smallest signed int type
		return descriptorpb.FieldDescriptorProto_TYPE_INT32, nil
	case mir_v1.Int16:
		// Note: Protobuf doesn't have native int16, using int32 as smallest signed int type
		return descriptorpb.FieldDescriptorProto_TYPE_INT32, nil
	case mir_v1.Int32:
		return descriptorpb.FieldDescriptorProto_TYPE_INT32, nil
	case mir_v1.Int64:
		return descriptorpb.FieldDescriptorProto_TYPE_INT64, nil
	case mir_v1.Float32:
		return descriptorpb.FieldDescriptorProto_TYPE_FLOAT, nil
	case mir_v1.Float64:
		return descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, nil
	case mir_v1.Message:
		return descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, nil
	case mir_v1.Bool:
		return descriptorpb.FieldDescriptorProto_TYPE_BOOL, nil
	case mir_v1.String:
		return descriptorpb.FieldDescriptorProto_TYPE_STRING, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedValueType, vt)
	}
}

type FieldSetterFn func(t time.Time) (protoreflect.Value, error)
type MsgSetterFn func(t time.Time) (proto.Message, error)

func SetTelemetryMsg(descMsg protoreflect.MessageDescriptor, tlmFieldsMap map[string]mir_v1.SwarmField) (MsgSetterFn, error) {
	setters, err := setMsgGenerators(descMsg, tlmFieldsMap)
	if err != nil {
		return nil, err
	}

	return func(t time.Time) (proto.Message, error) {
		msg := dynamicpb.NewMessage(descMsg)
		msgReflect := msg.ProtoReflect()

		for i := 0; i < descMsg.Fields().Len(); i++ {
			f := descMsg.Fields().Get(i)

			val, err := setters[i](t)
			if err != nil {
				return nil, err
			}
			msgReflect.Set(f, val)
		}

		return msg, nil
	}, nil
}

func setMsgGenerators(descMsg protoreflect.MessageDescriptor, tlmFieldsMap map[string]mir_v1.SwarmField) ([]FieldSetterFn, error) {
	fieldSetters := []FieldSetterFn{}

	for i := 0; i < descMsg.Fields().Len(); i++ {
		f := descMsg.Fields().Get(i)

		if f.Kind() == protoreflect.MessageKind {
			fMsg := f.Message()
			setters, err := setMsgGenerators(fMsg, tlmFieldsMap)
			if err != nil {
				return nil, err
			}
			msgSetter := func(t time.Time) (protoreflect.Value, error) {
				msg := dynamicpb.NewMessage(fMsg)
				msgReflect := msg.ProtoReflect()

				for i := 0; i < fMsg.Fields().Len(); i++ {
					val, err := setters[i](t)
					if err != nil {
						return protoreflect.Value{}, err
					}
					msgReflect.Set(fMsg.Fields().Get(i), val)
				}

				return protoreflect.ValueOfMessage(msg), err
			}

			fieldSetters = append(fieldSetters, msgSetter)
			continue
		}

		// Timestamp exception, no generator to attached
		tsType, ok := proto.GetExtension(f.Options(), devicev1.E_Timestamp).(devicev1.TimestampType)
		if ok && tsType != devicev1.TimestampType_TIMESTAMP_TYPE_UNSPECIFIED {
			fieldSetters = append(fieldSetters, SetFieldValueFn(f, nil))
			continue
		}

		tlmField, ok := tlmFieldsMap[f.TextName()]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrFieldNotFound, f.TextName())
		}
		gen, err := NewGenerator(tlmField.Generator)
		if err != nil {
			return nil, fmt.Errorf("%w: %w: %s", ErrCreatingGenerator, err, f.TextName())
		}
		fieldSetters = append(fieldSetters, SetFieldValueFn(f, gen))
	}

	return fieldSetters, nil
}

func SetFieldValueFn(field protoreflect.FieldDescriptor, gen *Generator) FieldSetterFn {
	switch field.Kind() {
	case protoreflect.FloatKind:
		return func(t time.Time) (protoreflect.Value, error) {
			val, err := gen.Generate(t)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfFloat32(float32(val)), nil
		}
	case protoreflect.DoubleKind:
		return func(t time.Time) (protoreflect.Value, error) {
			val, err := gen.Generate(t)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfFloat64(val), nil
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return func(t time.Time) (protoreflect.Value, error) {
			val, err := gen.Generate(t)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfInt32(int32(val)), nil
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		tsType, ok := proto.GetExtension(field.Options(), devicev1.E_Timestamp).(devicev1.TimestampType)
		if ok && tsType != devicev1.TimestampType_TIMESTAMP_TYPE_UNSPECIFIED {
			switch tsType {
			case devicev1.TimestampType_TIMESTAMP_TYPE_SEC:
				return func(t time.Time) (protoreflect.Value, error) {
					return protoreflect.ValueOfInt64(t.Unix()), nil
				}
			case devicev1.TimestampType_TIMESTAMP_TYPE_MILLI:
				return func(t time.Time) (protoreflect.Value, error) {
					return protoreflect.ValueOfInt64(t.UnixMilli()), nil
				}
			case devicev1.TimestampType_TIMESTAMP_TYPE_MICRO:
				return func(t time.Time) (protoreflect.Value, error) {
					return protoreflect.ValueOfInt64(t.UnixMicro()), nil
				}
			case devicev1.TimestampType_TIMESTAMP_TYPE_NANO:
				return func(t time.Time) (protoreflect.Value, error) {
					return protoreflect.ValueOfInt64(t.UnixNano()), nil
				}
			case devicev1.TimestampType_TIMESTAMP_TYPE_FRACTION:
				return func(t time.Time) (protoreflect.Value, error) {
					// v := m.Get(fieldsDesc.Get(tsFieldIndex))
					// mrNested := v.Message()
					// secondsField := mrNested.Descriptor().Fields().ByName("seconds")
					// nanosField := mrNested.Descriptor().Fields().ByName("nanos")
					// seconds := mrNested.Get(secondsField).Int()
					// nanos := mrNested.Get(nanosField).Int()
					// ts := seconds*1e9 + nanos
					// if ts == 0 {
					// 	return time.Now().UTC().UnixNano()
					// }
					// return ts
					return protoreflect.Value{}, nil
				}
			}
		} else {
			return func(t time.Time) (protoreflect.Value, error) {
				val, err := gen.Generate(t)
				if err != nil {
					return protoreflect.Value{}, err
				}
				return protoreflect.ValueOfInt64(int64(val)), nil
			}
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return func(t time.Time) (protoreflect.Value, error) {
			val, err := gen.Generate(t)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfUint32(uint32(val)), nil
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return func(t time.Time) (protoreflect.Value, error) {
			val, err := gen.Generate(t)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfUint64(uint64(val)), nil
		}
	default:
		return nil
	}
	return nil
}
