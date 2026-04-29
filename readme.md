# Glove

Typed drivers for external commands in Go.

Glove helps you wrap command-line tools with small, ergonomic Go drivers.
Instead of assembling command calls from loose strings, maps, and fragile argument slices, you describe the external
tool as a typed command surface and let Glove render, validate, resolve, and execute the command call.

It is not a CLI framework for building your own command-line app.
It is the other side of that equation: a library for driving
existing tools like `git`, `docker`, `kubectl`, `ffmpeg`, `terraform`, or internal company binaries from Go code.

```go
result := docker.Run(ctx,
    Compose{File: "compose.dev.yml"},
    Up{Detach: true, Service: "api"},
)
```

Constructed command:

```sh
docker compose --file compose.dev.yml up --detach api
```

Glove gives external CLIs a typed control surface without hiding the fact that they are still CLIs.

## Installation

```bash
go get github.com/qlustra/glove
```

## Why Glove?

Calling external tools from Go usually starts simple:

```go
exec.CommandContext(ctx, "git", "clone", "--branch", branch, repo, dir)
```

Then the integration grows.

You add optional flags. Repeated flags. Positional arguments. Environment variables. Working directories. Version checks.
Tool detection. Compatibility behavior. Test doubles. Debug logging. Dry-runs. Structured output handling.

Eventually, the command call becomes an untyped mini-language spread across string slices.

Glove is for that middle ground where a full SDK does not exist, is not worth writing,
or would be less honest than calling the tool directly — but raw `exec.Command` has become too loose.

## Core idea

Describe an external tool as a typed command tree:

```go
type Git struct {
    Bin   glove.Bin `bin:"git" version:"--version"`
    Clone Clone     `cmd:"clone"`
}

type Clone struct {
    Branch string `flag:"--branch,-b"`
    Depth  int    `flag:"--depth"`
    Repo   string `arg:"0,required"`
    Dir    string `arg:"1"`
}
```

Then call it using command frames:

```go
git := glove.New(Git{})

result := git.Run(ctx, Clone{
    Branch: "main",
    Depth:  1,
    Repo:   "https://github.com/acme/project.git",
    Dir:    "./project",
})
```

Constructed command:

```sh
git clone --branch main --depth 1 https://github.com/acme/project.git ./project
```

Each frame contributes a subcommand name, flags, positional arguments, compatibility metadata, and rendering behavior.

## Status

Glove is currently an early design/prototype concept.

The intended direction is:

* reflection-first API for fast iteration
* typed command trees using structs and tags
* explicit command frames for nested commands
* binary resolution through `PATH` or configured overrides
* optional version detection and compatibility rules
* injectable runners for testing and custom execution
* possible code generation later for stronger typed wrappers

The API shown in this README represents the target shape, not a finalized stable contract.

## Quick start

Define a small driver surface:

```go
package main

import (
    "context"
    "fmt"

    "github.com/qlustra/glove"
)

type Git struct {
    Bin   glove.Bin `bin:"git" version:"--version"`
    Clone Clone     `cmd:"clone"`
}

type Clone struct {
    Branch string `flag:"--branch,-b"`
    Depth  int    `flag:"--depth"`
    Repo   string `arg:"0,required"`
    Dir    string `arg:"1"`
}

func main() {
    ctx := context.Background()

    git := glove.New(Git{})

    result := git.Run(ctx, Clone{
        Branch: "main",
        Depth:  1,
        Repo:   "https://github.com/acme/project.git",
        Dir:    "./project",
    })

    if result.Err != nil {
        fmt.Println(string(result.Stderr))
        panic(result.Err)
    }

    fmt.Println(string(result.Stdout))
}
```

This renders and executes:

```sh
git clone --branch main --depth 1 https://github.com/acme/project.git ./project
```

## Representative examples

### Nested commands

Some tools use multi-level command trees. Glove models each level as a command frame.

