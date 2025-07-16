# Prerequisites

In this section, we will initialize the project and access the Mir Device SDK.

Make sure you have the Mir Server up & running and the Mir CLI ready to be used. Follow the [Running Mir Setup](../../running_mir/binary.md).

## Initialize Go project

```bash
go mod init github.com/<user/org>/<project>
```

## Access Mir Device SDK

Go packages are managed in GitHub repository.
Since the repository is private, you need to adjust your git configuration before you can execute this line.

```bash
go get github.com/maxthom/mir/
```

Make sure you have access to the [repository](https://github.com/maxthom/mir) on GitHub and your local env. is setup with an SSH key for authentication.

First, we need to tell Go to use the SSH protocol instead of HTTPS to access the GitHub repository.

```bash
# In ~/.gitconfig
[url "ssh://git@github.com/maxthom/mir"]
  insteadOf = https://github.com/maxthom/mir
```

Even though packages are stored in Git repositories, they get downloaded through Go mirror.
Therefore, we must tell Go to download it directly from the Git repository.

```bash
go env -w GOPRIVATE=github.com/maxthom/mir
```

If any import match the pattern `github.com/maxthom/mir/*`, Go will download the package directly from the Git repository.

Now, you can run

```bash
go get github.com/maxthom/mir/
```

Ready to roll!

## Mir tooling

Mir requires a set of utility tools to properly create devices:

- [protoc](https://grpc.io/docs/protoc-installation/): Protocol buffer compiler.

It must be manually installed via your package manager:

```bash
# Debian, Ubuntu, Raspian
sudo apt install protobuf-compiler
# Arch based
sudo pacman -S protobuf
```

The following can be installed via `go install` or using Mir CLI:

- [buf](https://github.com/bufbuild/buf/): Go bindings for protobuf compiler.
- [protoc-go-gen](https://github.com/bufbuild/buf/): Go bindings for protobuf compiler.

```bash
# Mir CLI
mir tools install

# Manually
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/buf/cmd/buf@latest
```
