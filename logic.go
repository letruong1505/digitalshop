package main

import (
	"sort"
	"strings"
	"time"
)

// ---------- kho & lô ----------

func lotsSum(p *Product) float64 {
	t := 0.0
	for _, l := range p.Lots { t += l.Qty }
	return r3(t)
}

// reconcileLots đảm bảo tổng lô không vượt tồn kho; bớt lô hết hạn/sớm nhất trước.
func reconcileLots(p *Product) {
	stock := p.Stock
	if stock < 0 { stock = 0 }
	sort.SliceStable(p.Lots, func(i, j int) bool { return p.Lots[i].Exp < p.Lots[j].Exp })
	over := r3(lotsSum(p) - stock)
	for i := 0; over > 0 && i < len(p.Lots); i++ {
		take := p.Lots[i].Qty
		if take > over { take = over }
		p.Lots[i].Qty = r3(p.Lots[i].Qty - take)
		over = r3(over - take)
	}
	out := p.Lots[:0]
	for _, l := range p.Lots { if l.Qty > 0.0005 { out = append(out, l) } }
	p.Lots = out
}

// takeStock trừ kho theo FEFO: lô còn hạn sớm nhất → phần không theo dõi hạn → lô đã hết hạn.
func takeStock(p *Product, q float64, td string) {
	p.Stock = r3(p.Stock - q)
	sort.SliceStable(p.Lots, func(i, j int) bool { return p.Lots[i].Exp < p.Lots[j].Exp })
	rem := q
	for i := range p.Lots {
		if rem <= 0 { break }
		if p.Lots[i].Exp != "" && p.Lots[i].Exp < td { continue }
		t := p.Lots[i].Qty
		if t > rem { t = rem }
		p.Lots[i].Qty = r3(p.Lots[i].Qty - t)
		rem = r3(rem - t)
	}
	reconcileLots(p) // phần còn lại tự trừ vào lô hết hạn nếu tồn không đủ
}

func expiredQty(p *Product, td string) float64 {
	t := 0.0
	for _, l := range p.Lots { if l.Exp != "" && l.Exp < td { t += l.Qty } }
	return r3(t)
}

func comboCost(s *State, p *Product) float64 {
	c := 0.0
	for _, it := range p.Items { if q := s.Products[it.PID]; q != nil { c += q.Cost * it.Qty } }
	return c
}

func comboStock(s *State, p *Product) float64 {
	best := -1.0
	for _, it := range p.Items {
		q := s.Products[it.PID]
		if q == nil || it.Qty <= 0 { return 0 }
		n := float64(int((q.Stock + 1e-9) / it.Qty))
		if best < 0 || n < best { best = n }
	}
	if best < 0 { return 0 }
	return best
}

// ---------- bán hàng ----------

type SaleReq struct {
	CID     string  `json:"cid"`
	Lines   []Line  `json:"lines"`
	Disc    float64 `json:"disc"`
	Method  string  `json:"method"` // cash | bank | debt
	Given   float64 `json:"given"`
	CustID  string  `json:"custId"`
	Note    string  `json:"note"`
	OrderID string  `json:"orderId"`
	// Hóa đơn bán lúc ngoại tuyến trên Android: ghi đúng giờ đã bán và đúng số tiền khách đã trả
	Offline   bool    `json:"offline"`
	Time      int64   `json:"time"`
	PromoDisc float64 `json:"promoDisc"`
	PromoNote string  `json:"promoNote"`
	Points    int     `json:"points"` // điểm khách muốn dùng để giảm giá
}

