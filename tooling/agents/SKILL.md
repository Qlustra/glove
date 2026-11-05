---
name: glove
description: Use this skill when working on the Glove Go library, designing or updating typed CLI drivers built with Glove, debugging `bin`/`cmd`/`flag`/`arg` declarations, changing render or execution behavior, or writing tests and docs for command-tree CLIs in this repository.
---

# Glove

Glove is a Go library for typed drivers over external commands. It models a conventional CLI as a declared command tree, compiles that tree once, then renders or runs explicit frame values.

Use this skill when the task involves:

- changing code in this repository
- adding or updating driver declarations, examples, or tests
- debugging compile-time schema errors or render-time frame/path errors
- reasoning about `Command`, `RunOptions`, `Runner`, or `ExecRunner`
- deciding whether a CLI is a good fit for Glove's model

Do not use Glove as if it were a shell parser or a general CLI framework. It is a poor fit for order-sensitive token streams such as `ffmpeg` or raw shell pipelines.

## Reference map

Load only the files relevant to the task.

- `README.md`: high-level library intent and quick start
- `docs/readme.md`: index into the docs set
- `docs/usage/model.md`: command-tree model, frame sequencing, good fit vs bad fit
- `docs/usage/tags.md`: exact tag grammar and supported field shapes
- `docs/usage/execution.md`: `Compile` vs `New`, `Command` vs `Run`, runtime options
- `docs/usage/validation.md`: compile-time and render-time validation rules
- `docs/examples/*.md`: example layouts for `git`, `docker`, and testing
- `glove_test.go`: behavioral coverage and the fastest way to confirm expected rendering
- `internal/driver/*.go`: source of truth for compilation and rendering rules
- `internal/command/*.go`: source of truth for `Command`, `RunOptions`, and `ExecRunner`

## Working model

Keep these invariants in mind when writing code or examples:

- The driver root declares exactly one `glove.Bin` field tagged with `bin:"..."`.
- Each `cmd` field is a child frame in a static command tree.
- Each value passed to `Driver.Command(...)`, `Run(...)`, or `RunWith(...)` is one frame in the selected path.
- Glove never infers missing intermediate commands.
- The root frame is optional at render time and should only be passed when root-level flags or args must render.
- Within a frame, flags always render before positional args.
- Rendering is deterministic by design. This matters for tests, examples, and API changes.

The mental model is `os/exec` plus a declared schema, not a higher-level orchestration system.

## Declaration rules

Glove exposes four tags: `bin`, `cmd`, `flag`, and `arg`.

### `bin`

- Allowed only on the driver root.
- Field type must be `glove.Bin`.
- There must be exactly one.
- `CompileOptions{Bin: ...}` may override the declared binary path at compile time.

### `cmd`

- Field type must be a struct or pointer to struct.
- Sibling command names must be unique.
- A frame type may appear only once in the entire driver tree.
- Every later frame in a render path must be a direct child of the previous frame.

### `flag`

Use `flag:"name[,alias...][,option...]"`.

Supported shapes:

- `bool`: render the flag token only when `true`
- scalar values: render `--flag value` when non-zero
- pointers to supported scalars: render when non-nil, including pointed zero values
- slices of supported scalars: repeat the flag once per item
- `map[string]string`: repeat as `--flag key=value`, sorted by key

Supported scalar forms:

- `string`
- signed integers
- unsigned integers
- `float32`, `float64`
- types implementing `encoding.TextMarshaler`
- types implementing `fmt.Stringer`

Flag options:

- `repeat`: valid only for slices and maps
- `list`: valid only for slice flags; renders one flag with a space-joined value
- `kv`: valid only for `map[string]string`
- `required`: valid for any flag shape; fails rendering when the flag would otherwise be omitted

Important constraints:

- pointer `bool` flags are rejected
- slice `bool` flags are rejected
- `repeat` and `list` cannot be combined
- flag names and aliases must be unique within one frame
- the first declared flag name is the rendered token; later names are aliases for validation only
- a required bool must be `true`, a required scalar must be non-zero, a required pointer must be non-nil, and a required slice or map must be non-empty
- use a pointer when a required flag needs to render a meaningful zero value

Use `list` for Go toolchain-style flags such as `-ldflags`, `-gcflags`, or `-asmflags` where one flag occurrence carries a joined value.

### `arg`

Use `arg:"position[,required][,tail]"`.

Supported shapes:

- scalar values
- pointers to supported scalars
- `[]string` with `tail`

