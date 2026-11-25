//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const onAndroid = false

func regQuery(args ...string) string {
	c := exec.Command("reg", append([]string{"query"}, args...)...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	b, _ := c.Output()
	return string(b)
}

func regValue(out, name string) string {
	for _, ln := range strings.Split(out, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 3 && strings.EqualFold(f[0], name) { return f[len(f)-1] }
	}
	return ""
}

// Mã gốc của máy: MachineGuid + số sê-ri ổ đĩa hệ thống.
func platformMachineRaw() string {
	guid := regValue(regQuery(`HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid"), "MachineGuid")
	var ser uint32
	root := os.Getenv("SystemDrive")
	if root == "" { root = "C:" }
	if p, err := syscall.UTF16PtrFromString(root + `\`); err == nil {
		kernel32.NewProc("GetVolumeInformationW").Call(uintptr(unsafe.Pointer(p)), 0, 0, uintptr(unsafe.Pointer(&ser)), 0, 0, 0, 0)
	}
	if guid == "" { h, _ := os.Hostname(); guid = "host-" + h }
	return "win|" + strings.ToLower(guid) + "|" + strconv.FormatUint(uint64(ser), 16)
}

const trialKey = `HKCU\Software\NamLinhGroup\DigitalShop`

func trialMarkGet() int64 {
	v := regValue(regQuery(trialKey, "/v", "t0"), "t0")
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func trialMarkSet(t int64) {
	c := exec.Command("reg", "add", trialKey, "/v", "t0", "/t", "REG_SZ", "/d", strconv.FormatInt(t, 10), "/f")
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	c.Run()
}
