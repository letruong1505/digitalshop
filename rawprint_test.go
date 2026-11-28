package main

import (
	"os"
	"path/filepath"
	"image"
	"image/color"
	"testing"
)

func TestRasterEscpos(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 800, 600))
	for i := range img.Pix { img.Pix[i] = 255 }
	for y := 100; y < 140; y++ { for x := 0; x < 800; x++ { img.Set(x, y, color.Black) } }
	rows, h := rasterize(img, 576)
	if h == 0 || len(rows[0]) != 72 { t.Fatalf("h=%d", h) }
	if h < 100 || h > 130 { t.Fatalf("cat day trang sai: %d", h) }
	d := escposBitmap(rows, 576, 3, true, true)
	if d[0] != 0x1B || d[1] != 0x40 || d[2] != 0x1D || d[3] != 0x76 || d[4] != 0x30 { t.Fatal("dau lenh") }
	if d[len(d)-9] != 0x1D || d[len(d)-5] != 0x1B || d[len(d)-4] != 0x70 { t.Fatalf("duoi lenh % x", d[len(d)-9:]) }
	if _, h2 := rasterize(image.NewNRGBA(image.Rect(0, 0, 100, 100)), 384); h2 != 0 { t.Fatal("anh trong phai rong") }
}

func TestRawPrintRender(t *testing.T) {
	if os.Getenv("DS_PRINT_BROWSER") == "" { t.Skip("cần DS_PRINT_BROWSER") }
	out := filepath.Join(t.TempDir(), "o.bin")
	t.Setenv("DS_PRINT_RAWFILE", out)
	body := `<div class="rc"><div class="c shop">Tạp hóa Nam Linh</div><hr><table><tr><td>Mì Hảo Hảo tôm chua cay</td><td class="r">9.000</td></tr></table><div class="big r">TỔNG 9.000</div></div>`
	if err := rawPrint(body, "80", "x", 2, true, false); err != nil { t.Fatal(err) }
	b, _ := os.ReadFile(out)
	if len(b) < 1000 { t.Fatalf("ít dữ liệu: %d", len(b)) }
	os.WriteFile("/tmp/raw_out.bin", b, 0o600)
}
