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
	Dir    string
	Env    map[string]string
	Runner Runner
}
```

Fields:

- `Dir`: working directory
- `Env`: extra environment variables appended to the current environment
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
- sorts environment keys for deterministic output
- captures stdout and stderr into memory
- rejects nil context and empty command program
