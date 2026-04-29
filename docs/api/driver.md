# Driver API

This page documents the exported declaration and driver surface.

## Types

### `Bin`

```go
type Bin struct{}
```

Marker type used on the driver root with a `bin:"..."` tag.

### `CompileOptions`

```go
type CompileOptions struct {
	Bin    string
	Runner Runner
}
```

Fields:

- `Bin`: overrides the declared binary name for the compiled driver
- `Runner`: installs the default runner used by `Driver.Run(...)`

Notable behavior:

- Glove accepts at most one `CompileOptions` value per `Compile(...)` or `New(...)` call

### `Driver[T]`

```go
type Driver[T any]
```

Opaque compiled driver for a declared command tree.

## Functions

### `Compile`

```go
func Compile[T any](decl T, options ...CompileOptions) (Driver[T], error)
```

Description:

- validates the driver declaration and returns a compiled reusable driver

Arguments:

- `decl`: root struct or pointer to struct describing the command tree
- `options`: optional compile-time overrides

Notable behavior:

- returns an error instead of panicking on invalid declarations
- installs `ExecRunner` when no default runner is provided

### `New`

```go
func New[T any](decl T, options ...CompileOptions) Driver[T]
```

Description:

- convenience wrapper around `Compile(...)`

Notable behavior:

- panics when the declaration is invalid

## Methods

### `Driver.Command`

```go
func (d Driver[T]) Command(frames ...any) (Command, error)
```

Description:

- renders a command without executing it

Arguments:

- `frames`: zero or more root/child frame values forming one path through the driver tree

Notable behavior:

- the first frame may be the root to render root-level flags or args
- later frames must be direct children of the current frame
- returns an error for invalid frame paths or invalid positional values

### `Driver.Run`

```go
func (d Driver[T]) Run(ctx context.Context, frames ...any) Result
```

Description:

- renders and executes a command using the driver's default runner

Notable behavior:

- equivalent to `RunWith(ctx, glove.RunOptions{}, frames...)`
- render failures are reported in `Result.Err`

### `Driver.RunWith`

```go
func (d Driver[T]) RunWith(ctx context.Context, options RunOptions, frames ...any) Result
```

Description:

- renders and executes a command using explicit runtime options

Notable behavior:

- `options.Runner` overrides the driver's default runner for that call

### `Driver.With`

```go
func (d Driver[T]) With(ctx context.Context, options RunOptions, frames ...any) Result
```

Description:

- short alias for `RunWith(...)`
