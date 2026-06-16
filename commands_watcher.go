package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/spf13/cobra"
)

func getDir(cmd *cobra.Command, args []string) (string, error) {
	var (
		dir string
		err error
	)

	if len(args) > 0 {
		dir = args[0]
	} else {
		dir, err = os.Getwd()
	}

	if !cmd.Flag("literal").Changed {
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	return dir, err
}

func commandWatchDir(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var dir, err = getDir(cmd, args)
	if err != nil {
		return err
	}

	_, ok := cnf.Files.Get(dir)
	if ok {
		return fmt.Errorf("Directory %q is already monitored.", dir)
	}

	cnf.Files.Set(dir, &configure.MonitoredObject{})
	err = configure.Write(cnf.Path, cnf)
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
	var dir, err = getDir(cmd, args)
	if err != nil {
		return err
	}

	ok := cnf.Files.Delete(dir)
	if !ok {
		return fmt.Errorf("Directory %q was already unmonitored", dir)
	}

	colYellow.Print("Removed directory from watch list: ")
	fmt.Println(dir)

	err = configure.Write(cnf.Path, cnf)
	if err != nil {
		return err
	}

	colBlue.Print("Rewrote configuration file: ")
	fmt.Println(cnf.Path)
	return nil
}

func commandWatcherAddAction(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var dir, err = getDir(cmd, args)
	if err != nil {
		return err
	}

	var (
		id         string
		typ        string
		size       uint64
		action     string
		supervised bool
	)

	if id, err = cmd.Flags().GetString("id"); err != nil {
		return err
	}
	if typ, err = cmd.Flags().GetString("type"); err != nil {
		return err
	}
	if size, err = cmd.Flags().GetUint64("size"); err != nil {
		return err
	}
	if action, err = cmd.Flags().GetString("action"); err != nil {
		return err
	}
	if supervised, err = cmd.Flags().GetBool("supervised"); err != nil {
		return err
	}

	obj, ok := cnf.Files.Get(dir)
	if !ok {
		obj = &configure.MonitoredObject{
			Actions: make([]configure.MonitoredObjectAction, 0),
		}
		cnf.Files.Set(dir, obj)
	}

	for _, action := range obj.Actions {
		if action.ID == id {
			return fmt.Errorf("action id %q is already in use.", action.ID)
		}
	}

	typ = strings.ToLower(typ)
	if !slices.Contains(configure.ACTION_TYPES, typ) {
		return fmt.Errorf("action must be one of %s", strings.Join(configure.ACTION_TYPES, ", "))
	}

	var actionObj = configure.MonitoredObjectAction{
		ID:         id,
		Action:     action,
		ActionType: action,
		Size:       size,
		Supervised: supervised,
	}

	obj.Actions = append(obj.Actions, actionObj)

	err = configure.Write(cnf.Path, cnf)
	if err != nil {
		return err
	}

	colBlue.Print("Rewrote configuration file: ")
	fmt.Println(cnf.Path)
	return nil
}

func commandWatcherRemAction(cnf *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var dir, err = getDir(cmd, args)
	if err != nil {
		return err
	}

	actionId, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}

	obj, ok := cnf.Files.Get(dir)
	if !ok {
		return fmt.Errorf("Directory %q is not monitored", dir)
	}

	if len(obj.Actions) == 0 {
		return fmt.Errorf("Directory %q has no actions.", dir)
	}

	var newActions = make([]configure.MonitoredObjectAction, len(obj.Actions)-1)
	for _, action := range obj.Actions {
		if action.ID == actionId {
			continue
		}
		newActions = append(newActions, action)
	}

	if len(obj.Actions) == len(newActions) {
		return fmt.Errorf("Action %q not found.", actionId)
	}

	obj.Actions = newActions

	err = configure.Write(cnf.Path, cnf)
	if err != nil {
		return err
	}

	colBlue.Print("Rewrote configuration file: ")
	fmt.Println(cnf.Path)
	return nil
}
