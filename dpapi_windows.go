//go:build windows

package main

// Mã hóa dữ liệu nhạy cảm (phiên đăng nhập cổng thuế) bằng DPAPI của Windows:
// chỉ đúng tài khoản Windows trên đúng máy này giải mã được.

import (
	"syscall"
	"unsafe"
)

var (
	crypt32       = syscall.NewLazyDLL("crypt32.dll")
	pProtectData   = crypt32.NewProc("CryptProtectData")
	pUnprotectData = crypt32.NewProc("CryptUnprotectData")
	pLocalFree     = kernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cb uint32
	pb *byte
}

func blobOf(b []byte) *dataBlob {
	if len(b) == 0 { return &dataBlob{} }
	return &dataBlob{cb: uint32(len(b)), pb: &b[0]}
}

func (d *dataBlob) bytes() []byte {
	if d.pb == nil || d.cb == 0 { return nil }
	return append([]byte(nil), unsafe.Slice(d.pb, d.cb)...)
}

const cryptprotectUIForbidden = 0x1

func protectData(b []byte) ([]byte, error) {
	var out dataBlob
	r, _, err := pProtectData.Call(uintptr(unsafe.Pointer(blobOf(b))), 0, 0, 0, 0, cryptprotectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 { return nil, err }
	defer pLocalFree.Call(uintptr(unsafe.Pointer(out.pb)))
	return out.bytes(), nil
}

func unprotectData(b []byte) ([]byte, error) {
	var out dataBlob
	r, _, err := pUnprotectData.Call(uintptr(unsafe.Pointer(blobOf(b))), 0, 0, 0, 0, cryptprotectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 { return nil, err }
	defer pLocalFree.Call(uintptr(unsafe.Pointer(out.pb)))
	return out.bytes(), nil
}
