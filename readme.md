# Glove

Typed drivers for external commands in Go.

Glove helps you wrap command-line tools with small, ergonomic Go drivers.
Instead of assembling command calls from loose strings, maps, and fragile argument slices, you describe the external
tool as a typed command surface and let Glove render, validate, resolve, and execute the command call.

It is not a CLI framework for building your own command-line app.
It is the other side of that equation: a library for driving
existing tools like `git`, `docker`, `kubectl`, `terraform`, or internal company binaries from Go code.

```go
type Git struct {
	Bin   glove.Bin `bin:"git"`
	Clone Clone     `cmd:"clone"`
}

type Clone struct {
	Branch string `flag:"--branch,-b"`
	Depth  int    `flag:"--depth"`
	Repo   string `arg:"0,required"`
	Dir    string `arg:"1"`
}
git := glove.New(Git{})

cmd, err := git.Command(Clone{
	Branch: "main",
	Depth:  1,
	Repo:   "https://github.com/acme/project.git",
	Dir:    "./project",
})
if err != nil {
	panic(err)
}

fmt.Println(cmd.String())
```

Constructed command:

```sh
git clone --branch main --depth 1 https://github.com/acme/project.git ./project
```

Glove gives external CLIs a typed control surface without hiding the fact that they are still CLIs.

## Installation

```bash
go get github.com/qlustra/glove
```

## Current scope

The current implementation is a deliberately small first pass:

- reflection-first driver compilation
- typed command trees using structs and tags
- explicit command frames for nested commands
- command inspection through `Driver.Command(...)`
- execution options through `RunOptions`
- default execution through `ExecRunner`
- injectable runners for tests and custom process control
- binary override through `CompileOptions`

Not implemented yet:

- version detection
- compatibility rules
- code generation

Glove currently targets conventional command-tree CLIs. Within a frame, flags render before positional arguments.
Order-sensitive token streams such as `ffmpeg` are not a good fit for the current model.

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/qlustra/glove"
)

type Git struct {
	Bin   glove.Bin `bin:"git"`
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

## Compile and run

Use `Compile(...)` when schema errors should be handled explicitly:

```go
git, err := glove.Compile(Git{})
if err != nil {
	panic(err)
}
```

Use `New(...)` when an invalid driver declaration should be treated as a programmer error:

```go
git := glove.New(Git{})
```

Both accept an optional `glove.CompileOptions`:

```go
git := glove.New(Git{}, glove.CompileOptions{
	Bin: "/usr/local/bin/git",
})
```

Execution settings are supplied separately through `glove.RunOptions`:

```go
opts := glove.RunOptions{
	Dir: "/workspace/repo",
	Env: map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
	},
}
```

You can pass them explicitly with `RunWith(...)` or use the shorter `With(...)` alias.

## Representative examples

### Nested commands

```go
type Docker struct {
	Bin     glove.Bin `bin:"docker"`
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

Usage:

```go
docker := glove.New(Docker{})

cmd, err := docker.Command(
	Compose{File: "compose.dev.yml"},
	Up{Detach: true, Service: "api"},
)
```

Constructed command:

```sh
docker compose --file compose.dev.yml up --detach api
```

### Root-level flags

Root fields with `flag` tags render before the first subcommand when you pass a root frame explicitly:

```go
type Docker struct {
	Bin     glove.Bin `bin:"docker"`
	Context string    `flag:"--context"`
	Compose Compose   `cmd:"compose"`
}
```

Usage:

```go
docker := glove.New(Docker{})

cmd, err := docker.Command(
	Docker{Context: "prod"},
	Compose{File: "compose.prod.yml"},
	Up{Detach: true, Service: "api"},
)
```

Constructed command:

```sh
docker --context prod compose --file compose.prod.yml up --detach api
```

### Repeated flags and tail args

Repeated flags are inferred from slices and maps. Tail args are declared with `arg:"n,tail"`.

```go
type Docker struct {
	Bin glove.Bin `bin:"docker"`
	Run Run       `cmd:"run"`
}

type Run struct {
	Env     map[string]string `flag:"--env,-e"`
	Volume  []string          `flag:"--volume,-v"`
	Image   string            `arg:"0,required"`
	Command []string          `arg:"1,tail"`
}
```

Usage:

```go
docker := glove.New(Docker{})

cmd, err := docker.Command(Run{
	Env: map[string]string{
		"APP_ENV": "dev",
		"DEBUG":   "1",
	},
	Volume: []string{
		"/host/app:/app",
		"/host/cache:/cache",
	},
	Image:   "golang:1.24",
	Command: []string{"go", "test", "./..."},
})
```

Constructed command:

```sh
docker run --env APP_ENV=dev --env DEBUG=1 --volume /host/app:/app --volume /host/cache:/cache golang:1.24 go test ./...
```

Map flags render in sorted key order for deterministic output.

### Command inspection and manual execution

`Driver.Command(...)` renders a `glove.Command` without executing it:

```go
cmd, err := git.Command(Clone{
	Branch: "main",
	Repo:   "https://github.com/acme/project.git",
})
if err != nil {
	panic(err)
}

result := cmd.RunWith(ctx, glove.RunOptions{
	Dir: "/tmp/work",
	Env: map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
	},
})
```

### Testing with a fake runner

```go
type FakeRunner struct {
	Commands []glove.Command
}

