# Access Mir Binary and SDK from private repository

Since the repository is private, you need to adjust your Git and Go configuration
before you can access the sdk or install the CLI. The goal is to be able to run those commands:

```bash
# Install CLI
go install github.com/maxthom/mir/cmds/mir@latest
# Import DeviceSDK to your project
go get github.com/maxthom/mir/
```

First, make sure you have access to the [repository](https://github.com/maxthom/mir) on
GitHub and your local env. is setup with an SSH key for authentication.

Second, we need to tell Go to use the SSH protocol instead of HTTP to access
the GitHub repository so it can pass credentials.

```bash
# In ~/.gitconfig
[url "ssh://git@github.com/maxthom/mir"]
  insteadOf = https://github.com/maxthom/mir
```

Even though Go packages are stored in Git repositories, they get downloaded through Go mirror.
Therefore, we must tell Go to download it directly from the Git repository.

```bash
go env -w GOPRIVATE=github.com/maxthom/mir
```

If any import match the pattern `github.com/maxthom/mir/*`, Go will download the package directly from the Git repository.

Now, you can run

```bash
# CLI
go install github.com/maxthom/mir/cmds/mir@latest
# DeviceSDK
go get github.com/maxthom/mir/
```

Ready to roll 🚀!
