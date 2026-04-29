# Docker example

This example shows nested commands, root-level flags, and repeated flags.

## Nested command tree

```go
type Docker struct {
	Bin     glove.Bin `bin:"docker"`
	Context string    `flag:"--context"`
	Compose Compose   `cmd:"compose"`
	Run     Run       `cmd:"run"`
}

type Compose struct {
	File string `flag:"--file,-f"`
	Up   Up     `cmd:"up"`
}

type Up struct {
	Detach  bool   `flag:"--detach,-d"`
	Service string `arg:"0"`
}
```

Render a nested path:

```go
docker := glove.New(Docker{})

cmd, err := docker.Command(
	Docker{Context: "prod"},
	Compose{File: "compose.prod.yml"},
	Up{Detach: true, Service: "api"},
)
if err != nil {
	panic(err)
}
```

Result:

```sh
docker --context prod compose --file compose.prod.yml up --detach api
```

The root frame is included explicitly because it carries the `--context` flag.

## Repeated flags and tail args

```go
type Run struct {
	Env     map[string]string `flag:"--env,-e,repeat,kv"`
	Volume  []string          `flag:"--volume,-v,repeat"`
	Image   string            `arg:"0,required"`
	Command []string          `arg:"1,tail"`
}
```

Render it:

```go
cmd, err := docker.Command(Run{
	Env: map[string]string{
		"APP_ENV": "dev",
		"DEBUG":   "1",
	},
	Volume: []string{
		"/host/app:/app",
		"/host/cache:/cache",
	},
	Image:   "golang:1.24",
	Command: []string{"go", "test", "./..."},
})
if err != nil {
	panic(err)
}
```

Result:

```sh
docker run --env APP_ENV=dev --env DEBUG=1 --volume /host/app:/app --volume /host/cache:/cache golang:1.24 go test ./...
```

Map-backed flags are sorted by key before rendering so command output stays deterministic.
