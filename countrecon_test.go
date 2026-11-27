package main

import (
	"testing"
	"time"
)

func TestCountRecon(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000, Cost: 6000}}, u)
	sp, _ := db.SaveParty("supp", Party{Name: "NCC"})
	// lần kiểm đầu: đếm được 100 → chưa có mốc trước
	c1, err := db.CreateCount(CountReq{Lines: []struct {
		PID    string  `json:"pid"`
		Actual float64 `json:"actual"`
		Note   string  `json:"note"`
	}{{PID: p.ID, Actual: 100, Note: "bể 2"}}}, u)
	if err != nil || c1.Lines[0].HasPrev || c1.Lines[0].Note != "bể 2" { t.Fatalf("%v %+v", err, c1) }
	time.Sleep(3 * time.Millisecond)
	// nhập 50, bán 30, trả NCC 5 → dự kiến 115
	db.CreateReceipt(ReceiptReq{CID: "r", SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 50, Price: 6000, Ratio: 1}}, Paid: 300000}, u)
	time.Sleep(3 * time.Millisecond)
	db.CreateInvoice(SaleReq{CID: "s", Lines: []Line{{PID: p.ID, Qty: 30, Price: 10000, Ratio: 1}}, Method: "cash"}, u)
	time.Sleep(3 * time.Millisecond)
	db.CreateSuppReturn(SuppReturnReq{SuppID: sp.ID, Lines: []Line{{PID: p.ID, Qty: 5}}}, u)
	c2, err := db.CreateCount(CountReq{Lines: []struct {
		PID    string  `json:"pid"`
		Actual float64 `json:"actual"`
		Note   string  `json:"note"`
	}{{PID: p.ID, Actual: 114}}}, u)
	l := c2.Lines[0]
	if err != nil || !l.HasPrev || l.Prev != 100 || l.In != 50 || l.Out != 35 || l.Expect != 115 || l.Before != 115 || l.After != 114 { t.Fatalf("%v %+v", err, l) }
}
