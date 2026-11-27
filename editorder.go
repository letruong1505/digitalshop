package main

import (
	"encoding/json"
	"strings"
)

// Sửa đơn sau thanh toán: hủy hóa đơn cũ và ghi hóa đơn mới trong cùng một lần lưu.
// Kho, công nợ và điểm được hoàn lại rồi tính lại; nếu hóa đơn mới không hợp lệ thì mọi thứ trở về như cũ.

const editWindowMs = 30 * 60 * 1000

func cloneJSON[T any](v *T) *T {
	if v == nil { return nil }
	b, _ := json.Marshal(v)
	var c T
	json.Unmarshal(b, &c)
	return &c
}

func (u *User) mayEditInvoice(inv *Invoice) bool {
	if u.can("manage") { return true }
	return u.can("sell") && inv.Staff == u.Name && nowMs()-inv.Time <= editWindowMs
}

func (db *DB) ReplaceInvoice(oldID string, req SaleReq, u *User) (*Invoice, error) {
	var out *Invoice
	err := db.Do(func(s *State, t *Tx) error {
		old := s.Invoices[oldID]
		if old == nil { return uerr("Không tìm thấy hóa đơn") }
		if old.Void { return uerr("Hóa đơn đã hủy, không sửa được") }
		if old.Src == "import" { return uerr("Hóa đơn nhập từ phần mềm cũ không sửa được") }
		if !u.mayEditInvoice(old) {
			return uerr("Chỉ sửa được hóa đơn của chính bạn trong 30 phút. Hóa đơn cũ hơn cần quyền hủy hóa đơn (quản lý)")
		}
		// lưu bản sao để hoàn tác nếu lỗi
		prods := map[string]*Product{}
		custs := map[string]*Party{}
		keep := func(pid string) {
			if p := s.Products[pid]; p != nil && prods[pid] == nil {
				prods[pid] = cloneJSON(p)
				for _, it := range p.Items { if q := s.Products[it.PID]; q != nil && prods[it.PID] == nil { prods[it.PID] = cloneJSON(q) } }
			}
		}
		for _, l := range old.Lines { keep(l.PID) }
		for _, l := range req.Lines { keep(l.PID) }
		for _, id := range []string{old.CustID, req.CustID} {
			if c := s.Customers[id]; c != nil && id != "" { custs[id] = cloneJSON(c) }
		}
		oldCopy := *old
		counters := map[string]int{}
		for k, v := range s.Counters { counters[k] = v }
		nrec := len(t.recs)
		rollback := func() {
			for id, p := range prods { s.Products[id] = p }
			for id, c := range custs { s.Customers[id] = c }
			*old = oldCopy
			s.Counters = counters
			t.recs = t.recs[:nrec]
		}
		if err := voidInvoiceTx(s, t, oldID, u); err != nil { rollback(); return err }
		old.Note = strings.TrimSpace(oldCopy.Note + " (sửa đơn bởi " + u.Name + ")")
		req.CID, req.Offline, req.OrderID = "", false, ""
		inv, err := createInvoiceTx(s, t, req, u)
		if err != nil { rollback(); return err }
		inv.Time = old.Time
		inv.EditOf = old.Code
		inv.OrderID = old.OrderID
		old.EditedTo = inv.Code
		t.put("invoice", old.ID, old)
		t.put("invoice", inv.ID, inv)
		if old.OrderID != "" {
			if o := s.Orders[old.OrderID]; o != nil && o.InvoiceID == old.ID { o.InvoiceID = inv.ID; t.put("order", o.ID, o) }
		}
		out = inv
		return nil
	})
	return out, err
}
