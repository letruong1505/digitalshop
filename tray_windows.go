//go:build windows

package main

// Chạy nền không có cửa sổ đen: biểu tượng ở khay hệ thống (cạnh đồng hồ),
// bấm vào để mở cửa sổ bán hàng, chuột phải để xem địa chỉ, mở thư mục dữ liệu hoặc tắt.
// Chỉ dùng lời gọi hệ thống Windows, không cần thư viện ngoài.

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassEx     = user32.NewProc("RegisterClassExW")
	pCreateWindowEx      = user32.NewProc("CreateWindowExW")
	pDefWindowProc       = user32.NewProc("DefWindowProcW")
	pGetMessage          = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessage     = user32.NewProc("DispatchMessageW")
	pPostMessage         = user32.NewProc("PostMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	pAppendMenu          = user32.NewProc("AppendMenuW")
	pSetMenuDefaultItem  = user32.NewProc("SetMenuDefaultItem")
	pTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	pDestroyMenu         = user32.NewProc("DestroyMenu")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pGetCursorPos        = user32.NewProc("GetCursorPos")
	pLoadImage           = user32.NewProc("LoadImageW")
	pLoadIcon            = user32.NewProc("LoadIconW")
	pMessageBox          = user32.NewProc("MessageBoxW")
	pRegisterWindowMsg   = user32.NewProc("RegisterWindowMessageW")
	pShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
	pGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmDestroy   = 0x0002
	wmCommand   = 0x0111
	wmNull      = 0x0000
	wmClose     = 0x0010
	wmLBtnUp    = 0x0202
	wmLBtnDbl   = 0x0203
	wmRBtnUp    = 0x0205
	wmTray      = 0x8001 // WM_APP+1
	nimAdd      = 0
	nimModify   = 1
	nimDelete   = 2
	nifMessage  = 0x01
	nifIcon     = 0x02
	nifTip      = 0x04
	nifInfo     = 0x10
	niifInfo    = 0x01
	mfString    = 0x0000
	mfGrayed    = 0x0001
	mfSeparator = 0x0800
	tpmReturn   = 0x0100
	tpmRight    = 0x0002
	tpmBottom   = 0x0020

	idOpen    = 1
	idFolder  = 2
	idQuit    = 3
	idDisplay = 4
)

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	X, Y    int32
}

type TrayOpts struct {
	Title    string   // dòng đầu menu
	Lines    []string // các dòng thông tin (không bấm được)
	Balloon  string   // thông báo khi vừa chạy
	OnOpen   func()
	OnFolder func()
	OnQuit   func()
	Display  func() // mở màn hình khách (có thể nil)
}

var (
	trayMu   sync.Mutex
	trayHwnd uintptr
	trayNid  notifyIconData
	trayOpts TrayOpts
	trayTbc  uintptr // thông điệp TaskbarCreated: Explorer khởi động lại thì thêm lại biểu tượng
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func copyU16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) { u = u[:len(dst)]; u[len(u)-1] = 0 }
	copy(dst, u)
}

// alert hiện hộp thông báo (thay cho cửa sổ đen khi có lỗi).
func alert(title, msg string) {
	pMessageBox.Call(0, uintptr(unsafe.Pointer(u16(msg))), uintptr(unsafe.Pointer(u16(title))), 0x10|0x10000) // MB_ICONERROR | MB_SETFOREGROUND
}

func confirmBox(title, msg string) bool {
	r, _, _ := pMessageBox.Call(trayHwnd, uintptr(unsafe.Pointer(u16(msg))), uintptr(unsafe.Pointer(u16(title))), 0x04|0x20|0x100|0x10000) // YESNO | QUESTION | DEFBUTTON2 | SETFOREGROUND
	return r == 6
}

