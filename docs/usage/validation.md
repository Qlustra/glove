# Validation

Glove validates both the declared driver and the frame values supplied at render time. The goal is to fail on malformed command schemas early and on invalid command paths precisely.

## Declaration validation

Compilation rejects invalid driver schemas before any command is rendered.

Rules enforced at compile time:

- tagged fields must be exported
- a field may not use more than one Glove tag
- the root must define exactly one `glove.Bin` field with a `bin` tag
- `bin` may not appear outside the root
- sibling `cmd` names must be unique
- a frame type may appear only once in the full driver tree
- flag names and aliases must be unique within one frame
- positional args must be contiguous starting at `0`
- a `tail` arg must be last
- a required positional arg may not follow an optional positional arg
- unsupported field shapes for `flag` and `arg` are rejected

Representative compile failures:

- missing `Bin` field on the root
- `map[int]string` used as a flag field
- `bool` used as a positional arg
- reused command names or frame types

## Frame validation

Rendering validates the command path formed by the supplied frames.

Rules enforced at render time:

- each frame must be a struct or pointer to struct
- frames must not be nil
- a child frame must directly follow its parent frame
- a frame type from another branch of the same driver is rejected
- a frame type not declared anywhere in the driver is rejected

This is why Glove requires explicit intermediate frames for nested commands.

## Positional validation

Glove also validates the frame values being rendered:

- missing required positional args fail
- once an optional positional arg is omitted, no later positional arg may be set
- tail args render only when the slice is non-empty

Example of an invalid positional gap:

```go
type Gap struct {
	Bin   glove.Bin `bin:"tool"`
	First string    `arg:"0,required"`
	Next  string    `arg:"1"`
	Last  *string   `arg:"2"`
}
```

Rendering `Gap{First: "one", Last: &last}` fails because position `1` was omitted.

## Run-time errors

`Driver.Run(...)` and `RunWith(...)` return a `glove.Result` even when rendering fails. In that case:

- `Result.Err` contains the failure
- `Result.Command` contains only the base command program when available

`ExecRunner` adds its own runtime checks:

- nil context is rejected
- empty command program is rejected
- process execution errors are returned in `Result.Err`

Use `Driver.Command(...)` directly when the caller wants to handle render failures before choosing whether to execute.
