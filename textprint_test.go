package main

import ("os";"path/filepath";"strings";"testing")

func TestTextFallback(t *testing.T) {
	body := `<div class="rc"><div class="c shop">Tạp hóa Đức Thắng</div><div class="c">Hóa đơn HD001</div><hr><table><tr><td>Mì tôm Hảo Hảo thùng 30 gói đặc biệt</td><td class="r">150.000</td></tr><tr><td>2 × 5.000</td><td class="r">10.000</td></tr></table><hr><div class="big">Tổng cộng 160.000đ</div></div>`
	lines := htmlToLines(body, 32)
	j := strings.Join(lines, "\n")
	for _, w := range []string{"Tap hoa Duc Thang", "150.000", "2 x 5.000", "Tong cong 160.000d"} {
		if !strings.Contains(j, w) { t.Fatalf("thiếu %q trong:\n%s", w, j) }
	}
	for _, l := range lines { if len(l) > 32 { t.Fatalf("dòng dài %d: %q", len(l), l) } }
	f := filepath.Join(t.TempDir(), "raw")
	t.Setenv("DS_PRINT_BROWSER", filepath.Join(t.TempDir(), "khong-co"))
	t.Setenv("DS_PRINT_RAWFILE", f)
	if err := rawPrint(body, "58", "x", 2, true, false); err != nil { t.Fatal(err) }
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "Tap hoa") { t.Fatalf("không in chữ dự phòng: %q", b) }
	t.Log("\n" + j)
}
