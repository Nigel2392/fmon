package main

import (
	"fmt"
	"os"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	colBlue   = color.RGB(100, 100, 255)
	colOrange = color.RGB(255, 200, 50)
	colYellow = color.RGB(255, 255, 100)
	colRed    = color.RGB(255, 100, 100)
)

type (
	CobraCommandFuncE     = func(*cobra.Command, []string) error
	CobraCommandFunc      = func(*cobra.Command, []string)
	FMonCommandFunc       = func(*configure.FilesystemMonitor, *cobra.Command, []string)
	FMonObjectCommandFunc = func(*configure.FilesystemMonitor, *configure.MonitoredObject, *cobra.Command, []string)
)

func preloadConfig() (*configure.FilesystemMonitor, CobraCommandFunc, func(FMonCommandFunc) CobraCommandFunc) {
	var conf = new(configure.FilesystemMonitor)
	var preload = func(cmd *cobra.Command, args []string) {
		config, err := configure.Read()
		if err != nil {
			colRed.Println(configure.ErrConfigRead)
			fmt.Println(err)
			os.Exit(1)
		}

		*conf = *config
	}
	var commandWithConf = func(fn FMonCommandFunc) CobraCommandFunc {
		return func(c *cobra.Command, s []string) {
			fn(conf, c, s)
		}
	}
	return conf, preload, commandWithConf
}

// func commandChain(fns ...CobraCommandFunc) CobraCommandFunc {
// 	return func(cmd *cobra.Command, args []string) {
// 		for _, fn := range fns {
// 			fn(cmd, args)
// 		}
// 	}
// }
