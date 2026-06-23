package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Nigel2392/fmon/watcher"
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

func commandInitConfig(cmd *cobra.Command, args []string) {
	var flagGlobal = cmd.Flag("global").Changed

	var example = configure.NewMonitorConfig("yaml", "")
	example.Files.Set("path/to/watched/directory_or_file", &configure.MonitoredObject{
		Actions: []configure.MonitoredObjectAction{
			{
				ID:         "my-unique-action-id",
				ActionType: configure.CREATE_ACTION,
				Size:       0,
				Debounce:   0.1,
				Action:     "path/to/js/or/shell/file",
			},
		},
	})

	existingPath, err := configure.ExistingPath()
	if err != nil && !errors.Is(err, configure.ErrConfigNotExists) {
		panic("error while checking existing configuration file path")
	}

	if existingPath != "" {
		colRed.Printf("Configuration file already exists at: ")
		fmt.Printf("%q\n", filepath.ToSlash(existingPath))
		colYellow.Print("Do you wish to overwrite it? [y/n] ")

		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))

		if !slices.Contains([]string{"y", "yes"}, response) {
			colYellow.Println("Aborting...")
			os.Exit(1)
		}

		// Create a dud to check service status
		var svc, err = service.New(
			watcher.NewWatcher(""),
			SERVICE_CONFIG,
		)
		if err != nil {
			colRed.Print("Error while creating service object: ")
			fmt.Println(err)
			os.Exit(1)
		}

		status, err := svc.Status()
		switch {
		case err == nil, status != service.StatusUnknown:
			colRed.Println("Service must be uninstalled first.")
			os.Exit(1)
		case !errors.Is(err, service.ErrNotInstalled):
			colRed.Print("Error while retrieving status for service: ")
			fmt.Println(err)
			os.Exit(1)
		default:
			colBlue.Println("Service not installed...")
		}

		var (
			isUserConfig   = configure.IsUserConfig(existingPath)
			movingToGlobal = flagGlobal && isUserConfig
			movingToUser   = !flagGlobal && !isUserConfig
		)
		if (movingToGlobal) || (movingToUser) {
			var dir = filepath.Dir(existingPath)
			if movingToGlobal {
				colRed.Print("Deleting old user config directory... ")
			} else {
				colRed.Print("Deleting old global configuration directory... ")
			}
			fmt.Println(dir)

			if err := os.RemoveAll(dir); err != nil {
				colRed.Print("Failed to remove old config file: ")
				fmt.Println(err)
				os.Exit(1)
			}

			// Reset the existing path so the rewriteConfig function can decide
			// the new path based on the 'global' parameter.
			existingPath = ""
		}

		colRed.Println("Overwriting config file...")
		example.Path = existingPath
	}

	if err := configure.Rewrite(example, flagGlobal); err != nil {
		colRed.Println(configure.ErrConfigWrite)
		fmt.Println(err)
		os.Exit(1)
	}

	colYellow.Print("Config written to: ")
	fmt.Println(example.Path)
}

func commandLocateConfig(config *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	colYellow.Print("Found config: ")
	fmt.Println(config.Path)
	return nil
}

func commandPrintConfig(config *configure.FilesystemMonitor, cmd *cobra.Command, args []string) error {
	var buf = new(bytes.Buffer)
	colBlue.Fprint(buf, "Configuration File: ")
	fmt.Fprintf(buf, "%s\n", config.Path)

	colOrange.Fprintln(buf, "Watching:")

	for k, v := range config.Files.Iter() {

		colYellow.Fprintf(buf, "  %s\n", k)

		if RUNTIME.Verbose {
			for _, action := range v.Actions {
				// ID
				// ActionType
				// Size
				// Debounce
				// Action
				// Target

				fmt.Fprintf(buf, "     ID: %s\n", action.ID)
				fmt.Fprintf(buf, "     action_type: %s\n", action.ActionType)
				fmt.Fprintf(buf, "     action: %s\n", action.Action)
				fmt.Fprintf(buf, "     debounce: %.2f\n", action.Debounce)

				if action.Cron != "" {
					fmt.Fprintf(buf, "     cron: %s\n", action.Cron)
				}

				if action.Size > 0 {
					fmt.Fprintf(buf, "     size: %d\n", action.Size)
				}
			}
		}
	}

	fmt.Print(buf.String())
	return nil
}
