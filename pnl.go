package main

// Kết quả kinh doanh (lãi/lỗ) và chốt ca cuối ngày.

import (
	"sort"
	"strings"
)

type PnL struct {
	From string `json:"from"`
	To   string `json:"to"`
	Revenue   float64            `json:"revenue"` // doanh thu thuần (đã trừ hàng khách trả)
	Returns   float64            `json:"returns"`
	COGS      float64            `json:"cogs"`
	Gross     float64            `json:"gross"`
	Loss      float64            `json:"loss"` // hàng hủy/hết hạn/kiểm kho thiếu (chỉ tính phần hao hụt, không tính phần tồn đầu kỳ nhập thêm), tính theo giá vốn
	Diff      float64            `json:"diff"` // chênh lệch tiền két khi chốt ca (thừa +, thiếu −)
	OtherIn   []map[string]any   `json:"otherIn"`
	OtherInT  float64            `json:"otherInT"`
	Expenses  []map[string]any   `json:"expenses"`
	ExpenseT  float64            `json:"expenseT"`
	Net       float64            `json:"net"`
	Draw      float64            `json:"draw"`    // chủ rút tiền chi tiêu (không phải chi phí)
	Capital   float64            `json:"capital"` // góp vốn (không phải doanh thu)
	Months    []map[string]any   `json:"months"`
}

// Các khoản trong sổ quỹ KHÔNG phải chi phí / thu nhập kinh doanh.
func pnlSkip(e *CashEntry) bool {
	if e.PartyType == "cust" || e.PartyType == "supp" { return true } // thu nợ khách, trả nợ NCC
	switch e.Cat {
	case "Hoàn tiền khách trả hàng", "NCC hoàn tiền trả hàng", "Góp vốn", "Rút tiền chi tiêu", "Tiền thừa cuối ngày":
		return true
	}
	return false
}

func sortedAmounts(m map[string]float64) []map[string]any {
	keys := make([]string, 0, len(m))
	for k := range m { keys = append(keys, k) }
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	out := []map[string]any{}
	for _, k := range keys { out = append(out, map[string]any{"cat": k, "amount": r0(m[k])}) }
	return out
}

func (db *DB) PnL(from, to string) *PnL {
	rp := db.Report(from, to, true)
	p := &PnL{From: from, To: to, Revenue: r0(rp.Rev), Returns: r0(rp.Returns), COGS: r0(rp.Cost)}
	p.Gross = p.Revenue - p.COGS
	in, out := map[string]float64{}, map[string]float64{}
	mon := map[string]*[3]float64{} // tháng → doanh thu, lãi gộp, chi phí ròng
	db.Read(func(s *State) {
		for _, e := range s.Cash {
			if e.Void || e.Kind == "adj" || !inRange(e.Time, from, to) { continue }
			if e.Cat == "Góp vốn" && e.Kind == "thu" { p.Capital += e.Amount }
			if e.Cat == "Rút tiền chi tiêu" && e.Kind == "chi" { p.Draw += e.Amount }
			if pnlSkip(e) { continue }
			c := e.Cat
			if c == "" { c = "Khác" }
			m := dayOf(e.Time)[:7]
			if mon[m] == nil { mon[m] = &[3]float64{} }
			if e.Kind == "thu" { in[c] += e.Amount; p.OtherInT += e.Amount; mon[m][2] -= e.Amount } else { out[c] += e.Amount; p.ExpenseT += e.Amount; mon[m][2] += e.Amount }
		}
		for _, c := range s.Counts {
			if !inRange(c.Time, from, to) { continue }
			for _, l := range c.Lines {
				if l.After < l.Before { p.Loss += (l.Before - l.After) * l.Cost }
			}
		}
		for _, cl := range s.Closings {
			if inRange(cl.Time, from, to) && !cl.Replaced { p.Diff += cl.Diff }
		}
		for _, inv := range s.Invoices {
			if inv.Void || !inRange(inv.Time, from, to) { continue }
			m := dayOf(inv.Time)[:7]
			if mon[m] == nil { mon[m] = &[3]float64{} }
			mon[m][0] += inv.Total
			mon[m][1] += inv.Total - inv.Cost
		}
		for _, rt := range s.Returns {
			if rt.Void || !inRange(rt.Time, from, to) { continue }
			m := dayOf(rt.Time)[:7]
			if mon[m] == nil { mon[m] = &[3]float64{} }
			cr := 0.0
			if rt.Restock { cr = rt.Cost }
			mon[m][0] -= rt.Total
			mon[m][1] -= rt.Total - cr
		}
	})
	p.Loss, p.Diff = r0(p.Loss), r0(p.Diff)
	p.OtherIn, p.Expenses = sortedAmounts(in), sortedAmounts(out)
	p.OtherInT, p.ExpenseT = r0(p.OtherInT), r0(p.ExpenseT)
	p.Net = p.Gross - p.Loss + p.Diff + p.OtherInT - p.ExpenseT
	keys := []string{}
	for k := range mon { keys = append(keys, k) }
	sort.Strings(keys)
	p.Months = []map[string]any{}
	for _, k := range keys {
		v := mon[k]
		p.Months = append(p.Months, map[string]any{"month": k, "rev": r0(v[0]), "gross": r0(v[1]), "net": r0(v[1] - v[2])})
	}
	return p
}

