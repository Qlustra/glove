package glove_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/qlustra/glove"
)

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

type Run struct {
	Env     map[string]string `flag:"--env,-e,repeat,kv"`
	Volume  []string          `flag:"--volume,-v,repeat"`
	Image   string            `arg:"0,required"`
	Command []string          `arg:"1,tail"`
}

type Git struct {
	Bin   glove.Bin `bin:"git"`
	Clone Clone     `cmd:"clone"`
}

type GoTool struct {
	Bin   glove.Bin `bin:"go"`
	Build Build     `cmd:"build"`
}

type Build struct {
	Output    string   `flag:"-o"`
	LinkFlags []string `flag:"-ldflags,list"`
	Packages  []string `arg:"0,tail"`
}

type Clone struct {
	Branch string `flag:"--branch,-b"`
	Repo   string `arg:"0,required"`
	Dir    string `arg:"1"`
}

func TestCommandRendersNestedFrames(t *testing.T) {
	driver, err := glove.Compile(Docker{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	cmd, err := driver.Command(
		Docker{Context: "prod"},
		Compose{File: "compose.prod.yml"},
		Up{Detach: true, Service: "api"},
	)
	if err != nil {
		t.Fatalf("render command: %v", err)
	}

	if cmd.Program != "docker" {
		t.Fatalf("program = %q, want docker", cmd.Program)
	}

	want := []string{
		"--context", "prod",
		"compose", "--file", "compose.prod.yml",
		"up", "--detach", "api",
	}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}

func TestCommandSortsMapFlags(t *testing.T) {
	driver := glove.New(Docker{})

	cmd, err := driver.Command(Run{
		Env: map[string]string{
			"DEBUG":   "1",
			"APP_ENV": "dev",
		},
		Volume: []string{
			"/host/app:/app",
			"/host/cache:/cache",
		},
		Image:   "golang:1.24",
		Command: []string{"go", "test", "./..."},
	})
	if err != nil {
		t.Fatalf("render command: %v", err)
	}

	want := []string{
		"run",
		"--env", "APP_ENV=dev",
		"--env", "DEBUG=1",
		"--volume", "/host/app:/app",
		"--volume", "/host/cache:/cache",
		"golang:1.24",
		"go", "test", "./...",
	}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}

func TestCommandRendersListStyleSliceFlag(t *testing.T) {
	driver := glove.New(GoTool{})

	cmd, err := driver.Command(Build{
		Output: "bin/sun",
		LinkFlags: []string{
			"-X main.BinaryVersion=0.0.1",
			"-X main.BuildDate=2026-05-07T19:03:17Z",
		},
		Packages: []string{"./cmd/sun"},
	})
	if err != nil {
		t.Fatalf("render command: %v", err)
	}

	want := []string{
		"build",
		"-o", "bin/sun",
		"-ldflags", "-X main.BinaryVersion=0.0.1 -X main.BuildDate=2026-05-07T19:03:17Z",
		"./cmd/sun",
	}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}

	wantString := "go build -o bin/sun -ldflags '-X main.BinaryVersion=0.0.1 -X main.BuildDate=2026-05-07T19:03:17Z' ./cmd/sun"
	if cmd.String() != wantString {
		t.Fatalf("string = %q, want %q", cmd.String(), wantString)
	}
}

func TestCompileRejectsNonContiguousArgs(t *testing.T) {
	type Invalid struct {
		Bin glove.Bin `bin:"tool"`
		One string    `arg:"0"`
		Two string    `arg:"2"`
	}

	_, err := glove.Compile(Invalid{})
	if err == nil {
		t.Fatal("expected compile error")
	}
}

func TestCompileRejectsListFlagOnNonSlice(t *testing.T) {
	type Invalid struct {
		Bin  glove.Bin `bin:"tool"`
		Flag string    `flag:"--flag,list"`
	}

	_, err := glove.Compile(Invalid{})
	if err == nil {
		t.Fatal("expected compile error")
	}
}

func TestCommandRejectsMissingIntermediateFrame(t *testing.T) {
	driver := glove.New(Docker{})

	_, err := driver.Command(Up{
		Detach:  true,
		Service: "api",
	})
	if err == nil {
		t.Fatal("expected path validation error")
	}
}

func TestCommandRejectsPositionalGaps(t *testing.T) {
	type Gap struct {
		Bin   glove.Bin `bin:"tool"`
		First string    `arg:"0,required"`
		Next  string    `arg:"1"`
		Last  *string   `arg:"2"`
	}

	driver := glove.New(Gap{})
	last := "tail"

	_, err := driver.Command(Gap{
		First: "one",
		Last:  &last,
	})
	if err == nil {
		t.Fatal("expected positional gap error")
	}
}

func TestNewPanicsOnInvalidSchema(t *testing.T) {
	type Invalid struct {
		Name string
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = glove.New(Invalid{})
}

func TestRunUsesInjectedRunner(t *testing.T) {
	runner := &captureRunner{
		result: glove.Result{
			Stdout: []byte("ok"),
			Err:    errors.New("boom"),
		},
	}

	driver := glove.New(Git{}, glove.CompileOptions{
		Bin:    "/usr/local/bin/git",
		Runner: runner,
	})

	result := driver.Run(context.Background(), Clone{
		Branch: "main",
		Repo:   "https://example.com/repo.git",
	})

	if result.Command.Program != "/usr/local/bin/git" {
		t.Fatalf("program = %q, want override", result.Command.Program)
	}
	if string(result.Stdout) != "ok" {
		t.Fatalf("stdout = %q, want ok", string(result.Stdout))
	}
	if result.Err == nil || result.Err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", result.Err)
	}

	want := []string{"clone", "--branch", "main", "https://example.com/repo.git"}
	if !slices.Equal(runner.command.Args, want) {
		t.Fatalf("runner args = %#v, want %#v", runner.command.Args, want)
	}
	if runner.options.Dir != "" {
		t.Fatalf("dir = %q, want empty", runner.options.Dir)
	}
}

func TestCommandRunWithPassesExecutionOptions(t *testing.T) {
	runner := &captureRunner{
		result: glove.Result{
			Stdout: []byte("ok"),
		},
	}

	cmd := glove.Command{
		Program: "git",
		Args:    []string{"status", "--short"},
	}

	result := cmd.RunWith(context.Background(), glove.RunOptions{
		Dir: "/workspace/repo",
		Env: map[string]string{
			"GIT_TERMINAL_PROMPT": "0",
		},
		Runner: runner,
	})

	if string(result.Stdout) != "ok" {
		t.Fatalf("stdout = %q, want ok", string(result.Stdout))
	}
	if runner.command.Program != "git" {
		t.Fatalf("program = %q, want git", runner.command.Program)
	}
	if runner.options.Dir != "/workspace/repo" {
		t.Fatalf("dir = %q, want /workspace/repo", runner.options.Dir)
	}
	if runner.options.Env["GIT_TERMINAL_PROMPT"] != "0" {
		t.Fatalf("env = %#v, want GIT_TERMINAL_PROMPT=0", runner.options.Env)
	}
}

func TestDriverRunWithOverridesDefaultRunner(t *testing.T) {
	defaultRunner := &captureRunner{
		result: glove.Result{Stdout: []byte("default")},
	}
	overrideRunner := &captureRunner{
		result: glove.Result{Stdout: []byte("override")},
	}

	driver := glove.New(Git{}, glove.CompileOptions{
		Runner: defaultRunner,
	})

	result := driver.RunWith(context.Background(), glove.RunOptions{
		Dir:    "/workspace/repo",
		Runner: overrideRunner,
	}, Clone{
		Branch: "main",
		Repo:   "https://example.com/repo.git",
	})

	if string(result.Stdout) != "override" {
		t.Fatalf("stdout = %q, want override", string(result.Stdout))
	}
	if defaultRunner.called {
		t.Fatal("default runner should not have been called")
	}
	if !overrideRunner.called {
		t.Fatal("override runner should have been called")
	}
	if overrideRunner.options.Dir != "/workspace/repo" {
		t.Fatalf("dir = %q, want /workspace/repo", overrideRunner.options.Dir)
	}
}

type captureRunner struct {
	command glove.Command
	options glove.RunOptions
	result  glove.Result
	called  bool
}

func (r *captureRunner) Run(_ context.Context, cmd glove.Command, options glove.RunOptions) glove.Result {
	r.called = true
	r.command = cmd
	r.options = options
	return r.result
}
