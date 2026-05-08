# Command API

This page documents the rendered command and execution surface.

## Types

### `Command`

```go
type Command struct {
	Program string
	Args    []string
}
```

Fields:

- `Program`: the executable path or binary name
- `Args`: argv items after the program

Methods:

```go
func (c Command) Argv() []string
func (c Command) String() string
func (c Command) Run(ctx context.Context) Result
func (c Command) RunWith(ctx context.Context, options RunOptions) Result
func (c Command) With(ctx context.Context, options RunOptions) Result
```

Notable behavior:

- `Argv()` returns `nil` only when both `Program` and `Args` are empty
- `String()` shell-quotes tokens for inspection and logging
- `RunWith(...)` injects the rendered command into the returned `Result`

### `RunOptions`

```go
type RunOptions struct {
	Dir              string
	Env              map[string]string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	DisableBuffering bool
	Runner           Runner
}
```

Fields:

- `Dir`: working directory
- `Env`: extra environment variables appended to the current environment
- `Stdin`: optional reader connected to the child process stdin
- `Stdout`: optional writer that receives stdout as it is produced
- `Stderr`: optional writer that receives stderr as it is produced
- `DisableBuffering`: when true, skips in-memory stdout/stderr capture
- `Runner`: per-call runner override

### `Result`

```go
type Result struct {
	Command Command
	Stdout  []byte
	Stderr  []byte
	Err     error
}
```

Description:

- contains the rendered command, captured output, and execution error

### `Runner`

```go
type Runner interface {
	Run(context.Context, Command, RunOptions) Result
}
```

Description:

- pluggable execution contract used by driver and command runs

### `ExecRunner`

```go
type ExecRunner struct{}
```

Methods:

```go
func (ExecRunner) Run(ctx context.Context, cmd Command, options RunOptions) Result
```

Notable behavior:

- delegates to `exec.CommandContext`
- applies `RunOptions.Dir` to the child process
- appends `RunOptions.Env` to `os.Environ()`
- connects `RunOptions.Stdin` to the child process stdin when provided
- mirrors stdout and stderr to `RunOptions.Stdout` and `RunOptions.Stderr` when provided
- skips `Result.Stdout` and `Result.Stderr` capture when `RunOptions.DisableBuffering` is true
- sorts environment keys for deterministic output
- captures stdout and stderr into memory by default
- rejects nil context and empty command program
