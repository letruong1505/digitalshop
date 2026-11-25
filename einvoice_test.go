package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInInvoiceFlow(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u1", Name: "Chủ", Role: "owner"}
	imp(t, db, u, "KiotViet_DanhSachSanPham.xlsx")

	var all []*InInvoice
	for _, f := range []string{"HD_DienLuc_C26TDL_5521.xml", "TaiVe_CongThue.zip"} {
		b, _ := os.ReadFile(filepath.Join("testdata", "hoadon", f))
		l, errs := ParseInvoiceFiles(f, b)
		if len(errs) > 0 { t.Fatalf("%s: %v", f, errs) }
		all = append(all, l...)
	}
	if len(all) != 3 { t.Fatalf("đọc được %d hóa đơn", len(all)) }
	res := db.AddInInvoices(all, "xml")
	if len(res.Added) != 3 { t.Fatalf("add %+v", res) }
	// nhập lại: trùng
	b, _ := os.ReadFile("testdata/hoadon/HD_HungPhat_C26THP_1052.xml")
	again, _ := ParseInvoiceFiles("x.xml", b)
	if r := db.AddInInvoices(again, "xml"); r.Dup != 1 || len(r.Added) != 0 { t.Fatalf("dup %+v", r) }

	var hp, el, lan *InInvoice
	db.Read(func(s *State) {
		for _, x := range s.InInvs {
			switch x.SellerMST {
			case "2901234567": hp = x
			case "2900325596": el = x
			case "8123456789": lan = x
			}
		}
	})
	if el.Status != "expense" { t.Fatalf("điện phải là chi phí: %s", el.Status) }
	if hp.Total != 2332080 { t.Fatalf("total %v", hp.Total) }
	sum := 0.0
	for i, l := range hp.Lines {
		t.Logf("%d k=%d %-50s %s x%v cost=%v → pid=%s auto=%s %s×%v cands=%v", i, l.Kind, l.Name, l.Unit, l.Qty, l.Cost, l.PID, l.Auto, l.PUnit, l.Ratio, l.Cands)
		sum += l.Cost * l.Qty
	}
	if d := sum - hp.Total; d > 5 || d < -5 { t.Fatalf("tổng giá vốn %v ≠ %v", sum, hp.Total) }
	if hp.Lines[0].Auto != "guess" || hp.Lines[0].Ratio != 30 { t.Fatalf("mì phải gợi ý thùng 30") }
	if ok, _, _ := hp.ready(); ok { t.Fatal("chưa duyệt mà đã sẵn sàng") }

	// chưa chọn dòng kẹo dừa → lỗi
	if _, err := db.ApproveInInvoice(hp.ID, nil, "debt", "", u); err == nil { t.Fatal("phải báo dòng chưa chọn") }
	var sels []InLineSel
	for i, l := range hp.Lines {
		switch {
		case l.Kind != 1 && l.Kind != 2:
		case l.PID == "":
			sels = append(sels, InLineSel{I: i, New: true})
		default:
			sels = append(sels, InLineSel{I: i, PID: l.PID, PUnit: l.PUnit, Ratio: l.Ratio})
		}
	}
	var mi *Product
	db.Read(func(s *State) { for _, p := range s.Products { if p.Code == "SP000001" { mi = p } } })
	stock0, cost0 := mi.Stock, mi.Cost
	rc, err := db.ApproveInInvoice(hp.ID, sels, "debt", "", u)
	if err != nil { t.Fatal(err) }
	t.Logf("phiếu %s total=%v debt=%v lines=%d", rc.Code, rc.Total, rc.DebtAmt, len(rc.Lines))
	db.Read(func(s *State) {
		x := s.InInvs[hp.ID]
		if x.Status != "done" || x.ReceiptCode != rc.Code { t.Errorf("status %s", x.Status) }
		sp := s.Suppliers[x.SuppID]
		if sp == nil || sp.TaxCode != "2901234567" || sp.Debt != rc.DebtAmt { t.Errorf("ncc %+v", sp) }
		m := s.Products[mi.ID]
		if m.Stock != stock0+150 { t.Errorf("tồn mì %v → %v", stock0, m.Stock) }
		t.Logf("mì giá vốn %v → %v", cost0, m.Cost)
		if len(s.InMap) < 5 { t.Errorf("nhớ %d dòng", len(s.InMap)) }
	})
	// hóa đơn lần sau cùng NCC → tự khớp từ trí nhớ
	b2 := []byte(string(b))
	nx, _ := ParseInvoiceFiles("y.xml", b2)
	nx[0].No = "1053"
	nx[0].Key = ""
	r := db.AddInInvoices(nx, "xml")
	db.Read(func(s *State) {
		x := s.InInvs[r.Added[0]]
		ok, a, n := x.ready()
		t.Logf("hóa đơn 1053: ready=%v %d/%d", ok, a, n)
		if !ok { t.Errorf("hóa đơn thứ hai phải tự khớp hết") }
	})
	// hủy phiếu → hóa đơn về chờ duyệt
	if err := db.VoidReceipt(rc.ID); err != nil { t.Fatal(err) }
	db.Read(func(s *State) { if s.InInvs[hp.ID].Status != "new" { t.Error("hủy phiếu chưa trả hóa đơn về") } })
	_ = lan
}