func (r *FakeRunner) Run(ctx context.Context, cmd glove.Command, opts glove.RunOptions) glove.Result {
	r.Commands = append(r.Commands, cmd)
	return glove.Result{Command: cmd, Stdout: []byte("ok")}
}
```

Usage:

```go
runner := &FakeRunner{}

git := glove.New(Git{}, glove.CompileOptions{
	Runner: runner,
})

_ = git.Run(ctx, Clone{
	Branch: "main",
	Repo:   "https://github.com/acme/project.git",
})
```

## Exported surface

```go
type Bin struct{}

type CompileOptions struct {
	Bin    string
	Runner Runner
}

type RunOptions struct {
	Dir    string
	Env    map[string]string
	Runner Runner
}

type Driver[T any] struct{ /* opaque */ }

func Compile[T any](decl T, options ...CompileOptions) (Driver[T], error)
func New[T any](decl T, options ...CompileOptions) Driver[T]

func (d Driver[T]) Command(frames ...any) (Command, error)
func (d Driver[T]) Run(ctx context.Context, frames ...any) Result
func (d Driver[T]) RunWith(ctx context.Context, options RunOptions, frames ...any) Result
func (d Driver[T]) With(ctx context.Context, options RunOptions, frames ...any) Result
```

Rendered commands and execution results use:

```go
type Command struct {
	Program string
	Args    []string
}

func (c Command) Argv() []string
func (c Command) String() string
func (c Command) Run(ctx context.Context) Result
func (c Command) RunWith(ctx context.Context, options RunOptions) Result
func (c Command) With(ctx context.Context, options RunOptions) Result

type Result struct {
	Command Command
	Stdout  []byte
	Stderr  []byte
	Err     error
}

type Runner interface {
	Run(context.Context, Command) Result
}
```

## Tag reference

### `bin`

Declares the default binary name for the driver root. It must be used on a `glove.Bin` field.

```go
Bin glove.Bin `bin:"git"`
```

### `cmd`

Declares a child command frame.

```go
Clone Clone `cmd:"clone"`
```

The field type must be a struct or pointer to struct.

### `flag`

Declares a named CLI flag.

```go
Branch string            `flag:"--branch,-b"`
Quiet  bool              `flag:"--quiet,-q"`
Env    map[string]string `flag:"--env,-e"`
Volume []string          `flag:"--volume,-v"`
```

Supported field shapes:

- `bool` renders as `--flag` when `true`
- scalar values render as `--flag value`
- `*scalar` renders when non-nil, even if the pointed value is a zero value
- `[]scalar` repeats the flag once per item
- `map[string]string` repeats the flag as `--flag key=value`

Glove renders the first declared flag name. Additional names are aliases used for declaration and validation only.

### `arg`

Declares a positional argument.

```go
Repo string   `arg:"0,required"`
Dir  string   `arg:"1"`
Tail []string `arg:"2,tail"`
```

Supported field shapes:

- scalar values
- `*scalar`
- `[]string` with `tail`

## Validation and conventions

Glove validates driver declarations up front:

- tagged fields must be exported
- the root declaration must define exactly one `bin` field
- each frame type may appear only once in a driver
- sibling `cmd` names must be unique
- positional arguments must be contiguous starting at `0`
- `tail` positional arguments must be last

Glove also enforces a few rendering rules:

- intermediate frames must be passed explicitly
- within a frame, flags render before positional arguments
- omitted optional positional arguments cannot be followed by later positional arguments
- `Command` carries only the rendered program and argv; execution settings live in `RunOptions`

## Non-goals

Glove does not try to:

- replace `os/exec`
- parse arbitrary shell strings
- model every possible CLI convention
- hide subprocess behavior behind an SDK-like facade

## License

Apache 2.0

Glove is maintained within the Qlustra Engineering Tooling Lab.
https://github.com/Qlustra