func (db *DB) CreateInvoice(req SaleReq, u *User) (*Invoice, error) {
	var out *Invoice
	err := db.Do(func(s *State, t *Tx) error {
		if req.CID != "" {
			if inv, ok := s.Invoices[req.CID]; ok { out = inv; return nil } // gửi lại do mạng chập chờn
		}
		if len(req.Lines) == 0 { return uerr("Hóa đơn chưa có mặt hàng nào") }
		td := today()
		need := map[string]float64{}
		lines := make([]Line, 0, len(req.Lines))
		sub, cost := 0.0, 0.0
		for _, l := range req.Lines {
			p := s.Products[l.PID]
			if p == nil { return uerr("Mặt hàng \"%s\" không còn trong danh mục", l.Name) }
			if l.Qty <= 0 { continue }
			if l.Ratio <= 0 { l.Ratio = 1 }
			if l.Price < 0 { return uerr("Giá bán không hợp lệ: %s", p.Name) }
			if !u.editPrice(&s.Settings) {
				l.Price = unitPrice(p, l.Unit, l.Ratio)
			}
			base := r3(l.Qty * l.Ratio)
			unitCost := 0.0
			if p.IsCombo {
				for _, it := range p.Items { need[it.PID] += r3(it.Qty * base) }
				unitCost = comboCost(s, p) * l.Ratio
			} else {
				need[p.ID] += base
				unitCost = p.Cost * l.Ratio
			}
			unit := l.Unit
			if unit == "" { unit = p.Unit }
			lines = append(lines, Line{PID: p.ID, Code: p.Code, Name: p.Name, Unit: unit, Ratio: l.Ratio, Qty: r3(l.Qty), Price: r0(l.Price), Cost: r0(unitCost)})
			sub += l.Qty * l.Price
			cost += l.Qty * unitCost
		}
		if len(lines) == 0 { return uerr("Hóa đơn chưa có mặt hàng nào") }
		for pid, q := range need {
			p := s.Products[pid]
			if p == nil { return uerr("Combo có thành phần đã bị xóa khỏi danh mục") }
			avail := p.Stock
			if s.Settings.BlockExpired { avail = r3(p.Stock - expiredQty(p, td)) }
			if !s.Settings.SellNegative && q > avail+1e-9 {
				if s.Settings.BlockExpired && expiredQty(p, td) > 0 {
					return uerr("Không đủ hàng còn hạn: %s chỉ còn %s %s (đã loại hàng hết hạn)", p.Name, fq(avail), p.Unit)
				}
				return uerr("Không đủ hàng: %s chỉ còn %s %s", p.Name, fq(avail), p.Unit)
			}
		}
		sub = r0(sub)
		promo, hits := promoDisc(s, lines, req.CustID, time.Now())
		saleTime := nowMs()
		if req.Offline {
			if req.Time > saleTime-30*24*3600*1000 && req.Time <= saleTime+5*60*1000 { saleTime = req.Time }
			promo, hits = req.PromoDisc, nil
			if promo < 0 { promo = 0 }
			if strings.TrimSpace(req.PromoNote) != "" { hits = []PromoHit{{Name: strings.TrimSpace(req.PromoNote)}} }
		}
		if promo > sub { promo = sub }
		disc := r0(req.Disc)
		if disc < 0 { disc = 0 }
		if disc > sub-promo { disc = sub - promo }
		var cust *Party
		if req.CustID != "" {
			cust = s.Customers[req.CustID]
			if cust == nil { return uerr("Không tìm thấy khách hàng") }
		}
		lo := s.Settings.Loyalty
		ptsUsed, ptsDisc := 0, 0.0
		if req.Points > 0 {
			if !lo.On { return uerr("Cửa hàng chưa bật tích điểm") }
			if cust == nil { return uerr("Chọn khách hàng để dùng điểm") }
			if req.Points > cust.Points { return uerr("Khách chỉ có %d điểm", cust.Points) }
			if lo.MinRedeem > 0 && req.Points < lo.MinRedeem { return uerr("Mỗi lần dùng tối thiểu %d điểm", lo.MinRedeem) }
			if mx := lo.maxRedeem(cust.Points, sub-promo-disc); req.Points > mx { return uerr("Hóa đơn này dùng tối đa %d điểm", mx) }
			ptsUsed, ptsDisc = req.Points, r0(float64(req.Points)*lo.PointValue)
			if ptsDisc > sub-promo-disc { ptsDisc = sub - promo - disc }
		}
		total := sub - disc - promo - ptsDisc
		promoNote := ""
		for i, h := range hits { if i > 0 { promoNote += ", " }; promoNote += h.Name }
		inv := &Invoice{ID: req.CID, Time: saleTime, Lines: lines, Sub: sub, Disc: disc + promo + ptsDisc, Promo: promo, PtsUsed: ptsUsed, PtsDisc: ptsDisc, PromoNote: promoNote, Total: total, Cost: r0(cost), Staff: u.Name, Note: strings.TrimSpace(req.Note)}
		if inv.ID == "" { inv.ID = newID() }
		if cust != nil { inv.CustID, inv.CustName = cust.ID, cust.Name }
		switch req.Method {
		case "bank":
			inv.Bank = total
		case "debt":
			if cust == nil { return uerr("Chọn khách hàng để ghi nợ") }
			inv.DebtAmt = total
		default:
			given := r0(req.Given)
			if given <= 0 { given = total }
			inv.Given = given
			if given >= total {
				inv.Cash = total
			} else {
				if cust == nil { return uerr("Khách còn thiếu %s. Chọn khách hàng để ghi nợ phần còn lại", money(total-given)) }
				inv.Cash = given
				inv.DebtAmt = total - given
			}
		}
		inv.Code = s.nextCode("HD", 6)
		for pid, q := range need {
			p := s.Products[pid]
			takeStock(p, q, td)
			p.Updated = nowMs()
			t.put("product", p.ID, p)
		}
		if cust != nil {
			cust.Debt = r0(cust.Debt + inv.DebtAmt)
			cust.Last = inv.Time
			inv.PtsEarned = lo.earn(total)
			cust.Points += inv.PtsEarned - ptsUsed
			if cust.Points < 0 { cust.Points = 0 }
			inv.PtsBal = cust.Points
			t.put("customer", cust.ID, cust)
		}
		if req.OrderID != "" {
			if o := s.Orders[req.OrderID]; o != nil && o.Status == "new" {
				o.Status, o.InvoiceID = "done", inv.ID
				inv.OrderID = o.ID
				t.put("order", o.ID, o)
			}
		}
		s.Invoices[inv.ID] = inv
		t.put("invoice", inv.ID, inv)
		t.put("counters", "", s.Counters)
		out = inv
		return nil
	})
	return out, err
}

