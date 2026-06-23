//go:build !windows
// +build !windows

package watcher

import "os/exec"

func hideCommandWindow(cmd *exec.Cmd) {

}
