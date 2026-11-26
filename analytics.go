package main

// Báo cáo và gợi ý combo — tính toán thuần, không cần AI.

import (
	"math"
	"sort"
	"time"
)

type Report struct {
	From     string           `json:"from"`
	To       string           `json:"to"`
	Rev      float64          `json:"rev"`
	Cost     float64          `json:"cost"`
	Profit   float64          `json:"profit"`
	N        int              `json:"n"`
	Disc     float64          `json:"disc"`
	Cash     float64          `json:"cash"`
	Bank     float64          `json:"bank"`
	Debt     float64          `json:"debt"`
	Voids    int              `json:"voids"`
	Series   []map[string]any `json:"series"`
	Hourly   bool             `json:"hourly"`
	Top      []map[string]any `json:"top"`
	Groups   []map[string]any `json:"groups"`
	Staff    []map[string]any `json:"staff"`
	Custs    []map[string]any `json:"custs"`
	Flow     map[string]float64 `json:"flow"`
	Imported int              `json:"imported"`
}

func inRange(ms int64, from, to string) bool {
	d := dayOf(ms)
	return d >= from && d <= to
}

func (db *DB) Report(from, to string, owner bool) *Report {
	r := &Report{From: from, To: to, Hourly: from == to, Flow: map[string]float64{}, Groups: []map[string]any{}, Staff: []map[string]any{}, Custs: []map[string]any{}}
	db.Read(func(s *State) {
		type agg struct{ name, unit string; qty, rev, prof float64; n int }
		top := map[string]*agg{}
		grp := map[string]*[2]float64{}
		staff := map[string]*[2]float64{}
		cust := map[string]*[3]float64{}
		series := map[string]*[3]float64{}
		for _, inv := range s.Invoices {
			if !inRange(inv.Time, from, to) { continue }
			if inv.Void { r.Voids++; continue }
			if inv.Src == "import" { r.Imported++ }
			r.N++
			r.Rev += inv.Total
			r.Cost += inv.Cost
			r.Disc += inv.Disc
			if inv.Src != "import" { r.Cash += inv.Cash; r.Bank += inv.Bank; r.Debt += inv.DebtAmt }
			k := dayOf(inv.Time)
			if r.Hourly { k = time.UnixMilli(inv.Time).Format("15") }
			if series[k] == nil { series[k] = &[3]float64{} }
			series[k][0] += inv.Total
			series[k][1] += inv.Total - inv.Cost
			series[k][2]++
			if inv.CustID != "" {
				if cust[inv.CustID] == nil { cust[inv.CustID] = &[3]float64{} }
				cust[inv.CustID][0] += inv.Total
				cust[inv.CustID][1]++
				cust[inv.CustID][2] += inv.Total - inv.Cost
			}
			st := inv.Staff
			if staff[st] == nil { staff[st] = &[2]float64{} }
			staff[st][0] += inv.Total
			staff[st][1]++
			share := 1.0
			if inv.Sub > 0 { share = inv.Total / inv.Sub }
			for _, l := range inv.Lines {
				id := l.PID
				if id == "" { id = "n:" + norm(l.Name) }
				a := top[id]
				if a == nil {
					a = &agg{name: l.Name, unit: l.Unit}
					if p := s.Products[l.PID]; p != nil { a.name, a.unit = p.Name, p.Unit }
					top[id] = a
				}
				rev := l.Qty * l.Price * share
				a.qty += l.Qty * l.Ratio
				a.rev += rev
				a.prof += rev - l.Qty*l.Cost
				a.n++
				g := "Chưa phân nhóm"
				if p := s.Products[l.PID]; p != nil && p.Group != "" { g = p.Group }
				if p := s.Products[l.PID]; p != nil && p.IsCombo { g = "Combo" }
				if grp[g] == nil { grp[g] = &[2]float64{} }
				grp[g][0] += rev
				grp[g][1] += rev - l.Qty*l.Cost
			}
		}
		r.Profit = r.Rev - r.Cost
		if r.Hourly {
			for h := 6; h <= 22; h++ {
				k := time.Date(2000, 1, 1, h, 0, 0, 0, time.Local).Format("15")
				v := series[k]
				if v == nil { v = &[3]float64{} }
				r.Series = append(r.Series, map[string]any{"k": k + "h", "rev": r0(v[0]), "profit": r0(v[1]), "n": v[2]})
			}
		} else {
			for d := from; d <= to; d = addDays(d, 1) {
				v := series[d]
				if v == nil { v = &[3]float64{} }
				r.Series = append(r.Series, map[string]any{"k": d, "rev": r0(v[0]), "profit": r0(v[1]), "n": v[2]})
				if len(r.Series) > 400 { break }
			}
		}
		tl := []map[string]any{}
		for id, a := range top {
			tl = append(tl, map[string]any{"id": id, "name": a.name, "unit": a.unit, "qty": r3(a.qty), "rev": r0(a.rev), "profit": r0(a.prof), "n": a.n})
		}
		sort.Slice(tl, func(i, j int) bool { return tl[i]["rev"].(float64) > tl[j]["rev"].(float64) })
		if len(tl) > 30 { tl = tl[:30] }
		r.Top = tl
		for g, v := range grp { r.Groups = append(r.Groups, map[string]any{"name": g, "rev": r0(v[0]), "profit": r0(v[1])}) }
		sort.Slice(r.Groups, func(i, j int) bool { return r.Groups[i]["rev"].(float64) > r.Groups[j]["rev"].(float64) })
		if len(cust) > 0 {
			spend := custSpendAll(s)
			tiers := tiersOf(s)
			for id, v := range cust {
				name, tier := inv0Name(s, id), ""
				if n := tierIndex(tiers, spend[id]); n > 0 { tier = tiers[n-1].Name }
				r.Custs = append(r.Custs, map[string]any{"id": id, "name": name, "rev": r0(v[0]), "n": v[1], "tier": tier, "spend": r0(spend[id]), "profit": r0(v[2]), "cost": r0(v[0] - v[2])})
			}
			// Mặc định xếp theo lợi nhuận (lãi gộp) cao → thấp; màn hình có thể đổi sang doanh thu.
			sort.Slice(r.Custs, func(i, j int) bool {
				a, b := r.Custs[i], r.Custs[j]
				if a["profit"].(float64) != b["profit"].(float64) { return a["profit"].(float64) > b["profit"].(float64) }
				return a["rev"].(float64) > b["rev"].(float64)
			})
			if len(r.Custs) > 300 { r.Custs = r.Custs[:300] }
		}
		for n, v := range staff { r.Staff = append(r.Staff, map[string]any{"name": n, "rev": r0(v[0]), "n": v[1]}) }
		// dòng tiền
		r.Flow["salesCash"], r.Flow["salesBank"] = r.Cash, r.Bank
		for _, rc := range s.Receipts {
			if rc.Void || !inRange(rc.Time, from, to) { continue }
			r.Flow["buyTotal"] += rc.Total
			if rc.Method == "bank" { r.Flow["buyBank"] += rc.Paid } else { r.Flow["buyCash"] += rc.Paid }
		}
		for _, e := range s.Cash {
			if e.Void || e.Kind == "adj" || !inRange(e.Time, from, to) { continue }
			k := e.Kind + "_" + e.Method
			r.Flow[k] += e.Amount
			if e.PartyType == "cust" && e.Kind == "thu" { r.Flow["debtCollected"] += e.Amount }
			if e.PartyType == "supp" && e.Kind == "chi" { r.Flow["suppPaid"] += e.Amount }
		}
		r.Flow["netCash"] = r.Flow["salesCash"] + r.Flow["thu_cash"] - r.Flow["chi_cash"] - r.Flow["buyCash"]
		r.Flow["netBank"] = r.Flow["salesBank"] + r.Flow["thu_bank"] - r.Flow["chi_bank"] - r.Flow["buyBank"]
	})
	r.Rev, r.Cost, r.Profit = r0(r.Rev), r0(r.Cost), r0(r.Profit)
	if !owner {
		r.Cost, r.Profit = 0, 0
		for _, x := range r.Top { x["profit"] = 0.0 }
		for _, x := range r.Groups { x["profit"] = 0.0 }
		for _, x := range r.Series { x["profit"] = 0.0 }
		for _, x := range r.Custs { x["profit"], x["cost"] = 0.0, 0.0 }
		sort.Slice(r.Custs, func(i, j int) bool { return r.Custs[i]["rev"].(float64) > r.Custs[j]["rev"].(float64) })
	}
	return r
}

