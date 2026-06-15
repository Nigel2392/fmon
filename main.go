package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/fmon/logo"
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

const (
	NAME       = "filesystem-monitor"
	NAME_SHORT = "fmon"
)

func main() {
	var wd, err = os.Getwd()
	if err != nil {
		panic("could not retrieve working directory")
	}

	logo.Print()

	RUNTIME.CurrentDir = wd
	configure.Setup(configure.PackageSetup{
		NameBase: "fmon",
	})

	/*
		--------------------
		Root Command Definition
		--------------------
	*/
	var root = &cobra.Command{
		Use:  "filesystem-monitor",
		Args: cobra.ArbitraryArgs,
	}

	/*
		--------------------
		Service Command Definition
		--------------------
	*/
	var preRun, changeService = preloadServiceFunc(
		&service.Config{
			Name:        "FMon",
			DisplayName: "FMon Watcher Service",
		},
	)

	var cobraService = &cobra.Command{
		Use:               "service",
		Short:             "Manage the service",
		PersistentPreRunE: preRun,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	var (
		cobraServiceInstall = &cobra.Command{
			Use:           "install",
			Short:         "Install the service",
			RunE:          changeService(cobraServiceInstall),
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		cobraServiceUninstall = &cobra.Command{
			Use:           "uninstall",
			Short:         "Uninstall the service",
			RunE:          changeService(cobraServiceUninstall),
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		cobraServiceStart = &cobra.Command{
			Use:           "start",
			Short:         "Start the service",
			RunE:          changeService(cobraServiceStart),
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		cobraServiceStop = &cobra.Command{
			Use:           "stop",
			Short:         "Stop the service",
			RunE:          changeService(cobraServiceStop),
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		cobraServiceStatus = &cobra.Command{
			Use:           "status",
			Short:         "Retrieve the service status",
			RunE:          changeService(cobraServiceStatus),
			SilenceUsage:  true,
			SilenceErrors: true,
		}
	)

	/*
		--------------------
		Watcher Command Definition
		--------------------
	*/
	var _, preloadConfig, changeConfig = preloadConfig()
	var changeObject = preloadObjectFunc(changeConfig)
	var cobraWatcher = &cobra.Command{
		Use:   "watcher",
		Short: "Manage the watcher",
		Aliases: []string{
			"w", "watch", ".",
		},
		PersistentPreRun: preloadConfig,
	}

	var (
		cobraWatcherAdd = &cobra.Command{
			Use: "add",
			Run: changeConfig(commandWatchDir),
			Aliases: []string{
				"a", "i", "n", "init", "new",
			},
		}
		cobraWatcherRem = &cobra.Command{
			Use: "remove",
			Run: changeConfig(commandUnwatchDir),
			Aliases: []string{
				"r", "rm", "delete", "del", "d",
			},
		}
		cobraWatcherAddAction = &cobra.Command{
			Use: "action",
			Run: changeObject(commandWatcherAddAction),
			Aliases: []string{
				"a",
			},
		}
		cobraWatcherRemAction = &cobra.Command{
			Use: "action",
			Run: changeObject(commandWatcherRemAction),
			Aliases: []string{
				"a",
			},
		}
	)

	/*
		--------------------
		Config Command Definition
		--------------------
	*/
	var cobraConfig = &cobra.Command{
		Use:   "config",
		Short: "Manage the configuration file",
		Aliases: []string{
			"c", "conf",
		},
	}

	var (
		cobraConfigInit = &cobra.Command{
			Use: "init",
			Run: commandInitConfig,
			Aliases: []string{
				"i", "n", "new",
			},
		}
		cobraConfigLocate = &cobra.Command{
			Use:    "locate",
			Run:    changeConfig(commandLocateConfig),
			PreRun: preloadConfig,
			Aliases: []string{
				"l", "find", "f",
			},
		}
		cobraConfigPrint = &cobra.Command{
			Use:    "print",
			Run:    changeConfig(commandPrintConfig),
			PreRun: preloadConfig,
			Aliases: []string{
				"p",
			},
		}
	)

	/*
		---------------
		Service Command
		---------------
	*/

	cobraService.AddCommand(
		cobraServiceInstall,
		cobraServiceUninstall,
		cobraServiceStart,
		cobraServiceStop,
		cobraServiceStatus,
	)

	/*
		---------------
		Watcher Command
		---------------
	*/
	cobraWatcher.Flags().FuncP("dir", "d", "The directory to perform the action on.", func(s string) error {
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
		colYellow.Printf("Using directory: %q", RUNTIME.CurrentDir)
		return nil
	})

	cobraWatcher.AddCommand(
		cobraWatcherAdd,
		cobraWatcherRem,
	)
	cobraWatcherAdd.AddCommand(
		cobraWatcherAddAction,
	)
	cobraWatcherRem.AddCommand(
		cobraWatcherRemAction,
	)

	/*
		---------------
		Config Command
		---------------
	*/
	cobraConfigInit.Flags().Bool(
		"global", false, "Write the config to a global location instead of a user specific location.",
	)

	cobraConfig.AddCommand(
		cobraConfigInit,
		cobraConfigLocate,
		cobraConfigPrint,
	)

	/*
		---------------
		Root Command
		---------------
	*/
	root.PersistentFlags().BoolVarP(
		&RUNTIME.Verbose, "verbose", "v", false, "Enable verbose output",
	)

	root.AddCommand(
		cobraService,
		cobraWatcher,
		cobraConfig,
	)

	if err := root.Execute(); err != nil {
		colRed.Print("Error during execution of program: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
