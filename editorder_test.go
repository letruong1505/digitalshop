package main

import "testing"

func TestReplaceInvoice(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	owner := &User{ID: "o", Name: "Chủ", Role: "owner"}
	cash := &User{ID: "c", Name: "Lan", Role: "cashier"}
	other := &User{ID: "d", Name: "Mai", Role: "cashier"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000, Cost: 6000}, SetStock: f100()}, owner)
	c, _ := db.SaveParty("cust", Party{Name: "Chị Hoa"})
	stock := func() float64 { var n float64; db.Read(func(s *State) { n = s.Products[p.ID].Stock }); return n }
	debt := func() float64 { var n float64; db.Read(func(s *State) { n = s.Customers[c.ID].Debt }); return n }
	st0 := stock()
	inv, err := db.CreateInvoice(SaleReq{CID: "x1", Lines: []Line{{PID: p.ID, Qty: 3, Price: 10000, Ratio: 1}}, Method: "cash"}, cash)
	if err != nil || stock() != st0-3 { t.Fatal(err, stock()) }
	// đổi sang 5 chai, ghi nợ
	n, err := db.ReplaceInvoice(inv.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 5, Price: 10000, Ratio: 1}}, Method: "debt", CustID: c.ID}, cash)
	if err != nil { t.Fatal(err) }
	if stock() != st0-5 || debt() != 50000 || n.EditOf != inv.Code || n.Total != 50000 { t.Fatalf("%v %v %+v", stock(), debt(), n) }
	db.Read(func(s *State) {
		o := s.Invoices[inv.ID]
		if !o.Void || o.EditedTo != n.Code { t.Fatalf("old %+v", o) }
	})
	// hóa đơn mới không hợp lệ (quá tồn) → mọi thứ như cũ
	if _, err := db.ReplaceInvoice(n.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 9999, Price: 10000, Ratio: 1}}, Method: "cash"}, cash); err == nil { t.Fatal("phai loi") }
	if stock() != st0-5 || debt() != 50000 { t.Fatalf("rollback: %v %v", stock(), debt()) }
	db.Read(func(s *State) { if s.Invoices[n.ID].Void { t.Fatal("hoa don bi huy sau loi") } })
	// người khác không sửa được; chủ sửa được
	if _, err := db.ReplaceInvoice(n.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 1, Price: 10000, Ratio: 1}}, Method: "cash"}, other); err == nil { t.Fatal("khong quyen") }
	// quá 30 phút: thu ngân không sửa được
	db.Do(func(s *State, tx *Tx) error { s.Invoices[n.ID].Time -= 31 * 60 * 1000; return nil })
	if _, err := db.ReplaceInvoice(n.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 1, Price: 10000, Ratio: 1}}, Method: "cash"}, cash); err == nil { t.Fatal("qua 30 phut") }
	m, err := db.ReplaceInvoice(n.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 1, Price: 10000, Ratio: 1}}, Method: "cash"}, owner)
	if err != nil || stock() != st0-1 || debt() != 0 || m.Time != n.Time { t.Fatalf("%v %v %v", err, stock(), debt()) }
	// hóa đơn đã hủy không sửa lại
	if _, err := db.ReplaceInvoice(n.ID, SaleReq{Lines: []Line{{PID: p.ID, Qty: 1, Price: 10000, Ratio: 1}}, Method: "cash"}, owner); err == nil { t.Fatal("da huy") }
}
