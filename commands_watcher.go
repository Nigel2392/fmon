package main

import (
	"fmt"
	"path/filepath"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/spf13/cobra"
)

func commandWatchDir(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var dir string = RUNTIME.CurrentDir
	if len(args) > 0 {
		dir = args[0]
	}

	if !cmd.Flag("literal").Changed {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
	}

	_, ok := cnf.Files.Get(dir)
	if ok {
		return fmt.Errorf("Directory %q is already monitored.", dir)
	}

	cnf.Files.Set(dir, &configure.MonitoredObject{})
	err := configure.Rewrite(cnf, true)
	if err != nil {
		return err
	}

	colBlue.Print("Added directory to watch list: ")
	fmt.Println(dir)

	colBlue.Print("Rewrote configuration file: ")
	fmt.Println(cnf.Path)
	return nil
}

func commandUnwatchDir(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var dir string = RUNTIME.CurrentDir
	if len(args) > 0 {
		dir = args[0]
	}

	if !cmd.Flag("literal").Changed {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
	}

	ok := cnf.Files.Delete(dir)
	if !ok {
		return fmt.Errorf("Directory %q was already unmonitored", dir)
	}

	colYellow.Print("Removed directory from watch list: ")
	fmt.Println(dir)

	err := configure.Rewrite(cnf, true)
	if err != nil {
		return err
	}

	colBlue.Print("Rewrote configuration file: ")
	fmt.Println(cnf.Path)
	return nil
}

func commandWatcherAddAction(cnf *configure.FilesystemMonitor, where *configure.MonitoredObject, cmd *cobra.Command, args []string) error {
	return nil
}

func commandWatcherRemAction(cnf *configure.FilesystemMonitor, where *configure.MonitoredObject, cmd *cobra.Command, args []string) error {
	return nil
}
