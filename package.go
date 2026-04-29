package glove

import (
	"github.com/qlustra/glove/internal/command"
	"github.com/qlustra/glove/internal/driver"
)

type Bin = driver.Bin

type CompileOptions = driver.CompileOptions
type Driver[T any] = driver.Driver[T]

type Command = command.Command
type RunOptions = command.RunOptions
type Result = command.Result
type Runner = command.Runner
type ExecRunner = command.ExecRunner

func Compile[T any](decl T, options ...CompileOptions) (Driver[T], error) {
	return driver.Compile(decl, options...)
}

func New[T any](decl T, options ...CompileOptions) Driver[T] {
	return driver.New(decl, options...)
}
