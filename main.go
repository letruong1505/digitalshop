package main

import (
	"embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed web
var webFS embed.FS

const (
	AppName    = "Digital Shop"
	AppVersion = "3.1.8"
	// TermsVersion: tăng khi sửa web/legal/DieuKhoanSuDung.txt
	TermsVersion = "1.1"
)

func main() {
	if os.Getenv("DS_UPDATED") != "" { time.Sleep(4 * time.Second) } // bản cũ còn đang tắt
	updCleanup()
	dir := dataDir()
	setupLog(dir)
	defer func() {
		if r := recover(); r != nil {
			fatal(fmt.Sprint("Phần mềm gặp lỗi và phải dừng: ", r, "\n\nDữ liệu đã bán vẫn được giữ. Hãy mở lại Digital Shop."))
		}
	}()
	ln, port, running := listen()
	if running {
		say("Digital Shop đang chạy rồi, mở lại cửa sổ bán hàng...")
		openBrowser("http://localhost:" + strconv.Itoa(port))
		time.Sleep(2 * time.Second)
		return
	}
	if ln == nil {
		fatal("Không mở được cổng mạng 8686–8690.\n\nCó thể một chương trình khác đang dùng các cổng này. Hãy khởi động lại máy rồi mở lại Digital Shop.")
		return
	}
	db, err := OpenDB(dir)
	if err != nil {
		fatal("Lỗi mở dữ liệu: " + err.Error() + "\n\nThư mục dữ liệu: " + dir + "\n\nCó thể khôi phục từ bản sao lưu trong thư mục SaoLuu.")
		return
	}
	if !db.HasBackupToday() { db.Backup("") }
	go db.SyncExternal(false)
	gdt.Init(dir)
	licInit(dir)
	licInfoFn = func() (shop, phone, addr string) {
		db.Read(func(st *State) { shop, phone, addr = st.Settings.ShopName, st.Settings.Phone, st.Settings.Addr })
		return
	}
	srv := NewServer(db, port)
	go srv.payFeedLoop()
	url := "http://localhost:" + strconv.Itoa(port)
	lan := lanAddrs(port)
	say("==============================================")
	say("  DIGITAL SHOP " + AppVersion + " — phần mềm bán hàng & quản lý kho")
	say("  Trên máy này:      " + url)
	for _, a := range lan { say("  Điện thoại/máy khác (cùng wifi): " + a) }
	say("  Dữ liệu lưu tại:   " + dir)
	say("==============================================")
	if os.Getenv("TAPHOAX_NOBROWSER") == "" { go openBrowser(url) }
	go func() {
		n := 0
		for range time.Tick(2 * time.Minute) {
			n++
			if n%15 == 0 && !db.HasBackupToday() { db.Backup("") }
			db.SyncExternal(false) // USB vừa cắm vào sẽ được chép trong vòng 2 phút
		}
	}()
	hs := &http.Server{Handler: srv, ReadHeaderTimeout: 20 * time.Second}
	go hs.Serve(ln)
	if hasTray {
		lines := []string{"Trên máy này: " + url}
		for i, a := range lan {
			if i == 2 { break }
			lines = append(lines, "Điện thoại (cùng wifi): "+a)
		}
		go runTray(TrayOpts{
			Title:   AppName + " " + AppVersion + " — đang chạy",
			Lines:   lines,
			Balloon: "Phần mềm chạy nền ở biểu tượng này. Bấm để mở cửa sổ bán hàng, chuột phải để tắt.",
			OnOpen:  func() { openBrowser(url) },
			OnFolder: func() { openFolder(dir) },
			OnQuit:  srv.Quit,
			Display: func() {
				var k string
				db.Read(func(st *State) { k = st.Settings.DisplayKey })
				openBrowserPlain(url + "/khach.html?k=" + k)
			},
		})
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-srv.quit:
	}
	say("Đang lưu dữ liệu và tắt...")
	trayRemove()
	hs.Close()
	db.Close()
}

var logw io.Writer = os.Stdout

// setupLog ghi nhật ký ra file TapHoaX.log trong thư mục dữ liệu (bản Windows không có cửa sổ đen để in ra).
func setupLog(dir string) {
	p := filepath.Join(dir, "TapHoaX.log")
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 { os.Rename(p, p+".cu") }
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil { return }
	if runtime.GOOS == "windows" { logw = f } else { logw = io.MultiWriter(os.Stdout, f) }
	fmt.Fprintln(f, "\n--- "+time.Now().Format("02/01/2006 15:04:05")+" khởi động "+AppVersion)
}

func say(s string) { fmt.Fprintln(logw, s) }

func fatal(msg string) {
	say("LỖI: " + msg)
	alert(AppName, msg)
	if runtime.GOOS != "windows" { pause() }
}

func pause() {
	fmt.Println("Nhấn Enter để đóng.")
	io.ReadAll(io.LimitReader(os.Stdin, 1))
}

func dataDir() string {
	if d := os.Getenv("TAPHOAX_DATA"); d != "" { return d }
	exe, err := os.Executable()
	if err == nil {
		// Bản cài đặt ghi sẵn nơi lưu dữ liệu (ProgramData) vào datadir.txt cạnh file exe.
		if b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "datadir.txt")); err == nil {
			if d := strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff")); d != "" {
				if os.MkdirAll(d, 0o755) == nil { return d }
			}
		}
		d := filepath.Join(filepath.Dir(exe), "DuLieu_TapHoaX")
		if os.MkdirAll(d, 0o755) == nil {
			test := filepath.Join(d, ".ghi-thu")
			if f, err := os.Create(test); err == nil { f.Close(); os.Remove(test); return d }
		}
	}
	base, err := os.UserConfigDir()
	if err != nil { base = "." }
	d := filepath.Join(base, "TapHoaX")
	os.MkdirAll(d, 0o755)
	return d
}

func listen() (net.Listener, int, bool) {
	for port := 8686; port <= 8690; port++ {
		ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err == nil { return ln, port, false }
		cl := &http.Client{Timeout: 2 * time.Second}
		if resp, err := cl.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/api/ping"); err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(b) == "taphoax" { return nil, port, true }
		}
	}
	return nil, 0, false
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		// Mở dạng cửa sổ ứng dụng (không thanh địa chỉ) bằng Edge hoặc Chrome nếu có.
		cands := []string{
			os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`,
			os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application\msedge.exe`,
			os.Getenv("ProgramFiles") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("ProgramFiles(x86)") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("LocalAppData") + `\Google\Chrome\Application\chrome.exe`,
		}
		for _, c := range cands {
			if _, err := os.Stat(c); err == nil {
				if exec.Command(c, "--app="+url, "--start-maximized").Start() == nil { return }
			}
		}
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

// openBrowserPlain mở trang trong trình duyệt mặc định (cửa sổ thường, kéo được sang màn hình thứ hai).
func openBrowserPlain(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

func openFolder(dir string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("explorer", dir).Start()
	case "darwin":
		exec.Command("open", dir).Start()
	default:
		exec.Command("xdg-open", dir).Start()
	}
}
