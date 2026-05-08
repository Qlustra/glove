package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sort"
)

var (
	errCommandProgramEmpty = errors.New("command program must not be empty")
	errRunContextNil       = errors.New("context must not be nil")
)

type Result struct {
	Command Command
	Stdout  []byte
	Stderr  []byte
	Err     error
}

type RunOptions struct {
	Dir              string
	Env              map[string]string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	DisableBuffering bool
	Runner           Runner
}

type Runner interface {
	Run(context.Context, Command, RunOptions) Result
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, cmd Command, options RunOptions) Result {
	result := Result{Command: cmd}

	if ctx == nil {
		result.Err = errRunContextNil
		return result
	}
	if cmd.Program == "" {
		result.Err = errCommandProgramEmpty
		return result
	}

	execCmd := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	if options.Dir != "" {
		execCmd.Dir = options.Dir
	}
	if len(options.Env) > 0 {
		execCmd.Env = append(os.Environ(), formatEnv(options.Env)...)
	}
	if options.Stdin != nil {
		execCmd.Stdin = options.Stdin
	}

	stdoutWriter, stdoutBuffer := outputWriter(options.Stdout, options.DisableBuffering)
	stderrWriter, stderrBuffer := outputWriter(options.Stderr, options.DisableBuffering)

	execCmd.Stdout = stdoutWriter
	execCmd.Stderr = stderrWriter

	result.Err = execCmd.Run()
	result.Stdout = outputBytes(stdoutBuffer)
	result.Stderr = outputBytes(stderrBuffer)

	return result
}

func formatEnv(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	formatted := make([]string, 0, len(keys))
	for _, key := range keys {
		formatted = append(formatted, key+"="+env[key])
	}
	return formatted
}

func outputWriter(sink io.Writer, disableBuffering bool) (io.Writer, *bytes.Buffer) {
	if disableBuffering {
		if sink == nil {
			return io.Discard, nil
		}
		return sink, nil
	}

	buffer := &bytes.Buffer{}
	if sink == nil {
		return buffer, buffer
	}

	return io.MultiWriter(buffer, sink), buffer
}

func outputBytes(buffer *bytes.Buffer) []byte {
	if buffer == nil {
		return nil
	}

	return buffer.Bytes()
}
