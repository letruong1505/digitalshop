package main

import (
	"testing"
	"time"
)

func TestCampaign(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "o", Name: "Chủ", Role: "owner"}
	a, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Mì", Group: "Mì", Unit: "gói", Price: 10000, Cost: 6000}, SetStock: f100()}, u)
	b, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Group: "Đồ uống", Unit: "chai", Price: 20000, Cost: 12000}, SetStock: f100()}, u)
	today := time.Now().Format("2006-01-02")
	camp := Campaign{Name: "Tết", Active: true, From: today, To: today, Lines: []CampLine{{Scope: "group", Target: "Mì", Kind: "pct", Value: 10}, {Scope: "product", Target: b.ID, Kind: "amt", Value: 3000}}}
	bad := camp; bad.Lines = nil
	if err := db.SaveCampaigns([]Campaign{bad}); err == nil { t.Fatal("chua co dong") }
	bad = camp; bad.To = "2000-01-01"
	if err := db.SaveCampaigns([]Campaign{bad}); err == nil { t.Fatal("ngay sai") }
	if err := db.SaveCampaigns([]Campaign{camp}); err != nil { t.Fatal(err) }
	// chiến dịch khác giảm cao hơn trên mặt hàng Mì → lấy mức cao hơn
	c2 := Campaign{Name: "Xả kho", Active: true, From: today, To: today, Lines: []CampLine{{Scope: "product", Target: a.ID, Kind: "pct", Value: 20}}}
	if err := db.SaveCampaigns([]Campaign{camp, c2}); err != nil { t.Fatal(err) }
	inv, err := db.CreateInvoice(SaleReq{CID: "c1", Lines: []Line{{PID: a.ID, Qty: 2, Price: 10000, Ratio: 1}, {PID: b.ID, Qty: 1, Price: 20000, Ratio: 1}}, Method: "cash"}, u)
	if err != nil { t.Fatal(err) }
	// Mì: 20% của 20.000 = 4.000 (hơn 10% = 2.000); Nước: 3.000
	if inv.Promo != 7000 || inv.Total != 33000 { t.Fatalf("%+v", inv) }
	if inv.Lines[0].PromoName != "Xả kho" || inv.Lines[0].PromoDisc != 4000 || inv.Lines[1].PromoName != "Tết" { t.Fatalf("%+v", inv.Lines) }
	rep := db.PromoReport("2000-01-01", "2100-01-01")
	if len(rep) != 2 || rep[0].Name != "Xả kho" || rep[0].Amount != 4000 || rep[1].Amount != 3000 { t.Fatalf("%+v", rep) }
	// thu ngân tự hạ giá → không tính khuyến mãi cho dòng đó
	inv2, _ := db.CreateInvoice(SaleReq{CID: "c2", Lines: []Line{{PID: a.ID, Qty: 1, Price: 9000, Ratio: 1}}, Method: "cash"}, u)
	if inv2.Promo != 0 { t.Fatalf("khuyen mai tren gia sua: %+v", inv2) }
	// tắt chiến dịch / hết hạn
	camp.Active, c2.Active = false, false
	db.SaveCampaigns([]Campaign{camp, c2})
	inv3, _ := db.CreateInvoice(SaleReq{CID: "c3", Lines: []Line{{PID: a.ID, Qty: 1, Price: 10000, Ratio: 1}}, Method: "cash"}, u)
	if inv3.Promo != 0 { t.Fatal("van con giam") }
}
