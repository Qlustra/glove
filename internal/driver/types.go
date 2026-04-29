package driver

import "github.com/qlustra/glove/internal/command"

type Bin struct{}

type CompileOptions struct {
	Bin    string
	Runner command.Runner
}

type Driver[T any] struct {
	compiled *schema
	runner   command.Runner
}