```go
type Docker struct {
    Bin     glove.Bin `bin:"docker" version:"--version"`
    Compose Compose   `cmd:"compose"`
}

type Compose struct {
    File string `flag:"--file,-f"`
    Up   Up     `cmd:"up"`
}

type Up struct {
    Detach  bool   `flag:"--detach,-d"`
    Service string `arg:"0"`
}
```

Call it like this:

```go
docker := glove.New(Docker{})

result := docker.Run(ctx,
    Compose{File: "compose.dev.yml"},
    Up{Detach: true, Service: "api"},
)
```

Constructed command:

```sh
docker compose --file compose.dev.yml up --detach api
```

### Root-level flags

Some tools accept flags before the subcommand.

```go
type Docker struct {
    Bin     glove.Bin `bin:"docker" version:"--version"`
    Context string    `flag:"--context"`
    Compose Compose   `cmd:"compose"`
}
```

Usage:

```go
docker := glove.New(Docker{})

result := docker.Run(ctx,
    Docker{Context: "prod"},
    Compose{File: "compose.prod.yml"},
    Up{Detach: true, Service: "api"},
)
```

Constructed command:

```sh
docker --context prod compose --file compose.prod.yml up --detach api
```

Depending on final API design, root configuration may also be supplied through the driver instance instead of as a frame.

### Repeated flags

Repeated flags are represented with slices.

```go
type Run struct {
    Env     map[string]string `flag:"--env,-e,repeat,kv"`
    Volume  []string          `flag:"--volume,-v,repeat"`
    Image   string            `arg:"0,required"`
    Command []string          `arg:"1,tail"`
}
```

Usage:

```go
result := docker.Run(ctx, Run{
    Env: map[string]string{
        "APP_ENV": "dev",
        "DEBUG":   "1",
    },
    Volume: []string{
        "/host/app:/app",
        "/host/cache:/cache",
    },
    Image: "golang:1.24",
    Command: []string{
        "go", "test", "./...",
    },
})
```

Constructed command:

```sh
docker run --env APP_ENV=dev --env DEBUG=1 --volume /host/app:/app --volume /host/cache:/cache golang:1.24 go test ./...
```

### Optional and required positional arguments

```go
type Clone struct {
    Repo string `arg:"0,required"`
    Dir  string `arg:"1"`
}
```

If `Repo` is empty, Glove returns a validation error before executing the command.

If `Dir` is empty, it is omitted.

### Tail arguments

Tail arguments are useful when the command forwards raw arguments to another process.

```go
type Exec struct {
    Interactive bool     `flag:"--interactive,-i"`
    TTY         bool     `flag:"--tty,-t"`
    Container   string   `arg:"0,required"`
    Command     []string `arg:"1,tail"`
}
```

Usage:

```go
result := docker.Run(ctx, Exec{
    Interactive: true,
    TTY:         true,
    Container:   "app",
    Command:     []string{"bash", "-lc", "go test ./..."},
})
```

Constructed command:

```sh
docker exec --interactive --tty app bash -lc "go test ./..."
```

### Binary resolution

Glove can resolve the target binary from the driver declaration:

```go
type Git struct {
    Bin glove.Bin `bin:"git" version:"--version"`
}
```

By default, Glove resolves `git` through the current `PATH`.

You can override the binary path for controlled environments:

```go
git := glove.New(
    Git{},
    glove.WithBin("/usr/local/bin/git"),
)
```

This is useful for hermetic builds, CI jobs, portable toolchains, and tests.

### Version detection and compatibility

External CLIs are not stable APIs. Glove treats version and capability detection as first-class driver concerns.

```go
type Git struct {
    Bin glove.Bin `bin:"git" version:"--version"`
}

type Clone struct {
    Filter string `flag:"--filter" since:"2.19"`
    Repo   string `arg:"0,required"`
}
```

In strict compatibility mode, Glove can fail before execution if the installed binary does not support a requested flag.

```go
git := glove.New(
    Git{},
    glove.WithCompatibility(glove.Strict),
)
```

Compatibility modes may include:

```go
const (
    Strict CompatibilityMode = iota
    BestEffort
    IgnoreCompatibility
)
```