func unitPrice(p *Product, unit string, ratio float64) float64 {
	if ratio <= 1 || unit == p.Unit { return p.Price * ratio }
	for _, c := range p.Units { if c.Name == unit && c.Ratio == ratio { return c.Price } }
	return p.Price * ratio
}

func (db *DB) VoidInvoice(id string, u *User) error {
	return db.Do(func(s *State, t *Tx) error {
		inv := s.Invoices[id]
		if inv == nil { return uerr("Không tìm thấy hóa đơn") }
		if inv.Void { return uerr("Hóa đơn đã hủy trước đó") }
		if inv.Src != "import" {
			for _, l := range inv.Lines {
				p := s.Products[l.PID]
				if p == nil { continue }
				base := r3(l.Qty * l.Ratio)
				if p.IsCombo {
					for _, it := range p.Items {
						if q := s.Products[it.PID]; q != nil { q.Stock = r3(q.Stock + it.Qty*base); t.put("product", q.ID, q) }
					}
				} else {
					p.Stock = r3(p.Stock + base)
					t.put("product", p.ID, p)
				}
			}
			if inv.CustID != "" && inv.DebtAmt != 0 {
				if c := s.Customers[inv.CustID]; c != nil { c.Debt = r0(c.Debt - inv.DebtAmt); t.put("customer", c.ID, c) }
			}
			if inv.CustID != "" && (inv.PtsUsed != 0 || inv.PtsEarned != 0) {
				if c := s.Customers[inv.CustID]; c != nil {
					c.Points += inv.PtsUsed - inv.PtsEarned
					if c.Points < 0 { c.Points = 0 }
					t.put("customer", c.ID, c)
				}
			}
		}
		inv.Void, inv.VoidAt = true, nowMs()
		inv.Note = strings.TrimSpace(inv.Note + " (hủy bởi " + u.Name + ")")
		t.put("invoice", inv.ID, inv)
		return nil
	})
}

// ---------- nhập hàng ----------

type ReceiptReq struct {
	CID    string  `json:"cid"`
	SuppID string  `json:"suppId"`
	Lines  []Line  `json:"lines"`
	Disc   float64 `json:"disc"`
	Paid   float64 `json:"paid"`
	Method string  `json:"method"`
	Note   string  `json:"note"`
}

