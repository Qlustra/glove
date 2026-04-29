# Tag guide

Glove exposes four declaration tags: `bin`, `cmd`, `flag`, and `arg`. They define the command tree and how values are rendered.

## `bin`

`bin` declares the default program name for the driver root.

```go
Bin glove.Bin `bin:"git"`
```

Rules:

- the root must declare exactly one `bin` field
- the field type must be `glove.Bin`
- `bin` may only appear on the driver root

The declared binary can be overridden later with `glove.CompileOptions{Bin: ...}`.

## `cmd`

`cmd` declares a child command frame:

```go
Clone Clone `cmd:"clone"`
```

Rules:

- the field type must be a struct or pointer to struct
- sibling command names must be unique
- a frame type may only appear once in a driver

The field type becomes the frame value you pass when rendering that command.

## `flag`

`flag` declares a named CLI flag:

```go
Branch string            `flag:"--branch,-b"`
Quiet  bool              `flag:"--quiet,-q"`
Env    map[string]string `flag:"--env,-e,repeat,kv"`
Volume []string          `flag:"--volume,-v,repeat"`
```

The first token is the rendered flag name. Additional flag names are aliases for declaration and validation.

Supported shapes:

- `bool`: render the flag name when `true`
- scalar values such as `string`, integers, and floats: render `--flag value` when non-zero
- pointers to supported scalars: render when non-nil, including pointed zero values
- slices of supported scalars: repeat the flag once per item
- `map[string]string`: repeat the flag as `--flag key=value`, sorted by key

Additional notes:

- `repeat` is accepted only on slice and map flags
- `kv` is accepted only on `map[string]string`
- repeated rendering is inferred from the field type; the extra tokens mainly make the declaration explicit
- pointer `bool` and slice `bool` flags are not supported

## `arg`

`arg` declares a positional argument:

```go
Repo string   `arg:"0,required"`
Dir  string   `arg:"1"`
Tail []string `arg:"2,tail"`
```

The first token is the zero-based position. Supported options:

- `required`: the position must be present at render time
- `tail`: the field must be `[]string` and consumes the remaining argv tail

Supported shapes:

- scalar values such as `string`, integers, and floats
- pointers to supported scalars
- `[]string` with `tail`

`bool` positional args are not supported.

## Scalar conversion

Glove accepts these scalar forms in `flag` and `arg` fields:

- `string`
- signed integers
- unsigned integers
- `float32` and `float64`
- values implementing `encoding.TextMarshaler`
- values implementing `fmt.Stringer`

If the type does not match one of those shapes, compilation fails.

## Rendering rules

The tag grammar defines the output ordering:

- the program name comes from `bin` or `CompileOptions.Bin`
- child command tokens come from each `cmd` frame on the selected path
- flags render before positional args within the same frame
- optional positional gaps are not allowed once a later positional value is set

Use [validation](validation.md) for the full list of rejected declarations and render-time errors.
