//go:build !unix

package codexapp

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
