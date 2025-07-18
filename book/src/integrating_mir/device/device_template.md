
# Project template

The Mir CLI provides templates to initialize new projects with a basic layout. Inside the project folder, run the following:

```sh
# With Buf (recommended)
mir tools generate device_template github.com/<user/org>/<project>
# With Protoc
mir tools generate device_template --proto=protoc github.com/<user/org>/<project>
```

## Structure

The device template creates a complete Go project structure optimized for Mir development:

```
project/
├── cmd/
│   └── main.go               # Main application entry point with Mir SDK initialization
├── proto/                    # Protocol Buffer definitions directory
│   ├── mir/
│   │   └── device/
│   │       └── v1/
│   │           └── mir.proto # Mir Device SDK proto definitions
│   └── schema/               # Device-specific schema definitions
│       └── v1/
│           └── schema.proto  # Custom device schema template
├── buf.yaml                  # Buf configuration for proto management
├── buf.gen.yaml              # Buf code generation configuration
├── config.yaml               # Device configuration example
├── makefile/justfile         # Common tasks
├── USAGE.md                  # Usage documentation and getting started guide
└── go.mod
```

#### makefile/justfile

Common commands to help develop your device

- **`make/just proto`**: Generates Go code from Protocol Buffer definitions (using buf or protoc)
- **`make/just build`**: Compiles the device binary
- **`make/just run`**: Runs the device application for development

#### schema.proto

Device-specific Protocol Buffer schema definitions that define your device's communication interface for telemetry, commands and configuration.

#### mir.proto

Mir specific protobuf extentions used by the SDK. This file should not be edited.

#### config.yaml

Device configuration file with development-ready defaults.

#### buf.yaml (buf template only)

The buf.yaml file defines a workspace, which represents a directory or directories of Protobuf files that you want to treat as a unit.

#### buf.gen.yaml (buf template only)

buf.gen.yaml is a configuration file used by the buf generate command to generate integration code for the languages of your choice, in thise case: Go.

## Protobuf Files Management

The Mir CLI offers two approaches for managing Protocol Buffer files: the traditional `protoc` compiler and the modern `buf` tool. While both work seamlessly with Mir, **buf is strongly recommended** for new projects due to its superior developer experience and modern workflow.

**buf advantages:**
- **Faster compilation** with intelligent caching and parallel processing
- **Built-in linting** catches common protobuf issues before they become problems
- **Dependency management** handles external proto dependencies automatically
- **Breaking change detection** prevents accidental API changes
- **Better error messages** with clear guidance on how to fix issues
- **Simplified configuration** with declarative YAML files instead of complex command-line flags

**protoc advantages:**
- **Wider ecosystem support** with broader tooling compatibility
- **Lower learning curve** for teams already familiar with protoc workflows
- **Direct control** over compilation flags and plugin options

You can specify which approach to use when generating the device template.
