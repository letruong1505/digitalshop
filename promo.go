package main

// Khuyến mãi tự động theo mặt hàng/nhóm hàng, theo khung giờ (kể cả qua nửa đêm), theo ngày trong tuần, theo
// khoảng ngày và theo hạng khách hàng. Hạng khách tính từ tổng tiền đã mua (không tính hóa đơn đã hủy).

import (
	"sort"
	"strings"
	"time"
)

type Tier struct {
	Name string  `json:"name"`
	Min  float64 `json:"min"` // tổng đã mua tối thiểu để đạt hạng
}

type Promo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Active bool     `json:"active"`
	Kind   string   `json:"kind"`   // pct: giảm theo %; amt: giảm số tiền trên mỗi đơn vị bán
	Value  float64  `json:"value"`
	Scope  string   `json:"scope"`  // all | group | product
	Target string   `json:"target"` // tên nhóm hàng hoặc mã mặt hàng
	Days   []int    `json:"days"`   // 0 = Chủ nhật … 6 = thứ bảy; rỗng = mọi ngày
	From   string   `json:"from"`   // yyyy-mm-dd, rỗng = không giới hạn
	To     string   `json:"to"`
	TStart string   `json:"tStart"` // HH:MM, rỗng = cả ngày
	TEnd   string   `json:"tEnd"`
	MinTier int     `json:"minTier"` // 0 = mọi khách; n = cần khách đạt hạng thứ n trở lên (tính từ 1)
}

func defaultTiers() []Tier {
	return []Tier{{"Bạc", 2000000}, {"Vàng", 10000000}, {"Kim cương", 30000000}}
}

func tiersOf(s *State) []Tier {
	if len(s.Settings.Tiers) == 0 { return defaultTiers() }
	return s.Settings.Tiers
}

// tierIndex trả về 0 nếu chưa đạt hạng nào, n nếu đạt hạng thứ n (tiers xếp tăng dần theo Min).
func tierIndex(tiers []Tier, spend float64) int {
	n := 0
	for i, t := range tiers { if spend >= t.Min { n = i + 1 } }
	return n
}

func custSpendAll(s *State) map[string]float64 {
	out := map[string]float64{}
	for _, inv := range s.Invoices {
		if inv.Void || inv.CustID == "" { continue }
		out[inv.CustID] += inv.Total
	}
	return out
}

func hhmm(s string) int {
	if len(s) != 5 || s[2] != ':' { return -1 }
	h, m := int(num(s[:2])), int(num(s[3:]))
	if h < 0 || h > 23 || m < 0 || m > 59 { return -1 }
	return h*60 + m
}

func promoOn(p *Promo, now time.Time) bool {
	if !p.Active { return false }
	d := now.Format("2006-01-02")
	if p.From != "" && d < p.From { return false }
	if p.To != "" && d > p.To { return false }
	if len(p.Days) > 0 {
		ok := false
		for _, x := range p.Days { if x == int(now.Weekday()) { ok = true } }
		if !ok { return false }
	}
	a, b := hhmm(p.TStart), hhmm(p.TEnd)
	if a >= 0 && b >= 0 && a != b {
		cur := now.Hour()*60 + now.Minute()
		if a < b { if cur < a || cur >= b { return false } } else if cur < a && cur >= b { return false } // qua nửa đêm
	}
	return true
}

type PromoHit struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// promoDisc tính tổng giảm giá tự động cho các dòng hàng. Mỗi dòng chỉ lấy khuyến mãi có lợi nhất cho khách (không cộng dồn).
func promoDisc(s *State, lines []Line, custID string, now time.Time) (float64, []PromoHit) {
	var act []*Promo
	needSpend := false
	for i := range s.Settings.Promos {
		p := &s.Settings.Promos[i]
		if promoOn(p, now) && p.Value > 0 { act = append(act, p); if p.MinTier > 0 { needSpend = true } }
	}
	if len(act) == 0 { return 0, nil }
	tier := 0
	if needSpend && custID != "" {
		spend := 0.0
		for _, inv := range s.Invoices { if !inv.Void && inv.CustID == custID { spend += inv.Total } }
		tier = tierIndex(tiersOf(s), spend)
	}
	byName := map[string]float64{}
	total := 0.0
	for _, l := range lines {
		amount := l.Qty * l.Price
		if amount <= 0 { continue }
		pr := s.Products[l.PID]
		best, bestName := 0.0, ""
		for _, p := range act {
			if p.MinTier > 0 && tier < p.MinTier { continue }
			switch p.Scope {
			case "product": if p.Target != l.PID { continue }
			case "group":
				if pr == nil || norm(pr.Group) != norm(p.Target) || p.Target == "" { continue }
			}
			d := 0.0
			if p.Kind == "amt" { d = p.Value * l.Qty } else { d = amount * p.Value / 100 }
			if d > amount { d = amount }
			if d > best { best, bestName = d, p.Name }
		}
		if best > 0 { total += best; byName[bestName] += best }
	}
	hits := []PromoHit{}
	for n, a := range byName { hits = append(hits, PromoHit{Name: n, Amount: r0(a)}) }
	sort.Slice(hits, func(i, j int) bool { return strings.Compare(hits[i].Name, hits[j].Name) < 0 })
	return r0(total), hits
}

// SavePromos thay toàn bộ danh sách khuyến mãi và các mức hạng khách.
func (db *DB) SavePromos(promos []Promo, tiers []Tier) error {
	return db.Do(func(s *State, t *Tx) error {
		clean := make([]Promo, 0, len(promos))
		for _, p := range promos {
			p.Name = strings.TrimSpace(p.Name)
			if p.Name == "" { return uerr("Nhập tên chương trình khuyến mãi") }
			if p.ID == "" { p.ID = newID() }
			if p.Kind != "amt" { p.Kind = "pct" }
			if p.Value <= 0 { return uerr("Mức giảm của \"%s\" phải lớn hơn 0", p.Name) }
			if p.Kind == "pct" && p.Value > 100 { return uerr("Giảm theo %% của \"%s\" tối đa 100", p.Name) }
			if p.Scope != "group" && p.Scope != "product" { p.Scope, p.Target = "all", "" }
			if p.Scope != "all" && p.Target == "" { return uerr("Chọn mặt hàng hoặc nhóm hàng cho \"%s\"", p.Name) }
			if (p.TStart != "" || p.TEnd != "") && (hhmm(p.TStart) < 0 || hhmm(p.TEnd) < 0) { return uerr("Khung giờ của \"%s\" nhập dạng 08:00", p.Name) }
			if p.From != "" && p.To != "" && p.From > p.To { return uerr("Ngày bắt đầu của \"%s\" phải trước ngày kết thúc", p.Name) }
			clean = append(clean, p)
		}
		ts := make([]Tier, 0, len(tiers))
		for _, x := range tiers {
			x.Name = strings.TrimSpace(x.Name)
			if x.Name == "" { continue }
			if x.Min < 0 { return uerr("Mức tiền của hạng \"%s\" không hợp lệ", x.Name) }
			ts = append(ts, x)
		}
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Min < ts[j].Min })
		for i := range clean { if clean[i].MinTier > len(ts) && len(ts) > 0 { clean[i].MinTier = len(ts) } }
		s.Settings.Promos, s.Settings.Tiers = clean, ts
		t.put("settings", "", s.Settings)
		return nil
	})
}
