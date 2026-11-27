package main

import "testing"

func TestSuppReturn(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Sữa", Unit: "hộp", Price: 12000, Cost: 8000}, SetStock: f100()}, u)
	sp, _ := db.SaveParty("supp", Party{Name: "NCC A"})
	sp2, _ := db.SaveParty("supp", Party{Name: "NCC B"})
	// nhập 50 hộp × 8.000 = 400.000, trả 100.000, nợ 300.000
	if _, err := db.CreateReceipt(ReceiptReq{CID: "r1", SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 50, Price: 8000, Ratio: 1}}, Paid: 100000}, u); err != nil { t.Fatal(err) }
	debt := func() float64 { var d float64; db.Read(func(s *State) { d = s.Suppliers[sp.ID].Debt }); return d }
	stock := func() float64 { var d float64; db.Read(func(s *State) { d = s.Products[p.ID].Stock }); return d }
	if debt() != 300000 || stock() != 150 { t.Fatalf("%v %v", debt(), stock()) }
	// NCC khác chưa nhập hàng này
	if _, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp2.ID, Lines: []Line{{PID: p.ID, Qty: 1}}, ToDebt: true}, u); err == nil { t.Fatal("tra NCC khac") }
	// quá số đã nhập
	if _, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 51}}, ToDebt: true}, u); err == nil { t.Fatal("qua so luong") }
	// giá cao hơn giá nhập
	if _, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 1, Price: 9000}}, ToDebt: true}, u); err == nil { t.Fatal("gia cao") }
	// trả 10 hộp = 80.000 trừ nợ → 220.000, kho 140
	r, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 10}}, ToDebt: true}, u)
	if err != nil || r.Total != 80000 || r.ApplyDebt != 80000 || r.Refund != 0 || debt() != 220000 || stock() != 140 { t.Fatalf("%v %+v %v %v", err, r, debt(), stock()) }
	// trả 40 hộp = 320.000: trừ 220.000 nợ, NCC hoàn 100.000 tiền mặt
	r2, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 40}}, ToDebt: true, RefundMethod: "cash"}, u)
	if err != nil || r2.ApplyDebt != 220000 || r2.Refund != 100000 || r2.CashID == "" || debt() != 0 || stock() != 100 { t.Fatalf("%v %+v", err, r2) }
	var thu float64
	db.Read(func(s *State) { e := s.Cash[r2.CashID]; thu = e.Amount; if e.Kind != "thu" { t.Fatal("kind") } })
	if thu != 100000 { t.Fatal(thu) }
	// hết hàng trả được
	if _, err := db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 1}}}, u); err == nil { t.Fatal("het") }
	// hủy phiếu 2: nợ 220.000, kho 140, phiếu thu bị hủy
	if err := db.VoidSuppReturn(r2.ID); err != nil || debt() != 220000 || stock() != 140 { t.Fatalf("%v %v %v", err, debt(), stock()) }
	db.Read(func(s *State) { if !s.Cash[r2.CashID].Void { t.Fatal("cash not void") } })
}
