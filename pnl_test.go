package main

import "testing"

func TestPnLAndClosing(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000, Cost: 6000}, SetStock: f100()}, u)
	c, _ := db.SaveParty("cust", Party{Name: "Hoa"})
	day := dayOf(nowMs())
	// bán 20 chai = 200.000 (vốn 120.000) tiền mặt
	if _, err := db.CreateInvoice(SaleReq{CID: "a", Lines: []Line{{PID: p.ID, Qty: 20, Price: 10000, Ratio: 1}}, Method: "cash", CustID: c.ID}, u); err != nil { t.Fatal(err) }
	// chi tiền điện 30.000, thu khác 5.000, rút 50.000, thu nợ khách 0
	for _, e := range []CashEntry{{Kind: "chi", Method: "cash", Amount: 30000, Cat: "Tiền điện"}, {Kind: "thu", Method: "cash", Amount: 5000, Cat: "Thu khác"}, {Kind: "chi", Method: "cash", Amount: 50000, Cat: "Rút tiền chi tiêu"}} {
		if _, err := db.CreateCash(e, u); err != nil { t.Fatal(err) }
	}
	// khách trả 5 chai (50.000, vốn 30.000, nhập lại kho) hoàn tiền mặt
	if _, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 5}}, Restock: true, RefundMethod: "cash"}, u); err != nil { t.Fatal(err) }
	pl := db.PnL(day, day)
	// doanh thu thuần 150.000, vốn 90.000, lãi gộp 60.000; chi phí 30.000; thu khác 5.000 → 35.000
	if pl.Revenue != 150000 || pl.COGS != 90000 || pl.Gross != 60000 || pl.ExpenseT != 30000 || pl.OtherInT != 5000 || pl.Net != 35000 || pl.Draw != 50000 {
		t.Fatalf("%+v", pl)
	}
	// chốt ca: tiền mặt = bán 200.000 + thu 5.000 − chi 30.000 − rút 50.000 − hoàn khách 50.000 = 75.000
	cp := db.ClosePreview(day)
	if cp.Net != 75000 || cp.Expected != 75000 || cp.Opening != 0 { t.Fatalf("%+v", cp) }
	if _, err := db.CreateClosing(day, 70000, 80000, "", u); err == nil { t.Fatal("kept>counted") }
	cl, err := db.CreateClosing(day, 70000, 20000, "thiếu", u)
	if err != nil || cl.Diff != -5000 || cl.Take != 50000 { t.Fatalf("%v %+v", err, cl) }
	if pl := db.PnL(day, day); pl.Diff != -5000 || pl.Net != 30000 { t.Fatalf("%+v", pl) }
	// chốt lại cùng ngày thay thế bản cũ
	cl2, _ := db.CreateClosing(day, 75000, 75000, "", u)
	if cl2.Diff != 0 || len(db.Closings(10)) != 1 { t.Fatal("replace") }
	// ngày mai mở ca với tiền để lại
	var open float64
	db.Read(func(s *State) { open = lastClosingBefore(s, "2999-01-01").Kept })
	if open != 75000 { t.Fatal(open) }
}
