package main

// In phiếu qua máy chủ: điện thoại (app Android) không in trực tiếp được, nên gửi nội dung phiếu về máy tính chạy
// Digital Shop; máy tính mở một cửa sổ trình duyệt ẩn danh ở chế độ in thẳng (--kiosk-printing) để in ra máy in mặc định
// của Windows rồi tự đóng. Máy in phiếu cần được đặt làm máy in mặc định của Windows.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type PrintReq struct {
	HTML  string `json:"html"`
	Paper string `json:"paper"` // 58 | 80 | a5 | a4
}

const printCSS = `html,body{margin:0;background:#fff}
#print{display:block;position:absolute;left:0;top:0;color:#000;background:#fff;font:12px/1.35 Arial,"Segoe UI",sans-serif}
#print .rc{width:72mm;padding:3mm 2mm}
#print .rc.p58{width:48mm;font-size:11px}
#print .rc.a5{width:140mm;padding:8mm;font-size:13px}
#print .pdoc{width:190mm;padding:8mm;font-size:12px}
#print .pdoc h2{font-size:18px}
#print .pdoc table.pt{border:1px solid #000}
#print .pdoc table.pt th,#print .pdoc table.pt td{border:1px solid #000;padding:3px 5px;vertical-align:top}
#print .pdoc table.pt td:nth-child(n+3){min-width:0}
#print .pdoc tr.tot td{background:#eee}
#print .pdoc table.sig{margin-top:18px}
#print .pdoc table.sig td{height:30mm;width:33%}
#print .pdoc .ps{font-size:.85em}
#print .c{text-align:center}
#print .shop{font-size:15px;font-weight:800;text-transform:uppercase}
#print hr{border:0;border-top:1px dashed #000;margin:5px 0}
#print table{width:100%;border-collapse:collapse;font-size:inherit}
#print td{padding:1px 0;border:0;vertical-align:top}
#print .r{text-align:right}
#print .big{font-size:15px;font-weight:800}
#print .qr{width:30mm;height:30mm;margin:4px auto;display:block}
@media print{@page{margin:0}}`

var (
	reBlock  = regexp.MustCompile(`(?is)<\s*(script|iframe|object|embed|style)\b.*?</\s*(script|iframe|object|embed|style)\s*>`)
	reScript = regexp.MustCompile(`(?is)<\s*/?\s*(script|iframe|object|embed|style|link|meta|base|form)\b[^>]*>`)
	reOn     = regexp.MustCompile(`(?i)\s(on\w+|srcdoc|formaction)\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	reURL    = regexp.MustCompile(`(?i)\s(href|src)\s*=\s*("[^"]*"|'[^']*')`)
)

func cleanPrintHTML(h string) string {
	h = reBlock.ReplaceAllString(h, "")
	h = reScript.ReplaceAllString(h, "")
	h = reOn.ReplaceAllString(h, "")
	h = reURL.ReplaceAllStringFunc(h, func(m string) string {
		l := strings.ToLower(m)
		if strings.Contains(l, `"data:image/`) || strings.Contains(l, `'data:image/`) { return m }
		return ""
	})
	return h
}

// printPage dựng trang HTML tự in. Mã nonce chỉ cho phép đoạn script của chương trình chạy.
func printPage(body, paper string) string {
	nb := make([]byte, 12)
	rand.Read(nb)
	nonce := hex.EncodeToString(nb)
	w := 0
	switch paper {
	case "58":
		w = 58
	case "80":
		w = 80
	}
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>In phiếu</title>
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:; script-src 'nonce-%s'">
<style>%s</style></head><body><div id="print">%s</div>
<script nonce="%s">
(function(){var W=%d,P=document.getElementById('print');
if(W){var h=P.getBoundingClientRect().height,mm=Math.ceil(h*25.4/96)+2,s=document.createElement('style');s.textContent='@page{size:'+W+'mm '+mm+'mm;margin:0}';document.head.appendChild(s);}
window.onafterprint=function(){setTimeout(function(){window.close()},500)};
setTimeout(function(){window.print()},300);setTimeout(function(){window.close()},20000);})();
</script></body></html>`, nonce, printCSS, body, nonce, w)
}

func printBrowser() string {
	if p := os.Getenv("DS_PRINT_BROWSER"); p != "" { return p }
	if runtime.GOOS != "windows" { return "" }
	for _, c := range []string{
		os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`,
		os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application\msedge.exe`,
		os.Getenv("ProgramFiles") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("ProgramFiles(x86)") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("LocalAppData") + `\Google\Chrome\Application\chrome.exe`,
	} {
		if _, err := os.Stat(c); err == nil { return c }
	}
	return ""
}

type printer struct {
	mu   sync.Mutex // in lần lượt từng phiếu
	dir  string
	last string
	at   time.Time
}

var prn = &printer{}

// run in một phiếu (chặn tới khi cửa sổ in đóng, tối đa 30 giây).
func (p *printer) run(body, paper string) error {
	br := printBrowser()
	if br == "" { return uerr("Máy tính chạy Digital Shop chưa có trình duyệt Edge/Chrome để in. Hãy cài Edge hoặc Chrome trên máy đó.") }
	p.mu.Lock()
	defer p.mu.Unlock()
	dir, err := os.MkdirTemp("", "dsprint")
	if err != nil { return err }
	defer os.RemoveAll(dir)
	f := filepath.Join(dir, "phieu.html")
	if err := os.WriteFile(f, []byte(printPage(body, paper)), 0o600); err != nil { return err }
	u := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(f), "/")}).String()
	cmd := exec.Command(br, "--user-data-dir="+filepath.Join(dir, "profile"), "--kiosk-printing", "--app="+u, "--window-size=360,240",
		"--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-sync", "--disable-gpu")
	if err := cmd.Start(); err != nil { return uerr("Không mở được trình duyệt để in: %v", err) }
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		<-done
	}
	return nil
}

func (s *Server) printRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/print", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		off := false
		s.db.Read(func(st *State) { off = st.Settings.NoRemotePrint })
		if off { failErr(w, uerr("Máy tính đang tắt chức năng nhận lệnh in từ thiết bị khác (Cài đặt → In phiếu)")); return }
		r.Body = http.MaxBytesReader(w, r.Body, 600<<10)
		var req PrintReq
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		body := strings.TrimSpace(cleanPrintHTML(req.HTML))
		if body == "" { failErr(w, uerr("Không có nội dung để in")); return }
		if printBrowser() == "" { failErr(w, uerr("Máy tính chạy Digital Shop chưa có Edge/Chrome để in. Hãy cài trên máy đó.")); return }
		go func() {
			if err := prn.run(body, req.Paper); err != nil { fmt.Println("In phiếu lỗi:", html.EscapeString(err.Error())) }
		}()
		writeJSON(w, map[string]bool{"ok": true})
	}))
}
