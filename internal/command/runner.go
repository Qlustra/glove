package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
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

type Runner interface {
	Run(context.Context, Command) Result
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, cmd Command) Result {
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
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	if len(cmd.Env) > 0 {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	result.Err = execCmd.Run()
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	return result
}
