package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/spf13/cobra"
)

var RUNTIME = program{}

type program struct {
	Verbose    bool
	NoLogo     bool
	CurrentDir string
}

func preloadObjectFunc(change func(FMonCommandFunc) CobraCommandFuncE) func(FMonObjectCommandFunc) CobraCommandFuncE {
	return func(focf FMonObjectCommandFunc) CobraCommandFuncE {
		var inner = func(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
			var obj = cnf.Files.GetOrDefault(RUNTIME.CurrentDir, &configure.MonitoredObject{})
			return focf(cnf, obj, cmd, args)
		}

		return change(inner)
	}
}

func setRuntimeWd(s string) error {
	absTarget, err := filepath.Abs(s)
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(RUNTIME.CurrentDir, absTarget)
	if err != nil {
		// If Rel fails, they are fundamentally incompatible
		// (example: C:/ vs D:/ on Windows).
		return fmt.Errorf("Path not in working directory, error: %w", err)
	}

	relSlash := filepath.ToSlash(rel)

	if relSlash == ".." || strings.HasPrefix(relSlash, "../") {
		return fmt.Errorf(
			"Path is outside of the current working directory: %q",
			relSlash,
		)
	}

	RUNTIME.CurrentDir = absTarget
	return nil
}
