package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/wentx/henetdns/internal/cli"
	"github.com/wentx/henetdns/internal/errs"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, errs.ErrInvalidInput):
		return 2
	case errors.Is(err, errs.ErrAuthRequired):
		return 3
	case errors.Is(err, errs.ErrRemote):
		return 4
	case errors.Is(err, errs.ErrParseChanged):
		return 5
	case errors.Is(err, errs.ErrStore):
		return 6
	default:
		return 1
	}
}
