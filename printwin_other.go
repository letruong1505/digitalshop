//go:build !windows

package main

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}

func listPrinters() (names []string, def string) { return nil, "" }

func sendRaw(printer string, data []byte) error {
	return uerr("In thẳng ra máy in chỉ dùng được trên máy tính Windows")
}