func (db *DB) CreateReceipt(req ReceiptReq, u *User) (*Receipt, error) {
	var out *Receipt
	err := db.Do(func(s *State, t *Tx) error {
		if req.CID != "" { if r, ok := s.Receipts[req.CID]; ok { out = r; return nil } }
		if len(req.Lines) == 0 { return uerr("Phiếu nhập chưa có mặt hàng") }
		rc := &Receipt{ID: req.CID, Time: nowMs(), Staff: u.Name, Note: strings.TrimSpace(req.Note), Method: req.Method}
		if rc.ID == "" { rc.ID = newID() }
		if rc.Method != "bank" { rc.Method = "cash" }
		var sup *Party
		if req.SuppID != "" {
			sup = s.Suppliers[req.SuppID]
			if sup == nil { return uerr("Không tìm thấy nhà cung cấp") }
			rc.SuppID, rc.SuppName = sup.ID, sup.Name
		}
		sub := 0.0
		for _, l := range req.Lines {
			p := s.Products[l.PID]
			if p == nil || p.IsCombo { return uerr("Mặt hàng không hợp lệ trong phiếu nhập") }
			if l.Qty <= 0 { continue }
			if l.Ratio <= 0 { l.Ratio = 1 }
			if l.Unit == "" { l.Unit = p.Unit }
			rc.Lines = append(rc.Lines, Line{PID: p.ID, Code: p.Code, Name: p.Name, Unit: l.Unit, Ratio: l.Ratio, Qty: r3(l.Qty), Price: r0(l.Price), Exp: l.Exp})
			sub += l.Qty * l.Price
		}
		if len(rc.Lines) == 0 { return uerr("Phiếu nhập chưa có số lượng") }
		rc.Sub = r0(sub)
		rc.Disc = r0(req.Disc)
		if rc.Disc < 0 { rc.Disc = 0 }
		if rc.Disc > rc.Sub { rc.Disc = rc.Sub }
		rc.Total = rc.Sub - rc.Disc
		rc.Paid = r0(req.Paid)
		if rc.Paid > rc.Total { rc.Paid = rc.Total }
		if rc.Paid < 0 { rc.Paid = 0 }
		rc.DebtAmt = rc.Total - rc.Paid
		if rc.DebtAmt > 0 && sup == nil { return uerr("Chọn nhà cung cấp để ghi nợ phần chưa trả %s", money(rc.DebtAmt)) }
		factor := 1.0
		if rc.Sub > 0 { factor = rc.Total / rc.Sub }
		td := today()
		for _, l := range rc.Lines {
			p := s.Products[l.PID]
			base := r3(l.Qty * l.Ratio)
			uc := l.Price / l.Ratio * factor
			s0 := p.Stock
			if s0 < 0 { s0 = 0 }
			if s0+base > 0 { p.Cost = r0((s0*p.Cost + base*uc) / (s0 + base)) }
			p.Stock = r3(p.Stock + base)
			if l.Exp != "" {
				p.Lots = append(p.Lots, Lot{ID: newID(), Qty: base, Exp: l.Exp, In: td, Ref: rc.ID})
			}
			reconcileLots(p)
			p.Updated = nowMs()
			t.put("product", p.ID, p)
		}
		if sup != nil {
			sup.Debt = r0(sup.Debt + rc.DebtAmt)
			sup.Last = rc.Time
			t.put("supplier", sup.ID, sup)
		}
		rc.Code = s.nextCode("PN", 6)
		s.Receipts[rc.ID] = rc
		t.put("receipt", rc.ID, rc)
		t.put("counters", "", s.Counters)
		out = rc
		return nil
	})
	return out, err
}

func (db *DB) VoidReceipt(id string) error {
	return db.Do(func(s *State, t *Tx) error {
		rc := s.Receipts[id]
		if rc == nil { return uerr("Không tìm thấy phiếu nhập") }
		if rc.Void { return uerr("Phiếu đã hủy") }
		for _, l := range rc.Lines {
			p := s.Products[l.PID]
			if p == nil { continue }
			base := r3(l.Qty * l.Ratio)
			for i := range p.Lots {
				if p.Lots[i].Ref == rc.ID {
					take := p.Lots[i].Qty
					if take > base { take = base }
					p.Lots[i].Qty = r3(p.Lots[i].Qty - take)
				}
			}
			p.Stock = r3(p.Stock - base)
			reconcileLots(p)
			t.put("product", p.ID, p)
		}
		if rc.SuppID != "" && rc.DebtAmt != 0 {
			if sp := s.Suppliers[rc.SuppID]; sp != nil { sp.Debt = r0(sp.Debt - rc.DebtAmt); t.put("supplier", sp.ID, sp) }
		}
		rc.Void = true
		t.put("receipt", rc.ID, rc)
		// phiếu tạo từ hóa đơn đầu vào: đưa hóa đơn về trạng thái chờ duyệt
		for _, x := range s.InInvs {
			if x.ReceiptID == rc.ID { x.Status, x.ReceiptID, x.ReceiptCode, x.Done = "new", "", "", 0; t.put("inInvoice", x.ID, x) }
		}
		return nil
	})
}