// ---------- chốt ca cuối ngày ----------

type Closing struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Date     string  `json:"date"`
	Time     int64   `json:"time"`
	Opening  float64 `json:"opening"`  // tiền để lại két từ lần chốt trước
	Net      float64 `json:"net"`      // thu − chi tiền mặt trong ngày (theo phần mềm)
	Expected float64 `json:"expected"` // tiền mặt lẽ ra có trong két
	Counted  float64 `json:"counted"`  // tiền đếm thực tế
	Diff     float64 `json:"diff"`     // counted − expected
	Kept     float64 `json:"kept"`     // tiền để lại két cho ngày mai
	Take     float64 `json:"take"`     // tiền mang đi (counted − kept)
	Rev      float64 `json:"rev"`
	N        int     `json:"n"`
	Bank     float64 `json:"bank"`
	Debt     float64 `json:"debt"`
	Note     string  `json:"note"`
	Staff    string  `json:"staff"`
	Replaced bool    `json:"replaced,omitempty"`
}

type ClosePreview struct {
	Date     string   `json:"date"`
	Opening  float64  `json:"opening"`
	OpenFrom string   `json:"openFrom"` // ngày chốt gần nhất trước đó
	Sales    float64  `json:"sales"`    // tiền mặt bán hàng
	Thu      float64  `json:"thu"`
	Chi      float64  `json:"chi"`
	Buy      float64  `json:"buy"`
	Net      float64  `json:"net"`
	Expected float64  `json:"expected"`
	Rev      float64  `json:"rev"`
	N        int      `json:"n"`
	Bank     float64  `json:"bank"`
	Debt     float64  `json:"debt"`
	Existing *Closing `json:"existing"`
}

func lastClosingBefore(s *State, date string) *Closing {
	var best *Closing
	for _, c := range s.Closings {
		if c.Replaced || c.Date >= date { continue }
		if best == nil || c.Date > best.Date || (c.Date == best.Date && c.Time > best.Time) { best = c }
	}
	return best
}

func (db *DB) ClosePreview(date string) *ClosePreview {
	rp := db.Report(date, date, true)
	cp := &ClosePreview{Date: date, Rev: r0(rp.Rev), N: rp.N, Bank: r0(rp.Bank), Debt: r0(rp.Debt)}
	cp.Sales, cp.Buy = r0(rp.Cash), r0(rp.Flow["buyCash"])
	cp.Thu, cp.Chi = r0(rp.Flow["thu_cash"]), r0(rp.Flow["chi_cash"])
	cp.Net = r0(rp.Flow["netCash"])
	db.Read(func(s *State) {
		if c := lastClosingBefore(s, date); c != nil { cp.Opening, cp.OpenFrom = c.Kept, c.Date }
		for _, c := range s.Closings { if c.Date == date && !c.Replaced { x := *c; cp.Existing = &x } }
	})
	cp.Expected = cp.Opening + cp.Net
	return cp
}

func (db *DB) CreateClosing(date string, counted, kept float64, note string, u *User) (*Closing, error) {
	var out *Closing
	cp := db.ClosePreview(date)
	err := db.Do(func(s *State, t *Tx) error {
		if len(date) != 10 { return uerr("Chọn ngày cần chốt") }
		if counted < 0 || kept < 0 { return uerr("Số tiền không hợp lệ") }
		if kept > counted { return uerr("Tiền để lại két (%s) không được lớn hơn tiền đã đếm (%s)", money(kept), money(counted)) }
		for _, c := range s.Closings {
			if c.Date == date && !c.Replaced { c.Replaced = true; t.put("closing", c.ID, c) }
		}
		c := &Closing{ID: newID(), Code: s.nextCode("CC", 6), Date: date, Time: nowMs(), Opening: cp.Opening, Net: cp.Net, Expected: cp.Expected,
			Counted: r0(counted), Diff: r0(counted - cp.Expected), Kept: r0(kept), Take: r0(counted - kept), Rev: cp.Rev, N: cp.N, Bank: cp.Bank, Debt: cp.Debt,
			Note: strings.TrimSpace(note), Staff: u.Name}
		s.Closings[c.ID] = c
		t.put("closing", c.ID, c)
		t.put("counters", "", s.Counters)
		out = c
		return nil
	})
	return out, err
}

func (db *DB) Closings(limit int) []*Closing {
	var out []*Closing
	db.Read(func(s *State) { for _, c := range s.Closings { if !c.Replaced { out = append(out, c) } } })
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if limit > 0 && len(out) > limit { out = out[:limit] }
	if out == nil { out = []*Closing{} }
	return out
}
