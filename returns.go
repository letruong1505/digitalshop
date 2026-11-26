package main

// Khách trả hàng: giá trị hàng trả được trừ vào công nợ của khách; phần còn dư hoàn tiền (mặt/chuyển khoản)
// hoặc giữ lại làm số dư cho khách. Hàng còn bán được thì nhập lại kho. Báo cáo trừ doanh thu theo hàng trả.
// Chỉ nhận hàng khách đã mua (kiểm tra theo hóa đơn), không quá số lượng đã mua trừ số đã trả, không quá giá đã bán.

import (
	"sort"
	"strings"
)

type CustReturn struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	Time         int64   `json:"time"`
	CustID       string  `json:"custId"`
	CustName     string  `json:"custName"`
	Lines        []Line  `json:"lines"`
	Total        float64 `json:"total"`        // giá trị hàng khách trả
	Cost         float64 `json:"cost"`         // giá vốn của hàng trả
	Restock      bool    `json:"restock"`      // đã nhập lại kho
	ApplyDebt    float64 `json:"applyDebt"`    // phần trừ vào nợ
	Refund       float64 `json:"refund"`       // phần còn lại
	RefundMethod string  `json:"refundMethod"` // cash | bank | credit (giữ làm số dư)
	CashID       string  `json:"cashId,omitempty"`
	DebtAfter    float64 `json:"debtAfter"`
	Note         string  `json:"note"`
	Staff        string  `json:"staff"`
	Void         bool    `json:"void"`
}

