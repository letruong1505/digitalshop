package main

// Đối chiếu kiểm kho: với mỗi mặt hàng, tồn dự kiến = số sau lần kiểm/xuất hủy gần nhất + nhập kho − bán ra
// + khách trả (nhập lại kho) − trả nhà cung cấp, tính từ sau thời điểm lần kiểm đó. Hàng chưa từng kiểm thì chưa đối chiếu được.

type reconRow struct {
	prevTime                  int64
	prev, in, out, expect     float64
}

func countRecon(s *State, want map[string]bool) map[string]*reconRow {
	out := map[string]*reconRow{}
	for _, c := range s.Counts {
		for _, l := range c.Lines {
			if !want[l.PID] { continue }
			r := out[l.PID]
			if r == nil || c.Time > r.prevTime { out[l.PID] = &reconRow{prevTime: c.Time, prev: l.After} }
		}
	}
	add := func(l Line, t int64, sign float64, in bool) {
		p := s.Products[l.PID]
		if p == nil { return }
		base := l.Qty * ratioOf(l)
		bump := func(pid string, q float64) {
			r := out[pid]
			if r == nil || t <= r.prevTime { return }
			if in { r.in += q } else { r.out += q }
		}
		if p.IsCombo { for _, it := range p.Items { bump(it.PID, it.Qty*base) }; return }
		bump(l.PID, base)
	}
	for _, rc := range s.Receipts { if !rc.Void { for _, l := range rc.Lines { add(l, rc.Time, 1, true) } } }
	for _, r := range s.Returns { if !r.Void && r.Restock { for _, l := range r.Lines { add(l, r.Time, 1, true) } } }
	for _, iv := range s.Invoices { if !iv.Void { for _, l := range iv.Lines { add(l, iv.Time, -1, false) } } }
	for _, r := range s.SuppReturns { if !r.Void { for _, l := range r.Lines { add(l, r.Time, -1, false) } } }
	for _, r := range out { r.in, r.out = r3(r.in), r3(r.out); r.expect = r3(r.prev + r.in - r.out) }
	return out
}
