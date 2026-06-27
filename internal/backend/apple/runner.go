package apple

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type CommandRunner interface {
	Run(ctx context.Context, args ...string) (CommandResult, error)
	RunInput(ctx context.Context, stdin io.Reader, args ...string) (CommandResult, error)
	Start(ctx context.Context, stdout, stderr io.Writer, args ...string) (func() error, error)
}

type ExecRunner struct {
	Binary string
}

func (r ExecRunner) Run(ctx context.Context, args ...string) (CommandResult, error) {
	return r.RunInput(ctx, nil, args...)
}

func (r ExecRunner) RunInput(ctx context.Context, stdin io.Reader, args ...string) (CommandResult, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	return result, &CommandError{Args: args, Result: result, Err: err}
}

func (r ExecRunner) Start(ctx context.Context, stdout, stderr io.Writer, args ...string) (func() error, error) {
	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, &CommandError{Args: args, Err: err}
	}
	return func() error {
		if err := cmd.Wait(); err != nil {
			var exitErr *exec.ExitError
			result := CommandResult{}
			if errors.As(err, &exitErr) {
				result.ExitCode = exitErr.ExitCode()
			}
			return &CommandError{Args: args, Result: result, Err: err}
		}
		return nil
	}, nil
}

type CommandError struct {
	Args   []string
	Result CommandResult
	Err    error
}

func (e *CommandError) Error() string {
	message := string(bytes.TrimSpace(e.Result.Stderr))
	if message == "" {
		message = e.Err.Error()
	}
	return fmt.Sprintf("container %v: %s", e.Args, message)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}
