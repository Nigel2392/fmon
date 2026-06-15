package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/spf13/cobra"
)

func commandInitConfig(cmd *cobra.Command, args []string) {
	var flagGlobal = cmd.Flag("global").Changed

	var example = configure.NewMonitorConfig("yaml", "")

	example.Files.Set("path/to/watched/directory_or_file", &configure.MonitoredObject{
		Actions: []configure.MonitoredObjectAction{
			{},
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

		var (
			isUserConfig   = configure.IsUserConfig(existingPath)
			movingToGlobal = flagGlobal && isUserConfig
			movingToUser   = !flagGlobal && !isUserConfig
		)
		if (movingToGlobal) || (movingToUser) {

			var dir = filepath.Dir(existingPath)
			if movingToGlobal {
				colRed.Printf("Deleting old user config directory %q...\n", dir)
			} else {
				colRed.Printf("Deleting old global configuration directory %q...\n", dir)
			}

			if err := os.RemoveAll(dir); err != nil {
				colRed.Println("Failed to remove old config file: ", err)
				os.Exit(1)
			}

			// Reset the existing path so the rewriteConfig function can decide
			// based on the 'global' parameter.
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

func commandLocateConfig(config *configure.FilesystemMonitor, cmd *cobra.Command, args []string) {
	colYellow.Print("Found config: ")
	fmt.Println(config.Path)
}

func commandPrintConfig(config *configure.FilesystemMonitor, cmd *cobra.Command, args []string) {
	var buf = new(bytes.Buffer)
	colBlue.Fprint(buf, "Configuration File: ")
	fmt.Fprintf(buf, "%s\n", config.Path)

	colOrange.Fprintln(buf, "Watching:")

	for k, v := range config.Files.Iter() {

		colYellow.Fprintf(buf, "  %s\n", k)

		if RUNTIME.Verbose {
			for _, action := range v.Actions {
				// ActionType
				// Size
				// Action
				// Supervised

				fmt.Fprintf(buf, "     action_type: %s\n", action.ActionType)
				fmt.Fprintf(buf, "     supervised: %t\n", action.Supervised)
				fmt.Fprintf(buf, "     action: %s\n", action.Action)

				if action.Size > 0 {
					fmt.Fprintf(buf, "     size: %d\n", action.Size)
				}
			}
		}
	}

	fmt.Print(buf.String())
}
