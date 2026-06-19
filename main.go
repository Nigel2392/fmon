package main

import (
	"fmt"
	"os"

	"github.com/Nigel2392/fmon/logo"
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

const (
	NAME       = "filesystem-monitor"
	NAME_SHORT = "fmon"
)

var SERVICE_CONFIG = &service.Config{
	Name:        NAME_SHORT,
	DisplayName: "FMon Watcher Service",
	Description: "Service keeps track of configuration changes and watches the configured directories.",
	Arguments:   []string{"service", "run"},
}

func main() {
	configure.Setup(configure.PackageSetup{
		NameBase: "fmon",
	})

	/*
		--------------------
		Root Command Definition
		--------------------
	*/
	var root = &cobra.Command{
		Use:           NAME,
		Args:          cobra.ArbitraryArgs,
		PreRun:        printLogo,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	/*
		--------------------
		Service Command Definition
		--------------------
	*/
	var preRun, changeService = preloadServiceFunc(
		SERVICE_CONFIG,
	)

	var cobraService = &cobra.Command{
		Use:               "service",
		Short:             "Manage the service",
		PersistentPreRunE: preRun,
	}

	var (
		cobraServiceInstall = &cobra.Command{
			Use:    "install",
			Short:  "Install the service",
			PreRun: printLogo,
			RunE:   changeService(commandServiceInstall),
		}
		cobraServiceUninstall = &cobra.Command{
			Use:    "uninstall",
			Short:  "Uninstall the service",
			PreRun: printLogo,
			RunE:   changeService(commandServiceUninstall),
		}
		cobraServiceStart = &cobra.Command{
			Use:    "start",
			Short:  "Start the service",
			PreRun: printLogo,
			RunE:   changeService(commandServiceStart),
		}
		cobraServiceRun = &cobra.Command{
			Use:   "run",
			Short: "Run the service in interactive mode",
			RunE:  changeService(commandServiceRun),
		}
		cobraServiceStop = &cobra.Command{
			Use:    "stop",
			Short:  "Stop the service",
			PreRun: printLogo,
			RunE:   changeService(commandServiceStop),
		}
		cobraServiceStatus = &cobra.Command{
			Use:    "status",
			Short:  "Retrieve the service status",
			PreRun: printLogo,
			RunE:   changeService(commandServiceStatus),
		}
	)

	/*
		--------------------
		Watcher Command Definition
		--------------------
	*/
	var _, preloadConfig, changeConfig = preloadConfig()
	var cobraWatcher = &cobra.Command{
		Use:   "watcher",
		Short: "Manage the watcher",
		Aliases: []string{
			"w", "watch", ".",
		},
		PersistentPreRun: commandChain(
			printLogo,
			preloadConfig,
		),
	}

	var (
		cobraWatcherAdd = &cobra.Command{
			Use:  "add",
			RunE: changeConfig(commandWatchDir),
			Aliases: []string{
				"a", "i", "n", "init", "new",
			},
		}
		cobraWatcherRem = &cobra.Command{
			Use:  "remove",
			RunE: changeConfig(commandUnwatchDir),
			Aliases: []string{
				"r", "rm", "delete", "del", "d",
			},
		}
		cobraWatcherAddAction = &cobra.Command{
			Use:  "action",
			RunE: changeConfig(commandWatcherAddAction),
			Aliases: []string{
				"a",
			},
		}
		cobraWatcherRemAction = &cobra.Command{
			Use:  "action",
			RunE: changeConfig(commandWatcherRemAction),
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
		Use:              "config",
		Short:            "Manage the configuration file",
		PersistentPreRun: printLogo,
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
			RunE:   changeConfig(commandLocateConfig),
			PreRun: preloadConfig,
			Aliases: []string{
				"l", "find", "f",
			},
		}
		cobraConfigPrint = &cobra.Command{
			Use:    "print",
			RunE:   changeConfig(commandPrintConfig),
			PreRun: preloadConfig,
			Aliases: []string{
				"p",
			},
		}
	)

	/*
		---------------
		Watcher Command
		---------------
	*/
	cobraWatcher.PersistentFlags().BoolP(
		"literal", "l", false,
		"If set, the directory argument is treated as a literal path and will undergo no transformations.",
	)

	var addFlags = cobraWatcherAddAction.Flags()
	addFlags.StringP("id", "i", "", "The ID for the action")
	addFlags.StringP("type", "t", "", "The type for the action")
	addFlags.Uint64P("size", "s", 0, "The size for the action (in bytes).")
	addFlags.Float64P("debounce", "d", 0.1, "The debounce time in seconds (0.1 is minimum)")
	addFlags.StringP("action", "a", "", "The action to perform or the path to a js which handles the action.")
	addFlags.StringP("target", "x", "", "The file to perform an action on")
	// addFlags.Bool("supervised", false, "Make the action supervised.")
	cobraWatcherAddAction.MarkFlagRequired("id")
	cobraWatcherAddAction.MarkFlagRequired("type")
	cobraWatcherAddAction.MarkFlagRequired("action")

	var remFlags = cobraWatcherRemAction.Flags()
	remFlags.StringP("id", "i", "", "The ID of the action to remove")
	cobraWatcherRemAction.MarkFlagRequired("id")

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
		Service Command
		---------------
	*/
	cobraService.AddCommand(
		cobraServiceInstall,
		cobraServiceUninstall,
		cobraServiceStart,
		cobraServiceRun,
		cobraServiceStop,
		cobraServiceStatus,
	)

	/*
		---------------
		Root Command
		---------------
	*/
	root.PersistentFlags().BoolVarP(
		&RUNTIME.Verbose, "verbose", "v", false, "Enable verbose output",
	)
	root.PersistentFlags().BoolVarP(
		&RUNTIME.NoLogo, "quiet", "q", false, "Stop the logo from printing",
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

func printLogo(_ *cobra.Command, _ []string) {
	if RUNTIME.NoLogo {
		return
	}
	logo.Print()
}
