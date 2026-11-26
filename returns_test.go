package main

import "testing"

func TestCustomerReturn(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000, Cost: 6000}, SetStock: f100()}, u)
	c, _ := db.SaveParty("cust", Party{Name: "Chị Hoa"})
	// mua 10 chai ghi nợ hết: nợ 100.000
	if _, err := db.CreateInvoice(SaleReq{CID: "i1", Lines: []Line{{PID: p.ID, Qty: 10, Price: 10000, Ratio: 1}}, Method: "debt", CustID: c.ID}, u); err != nil { t.Fatal(err) }
	debt := func() float64 { var d float64; db.Read(func(s *State) { d = s.Customers[c.ID].Debt }); return d }
	stock := func() float64 { var d float64; db.Read(func(s *State) { d = s.Products[p.ID].Stock }); return d }
	if debt() != 100000 || stock() != 90 { t.Fatalf("debt=%v stock=%v", debt(), stock()) }

	// trả quá số đã mua / giá quá cao bị chặn
	if _, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 11, Price: 10000}}, ToDebt: true, Restock: true}, u); err == nil { t.Fatal("tra qua so luong") }
	if _, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 1, Price: 20000}}, ToDebt: true}, u); err == nil { t.Fatal("gia qua cao") }

	// trả 3 chai: trừ nợ 30.000, nhập lại kho
	r, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 3}}, ToDebt: true, Restock: true}, u)
	if err != nil || r.Total != 30000 || r.ApplyDebt != 30000 || r.Refund != 0 || r.DebtAfter != 70000 { t.Fatalf("%v %+v", err, r) }
	if debt() != 70000 || stock() != 93 { t.Fatalf("debt=%v stock=%v", debt(), stock()) }
	rep := db.Report("2000-01-01", "2999-12-31", true)
	if rep.Rev != 70000 || rep.Cost != 42000 || rep.Profit != 28000 || rep.Returns != 30000 { t.Fatalf("report %v %v %v %v", rep.Rev, rep.Cost, rep.Profit, rep.Returns) }
	if rep.Custs[0]["rev"].(float64) != 70000 || rep.Custs[0]["profit"].(float64) != 28000 { t.Fatalf("%v", rep.Custs[0]) }

	// còn trả được 7; trả 7 chai hỏng (không nhập lại kho): nợ 70.000 → 0, không hoàn tiền; giá vốn vẫn mất
	r2, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 7}}, ToDebt: true, Restock: false}, u)
	if err != nil || r2.DebtAfter != 0 || r2.Refund != 0 { t.Fatalf("%v %+v", err, r2) }
	if stock() != 93 { t.Fatalf("stock=%v", stock()) }
	if _, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 1}}, ToDebt: true}, u); err == nil { t.Fatal("het hang tra") }
	rep = db.Report("2000-01-01", "2999-12-31", true)
	if rep.Rev != 0 || rep.Cost != 42000 || rep.Profit != -42000 { t.Fatalf("report %v %v %v", rep.Rev, rep.Cost, rep.Profit) }

	// hủy phiếu 2: nợ trở lại 70.000
	if err := db.VoidReturn(r2.ID); err != nil || debt() != 70000 { t.Fatalf("%v %v", err, debt()) }

	// khách không nợ trả 2 chai, hoàn tiền mặt 20.000: có phiếu chi
	c2, _ := db.SaveParty("cust", Party{Name: "Anh Bình"})
	if _, err := db.CreateInvoice(SaleReq{CID: "i2", Lines: []Line{{PID: p.ID, Qty: 5, Price: 10000, Ratio: 1}}, Method: "cash", CustID: c2.ID, Given: 50000}, u); err != nil { t.Fatal(err) }
	r3, err := db.CreateReturn(ReturnReq{CustID: c2.ID, Lines: []Line{{PID: p.ID, Qty: 2}}, ToDebt: true, Restock: true, RefundMethod: "cash"}, u)
	if err != nil || r3.ApplyDebt != 0 || r3.Refund != 20000 || r3.CashID == "" { t.Fatalf("%v %+v", err, r3) }
	var chi float64
	db.Read(func(s *State) { for _, e := range s.Cash { if e.Kind == "chi" && !e.Void { chi += e.Amount } } })
	if chi != 20000 { t.Fatalf("chi=%v", chi) }
	// giữ làm số dư
	r4, err := db.CreateReturn(ReturnReq{CustID: c2.ID, Lines: []Line{{PID: p.ID, Qty: 1}}, Restock: true, RefundMethod: "credit"}, u)
	if err != nil || r4.DebtAfter != -10000 { t.Fatalf("%v %+v", err, r4) }
	// mở lại dữ liệu: phiếu trả hàng phải còn nguyên
	db2, err := OpenDB(dir)
	if err != nil { t.Fatal(err) }
	n := 0
	db2.Read(func(s *State) { for _, r := range s.Returns { if !r.Void { n++ } } })
	if n != 3 { t.Fatalf("sau khi mo lai con %d phieu (can 3: r, r3, r4)", n) }
}
