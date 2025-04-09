package mir

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type builder struct {
	fileOpts            []func(*mir_config.MirConfig)
	deviceId            *string
	target              *string
	logLevel            *LogLevel
	logWriters          []io.Writer
	schema              *descriptorpb.FileDescriptorSet
	noSchemaOnBoot      *bool
	telemetryModuleFlag bool
	excludeMirProtoFlag bool
	storeOpts           StoreOptions
}

type configFormat string
type LogLevel string

func (l LogLevel) String() string {
	return string(l)
}

const (
	Yaml            configFormat = "yaml"
	Json            configFormat = "json"
	LogLevelTrace   LogLevel     = "trace"
	LogLevelDebug   LogLevel     = "debug"
	LogLevelInfo    LogLevel     = "info"
	LogLevelWarning LogLevel     = "warn"
	LogLevelError   LogLevel     = "error"
	LogLevelFatal   LogLevel     = "fatal"
)

// Builder pattern to get your Mir
// device built and launched !
// Configure logging, device authentication and config loading.
func Builder() builder {
	tar := "nats://127.0.0.1:4222"
	return builder{
		schema: new(descriptorpb.FileDescriptorSet),
		target: &tar,
	}
}

// Set unique ID of the device
// Can be retrieved from creating a new device
// or be used with an Autoprovisioner.
// See b.DeviceProvisioner(p Provisioner)
func (b builder) DeviceId(id string) builder {
	b.deviceId = &id
	return b
}

// Specify the url of the server instance
// default to nats://127.0.0.1:4222
func (b builder) Target(t string) builder {
	b.target = &t
	return b
}

// Use a configuration file to load the
// device_id and the target. Specifying those configs
// in the builder pattern have greater priority
// then loading from the config file.
// The file is loaded in folder /etc/mir/device.[json|yaml]
// If not running in a container, it will also load from path
// $HOME/.config/mir/device.[json|yaml]
func (b builder) DefaultConfigFile(f configFormat) builder {
	format := mir_config.Yaml
	fileName := "mir/device.yaml"
	if f == Json {
		format = mir_config.Json
		fileName = "mir/device.json"
	}
	b.fileOpts = append(b.fileOpts,
		mir_config.WithEtcFilePath(fileName, format, false),
		mir_config.WithXdgConfigHomeFilePath(fileName, format, true),
	)
	return b
}

// Use a configuration file to load the
// device_id and the target. Specifying those configs
// in the builder pattern have greater priority
// then loading from the config file.
func (b builder) CustomConfigFile(fullPath string, f configFormat) builder {
	format := mir_config.Yaml
	if f == Json {
		format = mir_config.Json
	}
	b.fileOpts = append(b.fileOpts, mir_config.WithFilePath(fullPath, format, false))

	return b
}

// Load config from environment variables following this nomenclature
// - prefix with MIR
// - __ to represent nesting
// - They follow the json or yaml configuration layout
// These have priority over the config from files
func (b builder) EnvVars() builder {
	b.fileOpts = append(b.fileOpts, mir_config.WithEnvVars("MIR"))
	return b
}

// Set loglevel of the logger. Default to info
func (b builder) LogLevel(l LogLevel) builder {
	b.logLevel = &l
	return b
}

// Use a custom io.Writer to store the logs of the Mir SDK.
// Default to os.sdtout. Useful to combine with file logging.
// Can be stacked by calling it multiple times.
// TODO a writer for sending the logs upstream
func (b builder) LogWriter(w io.Writer) builder {
	b.logWriters = append(b.logWriters, w)
	return b
}

// Use a set of io.Writer to store the logs of the Mir SDK.
// Default to os.sdtout. Useful to combine with file logging.
// Can be stacked by calling it multiple times.
// TODO a writer for sending the logs upstream
func (b builder) LogWriters(writers []io.Writer) builder {
	b.logWriters = append(b.logWriters, writers...)
	return b
}

func (b builder) LogPretty(colors bool) builder {
	b.logWriters = append(b.logWriters, zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: !colors,
	})
	return b
}

func (b builder) Schema(s ...protoreflect.FileDescriptor) builder {
	if len(s) > 0 {
		b.telemetryModuleFlag = true
	}
	for _, f := range s {
		b.schema.File = append(b.schema.File,
			protodesc.ToFileDescriptorProto(f))
	}
	return b
}
func (b builder) SchemaProto(s ...*descriptorpb.FileDescriptorProto) builder {
	if len(s) > 0 {
		b.telemetryModuleFlag = true
	}
	b.schema.File = append(b.schema.File, s...)
	return b
}

func (b builder) ExcludeMirSchema() builder {
	b.excludeMirProtoFlag = true
	return b
}

// Send the device schema on device boot.
// Default to true
func (b builder) ExcludeSchemaOnLaunch() builder {
	t := true
	b.noSchemaOnBoot = &t
	return b
}

