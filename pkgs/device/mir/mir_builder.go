package mir

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	"github.com/maxthom/mir/internal/libs/systemid"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type builder struct {
	fileOpts            []func(*mir_config.MirConfig)
	deviceId            *string
	deviceIdGenerator   *IdGenerator
	deviceIdPrefix      *IdPrefix
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
	return builder{
		schema: new(descriptorpb.FileDescriptorSet),
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

func (b builder) DeviceIdGenerator(t IdGenerator) builder {
	b.deviceIdGenerator = &t
	return b
}

func (b builder) DeviceIdPrefix(p IdPrefix) builder {
	b.deviceIdPrefix = &p
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

// Load config from environment variables following this nomenclature:
//   - __ to represent nesting
//   - _ for multiple words where the first letter after it becomes capitalize
//   - They follow the json or yaml configuration layout
//   - Use index to represent array elements MIR__SENSORS__0__NAME
//   - eg: MIR__DEVICE__ID for mir: device: id: <id>
//   - eg: MIR__LOG_LEVEL for mir: logLevel: <level>
//
// These have priority over the config from files
func (b builder) EnvVars() builder {
	b.fileOpts = append(b.fileOpts, mir_config.WithEnvVars(""))
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

type MirCfg struct {
	Mir  Config `json:"mir" yaml:"mir"`
	User any    `json:"user" yaml:"user"`
}

func (b builder) BuildWithExtraConfig(extraCfg any) (*Mir, error) {
	return b.build(extraCfg)
}

func (b builder) Build() (*Mir, error) {
	return b.build(nil)
}

// Return the Mir device object to
// be used to interact with the system
// TODO returns errors instead of logs, use error.wrap and error.join
func (b builder) build(extraCfg any) (*Mir, error) {
	cfg := MirCfg{
		Mir:  Config{},
		User: extraCfg,
	}
	var errs error
	var lookupFiles, foundFiles []string
	if len(b.fileOpts) > 0 {
		errs, lookupFiles, foundFiles = mir_config.New("mir", b.fileOpts...).LoadAndUnmarshal(&cfg)
	}

	// Top
	if b.target != nil {
		cfg.Mir.Target = *b.target
	} else if cfg.Mir.Target == "" {
		cfg.Mir.Target = "nats://127.0.0.1:4222"
	}
	if b.logLevel != nil {
		cfg.Mir.LogLevel = b.logLevel.String()
	} else if cfg.Mir.LogLevel == "" {
		cfg.Mir.LogLevel = "info"
	}

	// Device
	if b.deviceId != nil {
		cfg.Mir.Device.Id = *b.deviceId
	}
	if b.deviceIdGenerator != nil {
		cfg.Mir.Device.IdGenerator = b.deviceIdGenerator
	}
	if b.deviceIdPrefix != nil {
		cfg.Mir.Device.IdPrefix = b.deviceIdPrefix
	}
	if b.noSchemaOnBoot != nil {
		cfg.Mir.Device.NoSchemaOnBoot = *b.noSchemaOnBoot
	}

	// Store
	if b.storeOpts.FolderPath != "" {
		cfg.Mir.LocalStore.FolderPath = b.storeOpts.FolderPath
	} else if cfg.Mir.LocalStore.FolderPath == "" {
		cfg.Mir.LocalStore.FolderPath = filepath.Join(xdg.DataHome, "mir")
	}
	if b.storeOpts.InMemory {
		cfg.Mir.LocalStore.InMemory = b.storeOpts.InMemory
	}
	if b.storeOpts.DiskSpaceLimit > 0 {
		cfg.Mir.LocalStore.DiskSpaceLimit = b.storeOpts.DiskSpaceLimit
	} else if cfg.Mir.LocalStore.DiskSpaceLimit == 0 {
		cfg.Mir.LocalStore.DiskSpaceLimit = 85
	}
	if b.storeOpts.RetentionLimit > 0 {
		cfg.Mir.LocalStore.RetentionLimit = b.storeOpts.RetentionLimit
	} else if cfg.Mir.LocalStore.RetentionLimit == 0 {
		cfg.Mir.LocalStore.RetentionLimit = time.Minute * 10080 // A week
	}
	if b.storeOpts.PersistenceType != "" {
		cfg.Mir.LocalStore.PersistenceType = b.storeOpts.PersistenceType
	} else if cfg.Mir.LocalStore.PersistenceType == "" {
		cfg.Mir.LocalStore.PersistenceType = PersistentTypeOnlyIfOffline
	}

	if len(b.logWriters) == 0 {
		b.logWriters = append(b.logWriters, os.Stdout)
	}

	l := mir_log.Setup(
		mir_log.WithLogLevel(cfg.Mir.LogLevel),
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

	fieldsErr := []string{}
	if cfg.Mir.Device.Id == "" {
		if cfg.Mir.Device.IdGenerator == nil || (cfg.Mir.Device.IdGenerator != nil && !cfg.Mir.Device.IdGenerator.IsActive()) {
			fieldsErr = append(fieldsErr, "Device.Id or Device.IdGenerator is required to identify the device")
		}
	}
	if cfg.Mir.Target == "" {
		fieldsErr = append(fieldsErr, "Target is required to connect to the server")
	}
	if cfg.Mir.LocalStore.PersistenceType != PersistentTypeNoStorage &&
		cfg.Mir.LocalStore.PersistenceType != PersistentTypeOnlyIfOffline &&
		cfg.Mir.LocalStore.PersistenceType != PersistentTypeAlways {
		fieldsErr = append(fieldsErr, "Invalid local store persistence type [nostorage|ifoffline|always]")
	}
	if cfg.Mir.LocalStore.DiskSpaceLimit < 0 || cfg.Mir.LocalStore.DiskSpaceLimit > 99 {
		fieldsErr = append(fieldsErr, "Disk space limit must be a valid pourcentage between 0 and 99")
	}

	if cfg.Mir.Device.Id == "" && cfg.Mir.Device.IdGenerator != nil && cfg.Mir.Device.IdGenerator.IsActive() {
		structId, err := systemid.GetShortDeviceID(cfg.Mir.Device.IdGenerator.ToSystemIdOpts())
		if err != nil {
			return nil, fmt.Errorf("error generating device id: %w", err)
		}
		cfg.Mir.Device.Id = structId
	}
	if cfg.Mir.Device.IdPrefix != nil && cfg.Mir.Device.IdPrefix.IsActive() {
		prefix, err := systemid.GetPrefix(cfg.Mir.Device.IdPrefix.ToSystemIdPrefixOpts())
		if err != nil {
			return nil, fmt.Errorf("error generating device prefix: %w", err)
		}
		cfg.Mir.Device.Id = fmt.Sprintf("%s_%s", prefix, cfg.Mir.Device.Id)
	}

	if prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(cfg); err != nil {
		l.Error().Err(err).Msg("Error marshalling config")
	} else {
		l.Info().Str("config", string(prettyCfg)).Msg("")
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

	store, err := NewStore(cfg.Mir.LocalStore)
	if err != nil {
		return nil, fmt.Errorf("error creating store: %w", err)
	}
	return &Mir{
		cfg:         cfg.Mir,
		l:           l.With().Str("device_id", cfg.Mir.Device.Id).Logger(),
		cleanLogger: cleanLogger.With().Str("device_id", cfg.Mir.Device.Id).Logger(),
		store:       store,
		schema:      b.schema,
		schemaReg:   reg,
		cmdHandlers: make(map[string]cmdHandlerValue),
		cfgHandlers: make(map[string]cfgHandlerValue),
	}, nil
}

func getMacAddr() ([]string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range ifas {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
		}
	}
	return as, nil
}
