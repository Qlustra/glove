package command

import (
	"context"
	"strings"
)

type Command struct {
	Program string
	Args    []string
}

func (c Command) Argv() []string {
	if c.Program == "" && len(c.Args) == 0 {
		return nil
	}

	argv := make([]string, 0, 1+len(c.Args))
	if c.Program != "" {
		argv = append(argv, c.Program)
	}
	argv = append(argv, c.Args...)
	return argv
}

func (c Command) String() string {
	argv := c.Argv()
	if len(argv) == 0 {
		return ""
	}

	quoted := make([]string, len(argv))
	for i, token := range argv {
		quoted[i] = shellQuote(token)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(token string) string {
	if token == "" {
		return "''"
	}

	if isShellSafe(token) {
		return token
	}

	return "'" + strings.ReplaceAll(token, "'", `'"'"'`) + "'"
}

func isShellSafe(token string) bool {
	for _, r := range token {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}

		switch r {
		case '-', '_', '.', '/', ':', ',', '=', '+', '@', '%':
			continue
		default:
			return false
		}
	}

	return true
}

func (c Command) Run(ctx context.Context) Result {
	return c.RunWith(ctx, RunOptions{})
}

func (c Command) RunWith(ctx context.Context, options RunOptions) Result {
	runner := options.Runner
	if runner == nil {
		runner = ExecRunner{}
	}

	result := runner.Run(ctx, c, options)
	result.Command = c
	return result
}

func (c Command) With(ctx context.Context, options RunOptions) Result {
	return c.RunWith(ctx, options)
}
