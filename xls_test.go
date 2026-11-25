package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// So sánh từng ô giữa bản .xlsx và bản .xls (LibreOffice chuyển đổi) của cùng file mẫu.
func TestXLSMatchesXLSX(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.xls")
	if len(files) == 0 { t.Skip("không có file .xls mẫu") }
	for _, f := range files {
		xb, _ := os.ReadFile(f)
		yb, err := os.ReadFile(f + "x")
		if err != nil { continue }
		a, err := ReadTable(f, xb)
		if err != nil { t.Fatalf("%s: %v", f, err) }
		b, err := ReadTable(f+"x", yb)
		if err != nil { t.Fatalf("%s: %v", f, err) }
		if len(a) < 1 || len(b) < 1 { t.Fatalf("%s: không có sheet", f) }
		ra, rb := a[0].Rows, b[0].Rows
		// bỏ dòng rỗng của bản xlsx
		var rb2 [][]string
		for _, r := range rb { if strings.TrimSpace(strings.Join(r, "")) != "" { rb2 = append(rb2, r) } }
		if len(ra) != len(rb2) { t.Errorf("%s: %d dòng vs %d", f, len(ra), len(rb2)); continue }
		diff := 0
		for i := range ra {
			for j := 0; j < max(len(ra[i]), len(rb2[i])); j++ {
				va, vb := "", ""
				if j < len(ra[i]) { va = ra[i][j] }
				if j < len(rb2[i]) { vb = rb2[i][j] }
				if cell(va) != cell(vb) {
					if fa, ok := parseNum(va); ok { if fb, ok2 := parseNum(vb); ok2 && fa == fb { continue } }
					if diff < 5 { t.Errorf("%s [%d,%d]: %q vs %q", f, i, j, va, vb) }
					diff++
				}
			}
		}
		t.Logf("%s: %d sheet, %d dòng, khác %d ô", filepath.Base(f), len(a), len(ra), diff)
		p, err := NewImport(filepath.Base(f), xb, "auto")
		if err != nil { t.Fatalf("%s import: %v", f, err) }
		t.Logf("   → nhận ra %s / %s, %d dòng", p.Source, p.Kind, p.Rows)
	}
}
