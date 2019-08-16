// +build darwin linux

package cmd

import "syscall"

var (
	sysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
)
