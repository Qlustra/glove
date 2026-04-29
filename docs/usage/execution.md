# Execution

Glove separates declaration, rendering, and execution. You compile a schema once, render commands when needed, and choose whether a run should use the default runner or an injected override.

## Compile vs `New`

Use `glove.Compile(...)` when schema errors should be handled explicitly:

```go
driver, err := glove.Compile(Git{})
if err != nil {
	panic(err)
}
```

Use `glove.New(...)` when an invalid declaration is a programmer error:

```go
driver := glove.New(Git{})
```

Both accept at most one `glove.CompileOptions` value:

```go
driver := glove.New(Git{}, glove.CompileOptions{
	Bin: "/usr/local/bin/git",
})
```

`CompileOptions` controls:

- `Bin`: override the declared program name
- `Runner`: install a default runner for future `Run(...)` calls

## Render without executing

`Driver.Command(...)` returns a `glove.Command`:

```go
cmd, err := driver.Command(Clone{
	Branch: "main",
	Repo:   "https://example.com/repo.git",
})
```

Useful methods:

- `Argv()` returns the rendered argv slice including the program
- `String()` returns a shell-quoted command string for inspection or logging
- `Run(...)` and `RunWith(...)` execute that one rendered command

This is the right path when you want inspection, logging, batching, or custom process orchestration before execution.

## Run through the driver

`Driver.Run(...)` renders and executes in one step:

```go
result := driver.Run(ctx, Clone{
	Branch: "main",
	Repo:   "https://example.com/repo.git",
})
```

`Driver.RunWith(...)` adds execution settings:

```go
result := driver.RunWith(ctx, glove.RunOptions{
	Dir: "/workspace/repo",
	Env: map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
	},
}, Clone{
	Branch: "main",
	Repo:   "https://example.com/repo.git",
})
```

`Driver.With(...)` is a short alias for `RunWith(...)`.

## Run options

`glove.RunOptions` controls runtime concerns only:

```go
type RunOptions struct {
	Dir    string
	Env    map[string]string
	Runner glove.Runner
}
```

Behavior:

- `Dir` sets the working directory
- `Env` is appended to the current process environment
- `Runner` overrides the driver's default runner for one call

The environment map is sorted by key before execution so the generated environment slice is deterministic.

## Default runner

When no runner is provided, Glove uses `glove.ExecRunner`, which delegates to `os/exec`.

Important behavior:

- `ExecRunner` requires a non-nil context
- a command must have a non-empty program
- stdout and stderr are captured into `glove.Result`

## Custom runners

Inject a runner when you want tests, tracing, or custom process control:

```go
type CaptureRunner struct {
	Commands []glove.Command
}

func (r *CaptureRunner) Run(_ context.Context, cmd glove.Command, opts glove.RunOptions) glove.Result {
	r.Commands = append(r.Commands, cmd)
	return glove.Result{Command: cmd, Stdout: []byte("ok")}
}
```

Install it once:

```go
driver := glove.New(Git{}, glove.CompileOptions{
	Runner: runner,
})
```

Or override it for one run:

```go
result := driver.RunWith(ctx, glove.RunOptions{
	Runner: runner,
}, Clone{Repo: "https://example.com/repo.git"})
```

Use [testing examples](../examples/testing.md) for assertion patterns.