// Set persistent device store options
// Path:
// default to $XDG_DATA_HOME/mir/mir.db
// On linux, that is $HOME/.local/share/mir/mir.db
func (b builder) Store(opts StoreOptions) builder {
	b.storeOpts = opts
	return b
}

// Return the Mir device object to
// be used to interact with the system
// TODO returns errors instead of logs, use error.wrap and error.join
func (b builder) Build() (*Mir, error) {
	c := Cfg{}

	var errs error
	var lookupFiles, foundFiles []string
	if len(b.fileOpts) > 0 {
		errs, lookupFiles, foundFiles = mir_config.New("mir", b.fileOpts...).LoadAndUnmarshal(&c)
	}
	if b.deviceId != nil {
		c.DeviceId = *b.deviceId
	}
	if b.target != nil {
		c.Target = *b.target
	}
	if b.logLevel != nil {
		c.LogLevel = b.logLevel.String()
	} else {
		c.LogLevel = "info"
	}
	if b.noSchemaOnBoot != nil {
		c.NoSchemaOnBoot = *b.noSchemaOnBoot
	}

	if len(b.logWriters) == 0 {
		b.logWriters = append(b.logWriters, os.Stdout)
	}

	l := mir_log.Setup(
		mir_log.WithLogLevel(c.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithCustomWriters(b.logWriters),
	)
	cleanLogger := l.With().Logger()
	l = l.With().Str("source", "mir").Logger()

	if errs != nil {
		l.Error().Err(errs).Msg("Error while loading configuration")
		return nil, errs
	}
	if len(b.fileOpts) > 0 {
		l.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")
	}

	if b.storeOpts.Path == "" {
		c.Store.Path = b.storeOpts.Path
	}
	if b.storeOpts.InMemory {
		c.Store.InMemory = b.storeOpts.InMemory
	}
	if b.storeOpts.Msgs.DiskSpaceLimit > 0 {
		c.Store.Msgs.DiskSpaceLimit = b.storeOpts.Msgs.DiskSpaceLimit
	}
	if b.storeOpts.Msgs.RententionLimit > 0 {
		c.Store.Msgs.RententionLimit = b.storeOpts.Msgs.RententionLimit
	}
	if b.storeOpts.Msgs.MsgStorageType != StorageTypeNone {
		c.Store.Msgs.MsgStorageType = b.storeOpts.Msgs.MsgStorageType
	}
	if b.storeOpts.Path == "" && c.Store.Path == "" {
		c.Store.Path = filepath.Join(xdg.DataHome, "mir", "mir.db")
	}
	if b.storeOpts.Msgs.DiskSpaceLimit == 0 && c.Store.Msgs.DiskSpaceLimit == 0 {
		c.Store.Msgs.DiskSpaceLimit = 85
	}
	if b.storeOpts.Msgs.RententionLimit == 0 && c.Store.Msgs.RententionLimit == 0 {
		c.Store.Msgs.RententionLimit = JsonReadableDuration(time.Minute * 10080) // A week
	}
	if b.storeOpts.Msgs.MsgStorageType == StorageTypeNone && c.Store.Msgs.MsgStorageType == StorageTypeNone {
		c.Store.Msgs.MsgStorageType = StorageTypeOnlyIfOffline
	}

	if prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(c); err != nil {
		l.Error().Err(err).Msg("Error marshalling config")
	} else {
		l.Info().Str("config", string(prettyCfg)).Msg("")
	}

	fieldsErr := []string{}
	if c.DeviceId == "" {
		fieldsErr = append(fieldsErr, "DeviceId is required to identity the device")
	}
	if c.Target == "" {
		fieldsErr = append(fieldsErr, "Target is required to connect to the server")
	}
	if len(fieldsErr) > 0 {
		return nil, MirBuilderFieldsError{
			Fields: fieldsErr,
		}
	}

	// Add descriptordb for mir options imports
	b.schema.File = append(b.schema.File,
		protodesc.ToFileDescriptorProto(
			descriptorpb.File_google_protobuf_descriptor_proto,
		),
	)
	if !b.excludeMirProtoFlag {
		b.schema.File = append(b.schema.File,
			protodesc.ToFileDescriptorProto(
				devicev1.File_mir_device_v1_mir_proto,
			),
		)
	}

	reg, err := protodesc.NewFiles(b.schema)
	if err != nil {
		return nil, fmt.Errorf("error creating schema registry: %w", err)
	}

	store, err := NewStore(c.Store)
	if err != nil {
		return nil, fmt.Errorf("error creating store: %w", err)
	}
	return &Mir{
		cfg:         c,
		l:           l.With().Str("device_id", c.DeviceId).Logger(),
		cleanLogger: cleanLogger.With().Str("device_id", c.DeviceId).Logger(),
		store:       store,
		schema:      b.schema,
		schemaReg:   reg,
		cmdHandlers: make(map[string]cmdHandlerValue),
		cfgHandlers: make(map[string]cfgHandlerValue),
	}, nil
}
