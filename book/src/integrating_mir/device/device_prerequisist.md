# Prerequisist

In this section, we will intialize the project and access the Mir Device SDK.

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

First of, we need to tell git to use the SSH protocol to access the GitHub repository.

```bash
# In ~/.gitconfig
[url "ssh://git@github.com/"]
  insteadOf = https://github.com
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

- [buf](https://github.com/air-verse/air): A modern protobuf schema manager.
- [protoc](https://github.com/bufbuild/buf/): The protobuf compiler.

They can be installed via `go install` or using Mir CLI:

```bash
# Mir CLI
mir tools install

# Manually
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/buf/cmd/buf@latest
```