// ---------- gợi ý combo ----------

type ComboSuggest struct {
	Kind    string      `json:"kind"` // pair | expiry | slow
	Items   []ComboItem `json:"items"`
	Names   []string    `json:"names"`
	Retail  float64     `json:"retail"`
	Price   float64     `json:"price"`
	Cost    float64     `json:"cost"`
	Margin  float64     `json:"margin"`
	Save    float64     `json:"save"`
	Count   int         `json:"count"`
	Conf    float64     `json:"conf"`
	Lift    float64     `json:"lift"`
	Reason  string      `json:"reason"`
	Name    string      `json:"name"`
	Urgency int         `json:"urgency"`
}

func roundPrice(x float64) float64 {
	step := 500.0
	if x >= 100000 { step = 1000 }
	return math.Floor(x/step) * step
}

func makeCombo(s *State, items []ComboItem, discPct, minMargin float64) (ComboSuggest, bool) {
	c := ComboSuggest{Items: items}
	for _, it := range items {
		p := s.Products[it.PID]
		if p == nil { return c, false }
		c.Names = append(c.Names, p.Name)
		c.Retail += p.Price * it.Qty
		c.Cost += p.Cost * it.Qty
	}
	if c.Retail <= 0 { return c, false }
	price := roundPrice(c.Retail * (1 - discPct/100))
	// giữ biên lãi tối thiểu nếu biết giá vốn
	if c.Cost > 0 {
		floor := c.Cost * (1 + minMargin/100)
		if price < floor { price = math.Ceil(floor/500) * 500 }
	}
	if price >= c.Retail { price = roundPrice(c.Retail - 500) }
	if price <= 0 || price >= c.Retail { return c, false }
	c.Price = price
	c.Save = c.Retail - price
	if price > 0 && c.Cost > 0 { c.Margin = math.Round((price-c.Cost)/price*1000) / 10 }
	return c, true
}

