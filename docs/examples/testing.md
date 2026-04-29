# Testing example

Glove is designed so tests can assert rendered commands without spawning subprocesses.

## Capture runner

```go
type CaptureRunner struct {
	Command glove.Command
	Options glove.RunOptions
	Result  glove.Result
	Called  bool
}

func (r *CaptureRunner) Run(_ context.Context, cmd glove.Command, opts glove.RunOptions) glove.Result {
	r.Called = true
	r.Command = cmd
	r.Options = opts
	return r.Result
}
```

## Install it on the driver

```go
runner := &CaptureRunner{
	Result: glove.Result{
		Stdout: []byte("ok"),
	},
}

git := glove.New(Git{}, glove.CompileOptions{
	Runner: runner,
})
```

## Assert rendered output

```go
result := git.Run(context.Background(), Clone{
	Branch: "main",
	Repo:   "https://example.com/repo.git",
})

if result.Err != nil {
	t.Fatalf("run: %v", result.Err)
}

want := []string{"clone", "--branch", "main", "https://example.com/repo.git"}
if !slices.Equal(runner.Command.Args, want) {
	t.Fatalf("args = %#v, want %#v", runner.Command.Args, want)
}
```

This pattern keeps the test focused on Glove's output contract.

## Override the runner for one call

Per-run overrides are useful when one test wants a different behavior than the driver's default:

```go
result := git.RunWith(context.Background(), glove.RunOptions{
	Dir:    "/workspace/repo",
	Runner: runner,
}, Clone{
	Repo: "https://example.com/repo.git",
})
```

`RunOptions.Runner` takes precedence over the driver's installed runner.
