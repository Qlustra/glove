# Glove model

Glove treats an external CLI as a declared command tree. You describe that tree with plain Go structs, compile it once, then pass frame values that render one path through the tree.

## Declaring a driver

The driver root declares the binary and the reachable subcommands:

```go
type Git struct {
	Bin   glove.Bin `bin:"git"`
	Clone Clone     `cmd:"clone"`
}

type Clone struct {
	Branch string `flag:"--branch,-b"`
	Repo   string `arg:"0,required"`
	Dir    string `arg:"1"`
}
```

`glove.Compile(...)` validates that declaration and builds the reusable schema. `glove.New(...)` does the same thing, but panics instead of returning an error.

## Frames

Each struct value passed to `Driver.Command(...)`, `Run(...)`, or `RunWith(...)` is one frame in the command path:

```go
cmd, err := git.Command(Clone{
	Branch: "main",
	Repo:   "https://example.com/repo.git",
	Dir:    "./repo",
})
```

Nested commands use multiple frames in order:

```go
cmd, err := docker.Command(
	Docker{Context: "prod"},
	Compose{File: "compose.prod.yml"},
	Up{Detach: true, Service: "api"},
)
```

The key rule is explicitness:

- Glove does not infer missing intermediate commands
- the first frame may be the root only when you want root-level flags or args rendered
- every later frame must be a direct child of the previous frame

## Rendering model

For a valid frame sequence, Glove renders:

1. the configured program name
2. each command token in the selected path
3. each frame's flags
4. each frame's positional arguments

Within one frame, flags always render before positional arguments. That makes the output deterministic, but it also means Glove is designed for conventional command-tree CLIs rather than tools that need fully custom token ordering.

## Good fit

Glove works well when:

- a binary has a stable command tree
- flags and positional arguments can be modeled declaratively
- you want reusable command rendering in library code
- you need testable subprocess orchestration without stringly typed helpers

## Poor fit

Glove is a poor fit when:

- token order must be interleaved manually
- the CLI surface is better expressed as a raw shell pipeline
- you need a framework for building your own end-user CLI

Order-sensitive tools such as `ffmpeg` are the clearest example of a bad fit for the current model.

## What stays explicit

Glove adds structure, not abstraction over subprocess reality:

- binaries are still external processes
- rendered commands are still visible as argv
- working directory and environment are runtime concerns in `RunOptions`
- execution is still swappable through a `Runner`

Use the [tags guide](tags.md) for the declaration grammar and [execution](execution.md) for the runtime model.
