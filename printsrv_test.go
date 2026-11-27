package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanPrintHTML(t *testing.T) {
	in := `<div class="rc" onclick="x()"><b>Hi</b><script>alert(1)</script><img src="http://evil/x.png"><img src="data:image/png;base64,AAA" onerror='y()'><iframe src=a></iframe><link rel=stylesheet href=x><a href="http://e">l</a></div>`
	out := cleanPrintHTML(in)
	for _, bad := range []string{"script", "alert", "onclick", "onerror", "iframe", "evil", "<link", "http://e"} {
		if strings.Contains(strings.ToLower(out), bad) { t.Fatalf("còn %q: %s", bad, out) }
	}
	if !strings.Contains(out, "<b>Hi</b>") || !strings.Contains(out, "data:image/png") { t.Fatal(out) }
	pg := printPage(out, "80")
	if !strings.Contains(pg, "script-src 'nonce-") || !strings.Contains(pg, "W=80") { t.Fatal("page") }
}

// Giả lập trình duyệt: ghi lại tham số và đường dẫn trang để kiểm tra phiếu đã được dựng.
func TestPrintRun(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log.txt")
	fake := filepath.Join(dir, "fakebrowser.sh")
	os.WriteFile(fake, []byte("#!/bin/sh\nfor a in \"$@\"; do echo \"$a\" >> "+log+"; case \"$a\" in --app=file://*) cat \"${a#--app=file://}\" >> "+log+";; esac; done\n"), 0o755)
	t.Setenv("DS_PRINT_BROWSER", fake)
	if err := prn.run(`<div class="rc">PHIẾU THỬ 123</div>`, "58"); err != nil { t.Fatal(err) }
	b, _ := os.ReadFile(log)
	s := string(b)
	for _, w := range []string{"--kiosk-printing", "PHIẾU THỬ 123", "W=58"} {
		if !strings.Contains(s, w) { t.Fatalf("thiếu %q trong %s", w, s) }
	}
	_ = time.Second
}