// ReturnLineInfo: hàng khách đã mua và còn trả được (tính theo đơn vị gốc của mặt hàng).
type ReturnLineInfo struct {
	PID      string  `json:"pid"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Bought   float64 `json:"bought"`
	Returned float64 `json:"returned"`
	Left     float64 `json:"left"`
	Price    float64 `json:"price"` // giá bán thực tế bình quân (đã trừ giảm giá) cho 1 đơn vị gốc
	MaxPrice float64 `json:"maxPrice"`
	Cost     float64 `json:"cost"` // giá vốn bình quân cho 1 đơn vị gốc
	Last     int64   `json:"last"`
}

func ratioOf(l Line) float64 { if l.Ratio <= 0 { return 1 }; return l.Ratio }

func returnable(s *State, custID string) map[string]*ReturnLineInfo {
	out := map[string]*ReturnLineInfo{}
	sumPrice, sumCost := map[string]float64{}, map[string]float64{}
	for _, inv := range s.Invoices {
		if inv.CustID != custID || inv.Void || inv.Src == "import" { continue }
		share := 1.0
		if inv.Sub > 0 { share = inv.Total / inv.Sub }
		for _, l := range inv.Lines {
			p := s.Products[l.PID]
			if p == nil || l.Qty <= 0 { continue }
			base := l.Qty * ratioOf(l)
			x := out[l.PID]
			if x == nil { x = &ReturnLineInfo{PID: l.PID, Name: p.Name, Unit: p.Unit}; out[l.PID] = x }
			x.Bought += base
			sumPrice[l.PID] += l.Qty * l.Price * share
			sumCost[l.PID] += l.Qty * l.Cost
			if u := l.Price * share / ratioOf(l); u > x.MaxPrice { x.MaxPrice = u }
			if inv.Time > x.Last { x.Last = inv.Time }
		}
	}
	for _, rt := range s.Returns {
		if rt.CustID != custID || rt.Void { continue }
		for _, l := range rt.Lines { if x := out[l.PID]; x != nil { x.Returned += l.Qty * ratioOf(l) } }
	}
	for id, x := range out {
		x.Bought, x.Returned = r3(x.Bought), r3(x.Returned)
		x.Left = r3(x.Bought - x.Returned)
		if x.Bought > 0 { x.Price, x.Cost = r0(sumPrice[id]/x.Bought), sumCost[id]/x.Bought }
		x.MaxPrice = r0(x.MaxPrice + 0.49)
	}
	return out
}

func returnableList(s *State, custID string) []*ReturnLineInfo {
	m := returnable(s, custID)
	out := make([]*ReturnLineInfo, 0, len(m))
	for _, x := range m { out = append(out, x) }
	sort.Slice(out, func(i, j int) bool { return out[i].Last > out[j].Last })
	if len(out) > 150 { out = out[:150] }
	return out
}

type ReturnReq struct {
	CustID       string `json:"custId"`
	Lines        []Line `json:"lines"` // pid, qty (đơn vị gốc), price (giá trả cho 1 đơn vị gốc)
	Restock      bool   `json:"restock"`
	ToDebt       bool   `json:"toDebt"`
	RefundMethod string `json:"refundMethod"`
	Note         string `json:"note"`
}

func stockMove(s *State, t *Tx, l Line, sign float64) {
	p := s.Products[l.PID]
	if p == nil { return }
	base := r3(l.Qty * ratioOf(l))
	if p.IsCombo {
		for _, it := range p.Items { if q := s.Products[it.PID]; q != nil { q.Stock = r3(q.Stock + sign*it.Qty*base); t.put("product", q.ID, q) } }
		return
	}
	p.Stock = r3(p.Stock + sign*base)
	t.put("product", p.ID, p)
}

func (db *DB) CreateReturn(req ReturnReq, u *User) (*CustReturn, error) {
	var out *CustReturn
	err := db.Do(func(s *State, t *Tx) error {
		c := s.Customers[req.CustID]
		if c == nil { return uerr("Không tìm thấy khách hàng") }
		if len(req.Lines) == 0 { return uerr("Chọn hàng khách trả") }
		info := returnable(s, c.ID)
		lines := []Line{}
		var total, cost float64
		seen := map[string]bool{}
		for _, l := range req.Lines {
			if l.Qty <= 0 { continue }
			x := info[l.PID]
			if x == nil { return uerr("Khách chưa mua mặt hàng này, không nhận trả") }
			if seen[l.PID] { return uerr("Mỗi mặt hàng chỉ nhập một dòng") }
			seen[l.PID] = true
			qty := r3(l.Qty)
			if qty > x.Left+0.0005 { return uerr("%s: khách đã mua %s, đã trả %s, chỉ còn trả được %s %s", x.Name, fq(x.Bought), fq(x.Returned), fq(x.Left), x.Unit) }
			price := l.Price
			if price <= 0 { price = x.Price }
			if price > x.MaxPrice { return uerr("%s: giá trả %s lớn hơn giá khách đã mua (%s)", x.Name, money(price), money(x.MaxPrice)) }
			nl := Line{PID: l.PID, Name: x.Name, Unit: x.Unit, Ratio: 1, Qty: qty, Price: r0(price), Cost: x.Cost}
			lines = append(lines, nl)
			total += qty * nl.Price
			cost += qty * nl.Cost
		}
		if len(lines) == 0 { return uerr("Nhập số lượng khách trả") }
		total, cost = r0(total), r0(cost)
		applyDebt := 0.0
		if req.ToDebt && c.Debt > 0 { applyDebt = total; if c.Debt < applyDebt { applyDebt = c.Debt } }
		refund := r0(total - applyDebt)
		method := req.RefundMethod
		if method != "bank" && method != "credit" { method = "cash" }
		now := nowMs()
		r := &CustReturn{ID: newID(), Code: s.nextCode("TH", 6), Time: now, CustID: c.ID, CustName: c.Name, Lines: lines, Total: total, Cost: cost,
			Restock: req.Restock, ApplyDebt: applyDebt, Refund: refund, RefundMethod: method, Note: strings.TrimSpace(req.Note), Staff: u.Name}
		if refund == 0 { r.RefundMethod = "" }
		if r.Restock { for _, l := range lines { stockMove(s, t, l, +1) } }
		c.Debt = r0(c.Debt - applyDebt)
		if refund > 0 && method == "credit" { c.Debt = r0(c.Debt - refund) }
		if refund > 0 && method != "credit" {
			e := &CashEntry{ID: newID(), Code: s.nextCode("PC", 6), Time: now, Kind: "chi", Method: method, Amount: refund, Cat: "Hoàn tiền khách trả hàng",
				Note: "Khách " + c.Name + " trả hàng " + r.Code, Staff: u.Name}
			s.Cash[e.ID] = e
			t.put("cash", e.ID, e)
			r.CashID = e.ID
		}
		c.Last = now
		r.DebtAfter = c.Debt
		t.put("customer", c.ID, c)
		s.Returns[r.ID] = r
		t.put("return", r.ID, r)
		t.put("counters", "", s.Counters)
		out = r
		return nil
	})
	return out, err
}

func (db *DB) VoidReturn(id string) error {
	return db.Do(func(s *State, t *Tx) error {
		r := s.Returns[id]
		if r == nil || r.Void { return uerr("Không tìm thấy phiếu trả hàng") }
		if c := s.Customers[r.CustID]; c != nil {
			c.Debt = r0(c.Debt + r.ApplyDebt)
			if r.RefundMethod == "credit" { c.Debt = r0(c.Debt + r.Refund) }
			t.put("customer", c.ID, c)
		}
		if r.Restock { for _, l := range r.Lines { stockMove(s, t, l, -1) } }
		if e := s.Cash[r.CashID]; e != nil && !e.Void { e.Void = true; t.put("cash", e.ID, e) }
		r.Void = true
		t.put("return", r.ID, r)
		return nil
	})
}
