package main

import "testing"

// Khách mua nhiều nhưng lãi mỏng phải xếp sau khách mua ít mà lãi dày.
func TestCustomerProfitRank(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	thin, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Gạo", Unit: "bao", Price: 100000, Cost: 95000}, SetStock: f100()}, u)
	fat, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Mì", Unit: "thùng", Price: 100000, Cost: 60000}, SetStock: f100()}, u)
	big, _ := db.SaveParty("cust", Party{Name: "Đại lý A"})
	small, _ := db.SaveParty("cust", Party{Name: "Cô B"})
	if _, err := db.CreateInvoice(SaleReq{CID: "a1", Lines: []Line{{PID: thin.ID, Qty: 40, Price: 100000, Cost: 95000, Ratio: 1}}, Method: "cash", CustID: big.ID}, u); err != nil { t.Fatal(err) }
	if _, err := db.CreateInvoice(SaleReq{CID: "b1", Lines: []Line{{PID: fat.ID, Qty: 10, Price: 100000, Ratio: 1}}, Method: "cash", CustID: small.ID}, u); err != nil { t.Fatal(err) }
	r := db.Report("2000-01-01", "2999-12-31", true)
	if len(r.Custs) != 2 { t.Fatalf("custs=%v", r.Custs) }
	if r.Custs[0]["name"] != "Cô B" || r.Custs[0]["profit"].(float64) != 400000 { t.Fatalf("hang 1 phai la Co B lai 400.000: %v", r.Custs) }
	if r.Custs[1]["profit"].(float64) != 200000 || r.Custs[1]["rev"].(float64) != 4000000 { t.Fatalf("Dai ly A: %v", r.Custs[1]) }
	r = db.Report("2000-01-01", "2999-12-31", false)
	if r.Custs[0]["name"] != "Đại lý A" || r.Custs[0]["profit"].(float64) != 0 { t.Fatalf("khong co quyen xem gia von: %v", r.Custs) }
}

func TestDebtPayCap(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	c, _ := db.SaveParty("cust", Party{Name: "Chị Hoa"})
	if _, err := db.CreateCash(CashEntry{Kind: "adj", PartyType: "cust", PartyID: c.ID, Amount: 500000}, u); err != nil { t.Fatal(err) }
	if _, err := db.CreateCashCap(CashEntry{Kind: "thu", PartyType: "cust", PartyID: c.ID, Amount: 600000}, u, true); err == nil { t.Fatal("thu qua so no phai bi chan") }
	e, err := db.CreateCashCap(CashEntry{Kind: "thu", Method: "bank", PartyType: "cust", PartyID: c.ID, Amount: 200000}, u, true)
	if err != nil || e.DebtAfter != 300000 || e.Method != "bank" { t.Fatalf("%v %+v", err, e) }
	if _, err := db.CreateCashCap(CashEntry{Kind: "thu", PartyType: "cust", PartyID: c.ID, Amount: 300000}, u, true); err != nil { t.Fatal(err) }
	if _, err := db.CreateCashCap(CashEntry{Kind: "thu", PartyType: "cust", PartyID: c.ID, Amount: 1}, u, true); err == nil { t.Fatal("het no thi khong thu them") }
}
