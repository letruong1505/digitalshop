//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

const updSupported = true

// updSwap: đổi tên exe đang chạy thành .old (Windows cho phép), đặt file mới vào chỗ cũ.
func updSwap(exe, tmp string) error {
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil { return uerr("Không thay được chương trình (%v). Hãy cài lại bằng bộ cài mới nhất.", err) }
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return uerr("Không đặt được bản mới (%v).", err)
	}
	return nil
}

// updRestart mở bản mới tách riêng khỏi tiến trình này; bản mới đợi vài giây cho bản cũ tắt hẳn.
func updRestart() {
	exe, err := os.Executable()
	if err != nil { return }
	c := exec.Command(exe)
	c.Env = append(os.Environ(), "DS_UPDATED=1")
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08 | 0x200} // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	c.Start()
}
