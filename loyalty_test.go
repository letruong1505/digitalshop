package main

import "testing"

func TestLoyalty(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000, Cost: 6000}, SetStock: f100()}, u)
	c, _ := db.SaveParty("cust", Party{Name: "Chị Hoa"})
	pts := func() int { var n int; db.Read(func(s *State) { n = s.Customers[c.ID].Points }); return n }
	sale := func(cid string, qty float64, points int) (*Invoice, error) {
		return db.CreateInvoice(SaleReq{CID: cid, Lines: []Line{{PID: p.ID, Qty: qty, Price: 10000, Ratio: 1}}, Method: "cash", CustID: c.ID, Points: points}, u)
	}
	// chưa bật: không tích, không dùng được
	if i, _ := sale("a0", 1, 0); i.PtsEarned != 0 || pts() != 0 { t.Fatal("chua bat") }
	if _, err := sale("a1", 1, 5); err == nil { t.Fatal("dung diem khi chua bat") }
	if err := db.SaveLoyalty(Loyalty{On: true, EarnPer: 10000, PointValue: 100, MinRedeem: 10, MaxPct: 50}); err != nil { t.Fatal(err) }
	if err := db.SaveLoyalty(Loyalty{On: true, EarnPer: 100, PointValue: 500}); err == nil { t.Fatal("cho phep lo") }
	// mua 100.000 → 10 điểm
	i1, err := sale("b1", 10, 0)
	if err != nil || i1.PtsEarned != 10 || pts() != 10 || i1.PtsBal != 10 { t.Fatalf("%v %+v", err, i1) }
	// kiểm tra giới hạn
	if _, err := sale("b2", 10, 50); err == nil { t.Fatal("vuot so diem") }
	if _, err := sale("b2", 10, 5); err == nil { t.Fatal("duoi toi thieu") }
	// mua 200.000, dùng 10 điểm → giảm 1.000, trả 199.000 → +19 điểm
	i2, err := sale("b3", 20, 10)
	if err != nil || i2.PtsDisc != 1000 || i2.Total != 199000 || i2.Disc != 1000 || i2.PtsEarned != 19 { t.Fatalf("%v %+v", err, i2) }
	if pts() != 19 { t.Fatalf("pts=%d", pts()) }
	// gửi lại cùng mã không cộng lần hai
	sale("b3", 20, 10)
	if pts() != 19 { t.Fatalf("lap: %d", pts()) }
	// hủy hóa đơn 2: trả điểm đã dùng, thu hồi điểm đã tích → 10
	if err := db.VoidInvoice("b3", u); err != nil || pts() != 10 { t.Fatalf("%v %d", err, pts()) }
	// trả hàng 5 chai (50.000) thu hồi 5 điểm
	r, err := db.CreateReturn(ReturnReq{CustID: c.ID, Lines: []Line{{PID: p.ID, Qty: 5}}, Restock: true, RefundMethod: "cash"}, u)
	if err != nil || r.PtsTaken != 5 || pts() != 5 { t.Fatalf("%v %+v %d", err, r, pts()) }
	if err := db.VoidReturn(r.ID); err != nil || pts() != 10 { t.Fatalf("%v %d", err, pts()) }
	// điều chỉnh tay
	if _, err := db.AdjustPoints(c.ID, -11, "x", u); err == nil { t.Fatal("tru qua") }
	if _, err := db.AdjustPoints(c.ID, 40, "tặng sinh nhật", u); err != nil || pts() != 50 { t.Fatalf("%d", pts()) }
	// tối đa 50% hóa đơn: 30.000 → 15.000/100 = 150 điểm, bị chặn bởi số dư 50
	var mx int
	db.Read(func(s *State) { mx = s.Settings.Loyalty.maxRedeem(s.Customers[c.ID].Points, 30000) })
	if mx != 50 { t.Fatalf("max=%d", mx) }
}