func addIcon(withBalloon bool) {
	nid := trayNid
	nid.UFlags = nifMessage | nifIcon | nifTip
	if withBalloon && trayOpts.Balloon != "" {
		nid.UFlags |= nifInfo
		copyU16(nid.SzInfo[:], trayOpts.Balloon)
		copyU16(nid.SzInfoTitle[:], AppName+" đang chạy")
		nid.DwInfoFlags = niifInfo
	}
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func showMenu(hwnd uintptr) {
	m, _, _ := pCreatePopupMenu.Call()
	if m == 0 { return }
	defer pDestroyMenu.Call(m)
	add := func(flags uintptr, id uintptr, text string) {
		pAppendMenu.Call(m, flags, id, uintptr(unsafe.Pointer(u16(text))))
	}
	add(mfString|mfGrayed, 0, trayOpts.Title)
	for _, l := range trayOpts.Lines { add(mfString|mfGrayed, 0, l) }
	add(mfSeparator, 0, "")
	add(mfString, idOpen, "Mở cửa sổ bán hàng")
	if trayOpts.Display != nil { add(mfString, idDisplay, "Mở màn hình cho khách") }
	add(mfString, idFolder, "Mở thư mục dữ liệu")
	add(mfSeparator, 0, "")
	add(mfString, idQuit, "Tắt "+AppName)
	pSetMenuDefaultItem.Call(m, idOpen, 0)
	var pt struct{ X, Y int32 }
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, tpmReturn|tpmRight|tpmBottom, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	pPostMessage.Call(hwnd, wmNull, 0, 0)
	switch cmd {
	case idOpen:
		go trayOpts.OnOpen()
	case idDisplay:
		go trayOpts.Display()
	case idFolder:
		go trayOpts.OnFolder()
	case idQuit:
		if confirmBox(AppName, "Tắt phần mềm bán hàng?\n\nMáy tính tiền, điện thoại và màn hình khách sẽ không dùng được cho đến khi mở lại.") {
			trayRemove()
			go trayOpts.OnQuit()
		}
	}
}

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch uint32(msg) {
	case wmTray:
		switch uint32(lp) & 0xFFFF {
		case wmLBtnUp, wmLBtnDbl:
			go trayOpts.OnOpen()
		case wmRBtnUp:
			showMenu(hwnd)
		}
		return 0
	case wmClose:
		trayRemove()
		pPostQuitMessage.Call(0)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	if trayTbc != 0 && msg == trayTbc {
		addIcon(false)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

// runTray tạo biểu tượng khay và chạy vòng nhận thông điệp; trả về khi biểu tượng bị gỡ.
func runTray(o TrayOpts) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	trayOpts = o
	hinst, _, _ := pGetModuleHandle.Call(0)
	cls := u16("TapHoaXTray")
	wc := wndClassEx{LpfnWndProc: syscall.NewCallback(wndProc), HInstance: hinst, LpszClassName: cls}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if r, _, _ := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 { return }
	hwnd, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16(AppName))), 0, 0, 0, 0, 0, 0, 0, hinst, 0)
	if hwnd == 0 { return }
	trayTbc, _, _ = pRegisterWindowMsg.Call(uintptr(unsafe.Pointer(u16("TaskbarCreated"))))
	// Biểu tượng số 1 trong file exe (installer/app.rc); 16×16 cho khay
	icon, _, _ := pLoadImage.Call(hinst, 1, 1, 16, 16, 0) // IMAGE_ICON
	if icon == 0 { icon, _, _ = pLoadIcon.Call(0, 32512) } // IDI_APPLICATION
	trayMu.Lock()
	trayHwnd = hwnd
	trayNid = notifyIconData{HWnd: hwnd, UID: 1, UCallbackMessage: wmTray, HIcon: icon}
	trayNid.CbSize = uint32(unsafe.Sizeof(trayNid))
	copyU16(trayNid.SzTip[:], AppName+" — bấm để mở, chuột phải để tắt")
	trayMu.Unlock()
	addIcon(true)
	var m winMsg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { break }
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// trayRemove gỡ biểu tượng (gọi khi tắt phần mềm từ bất kỳ đâu).
func trayRemove() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayHwnd == 0 { return }
	nid := trayNid
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	h := trayHwnd
	trayHwnd = 0
	pPostMessage.Call(h, wmClose, 0, 0)
}

const hasTray = true