Important constraints:

- positions are zero-based and must be contiguous starting at `0`
- `required` means the position must be present at render time
- `tail` is allowed only on `[]string`
- a `tail` arg must be last
- `bool` positional args are rejected
- pointer `bool` positional args are rejected
- once an optional positional arg is omitted, no later positional arg may be set
- a required positional arg may not follow an optional positional arg in the schema

## Execution model

Use the exported API according to the task:

- `glove.Compile(...)`: prefer when schema errors should be returned explicitly
- `glove.New(...)`: prefer when an invalid schema is a programmer error and panic is acceptable
- `Driver.Command(...)`: render without executing; use this for inspection, logging, tests, and batching
- `Driver.Run(...)`: render and execute with the driver's default runner
- `Driver.RunWith(...)` / `With(...)`: render and execute with explicit `RunOptions`

`Compile(...)` and `New(...)` accept at most one `CompileOptions` value.

### `RunOptions`

Runtime concerns belong in `RunOptions`, not in the driver declaration:

- `Dir`: child working directory
- `Env`: appended to `os.Environ()`, sorted by key for deterministic output
- `Stdin`: child stdin
- `Stdout`, `Stderr`: streaming sinks
- `DisableBuffering`: disables capture into `Result.Stdout` and `Result.Stderr`
- `Runner`: per-call runner override

### Runner behavior

Default behavior comes from `ExecRunner`:

- requires a non-nil context
- requires a non-empty command program
- captures stdout and stderr by default
- mirrors output to `Stdout` and `Stderr` when provided
- forwards only, without capture, when `DisableBuffering` is `true`

When render fails in `Driver.Run(...)` or `RunWith(...)`, Glove still returns a `Result`; `Result.Err` is set and `Result.Command` contains only the base program when available.

## Testing patterns

Prefer the smallest test that proves the contract.

### Rendering tests

Use `Driver.Command(...)` when the behavior under test is:

- frame sequencing
- flag ordering
- positional validation
- deterministic map or environment sorting
- shell-quoted output via `Command.String()`

Compare `cmd.Args` when asserting exact argv shape. Use `cmd.String()` only when the string form itself matters.

### Execution tests

Inject a custom `Runner` when the behavior under test is:

- which command would have been executed
- how `RunOptions` are passed through
- error propagation from the runner
- precedence between driver-level and per-call runners

`glove_test.go` already demonstrates the preferred patterns for capture runners, nested frames, list flags, positional-gap failures, and compile-time rejections.

## Change guidance for this repo

When implementing or updating Glove behavior:

- preserve the separation between declaration, rendering, and execution
- keep rendering deterministic unless the change intentionally alters that contract
- prefer explicit validation over implicit inference
- update docs when public behavior or constraints change
- add or adjust tests in `glove_test.go` for every exported behavior change

If you change tag semantics, field support, ordering, or validation, expect to touch:

- `docs/usage/tags.md`
- `docs/usage/validation.md`
- `docs/usage/model.md` or `docs/usage/execution.md` when behavior changes are conceptual
- `docs/examples/*.md` when example output changes
- `docs/api/*.md` when exported API behavior changes

## Workflow for common tasks

### Adding or changing a driver example

1. Read the closest example in `docs/examples/`.
2. Match the command tree explicitly with root and child frames.
3. Use tags that preserve conventional CLI ordering: command token, flags, then positional args.
4. Render with `Driver.Command(...)` first and verify `cmd.Args` or `cmd.String()`.

### Debugging a schema or render failure

1. Check `docs/usage/validation.md` for the rule class.
2. Inspect the declaration against `docs/usage/tags.md`.
3. If the failure is path-related, confirm every frame after the first is a direct child of the previous frame.
4. If the failure is positional, look for omitted optional args before later positions.
5. If the docs are ambiguous, confirm exact behavior in `internal/driver/compile.go` or `internal/driver/render.go`.

### Extending library behavior

1. Confirm the change still fits Glove's command-tree model.
2. Update compile/render logic in `internal/driver/` or execution logic in `internal/command/`.
3. Add focused tests before broad examples.
4. Update the relevant docs pages in the same change.
5. Run `go test ./...`.

## Validation

After changing code or docs in this repo, run:

```bash
go test ./...
```

For doc-only or skill-only changes, still sanity-check examples and terminology against:

- `README.md`
- `docs/usage/*.md`
- `glove_test.go`

If behavior and docs disagree, treat the tests and implementation as the source of truth, then bring the docs back into sync.
