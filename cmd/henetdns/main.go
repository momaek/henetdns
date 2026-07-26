package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/momaek/henetdns/internal/cli"
	"github.com/momaek/henetdns/internal/errs"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

// resolveVersion falls back to the module version recorded by the Go
// toolchain, so `go install ...@vX.Y.Z` builds report the release instead of
// "dev".
func resolveVersion() string {
	if version != "dev" && version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

func main() {
	if err := cli.Execute(resolveVersion()); err != nil {
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
