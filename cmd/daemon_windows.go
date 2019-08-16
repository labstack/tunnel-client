// +build windows
package cmd

import "syscall"

var (
	sysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
)
