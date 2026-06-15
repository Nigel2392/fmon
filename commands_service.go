package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Nigel2392/fmon/watcher"
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

type (
	FMonServiceFunc = func(service.Service, service.Config, *cobra.Command, []string) error
)

func ServiceExecutable() (string, error) {
	var executableName = configure.SETUP.NameBase
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	confDir, err := configure.Dir()
	if err != nil {
		return "", err
	}

	var targetExePath = filepath.Join(confDir, executableName)
	return targetExePath, nil
}

func preloadServiceFunc(c *service.Config) (CobraCommandFuncE, func(FMonServiceFunc) CobraCommandFuncE) {
	if c == nil {
		panic("service config is required.")
	}

	var svc service.Service
	var prerunE = func(cmd *cobra.Command, args []string) (err error) {
		targetExePath, err := ServiceExecutable()
		if err != nil {
			return err
		}

		c.Executable = targetExePath

		var w = watcher.NewWatcher(
			filepath.Dir(c.Executable),
		)
		svc, err = service.New(w, c)
		if err != nil {
			return err
		}

		return nil
	}

	var exec = func(fsf FMonServiceFunc) CobraCommandFuncE {
		return func(cmd *cobra.Command, args []string) error {
			var cpy = service.Config{
				Name:             c.Name,
				DisplayName:      c.DisplayName,
				Description:      c.Description,
				UserName:         c.UserName,
				Arguments:        slices.Clone(c.Dependencies),
				Executable:       c.Executable,
				Dependencies:     slices.Clone(c.Dependencies),
				WorkingDirectory: c.WorkingDirectory,
				ChRoot:           c.ChRoot,
				Option:           maps.Clone(c.Option),
				EnvVars:          maps.Clone(c.EnvVars),
			}

			var err = fsf(svc, cpy, cmd, args)
			if errors.Is(err, os.ErrPermission) {
				return fmt.Errorf(
					"Please retry this command while running as an administrator: %w",
					err,
				)
			}
			return err
		}
	}

	return prerunE, exec
}

func commandServiceInstall(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	status, err := svc.Status()
	if err != nil && !errors.Is(err, service.ErrNotInstalled) {
		return fmt.Errorf(
			"Error while retrieving service status: %w",
			err,
		)
	}

	if status != service.StatusUnknown {
		return errors.New("Service is already installed.")
	}

	if _, err := os.Stat(cnf.Executable); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		colBlue.Printf("Copying executable to: ")
		fmt.Println(cnf.Executable)

		currentExe, err := os.Executable()
		if err != nil {
			return err
		}

		currentExeF, err := os.Open(currentExe)
		if err != nil {
			return err
		}

		targetExe, err := os.Create(cnf.Executable)
		if err != nil {
			return err
		}

		if _, err := io.Copy(targetExe, currentExeF); err != nil {
			return err
		}
	}

	colBlue.Printf("Installing service %q\n", svc.String())
	return svc.Install()
}

func commandServiceUninstall(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	colRed.Printf("Uninstalling service %q\n", svc.String())
	return svc.Uninstall()
}

func commandServiceStart(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	return svc.Start()
}

func commandServiceRun(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	return svc.Run()
}

func commandServiceStop(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	return svc.Stop()
}

func commandServiceStatus(svc service.Service, cnf service.Config, cmd *cobra.Command, args []string) error {
	var status, err = svc.Status()
	var serviceNotInstalled = errors.Is(err, service.ErrNotInstalled)
	if err != nil && !errors.Is(err, service.ErrNotInstalled) {
		return err
	}

	colBlue.Print("Service Name: ")
	fmt.Println(svc.String())

	if cnf.Description != "" {
		colBlue.Print("Description: ")
		fmt.Println(cnf.Description)
	}

	var p string
	if cnf.Executable != "" {
		p, err = filepath.Abs(cnf.Executable)
	} else {
		p, err = os.Executable()
	}
	if err != nil {
		return fmt.Errorf("Failed to retrieve executable location: %w", err)
	}
	colBlue.Print("Executable: ")
	fmt.Println(p)

	if len(cnf.Dependencies) != 0 {
		colBlue.Println("Dependencies:")
		for _, dep := range cnf.Dependencies {
			fmt.Printf("  - %s\n", dep)
		}
	}

	if cnf.UserName != "" {
		colBlue.Print("User: ")
		fmt.Println(cnf.UserName)
	}

	switch {
	case status == service.StatusRunning:
		colYellow.Println("RUNNING")
	case status == service.StatusStopped:
		colYellow.Println("STOPPED")
	case serviceNotInstalled:
		colRed.Println("NOT INSTALLED")
	default:
		colRed.Println("UNKNOWN")
	}

	return nil
}