// ---------- kiểm kho / xuất hủy ----------

type CountReq struct {
	Lines []struct {
		PID    string  `json:"pid"`
		Actual float64 `json:"actual"`
	} `json:"lines"`
	Note string `json:"note"`
}

func (db *DB) CreateCount(req CountReq, u *User) (*Count, error) {
	var out *Count
	err := db.Do(func(s *State, t *Tx) error {
		c := &Count{ID: newID(), Time: nowMs(), Note: strings.TrimSpace(req.Note), Kind: "count", Staff: u.Name}
		for _, l := range req.Lines {
			p := s.Products[l.PID]
			if p == nil || p.IsCombo { continue }
			if l.Actual < 0 { return uerr("Số lượng thực tế không được âm: %s", p.Name) }
			c.Lines = append(c.Lines, CountLine{PID: p.ID, Name: p.Name, Unit: p.Unit, Before: p.Stock, After: r3(l.Actual), Cost: p.Cost})
			p.Stock = r3(l.Actual)
			reconcileLots(p)
			p.Updated = nowMs()
			t.put("product", p.ID, p)
		}
		if len(c.Lines) == 0 { return uerr("Phiếu kiểm chưa có mặt hàng") }
		c.Code = s.nextCode("KK", 6)
		s.Counts[c.ID] = c
		t.put("count", c.ID, c)
		t.put("counters", "", s.Counters)
		out = c
		return nil
	})
	return out, err
}

// DisposeExpired xuất hủy toàn bộ lô đã hết hạn của các mặt hàng chỉ định (rỗng = tất cả).
func (db *DB) DisposeExpired(pids []string, u *User) (*Count, error) {
	var out *Count
	err := db.Do(func(s *State, t *Tx) error {
		td := today()
		want := map[string]bool{}
		for _, id := range pids { want[id] = true }
		c := &Count{ID: newID(), Time: nowMs(), Note: "Xuất hủy hàng hết hạn", Kind: "dispose", Staff: u.Name}
		for _, p := range s.Products {
			if len(want) > 0 && !want[p.ID] { continue }
			q := expiredQty(p, td)
			if q <= 0 { continue }
			before := p.Stock
			keep := p.Lots[:0]
			for _, l := range p.Lots { if !(l.Exp != "" && l.Exp < td) { keep = append(keep, l) } }
			p.Lots = keep
			p.Stock = r3(p.Stock - q)
			c.Lines = append(c.Lines, CountLine{PID: p.ID, Name: p.Name, Unit: p.Unit, Before: before, After: p.Stock, Cost: p.Cost})
			t.put("product", p.ID, p)
		}
		if len(c.Lines) == 0 { return uerr("Không có hàng hết hạn để hủy") }
		c.Code = s.nextCode("XH", 6)
		s.Counts[c.ID] = c
		t.put("count", c.ID, c)
		t.put("counters", "", s.Counters)
		out = c
		return nil
	})
	return out, err
}

// ---------- sổ quỹ & công nợ ----------

func (db *DB) CreateCash(e CashEntry, u *User) (*CashEntry, error) { return db.CreateCashCap(e, u, false) }

