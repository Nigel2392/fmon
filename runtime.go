package main

import (
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/spf13/cobra"
)

var RUNTIME = program{}

type program struct {
	Verbose    bool
	CurrentDir string
}

func preloadObjectFunc(change func(FMonCommandFunc) CobraCommandFunc) func(FMonObjectCommandFunc) CobraCommandFunc {
	return func(focf FMonObjectCommandFunc) CobraCommandFunc {
		var inner = func(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) {
			var obj = cnf.Files.GetOrDefault(RUNTIME.CurrentDir, &configure.MonitoredObject{})
			focf(cnf, obj, cmd, args)
		}

		return change(inner)
	}
}