func (db *DB) SuggestCombos(days int) map[string]any {
	res := map[string]any{}
	db.Read(func(s *State) {
		since := time.Now().AddDate(0, 0, -days).UnixMilli()
		baskets := 0
		single := map[string]int{}
		qtySum := map[string]float64{}
		pair := map[[2]string]int{}
		lastSold := map[string]int64{}
		for _, inv := range s.Invoices {
			if inv.Void { continue }
			for _, l := range inv.Lines { if inv.Time > lastSold[l.PID] { lastSold[l.PID] = inv.Time } }
			if inv.Time < since { continue }
			set := map[string]bool{}
			for _, l := range inv.Lines {
				p := s.Products[l.PID]
				if p == nil || p.IsCombo { continue }
				set[l.PID] = true
				qtySum[l.PID] += l.Qty * l.Ratio
			}
			if len(set) == 0 { continue }
			baskets++
			ids := make([]string, 0, len(set))
			for id := range set { ids = append(ids, id); single[id]++ }
			sort.Strings(ids)
			if len(ids) > 25 { continue }
			for i := 0; i < len(ids); i++ { for j := i + 1; j < len(ids); j++ { pair[[2]string{ids[i], ids[j]}]++ } }
		}
		minMargin := s.Settings.MinMargin
		res["baskets"] = baskets
		res["days"] = days
		typQty := func(id string) float64 {
			if single[id] == 0 { return 1 }
			q := math.Round(qtySum[id] / float64(single[id]))
			if q < 1 { q = 1 }
			if q > 10 { q = 10 }
			return q
		}
		existing := map[string]bool{}
		for _, p := range s.Products {
			if p.IsCombo { ids := []string{}; for _, it := range p.Items { ids = append(ids, it.PID) }; sort.Strings(ids); existing[joinIDs(ids)] = true }
		}
		// 1) hay mua cùng nhau
		pairs := []ComboSuggest{}
		minCount := 3
		if baskets > 1500 { minCount = 8 }
		for k, n := range pair {
			if n < minCount || existing[joinIDs(k[:])] { continue }
			a, b := k[0], k[1]
			pa, pb := float64(single[a])/float64(baskets), float64(single[b])/float64(baskets)
			pab := float64(n) / float64(baskets)
			lift := pab / (pa * pb)
			if lift < 1.15 { continue }
			conf := math.Max(float64(n)/float64(single[a]), float64(n)/float64(single[b]))
			items := []ComboItem{{a, typQty(a)}, {b, typQty(b)}}
			c, ok := makeCombo(s, items, 4, minMargin)
			if !ok { continue }
			c.Kind, c.Count, c.Lift, c.Conf = "pair", n, math.Round(lift*100)/100, math.Round(conf*1000)/10
			lead, other := s.Products[a], s.Products[b]
			if float64(n)/float64(single[b]) > float64(n)/float64(single[a]) { lead, other = other, lead }
			c.Reason = itoa(int64(c.Conf)) + "% khách mua " + lead.Name + " cũng mua " + other.Name + " (" + itoa(int64(n)) + " đơn trong " + itoa(int64(days)) + " ngày)"
			c.Urgency = int(float64(n) * lift)
			pairs = append(pairs, c)
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].Urgency > pairs[j].Urgency })
		if len(pairs) > 12 { pairs = pairs[:12] }
		res["pairs"] = pairs
		// đối tác bán chạy nhất cho một mặt hàng
		partner := func(id string) string {
			best, bn := "", 0
			for k, n := range pair {
				o := ""
				if k[0] == id { o = k[1] } else if k[1] == id { o = k[0] } else { continue }
				if p := s.Products[o]; p == nil || p.Hidden || p.Stock <= 0 { continue }
				if n > bn { best, bn = o, n }
			}
			if best != "" { return best }
			bestN := 0
			for o, n := range single {
				if o == id { continue }
				if p := s.Products[o]; p == nil || p.Hidden || p.Stock <= 0 { continue }
				if n > bestN { best, bestN = o, n }
			}
			return best
		}
		// 2) xả hàng cận date
		td := today()
		warn := addDays(td, s.Settings.ExpWarn)
		exps := []ComboSuggest{}
		for _, p := range s.Products {
			if p.IsCombo || p.Hidden { continue }
			soon, q := "", 0.0
			for _, l := range p.Lots {
				if l.Exp >= td && l.Exp <= warn { q += l.Qty; if soon == "" || l.Exp < soon { soon = l.Exp } }
			}
			if q <= 0 { continue }
			mate := partner(p.ID)
			items := []ComboItem{{p.ID, 1}}
			if mate != "" { items = append(items, ComboItem{mate, 1}) }
			disc := 12.0
			left := daysBetween(td, soon)
			if left <= s.Settings.ExpDanger { disc = 20 }
			c, ok := makeCombo(s, items, disc, 0)
			if !ok { continue }
			c.Kind = "expiry"
			c.Urgency = 1000 - left
			c.Reason = "Còn " + fq(q) + " " + p.Unit + " hết hạn " + vnDate(soon) + " (còn " + itoa(int64(left)) + " ngày)"
			if mate != "" { c.Reason += ", ghép với " + s.Products[mate].Name + " hay bán kèm" }
			exps = append(exps, c)
		}
		sort.Slice(exps, func(i, j int) bool { return exps[i].Urgency > exps[j].Urgency })
		if len(exps) > 12 { exps = exps[:12] }
		res["expiry"] = exps
		// 3) hàng bán chậm
		slow := []ComboSuggest{}
		cut := time.Now().AddDate(0, 0, -30).UnixMilli()
		for _, p := range s.Products {
			if p.IsCombo || p.Hidden || p.Stock <= 0 || p.Created > cut { continue }
			if lastSold[p.ID] > cut { continue }
			mate := partner(p.ID)
			if mate == "" { continue }
			c, ok := makeCombo(s, []ComboItem{{p.ID, 1}, {mate, 1}}, 8, minMargin/2)
			if !ok { continue }
			c.Kind = "slow"
			c.Urgency = int(p.Stock * p.Cost / 1000)
			if lastSold[p.ID] == 0 { c.Reason = "Chưa bán được lần nào, tồn " + fq(p.Stock) + " " + p.Unit } else {
				c.Reason = "30 ngày chưa bán được, tồn " + fq(p.Stock) + " " + p.Unit + " (lần cuối " + vnDate(dayOf(lastSold[p.ID])) + ")"
			}
			slow = append(slow, c)
		}
		sort.Slice(slow, func(i, j int) bool { return slow[i].Urgency > slow[j].Urgency })
		if len(slow) > 10 { slow = slow[:10] }
		res["slow"] = slow
		// combo hiện có
		cur := []map[string]any{}
		sold := map[string]float64{}
		for _, inv := range s.Invoices {
			if inv.Void || inv.Time < since { continue }
			for _, l := range inv.Lines { sold[l.PID] += l.Qty }
		}
		for _, p := range s.Products {
			if !p.IsCombo { continue }
			cur = append(cur, map[string]any{"id": p.ID, "name": p.Name, "price": p.Price, "cost": r0(comboCost(s, p)), "sold": sold[p.ID], "stock": comboStock(s, p), "hidden": p.Hidden})
		}
		sort.Slice(cur, func(i, j int) bool { return cur[i]["sold"].(float64) > cur[j]["sold"].(float64) })
		res["current"] = cur
	})
	return res
}

func joinIDs(ids []string) string { s := ""; for _, i := range ids { s += i + "|" }; return s }

func daysBetween(a, b string) int {
	ta, e1 := time.ParseInLocation("2006-01-02", a, time.Local)
	tb, e2 := time.ParseInLocation("2006-01-02", b, time.Local)
	if e1 != nil || e2 != nil { return 0 }
	return int(math.Round(tb.Sub(ta).Hours() / 24))
}

func vnDate(d string) string {
	if len(d) != 10 { return d }
	return d[8:10] + "/" + d[5:7] + "/" + d[0:4]
}

func inv0Name(s *State, id string) string {
	if c := s.Customers[id]; c != nil { return c.Name }
	return "Khách đã xóa"
}