// CreateCashCap: capDebt = true thì phiếu thu nợ khách không được lớn hơn số khách đang nợ (chặn gõ thừa số 0).
func (db *DB) CreateCashCap(e CashEntry, u *User, capDebt bool) (*CashEntry, error) {
	var out *CashEntry
	err := db.Do(func(s *State, t *Tx) error {
		e.Amount = r0(e.Amount)
		if e.Kind != "thu" && e.Kind != "chi" && e.Kind != "adj" { return uerr("Loại phiếu không hợp lệ") }
		if e.Amount == 0 || (e.Kind != "adj" && e.Amount < 0) { return uerr("Nhập số tiền") }
		if e.Method != "bank" { e.Method = "cash" }
		e.ID, e.Time, e.Staff, e.Void = newID(), nowMs(), u.Name, false
		var party *Party
		switch e.PartyType {
		case "cust":
			party = s.Customers[e.PartyID]
		case "supp":
			party = s.Suppliers[e.PartyID]
		default:
			e.PartyType, e.PartyID = "", ""
		}
		if e.PartyType != "" {
			if party == nil { return uerr("Không tìm thấy đối tượng công nợ") }
			e.PartyName = party.Name
			if capDebt && e.PartyType == "cust" && e.Kind == "thu" {
				if party.Debt <= 0 { return uerr("%s hiện không còn nợ", party.Name) }
				if e.Amount > party.Debt+0.5 { return uerr("%s chỉ đang nợ %s, không thu quá số nợ", party.Name, money(party.Debt)) }
			}
			party.Debt = r0(party.Debt + debtEffect(e))
			e.DebtAfter = party.Debt
			party.Last = e.Time
			if e.PartyType == "cust" { t.put("customer", party.ID, party) } else { t.put("supplier", party.ID, party) }
		}
		switch e.Kind {
		case "thu":
			e.Code = s.nextCode("PT", 6)
		case "chi":
			e.Code = s.nextCode("PC", 6)
		default:
			e.Code = s.nextCode("DC", 6)
		}
		s.Cash[e.ID] = &e
		t.put("cash", e.ID, &e)
		t.put("counters", "", s.Counters)
		out = &e
		return nil
	})
	return out, err
}

// debtEffect: thay đổi công nợ của đối tượng khi ghi phiếu.
func debtEffect(e CashEntry) float64 {
	switch {
	case e.Kind == "adj":
		return e.Amount
	case e.PartyType == "cust" && e.Kind == "thu":
		return -e.Amount
	case e.PartyType == "cust" && e.Kind == "chi":
		return e.Amount
	case e.PartyType == "supp" && e.Kind == "chi":
		return -e.Amount
	case e.PartyType == "supp" && e.Kind == "thu":
		return e.Amount
	}
	return 0
}

func (db *DB) VoidCash(id string) error {
	return db.Do(func(s *State, t *Tx) error {
		e := s.Cash[id]
		if e == nil || e.Void { return uerr("Không tìm thấy phiếu") }
		if e.PartyType == "cust" {
			if c := s.Customers[e.PartyID]; c != nil { c.Debt = r0(c.Debt - debtEffect(*e)); t.put("customer", c.ID, c) }
		} else if e.PartyType == "supp" {
			if c := s.Suppliers[e.PartyID]; c != nil { c.Debt = r0(c.Debt - debtEffect(*e)); t.put("supplier", c.ID, c) }
		}
		e.Void = true
		t.put("cash", e.ID, e)
		return nil
	})
}

// ---------- mặt hàng ----------

type ProductReq struct {
	Product
	SetStock *float64 `json:"setStock"`
	StockExp string   `json:"stockExp"`
}

