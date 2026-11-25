//go:build windows

package main

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	k32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDrives = k32.NewProc("GetLogicalDrives")
	procGetDriveTypeW    = k32.NewProc("GetDriveTypeW")
	procGetVolumeInfoW   = k32.NewProc("GetVolumeInformationW")
	procGetDiskFreeW     = k32.NewProc("GetDiskFreeSpaceExW")
)

// listDrives liệt kê USB và ổ cứng khác ổ hệ thống (Windows).
func listDrives() []Drive {
	out := []Drive{}
	mask, _, _ := procGetLogicalDrives.Call()
	sys := strings.ToUpper(strings.TrimSuffix(os.Getenv("SystemDrive"), "\\"))
	if sys == "" { sys = "C:" }
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 { continue }
		letter := string(rune('A'+i)) + ":"
		if letter == sys || letter == "A:" || letter == "B:" { continue }
		root := letter + `\`
		rp, _ := syscall.UTF16PtrFromString(root)
		t, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rp)))
		kind := ""
		switch t {
		case 2:
			kind = "usb"
		case 3:
			kind = "disk"
		default:
			continue // CD, ổ mạng, ổ RAM
		}
		var free, total, totalFree uint64
		r, _, _ := procGetDiskFreeW.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&totalFree)))
		if r == 0 { continue } // khe đọc thẻ trống
		name := make([]uint16, 261)
		procGetVolumeInfoW.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&name[0])), 261, 0, 0, 0, 0, 0)
		label := syscall.UTF16ToString(name)
		out = append(out, Drive{Path: root, Label: label, Kind: kind, FreeGB: float64(free) / 1e9, TotalGB: float64(total) / 1e9})
	}
	return append(out, cloudFolders()...)
}
