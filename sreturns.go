package main

// Trả hàng nhà cung cấp: hàng lỗi, hết hạn, giao nhầm trả lại NCC. Chỉ trả hàng đã nhập từ chính NCC đó (theo phiếu nhập),
// không quá số đã nhập trừ số đã trả, không quá tồn kho hiện có. Giá trị hàng trả trừ vào công nợ phải trả NCC;
// phần dư NCC hoàn tiền (mặt/chuyển khoản, ghi vào sổ quỹ là khoản thu) hoặc giữ làm số dư để trừ vào lần nhập sau.

import (
	"sort"
	"strings"
)

type SuppReturn struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	Time         int64   `json:"time"`
	SuppID       string  `json:"suppId"`
	SuppName     string  `json:"suppName"`
	Lines        []Line  `json:"lines"`
	Total        float64 `json:"total"`
	ApplyDebt    float64 `json:"applyDebt"`
	Refund       float64 `json:"refund"`
	RefundMethod string  `json:"refundMethod"` // cash | bank | credit
	CashID       string  `json:"cashId,omitempty"`
	DebtAfter    float64 `json:"debtAfter"`
	Note         string  `json:"note"`
	Staff        string  `json:"staff"`
	Void         bool    `json:"void"`
}

type SuppReturnInfo struct {
	PID      string  `json:"pid"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Received float64 `json:"received"`
	Returned float64 `json:"returned"`
	Stock    float64 `json:"stock"`
	Left     float64 `json:"left"`  // còn trả được = min(đã nhập − đã trả, tồn kho)
	Price    float64 `json:"price"` // giá nhập bình quân cho 1 đơn vị gốc (đã trừ chiết khấu)
	MaxPrice float64 `json:"maxPrice"`
	Last     int64   `json:"last"`
}

func suppReturnable(s *State, suppID string) map[string]*SuppReturnInfo {
	out := map[string]*SuppReturnInfo{}
	sum := map[string]float64{}
	for _, rc := range s.Receipts {
		if rc.SuppID != suppID || rc.Void { continue }
		f := 1.0
		if rc.Sub > 0 { f = rc.Total / rc.Sub }
		for _, l := range rc.Lines {
			p := s.Products[l.PID]
			if p == nil || l.Qty <= 0 { continue }
			base := l.Qty * ratioOf(l)
			x := out[l.PID]
			if x == nil { x = &SuppReturnInfo{PID: l.PID, Name: p.Name, Unit: p.Unit}; out[l.PID] = x }
			x.Received += base
			sum[l.PID] += l.Qty * l.Price * f
			if u := l.Price * f / ratioOf(l); u > x.MaxPrice { x.MaxPrice = u }
			if rc.Time > x.Last { x.Last = rc.Time }
		}
	}
	for _, r := range s.SuppReturns {
		if r.SuppID != suppID || r.Void { continue }
		for _, l := range r.Lines { if x := out[l.PID]; x != nil { x.Returned += l.Qty } }
	}
	for id, x := range out {
		p := s.Products[id]
		x.Received, x.Returned = r3(x.Received), r3(x.Returned)
		x.Stock = p.Stock
		if x.Stock < 0 { x.Stock = 0 }
		x.Left = r3(x.Received - x.Returned)
		if x.Left > x.Stock { x.Left = x.Stock }
		if x.Received > 0 { x.Price = r0(sum[id] / x.Received) }
		x.MaxPrice = r0(x.MaxPrice + 0.49)
	}
	return out
}

func suppReturnableList(s *State, suppID string) []*SuppReturnInfo {
	m := suppReturnable(s, suppID)
	out := make([]*SuppReturnInfo, 0, len(m))
	for _, x := range m { out = append(out, x) }
	sort.Slice(out, func(i, j int) bool { return out[i].Last > out[j].Last })
	if len(out) > 150 { out = out[:150] }
	return out
}

type SuppReturnReq struct {
	SuppID       string `json:"suppId"`
	Lines        []Line `json:"lines"` // pid, qty (đơn vị gốc), price (giá trả cho 1 đơn vị gốc)
	ToDebt       bool   `json:"toDebt"`
	RefundMethod string `json:"refundMethod"`
	Note         string `json:"note"`
}

func (db *DB) CreateSuppReturn(req SuppReturnReq, u *User) (*SuppReturn, error) {
	var out *SuppReturn
	err := db.Do(func(s *State, t *Tx) error {
		sp := s.Suppliers[req.SuppID]
		if sp == nil { return uerr("Không tìm thấy nhà cung cấp") }
		if len(req.Lines) == 0 { return uerr("Chọn hàng trả nhà cung cấp") }
		info := suppReturnable(s, sp.ID)
		lines := []Line{}
		var total float64
		seen := map[string]bool{}
		for _, l := range req.Lines {
			if l.Qty <= 0 { continue }
			x := info[l.PID]
			if x == nil { return uerr("Chưa nhập mặt hàng này từ %s, không trả được", sp.Name) }
			if seen[l.PID] { return uerr("Mỗi mặt hàng chỉ nhập một dòng") }
			seen[l.PID] = true
			qty := r3(l.Qty)
			if qty > x.Left+0.0005 { return uerr("%s: đã nhập %s, đã trả %s, tồn kho %s, chỉ trả được %s %s", x.Name, fq(x.Received), fq(x.Returned), fq(x.Stock), fq(x.Left), x.Unit) }
			price := l.Price
			if price <= 0 { price = x.Price }
			if price > x.MaxPrice { return uerr("%s: giá trả %s lớn hơn giá đã nhập (%s)", x.Name, money(price), money(x.MaxPrice)) }
			lines = append(lines, Line{PID: l.PID, Name: x.Name, Unit: x.Unit, Ratio: 1, Qty: qty, Price: r0(price)})
			total += qty * r0(price)
		}
		if len(lines) == 0 { return uerr("Nhập số lượng trả") }
		total = r0(total)
		apply := 0.0
		if req.ToDebt && sp.Debt > 0 { apply = total; if sp.Debt < apply { apply = sp.Debt } }
		refund := r0(total - apply)
		method := req.RefundMethod
		if method != "bank" && method != "credit" { method = "cash" }
		now := nowMs()
		r := &SuppReturn{ID: newID(), Code: s.nextCode("TN", 6), Time: now, SuppID: sp.ID, SuppName: sp.Name, Lines: lines, Total: total,
			ApplyDebt: apply, Refund: refund, RefundMethod: method, Note: strings.TrimSpace(req.Note), Staff: u.Name}
		if refund == 0 { r.RefundMethod = "" }
		for _, l := range lines {
			p := s.Products[l.PID]
			p.Stock = r3(p.Stock - l.Qty)
			reconcileLots(p)
			p.Updated = now
			t.put("product", p.ID, p)
		}
		sp.Debt = r0(sp.Debt - apply)
		if refund > 0 && method == "credit" { sp.Debt = r0(sp.Debt - refund) }
		if refund > 0 && method != "credit" {
			e := &CashEntry{ID: newID(), Code: s.nextCode("PT", 6), Time: now, Kind: "thu", Method: method, Amount: refund, Cat: "NCC hoàn tiền trả hàng",
				Note: "Trả hàng nhà cung cấp " + sp.Name + " " + r.Code, Staff: u.Name}
			s.Cash[e.ID] = e
			t.put("cash", e.ID, e)
			r.CashID = e.ID
		}
		sp.Last = now
		r.DebtAfter = sp.Debt
		t.put("supplier", sp.ID, sp)
		s.SuppReturns[r.ID] = r
		t.put("sreturn", r.ID, r)
		t.put("counters", "", s.Counters)
		out = r
		return nil
	})
	return out, err
}

func (db *DB) VoidSuppReturn(id string) error {
	return db.Do(func(s *State, t *Tx) error {
		r := s.SuppReturns[id]
		if r == nil || r.Void { return uerr("Không tìm thấy phiếu trả hàng nhà cung cấp") }
		if sp := s.Suppliers[r.SuppID]; sp != nil {
			sp.Debt = r0(sp.Debt + r.ApplyDebt)
			if r.RefundMethod == "credit" { sp.Debt = r0(sp.Debt + r.Refund) }
			t.put("supplier", sp.ID, sp)
		}
		for _, l := range r.Lines {
			if p := s.Products[l.PID]; p != nil { p.Stock = r3(p.Stock + l.Qty); p.Updated = nowMs(); t.put("product", p.ID, p) }
		}
		if e := s.Cash[r.CashID]; e != nil && !e.Void { e.Void = true; t.put("cash", e.ID, e) }
		r.Void = true
		t.put("sreturn", r.ID, r)
		return nil
	})
}
