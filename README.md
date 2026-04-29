# Glove

Typed drivers for external commands in Go.

Glove lets you describe an external CLI as a validated Go command schema, then render and run commands through a small typed driver. Think: `os/exec` with declared command structure instead of ad hoc argv assembly.

## What it is

- a Go library for driving existing tools such as `git`, `docker`, `kubectl`, `terraform`, or internal binaries
- typed command trees declared with `bin`, `cmd`, `flag`, and `arg` tags
- explicit command rendering through `Driver.Command(...)` before execution
- pluggable execution through `RunOptions` and custom `Runner` implementations
- not a CLI framework, shell parser, or a fit for order-sensitive token streams such as `ffmpeg`

## Why it exists

Code that shells out to external tools usually drifts toward loose strings, hand-built argv slices, and duplicated validation. That is workable for one command and brittle once a project starts driving a real CLI surface.

Glove keeps the subprocess model explicit while giving it structure:

- command names live in one declared driver tree
- flag and positional rules are validated up front
- rendering and execution stay separate
- tests can inspect commands without spawning processes

See [Usage: model](docs/usage/model.md) for the operational model behind the library.

## Core idea

- a driver root declares one binary and the subcommands reachable from it
- each frame value supplies the flags and args for one command level
- intermediate frames are explicit, so the command path is never inferred
- within a frame, flags render before positional arguments
- compilation validates the declaration once; runs reuse the compiled schema

Glove is for Go projects that need a typed control surface over conventional command-tree CLIs.

## Install

```bash
go get github.com/qlustra/glove
```

## Quick start

```go
package main

import (
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
	git := glove.New(Git{})

	cmd, err := git.Command(Clone{ // build command, no execution
		Branch: "main",
		Depth:  1,
		Repo:   "https://github.com/acme/project.git",
		Dir:    "./project",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(cmd.String())
}
```

This renders to:

```sh
git clone --branch main --depth 1 https://github.com/acme/project.git ./project
```

Use `Run(...)` and `RunWith(...)` when you want Glove to execute the rendered command through a runner.

## Documentation

- Full docs: [Docs index](docs/readme.md)
- Getting started: [Usage guides](docs/usage/model.md)
- API: [API reference](docs/api/readme.md)
- Examples: [Git example](docs/examples/git.md)

## Status

Early-stage / experimental

## Development

```bash
go test ./...
```

## License

Apache 2.0

Conduit is maintained within the Qlustra Engineering Tooling Lab. https://github.com/Qlustra
