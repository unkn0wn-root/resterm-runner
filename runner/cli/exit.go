package cli

import (
	"errors"
	"fmt"

	"github.com/unkn0wn-root/resterm/headless"
)

type exitCoder interface {
	ExitCode() int
}

type ExitErr struct {
	Err  error
	Code int
}

func (e ExitErr) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e ExitErr) Unwrap() error {
	return e.Err
}

func (e ExitErr) ExitCode() int {
	if e.Code == 0 {
		return 1
	}
	return e.Code
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var ex exitCoder
	if errors.As(err, &ex) {
		return ex.ExitCode()
	}
	return 1
}

func usageErr(format string, args ...any) error {
	return ExitErr{Err: fmt.Errorf(format, args...), Code: headless.ExitUsage}
}

func runErr(err error) error {
	if headless.IsUsageError(err) {
		return usageErr("run: %w", err)
	}
	return err
}