func (db *DB) SaveProduct(req ProductReq, u *User) (*Product, error) {
	var out *Product
	err := db.Do(func(s *State, t *Tx) error {
		in := req.Product
		in.Name = strings.TrimSpace(in.Name)
		in.Unit = strings.TrimSpace(in.Unit)
		in.Code = strings.TrimSpace(in.Code)
		in.Barcode = strings.TrimSpace(in.Barcode)
		if in.Name == "" { return uerr("Nhập tên hàng") }
		if in.IsCombo {
			if in.Unit == "" { in.Unit = "combo" }
			if len(in.Items) == 0 { return uerr("Combo cần ít nhất một mặt hàng thành phần") }
			for _, it := range in.Items {
				q := s.Products[it.PID]
				if q == nil || q.IsCombo || it.Qty <= 0 { return uerr("Thành phần combo không hợp lệ") }
			}
		} else if in.Unit == "" {
			return uerr("Nhập đơn vị tính")
		}
		if in.Price < 0 || in.Cost < 0 { return uerr("Giá không được âm") }
		for _, other := range s.Products {
			if other.ID == in.ID { continue }
			if in.Code != "" && strings.EqualFold(other.Code, in.Code) { return uerr("Mã hàng %s đã dùng cho \"%s\"", in.Code, other.Name) }
			if in.Barcode != "" && other.Barcode == in.Barcode { return uerr("Mã vạch %s đã dùng cho \"%s\"", in.Barcode, other.Name) }
		}
		units := in.Units[:0]
		for _, c := range in.Units {
			c.Name = strings.TrimSpace(c.Name)
			if c.Name == "" || c.Ratio <= 1 { continue }
			if c.Price <= 0 { c.Price = r0(in.Price * c.Ratio) }
			units = append(units, c)
		}
		in.Units = units
		old := s.Products[in.ID]
		if old == nil {
			if in.ID == "" { in.ID = newID() }
			in.Created = nowMs()
			in.Stock, in.Lots = 0, nil
		} else {
			in.Stock, in.Lots, in.Created = old.Stock, old.Lots, old.Created
			if req.Lots != nil { // chỉnh hạn của lô (không đổi số lượng tổng)
				byID := map[string]Lot{}
				for _, l := range req.Lots { byID[l.ID] = l }
				lots := []Lot{}
				for _, l := range old.Lots {
					if nl, ok := byID[l.ID]; ok {
						l.Exp = nl.Exp
						if nl.Qty >= 0 { l.Qty = r3(nl.Qty) }
						lots = append(lots, l)
					}
				}
				in.Lots = lots
			}
		}
		if in.Code == "" { in.Code = s.nextCode("SP", 6); t.put("counters", "", s.Counters) }
		if in.IsCombo { in.Stock, in.Lots, in.Units = 0, nil, nil }
		in.Updated = nowMs()
		p := &in
		if req.SetStock != nil && !in.IsCombo && r3(*req.SetStock) != p.Stock {
			c := &Count{ID: newID(), Time: nowMs(), Kind: "count", Staff: u.Name, Note: "Sửa tồn khi lưu mặt hàng"}
			if old == nil { c.Note = "Tồn đầu kỳ" }
			c.Lines = []CountLine{{PID: p.ID, Name: p.Name, Unit: p.Unit, Before: p.Stock, After: r3(*req.SetStock), Cost: p.Cost}}
			if old == nil && req.StockExp != "" && *req.SetStock > 0 {
				p.Lots = append(p.Lots, Lot{ID: newID(), Qty: r3(*req.SetStock), Exp: req.StockExp, In: today()})
			}
			p.Stock = r3(*req.SetStock)
			c.Code = s.nextCode("KK", 6)
			s.Counts[c.ID] = c
			t.put("count", c.ID, c)
			t.put("counters", "", s.Counters)
		}
		reconcileLots(p)
		s.Products[p.ID] = p
		t.put("product", p.ID, p)
		out = p
		return nil
	})
	return out, err
}

func (db *DB) DeleteProduct(id string) error {
	return db.Do(func(s *State, t *Tx) error {
		p := s.Products[id]
		if p == nil { return uerr("Không tìm thấy mặt hàng") }
		for _, c := range s.Products {
			if c.IsCombo { for _, it := range c.Items { if it.PID == id { return uerr("Mặt hàng đang nằm trong combo \"%s\". Xóa hoặc sửa combo trước", c.Name) } } }
		}
		delete(s.Products, id)
		t.del("product", id)
		return nil
	})
}

func (db *DB) SaveParty(kind string, in Party) (*Party, error) {
	var out *Party
	err := db.Do(func(s *State, t *Tx) error {
		m, prefix, rk := s.Customers, "KH", "customer"
		if kind == "supp" { m, prefix, rk = s.Suppliers, "NCC", "supplier" }
		in.Name = strings.TrimSpace(in.Name)
		in.Phone = strings.TrimSpace(in.Phone)
		if in.Name == "" { return uerr("Nhập tên") }
		old := m[in.ID]
		if old == nil {
			if in.ID == "" { in.ID = newID() }
			in.Created, in.Last = nowMs(), nowMs()
			opening := r0(in.Debt)
			in.Debt = 0
			if in.Code == "" { in.Code = s.nextCode(prefix, 6) }
			p := &in
			m[p.ID] = p
			if opening != 0 {
				e := &CashEntry{ID: newID(), Time: nowMs(), Kind: "adj", Amount: opening, Cat: "Nợ đầu kỳ", PartyType: map[string]string{"customer": "cust", "supplier": "supp"}[rk], PartyID: p.ID, PartyName: p.Name, Staff: "hệ thống"}
				e.Code = s.nextCode("DC", 6)
				p.Debt = opening
				s.Cash[e.ID] = e
				t.put("cash", e.ID, e)
			}
			t.put("counters", "", s.Counters)
			t.put(rk, p.ID, p)
			out = p
			return nil
		}
		old.Name, old.Phone, old.Addr, old.Note = in.Name, in.Phone, in.Addr, in.Note
		if in.TaxCode != "" { old.TaxCode = in.TaxCode }
		if in.Code != "" { old.Code = in.Code }
		t.put(rk, old.ID, old)
		out = old
		return nil
	})
	return out, err
}

