package main

// In thẳng ra máy in nhiệt (ESC/POS) mà không mở cửa sổ trình duyệt nào:
// 1) trình duyệt Edge/Chrome chạy ẩn (headless) chụp phiếu thành ảnh,
// 2) đổi ảnh thành bitmap đen trắng đúng bề rộng máy in (384 điểm cho K58, 576 điểm cho K80),
// 3) gửi lệnh ESC/POS (có cắt giấy, mở két nếu bật) thẳng vào máy in Windows ở chế độ RAW.
// Làm bằng ảnh nên tiếng Việt có dấu in đúng trên mọi máy in, không phụ thuộc bảng mã của máy.

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func rawDots(paper string) (dots, cssW int) {
	if paper == "58" { return 384, 219 }
	return 576, 302 // K80: vùng in 72 mm
}

const rawScale = 2

// rawPage: trang HTML không có script, bề rộng đúng bằng bề rộng giấy.
func rawPage(body, paper string) string {
	_, w := rawDots(paper)
	fs := 12
	if paper == "58" { fs = 11 }
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:">
<style>%s
html,body{width:%dpx;margin:0;background:#fff;overflow:hidden}
#print{width:%dpx;font-size:%dpx;box-sizing:border-box}
#print .rc,#print .rc.p58,#print .rc.a5{width:%dpx;padding:4px 6px;box-sizing:border-box;font-size:%dpx}
#print .qr{width:%dpx;height:%dpx}</style></head><body><div id="print">%s</div></body></html>`, printCSS, w, w, fs, w, fs, w/2, w/2, body)
}

// rawRender chụp trang thành ảnh PNG qua trình duyệt ẩn.
func rawRender(body, paper string) (image.Image, error) {
	br := printBrowser()
	if br == "" { return nil, uerr("Máy tính chưa có Edge/Chrome để dựng phiếu. Hãy cài Edge hoặc Chrome.") }
	dir, err := os.MkdirTemp("", "dsraw")
	if err != nil { return nil, err }
	defer os.RemoveAll(dir)
	f := filepath.Join(dir, "phieu.html")
	if err := os.WriteFile(f, []byte(rawPage(body, paper)), 0o600); err != nil { return nil, err }
	out := filepath.Join(dir, "phieu.png")
	_, w := rawDots(paper)
	u := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(f), "/")}).String()
	cmd := exec.Command(br, "--headless=new", "--user-data-dir="+filepath.Join(dir, "profile"), "--screenshot="+out,
		fmt.Sprintf("--window-size=%d,%d", w, 7000), fmt.Sprintf("--force-device-scale-factor=%d", rawScale), "--hide-scrollbars",
		"--default-background-color=ffffffff", "--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-sync", "--disable-gpu", "--no-sandbox", u)
	hideWindow(cmd)
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil { return nil, uerr("Không chạy được trình duyệt để dựng phiếu: %v", err) }
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		<-done
	}
	b, err := os.ReadFile(out)
	if err != nil { return nil, uerr("Trình duyệt ẩn không tạo được ảnh phiếu (%s). Thử đóng hết cửa sổ Edge/Chrome rồi in lại", filepath.Base(br)) }
	return png.Decode(bytes.NewReader(b))
}

// rasterize: ảnh → bitmap 1 bit rộng `dots` điểm, cắt bỏ phần trắng phía dưới.
func rasterize(img image.Image, dots int) (rows [][]byte, h int) {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 { return nil, 0 }
	dh := sh * dots / sw
	if dh < 1 { dh = 1 }
	gray := func(x, y int) float64 { // trung bình vùng nguồn tương ứng một điểm đích
		x0, x1 := x*sw/dots, (x+1)*sw/dots
		y0, y1 := y*sh/dh, (y+1)*sh/dh
		if x1 <= x0 { x1 = x0 + 1 }
		if y1 <= y0 { y1 = y0 + 1 }
		var sum float64
		n := 0
		for yy := y0; yy < y1 && yy < sh; yy++ {
			for xx := x0; xx < x1 && xx < sw; xx++ {
				c := color.NRGBAModel.Convert(img.At(b.Min.X+xx, b.Min.Y+yy)).(color.NRGBA)
				a := float64(c.A) / 255
				l := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
				sum += l*a + 255*(1-a)
				n++
			}
		}
		if n == 0 { return 255 }
		return sum / float64(n)
	}
	stride := (dots + 7) / 8
	last := -1
	all := make([][]byte, dh)
	for y := 0; y < dh; y++ {
		row := make([]byte, stride)
		for x := 0; x < dots; x++ {
			if gray(x, y) < 170 { row[x/8] |= 0x80 >> uint(x%8); last = y }
		}
		all[y] = row
	}
	if last < 0 { return nil, 0 }
	return all[:last+1], last + 1
}

// escposBitmap dựng chuỗi lệnh: khởi tạo, ảnh (chia dải 100 dòng), đẩy giấy, cắt, mở két.
func escposBitmap(rows [][]byte, dots, feed int, cut, drawer bool) []byte {
	var o bytes.Buffer
	o.Write([]byte{0x1B, 0x40}) // ESC @
	stride := (dots + 7) / 8
	for i := 0; i < len(rows); i += 100 {
		j := i + 100
		if j > len(rows) { j = len(rows) }
		n := j - i
		o.Write([]byte{0x1D, 0x76, 0x30, 0x00, byte(stride), byte(stride >> 8), byte(n), byte(n >> 8)}) // GS v 0
		for _, r := range rows[i:j] { o.Write(r) }
	}
	if feed < 0 { feed = 0 }
	if feed > 30 { feed = 30 }
	o.Write([]byte{0x1B, 0x64, byte(feed)}) // ESC d n
	if cut { o.Write([]byte{0x1D, 0x56, 0x42, 0x00}) } // GS V 66 0: đẩy giấy tới vị trí cắt rồi cắt
	if drawer { o.Write([]byte{0x1B, 0x70, 0x00, 0x19, 0xFA}) } // ESC p: xung mở két
	return o.Bytes()
}

// rawPrint in một phiếu K58/K80 thẳng ra máy in `printer`.
func rawPrint(body, paper, printer string, feed int, cut, drawer bool) error {
	img, err := rawRender(body, paper)
	if err != nil { return err }
	dots, _ := rawDots(paper)
	rows, h := rasterize(img, dots)
	if h == 0 { return uerr("Phiếu trống, không có gì để in") }
	data := escposBitmap(rows, dots, feed, cut, drawer)
	if p := os.Getenv("DS_PRINT_RAWFILE"); p != "" { return os.WriteFile(p, data, 0o600) } // dùng khi kiểm thử
	return sendRaw(printer, data)
}
