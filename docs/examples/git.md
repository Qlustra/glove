# Git example

This example shows the smallest useful Glove driver: one binary and one subcommand.

## Declaration

```go
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
```

## Render first

```go
git := glove.New(Git{})

cmd, err := git.Command(Clone{
	Branch: "main",
	Depth:  1,
	Repo:   "https://github.com/acme/project.git",
	Dir:    "./project",
})
if err != nil {
	panic(err)
}

fmt.Println(cmd.String())
```

Result:

```sh
git clone --branch main --depth 1 https://github.com/acme/project.git ./project
```

This pattern is useful when the caller wants to log, inspect, or defer execution.

## Execute with runtime options

```go
result := git.RunWith(ctx, glove.RunOptions{
	Dir: "/workspace",
	Env: map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
	},
}, Clone{
	Branch: "main",
	Repo:   "https://github.com/acme/project.git",
	Dir:    "./project",
})

if result.Err != nil {
	panic(result.Err)
}
```

This keeps process concerns out of the driver declaration.

## Override the binary

```go
git := glove.New(Git{}, glove.CompileOptions{
	Bin: "/usr/local/bin/git",
})
```

Use this when the declaration should stay tool-shaped while deployment decides the concrete executable path.
