package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func imp(t *testing.T, db *DB, u *User, file string) (*Preview, *ImportResult) {
	b, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil { t.Fatal(err) }
	p, err := NewImport(file, b, "auto")
	if err != nil { t.Fatalf("%s: %v", file, err) }
	res, err := db.CommitImport(getJob(p.Token), p.Mapping, "upsert", u)
	if err != nil { t.Fatalf("%s commit: %v", file, err) }
	return p, res
}

func TestFlow(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u1", Name: "Chủ", Role: "owner"}

	p, r := imp(t, db, u, "KiotViet_DanhSachSanPham.xlsx")
	t.Logf("KV products: src=%s kind=%s header=%d map=%v res=%+v notes=%v", p.Source, p.Kind, p.Header, p.Mapping, r, p.Notes)
	if p.Source != "KiotViet" || p.Kind != "products" || r.Added != 10 || r.Units != 5 { t.Fatalf("kiotviet products wrong") }
	p, r = imp(t, db, u, "MISA_DanhSachVTHH.xlsx")
	t.Logf("MISA: src=%s kind=%s header=%d res=%+v", p.Source, p.Kind, p.Header, r)
	if p.Kind != "products" || r.Added != 4 || p.Header != 3 { t.Fatalf("misa wrong") }
	p, r = imp(t, db, u, "MISAeShop_HangHoa.xlsx")
	t.Logf("eShop: src=%s res=%+v", p.Source, r)
	p, r = imp(t, db, u, "KiotViet_DanhSachKhachHang.xlsx")
	t.Logf("KV cust: src=%s kind=%s res=%+v", p.Source, p.Kind, r)
	if p.Kind != "customers" || r.Added != 4 { t.Fatalf("customers wrong") }
	p, r = imp(t, db, u, "KiotViet_NhaCungCap.xlsx")
	t.Logf("KV supp: kind=%s res=%+v", p.Kind, r)
	if p.Kind != "suppliers" || r.Added != 2 { t.Fatalf("suppliers wrong") }
	p, r = imp(t, db, u, "KiotViet_ChiTietHoaDon.xlsx")
	t.Logf("KV inv: kind=%s res=%+v", p.Kind, r)
	if p.Kind != "invoices" || r.Added < 300 { t.Fatalf("invoices wrong") }

	var mi, gao *Product
	var hoa *Party
	db.Read(func(s *State) {
		for _, x := range s.Products {
			if x.Code == "SP000001" { mi = x }
			if x.Code == "GAO01" { gao = x }
		}
		for _, c := range s.Customers { if c.Code == "KH000001" { hoa = c } }
	})
	t.Logf("mi: %+v", *mi)
	t.Logf("gao: stock=%v cost=%v price=%v", gao.Stock, gao.Cost, gao.Price)
	if mi.Group != "Mì ăn liền" || len(mi.Units) != 1 || mi.Units[0].Ratio != 30 || mi.Cost != 3813 { t.Fatalf("mi wrong") }
	if gao.Stock != 50.5 || gao.Cost != 18500 { t.Fatalf("gao wrong") }
	if hoa.Debt != 185000 { t.Fatalf("hoa debt %v", hoa.Debt) }

	// nhập hàng có hạn sử dụng
	rc, err := db.CreateReceipt(ReceiptReq{Lines: []Line{{PID: mi.ID, Unit: "thùng", Ratio: 30, Qty: 2, Price: 114000, Exp: addDays(today(), 5)}}, Paid: 228000}, u)
	if err != nil { t.Fatal(err) }
	t.Logf("receipt %s total %v; mi stock %v cost %v lots %+v", rc.Code, rc.Total, mi.Stock, mi.Cost, mi.Lots)
	if mi.Stock != 300 || len(mi.Lots) != 1 { t.Fatalf("receipt wrong") }

	// bán: 1 thùng + 5 gói, khách đưa thiếu → ghi nợ
	inv, err := db.CreateInvoice(SaleReq{CID: "x1", Lines: []Line{{PID: mi.ID, Unit: "thùng", Ratio: 30, Qty: 1, Price: 125000}, {PID: mi.ID, Unit: "gói", Ratio: 1, Qty: 5, Price: 4500}}, Method: "cash", Given: 100000, CustID: hoa.ID}, u)
	if err != nil { t.Fatal(err) }
	t.Logf("inv %s total %v cash %v debt %v; mi stock %v lots %+v; hoa debt %v", inv.Code, inv.Total, inv.Cash, inv.DebtAmt, mi.Stock, mi.Lots, hoa.Debt)
	if inv.Total != 147500 || inv.DebtAmt != 47500 || mi.Stock != 265 || mi.Lots[0].Qty != 25 || hoa.Debt != 232500 { t.Fatalf("sale wrong") }
	// gửi lại cùng mã → không tạo trùng
	inv2, _ := db.CreateInvoice(SaleReq{CID: "x1", Lines: []Line{{PID: mi.ID, Qty: 1, Price: 1}}}, u)
	if inv2.Code != inv.Code { t.Fatalf("idempotency") }
	// thiếu hàng
	if _, err := db.CreateInvoice(SaleReq{Lines: []Line{{PID: gao.ID, Qty: 999, Price: 1}}}, u); err == nil || !strings.Contains(err.Error(), "Không đủ") { t.Fatalf("expect stock error, got %v", err) }
	// hủy hóa đơn
	if err := db.VoidInvoice(inv.ID, u); err != nil { t.Fatal(err) }
	if mi.Stock != 300 || hoa.Debt != 185000 { t.Fatalf("void wrong %v %v", mi.Stock, hoa.Debt) }
	// thu nợ
	if _, err := db.CreateCash(CashEntry{Kind: "thu", Amount: 85000, PartyType: "cust", PartyID: hoa.ID, Cat: "Thu nợ khách"}, u); err != nil { t.Fatal(err) }
	if hoa.Debt != 100000 { t.Fatalf("debt pay wrong") }

	// combo
	var trung, bia, thuoc *Product
	db.Read(func(s *State) {
		for _, x := range s.Products {
			switch x.Code { case "SP000004": trung = x; case "SP000002": bia = x; case "SP000008": thuoc = x }
		}
	})
	cb, err := db.SaveProduct(ProductReq{Product: Product{Name: "Combo bữa sáng", IsCombo: true, Price: 11000, Items: []ComboItem{{mi.ID, 1}, {trung.ID, 2}}}}, u)
	if err != nil { t.Fatal(err) }
	if _, err := db.CreateInvoice(SaleReq{Lines: []Line{{PID: cb.ID, Qty: 3, Price: 11000}}}, u); err != nil { t.Fatal(err) }
	if mi.Stock != 297 || trung.Stock != 54 { t.Fatalf("combo stock %v %v", mi.Stock, trung.Stock) }

	sg := db.SuggestCombos(90)
	pairs := sg["pairs"].([]ComboSuggest)
	t.Logf("baskets=%v pairs=%d", sg["baskets"], len(pairs))
	for _, c := range pairs[:min(4, len(pairs))] { t.Logf("  %v retail=%v price=%v margin=%v %s", c.Names, c.Retail, c.Price, c.Margin, c.Reason) }
	for _, c := range sg["expiry"].([]ComboSuggest) { t.Logf("  EXP %v price=%v %s", c.Names, c.Price, c.Reason) }
	_ = bia
	_ = thuoc

	// Zalo
	var ps *parser
	db.Read(func(s *State) { ps = newParser(s) })
	text := "Nguyễn Văn Tuấn: anh lấy 2 thùng bia sg với 1 cây thăng long, chiều 5h qua lấy\nchị Hoa xóm 3: e ơi mai cho chị 1 vỉ trứng, 10 gói hảo hảo nhé\nCô Lan: 2 chai nước mắm nam ngư + 1 gói omo"
	var ds []Draft
	db.Read(func(s *State) { ds = ps.ParseText(text) })
	for _, d := range ds {
		t.Logf("DRAFT %s (%s) due=%s %s unknown=%v", d.CustName, d.CustID, d.Due, d.DueTime, d.Unknown)
		for _, l := range d.Lines { t.Logf("    %v %s %s (ratio %v) <- %q", l.Qty, l.Unit, l.Name, l.Ratio, l.Text) }
	}
	if len(ds) != 3 { t.Fatalf("expected 3 drafts") }

	rep := db.Report(addDays(today(), -45), today(), true)
	t.Logf("report rev=%v profit=%v n=%v imported=%v top0=%v", rep.Rev, rep.Profit, rep.N, rep.Imported, rep.Top[0])

	// đọc lại từ đĩa
	db.Close()
	db2, err := OpenDB(dir)
	if err != nil { t.Fatal(err) }
	db2.Read(func(s *State) {
		if s.Products[mi.ID].Stock != 297 || s.Customers[hoa.ID].Debt != 100000 { t.Fatalf("reload mismatch") }
		t.Logf("reload ok: %d products %d invoices", len(s.Products), len(s.Invoices))
	})
	// xuất Excel rồi đọc lại
	var x []byte
	db2.Read(func(s *State) { x, _ = exportXLSX(s, "products", nil, true) })
	sh, err := ReadTable("a.xlsx", x)
	if err != nil || len(sh[0].Rows) < 10 { t.Fatalf("export roundtrip %v", err) }
	pv, err := NewImport("hang-hoa.xlsx", x, "auto")
	if err != nil { t.Fatal(err) }
	t.Logf("re-import detect src=%s kind=%s", pv.Source, pv.Kind)
}

func TestParseNum(t *testing.T) {
	cases := map[string]float64{"12.500": 12500, "12,5": 12.5, "1.234.567": 1234567, "1.234,5": 1234.5, "20k": 20000, "1,5tr": 1500000, "0.5": 0.5, NumMark + "3812.5": 3812.5, "4500đ": 4500, "125.000 đ": 125000}
	for in, want := range cases {
		if got := num(in); got != want { t.Errorf("%q => %v want %v", in, got, want) }
	}
}
