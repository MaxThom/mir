package mir_device

import (
	"fmt"

	"github.com/maxthom/mir/libs/boiler/mir_config"
)

type builder struct {
	fileOpts []func(*mir_config.MirConfig)
	deviceId *string
	target   *string
	logLevel *string
}

type configFormat string

const (
	Yaml configFormat = "yaml"
	Json configFormat = "json"
)

// TODO natsio configuration

// Builder pattern to get your Mir
// device built and launched !
// TODO explains options
func Builder() builder {
	return builder{}
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
	fileName := "device.yaml"
	if f == Json {
		format = mir_config.Json
		fileName = "device.json"
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
// These have priority over the config from files
func (b builder) EnvVars() builder {
	b.fileOpts = append(b.fileOpts, mir_config.WithEnvVars())
	return b
}

// Options with writer like file or sdt out?
// TBD
func (b builder) LogLevel(l string) builder {
	b.logLevel = &l
	return b
}

// Return the Mir device object to
// be used to interact with the system
func (b builder) Build() *Mir {
	// TODO connection to nats
	// TODO config
	// TODO logging
	c := cfg{}

	if len(b.fileOpts) > 0 {
		err, warns := mir_config.New("mir", b.fileOpts...).LoadAndUnmarshal(&c)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if warns != nil {
			fmt.Println(warns)
		}
	}
	if b.deviceId != nil {
		c.DeviceId = *b.deviceId
	}
	if b.target != nil {
		c.Target = *b.target
	}
	if b.logLevel != nil {
		c.LogLevel = *b.logLevel
	}
	return &Mir{
		cfg: c,
	}
}
