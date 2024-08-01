package mir

import (
	"io"
	"os"

	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type builder struct {
	fileOpts            []func(*mir_config.MirConfig)
	deviceId            *string
	target              *string
	logLevel            *logLevel
	logWriters          []io.Writer
	telemetrySchema     *descriptorpb.FileDescriptorSet
	telemetryModuleFlag bool
}

type configFormat string
type logLevel string

func (l logLevel) String() string {
	return string(l)
}

const (
	Yaml            configFormat = "yaml"
	Json            configFormat = "json"
	LogLevelTrace   logLevel     = "trace"
	LogLevelDebug   logLevel     = "debug"
	LogLevelInfo    logLevel     = "info"
	LogLevelWarning logLevel     = "warn"
	LogLevelError   logLevel     = "error"
	LogLevelFatal   logLevel     = "fatal"
)

// Builder pattern to get your Mir
// device built and launched !
// Configure logging, device authentication and config loading.
func Builder() builder {
	return builder{
		telemetrySchema: new(descriptorpb.FileDescriptorSet),
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
func (b builder) LogLevel(l logLevel) builder {
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

func (b builder) TelemetrySchema(s ...protoreflect.FileDescriptor) builder {
	if len(s) > 0 {
		b.telemetryModuleFlag = true
	}
	for _, f := range s {
		b.telemetrySchema.File = append(b.telemetrySchema.File,
			protodesc.ToFileDescriptorProto(f))
	}
	return b
}
func (b builder) TelemetrySchemaProto(s ...*descriptorpb.FileDescriptorProto) builder {
	if len(s) > 0 {
		b.telemetryModuleFlag = true
	}
	b.telemetrySchema.File = append(b.telemetrySchema.File, s...)
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

	if len(b.logWriters) == 0 {
		b.logWriters = append(b.logWriters, os.Stdout)
	}

	l := mir_log.Setup(
		mir_log.WithLogLevel(c.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithAppName("mir"),
		mir_log.WithCustomWriters(b.logWriters),
	)

	if errs != nil {
		l.Error().Err(errs).Msg("Error while loading configuration")
		return nil, errs
	}
	l.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")

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

	return &Mir{
		cfg:             c,
		l:               l,
		telemetrySchema: b.telemetrySchema,
	}, nil
}
