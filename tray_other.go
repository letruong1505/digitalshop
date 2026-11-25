//go:build !windows

package main

import "fmt"

type TrayOpts struct {
	Title    string
	Lines    []string
	Balloon  string
	OnOpen   func()
	OnFolder func()
	OnQuit   func()
	Display  func()
}

const hasTray = false

func runTray(o TrayOpts)        {}
func trayRemove()               {}
func alert(title, msg string)   { fmt.Println(title+":", msg) }
