package mir

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestLoadCompileConfig(t *testing.T) {
	mir, err := Builder().
		DeviceId("0xf86ea").
		Store(StoreOptions{InMemory: true}).
		Target("nats://127.0.0.1:4222").
		LogLevel(LogLevelInfo).
		//LogWriters([]io.Writer{os.Stdout}).
		//DefaultConfigFile(Yaml).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelInfo), mir.GetConfig().LogLevel)
}

func TestLoadFileConfigJson(t *testing.T) {
	// Marhal config struct instead
	cfg := `{
		"mir": {
			"target": "nats://127.0.0.1:4222",
			"logLevel": "info",
			"device": {
				"id": "0xf86ea"
			}
		}
	}`
	fileName, err := writeBytesToFile([]byte(cfg))
	defer os.Remove(fileName)

	mir, err := Builder().
		CustomConfigFile(fileName, Json).
		Store(StoreOptions{InMemory: true}).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelInfo), mir.GetConfig().LogLevel)
}

func TestLoadFileConfigYaml(t *testing.T) {
	// Marhal config struct instead
	cfg := `
mir:
    target: "nats://127.0.0.1:4222"
    logLevel: "info"
    device:
        id: "0xf86ea"
`
	fileName, err := writeBytesToFile([]byte(cfg))
	defer os.Remove(fileName)
	fmt.Println(fileName)

	mir, err := Builder().
		CustomConfigFile(fileName, Yaml).
		Store(StoreOptions{InMemory: true}).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelInfo), mir.GetConfig().LogLevel)
}

func TestLoadFileConfigMix(t *testing.T) {
	// Marhal config struct instead
	cfg := `
mir:
    device:
        id: "0xf86ea"
`
	fileName, err := writeBytesToFile([]byte(cfg))
	defer os.Remove(fileName)
	fmt.Println(fileName)

	mir, err := Builder().
		LogLevel(LogLevelInfo).
		Target("nats://127.0.0.1:4222").
		Store(StoreOptions{InMemory: true}).
		CustomConfigFile(fileName, Yaml).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelInfo), mir.GetConfig().LogLevel)
}

func TestLoadFileConfigMissingFields(t *testing.T) {
	// Marhal config struct instead
	cfg := ``
	fileName, err := writeBytesToFile([]byte(cfg))
	defer os.Remove(fileName)
	fmt.Println(fileName)

	mir, err := Builder().
		LogLevel(LogLevelInfo).
		Target("nats://127.0.0.1:4222").
		Store(StoreOptions{InMemory: true}).
		CustomConfigFile(fileName, Yaml).
		Build()

	assert.ErrorType(t, err, MirBuilderFieldsError{})
	assert.Equal(t, mir == nil, true)
}

// TODO one unit test with nested fields
// TODO one unit test with arrays
func TestLoadWithEnvVars(t *testing.T) {
	os.Setenv("MIR__DEVICE__ID", "0xf86ea")
	os.Setenv("MIR__TARGET", "nats://127.0.0.1:4222")
	os.Setenv("MIR__LOG_LEVEL", "info")
	defer os.Unsetenv("MIR__DEVICE__ID")
	defer os.Unsetenv("MIR__TARGET")
	defer os.Unsetenv("MIR__LOG_LEVEL")
	mir, err := Builder().
		EnvVars().
		Store(StoreOptions{InMemory: true}).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelInfo), mir.GetConfig().LogLevel)
}

func TestLoadConfigMix(t *testing.T) {
	os.Setenv("MIR__DEVICE__ID", "0xf86ea")
	defer os.Unsetenv("MIR__DEVICE__ID")

	cfg := `{
	"mir": {
			"target": "nats://127.0.0.1:4222"
		}
	}`
	fileName, err := writeBytesToFile([]byte(cfg))
	defer os.Remove(fileName)

	mir, err := Builder().
		LogLevel(LogLevelDebug).
		EnvVars().
		Store(StoreOptions{InMemory: true}).
		CustomConfigFile(fileName, Json).
		Build()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "0xf86ea", mir.GetConfig().Device.Id)
	assert.Equal(t, "nats://127.0.0.1:4222", mir.GetConfig().Target)
	assert.Equal(t, string(LogLevelDebug), mir.GetConfig().LogLevel)
}

func writeBytesToFile(b []byte) (string, error) {
	f, err := os.CreateTemp("", "device_*.json")
	if err != nil {
		return "", err
	}

	if _, err := f.Write(b); err != nil {
		return "", nil
	}
	if err := f.Close(); err != nil {
		return "", nil
	}
	return f.Name(), nil
}
