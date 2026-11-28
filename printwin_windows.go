//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	winspool          = syscall.NewLazyDLL("winspool.drv")
	pOpenPrinter      = winspool.NewProc("OpenPrinterW")
	pClosePrinter     = winspool.NewProc("ClosePrinter")
	pStartDocPrinter  = winspool.NewProc("StartDocPrinterW")
	pEndDocPrinter    = winspool.NewProc("EndDocPrinter")
	pStartPagePrinter = winspool.NewProc("StartPagePrinter")
	pEndPagePrinter   = winspool.NewProc("EndPagePrinter")
	pWritePrinter     = winspool.NewProc("WritePrinter")
	pEnumPrinters     = winspool.NewProc("EnumPrintersW")
	pGetDefaultPrn    = winspool.NewProc("GetDefaultPrinterW")
)

func hideWindow(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} }

type docInfo1 struct {
	name, output, datatype *uint16
}

// sendRaw gửi nguyên dữ liệu (ESC/POS) vào máy in Windows, bỏ qua trình điều khiển in.
func sendRaw(printer string, data []byte) error {
	name, err := syscall.UTF16PtrFromString(printer)
	if err != nil { return err }
	var h syscall.Handle
	if r, _, e := pOpenPrinter.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&h)), 0); r == 0 {
		return uerr("Không mở được máy in “%s”: %v", printer, e)
	}
	defer pClosePrinter.Call(uintptr(h))
	title, _ := syscall.UTF16PtrFromString("Digital Shop - phiếu")
	raw, _ := syscall.UTF16PtrFromString("RAW")
	di := docInfo1{name: title, datatype: raw}
	if r, _, e := pStartDocPrinter.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&di))); r == 0 {
		return uerr("Máy in “%s” không nhận lệnh in: %v", printer, e)
	}
	defer pEndDocPrinter.Call(uintptr(h))
	pStartPagePrinter.Call(uintptr(h))
	defer pEndPagePrinter.Call(uintptr(h))
	for off := 0; off < len(data); {
		n := len(data) - off
		if n > 32768 { n = 32768 }
		var wrote uint32
		if r, _, e := pWritePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&data[off])), uintptr(n), uintptr(unsafe.Pointer(&wrote))); r == 0 {
			return uerr("Gửi dữ liệu tới máy in “%s” bị lỗi: %v", printer, e)
		}
		if wrote == 0 { return uerr("Máy in “%s” không nhận dữ liệu", printer) }
		off += int(wrote)
	}
	return nil
}

// listPrinters: tên các máy in đã cài trên Windows và máy in mặc định.
func listPrinters() (names []string, def string) {
	const flags = 0x2 | 0x4 // LOCAL | CONNECTIONS
	var need, n uint32
	pEnumPrinters.Call(flags, 0, 4, 0, 0, uintptr(unsafe.Pointer(&need)), uintptr(unsafe.Pointer(&n)))
	if need > 0 {
		buf := make([]byte, need)
		if r, _, _ := pEnumPrinters.Call(flags, 0, 4, uintptr(unsafe.Pointer(&buf[0])), uintptr(need), uintptr(unsafe.Pointer(&need)), uintptr(unsafe.Pointer(&n))); r != 0 {
			sz := unsafe.Sizeof(uintptr(0)) * 2 + 4 // PRINTER_INFO_4: LPWSTR, LPWSTR, DWORD (căn lề theo con trỏ)
			sz = (sz + unsafe.Sizeof(uintptr(0)) - 1) &^ (unsafe.Sizeof(uintptr(0)) - 1)
			for i := uintptr(0); i < uintptr(n); i++ {
				p := *(**uint16)(unsafe.Pointer(&buf[i*sz]))
				if p != nil { names = append(names, syscall.UTF16ToString((*[512]uint16)(unsafe.Pointer(p))[:])) }
			}
		}
	}
	var sz uint32 = 260
	b := make([]uint16, sz)
	if r, _, _ := pGetDefaultPrn.Call(uintptr(unsafe.Pointer(&b[0])), uintptr(unsafe.Pointer(&sz))); r != 0 { def = syscall.UTF16ToString(b) }
	return
}