func (db *DB) DeleteParty(kind, id string) error {
	return db.Do(func(s *State, t *Tx) error {
		m, rk := s.Customers, "customer"
		if kind == "supp" { m, rk = s.Suppliers, "supplier" }
		p := m[id]
		if p == nil { return uerr("Không tìm thấy") }
		if p.Debt != 0 { return uerr("Còn công nợ %s, chưa xóa được", money(p.Debt)) }
		delete(m, id)
		t.del(rk, id)
		return nil
	})
}

// ---------- đơn đặt ----------

func (db *DB) SaveOrder(o Order, u *User) (*Order, error) {
	var out *Order
	err := db.Do(func(s *State, t *Tx) error {
		lines := o.Lines[:0]
		for _, l := range o.Lines {
			p := s.Products[l.PID]
			if p == nil || l.Qty <= 0 { continue }
			if l.Ratio <= 0 { l.Ratio = 1 }
			if l.Unit == "" { l.Unit = p.Unit }
			l.Name = p.Name
			if l.Price <= 0 { l.Price = unitPrice(p, l.Unit, l.Ratio) }
			lines = append(lines, l)
		}
		o.Lines = lines
		if len(o.Lines) == 0 { return uerr("Đơn chưa có mặt hàng nào khớp với kho") }
		if o.CustID != "" {
			if c := s.Customers[o.CustID]; c != nil { o.CustName = c.Name; if o.Phone == "" { o.Phone = c.Phone } } else { o.CustID = "" }
		}
		if strings.TrimSpace(o.CustName) == "" { o.CustName = "Khách lẻ" }
		old := s.Orders[o.ID]
		if old == nil {
			if o.ID == "" { o.ID = newID() }
			o.Time, o.Status, o.Staff = nowMs(), "new", u.Name
			o.Code = s.nextCode("DH", 6)
			t.put("counters", "", s.Counters)
		} else {
			o.Time, o.Code, o.Staff, o.Status, o.InvoiceID = old.Time, old.Code, old.Staff, old.Status, old.InvoiceID
		}
		p := &o
		s.Orders[p.ID] = p
		t.put("order", p.ID, p)
		out = p
		return nil
	})
	return out, err
}

func (db *DB) SetOrderStatus(id, st string) error {
	return db.Do(func(s *State, t *Tx) error {
		o := s.Orders[id]
		if o == nil { return uerr("Không tìm thấy đơn") }
		if st != "new" && st != "cancel" && st != "done" { return uerr("Trạng thái không hợp lệ") }
		o.Status = st
		t.put("order", o.ID, o)
		return nil
	})
}

// ---------- định dạng ----------

func money(x float64) string { return fnum(r0(x)) + "đ" }

func fnum(x float64) string {
	neg := x < 0
	if neg { x = -x }
	n := int64(r0(x))
	s := ""
	str := strings.TrimLeft(strings.Repeat(" ", 0)+itoa(n), " ")
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 { s += "." }
		s += string(c)
	}
	if neg { s = "-" + s }
	return s
}

func itoa(n int64) string {
	if n == 0 { return "0" }
	b := []byte{}
	for n > 0 { b = append([]byte{byte('0' + n%10)}, b...); n /= 10 }
	return string(b)
}

func fq(x float64) string {
	x = r3(x)
	if x == float64(int64(x)) { return fnum(x) }
	s := strings.TrimRight(strings.TrimRight(strings.Replace(ftoa3(x), ".", ",", 1), "0"), ",")
	return s
}

func ftoa3(x float64) string {
	neg := x < 0
	if neg { x = -x }
	i := int64(x)
	f := int64(r0((x - float64(i)) * 1000))
	if f >= 1000 { i++; f -= 1000 }
	s := itoa(i) + "." + strings.Repeat("0", 3-len(itoa(f))) + itoa(f)
	if f == 0 { s = itoa(i) + ".000" }
	if neg { s = "-" + s }
	return s
}

func addDays(d string, n int) string {
	t, err := time.ParseInLocation("2006-01-02", d, time.Local)
	if err != nil { return d }
	return t.AddDate(0, 0, n).Format("2006-01-02")
}