The exact compatibility API is still subject to design, but the goal is simple: make external tool drift visible and controllable.

### Dry runs and command inspection

Drivers should be easy to debug. Glove can expose rendered commands before execution.

```go
cmd, err := git.Command(Clone{
    Branch: "main",
    Repo:   "https://github.com/acme/project.git",
})

fmt.Println(cmd.String())
```

Example output:

```sh
git clone --branch main https://github.com/acme/project.git
```

This is useful for logs, tests, dry-runs, documentation, and reproducible bug reports.

### Testing with a fake runner

Glove separates command rendering from command execution.

```go
type FakeRunner struct {
    Commands []glove.Command
}

func (r *FakeRunner) Run(ctx context.Context, cmd glove.Command) glove.Result {
    r.Commands = append(r.Commands, cmd)
    return glove.Result{Stdout: []byte("ok")}
}
```

Usage:

```go
runner := &FakeRunner{}

git := glove.New(
    Git{},
    glove.WithRunner(runner),
)

_ = git.Run(ctx, Clone{
    Branch: "main",
    Repo:   "https://github.com/acme/project.git",
})

// Assert against runner.Commands[0]
```

This lets tests verify command construction without spawning real processes.

## Tag reference

### `bin`

Declares the default binary name for a tool.

```go
Bin glove.Bin `bin:"git"`
```

### `version`

Declares the command used to detect the installed tool version.

```go
Bin glove.Bin `bin:"git" version:"--version"`
```

### `cmd`

Declares a subcommand field.

```go
Clone Clone `cmd:"clone"`
```

### `flag`

Declares a named CLI flag.

```go
Branch string `flag:"--branch,-b"`
Quiet  bool   `flag:"--quiet,-q"`
```

Expected default rendering:

```text
bool true  -> --flag
string     -> --flag value
int        -> --flag value
slice      -> --flag value1 --flag value2
map kv     -> --flag key=value
```

### `arg`

Declares a positional argument.

```go
Repo string `arg:"0,required"`
Dir  string `arg:"1"`
```

### `tail`

Declares trailing passthrough arguments.

```go
Command []string `arg:"1,tail"`
```

## Design principles

### External CLIs are porcelain

Glove does not pretend that external command-line tools are stable APIs.
It gives you a typed adapter layer while preserving escape hatches for raw arguments,
version checks, compatibility policy, and direct command inspection.

### Typed where useful, raw where necessary

The happy path should be typed and ergonomic. The weird path should still be possible.

```go
type Clone struct {
    Branch    string   `flag:"--branch,-b"`
    Repo      string   `arg:"0,required"`
    ExtraArgs []string `arg:"1,tail"`
}
```

### Reflection first, codegen later

The initial API should be useful without generation steps.

Later, Glove may generate strongly typed wrappers from the same command tree declarations:

```go
docker.Compose(Compose{File: "compose.yml"}).Up(Up{Detach: true}).Run(ctx)
```

The reflection API should remain the foundation, with codegen as an optional ergonomic layer.

### Testability is part of the abstraction

A command driver that cannot be tested without spawning real tools is not much of a driver.
Glove should make rendered commands observable and execution replaceable.

## Prior art and non-goals

Glove is related to CLI parsing libraries in spirit, but it solves the inverse problem.

CLI frameworks help you build a command-line interface for your own program.

Glove helps your Go program drive external command-line interfaces as typed dependencies.

Non-goals:

* replacing `os/exec`
* becoming a shell
* parsing arbitrary shell strings
* hiding subprocess behavior
* pretending external tools are stable SDKs
* modeling every possible CLI convention on day one

## Roadmap

Possible early milestones:

* command frame rendering
* flag and positional argument tags
* nested command validation
* binary resolution
* injectable runners
* dry-run command inspection
* version detection
* compatibility constraints
* structured stderr/stdout result handling
* optional code generation

## License

Apache 2.0

Glove is maintained within the Qlustra Engineering Tooling Lab.
https://github.com/Qlustra
