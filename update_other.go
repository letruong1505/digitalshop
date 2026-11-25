//go:build !windows

package main

const updSupported = false

func updSwap(exe, tmp string) error { return uerr("Tự cập nhật chỉ dùng trên Windows.") }
func updRestart()                    {}
