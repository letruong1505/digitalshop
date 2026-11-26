package main

import "strings"

// Loyalty: tích điểm khách hàng. Mua hàng được điểm, đổi điểm thành giảm giá ở lần mua sau.
type Loyalty struct {
	On         bool    `json:"on"`
	EarnPer    float64 `json:"earnPer"`    // mỗi bao nhiêu đồng thanh toán được 1 điểm
	PointValue float64 `json:"pointValue"` // 1 điểm đổi được bao nhiêu đồng giảm giá
	MinRedeem  int     `json:"minRedeem"`  // số điểm tối thiểu mỗi lần đổi
	MaxPct     float64 `json:"maxPct"`     // tối đa bao nhiêu % giá trị hóa đơn được trả bằng điểm
}

// PtsEntry: lịch sử điều chỉnh điểm thủ công (cộng/trừ tay).
type PtsEntry struct {
	ID       string `json:"id"`
	Time     int64  `json:"time"`
	CustID   string `json:"custId"`
	CustName string `json:"custName"`
	Delta    int    `json:"delta"`
	Bal      int    `json:"bal"`
	Note     string `json:"note"`
	Staff    string `json:"staff"`
}

func (l Loyalty) earn(amount float64) int {
	if !l.On || l.EarnPer <= 0 || amount <= 0 { return 0 }
	return int(amount / l.EarnPer)
}

// maxRedeem: số điểm tối đa dùng được cho hóa đơn có giá trị payable (sau khuyến mãi và giảm giá tay).
func (l Loyalty) maxRedeem(balance int, payable float64) int {
	if !l.On || l.PointValue <= 0 || balance <= 0 || payable <= 0 { return 0 }
	capAmt := payable
	if l.MaxPct > 0 && l.MaxPct < 100 { capAmt = payable * l.MaxPct / 100 }
	n := int(capAmt / l.PointValue)
	if n > balance { n = balance }
	if l.MinRedeem > 0 && n < l.MinRedeem { return 0 }
	return n
}

func cleanLoyalty(l Loyalty) (Loyalty, error) {
	if !l.On { l.On = false }
	if l.EarnPer <= 0 { l.EarnPer = 10000 }
	if l.PointValue <= 0 { l.PointValue = 100 }
	if l.MinRedeem < 0 { l.MinRedeem = 0 }
	if l.MaxPct <= 0 || l.MaxPct > 100 { l.MaxPct = 100 }
	if l.PointValue*1 > l.EarnPer { return l, uerr("1 điểm đổi được nhiều hơn số tiền cần mua để có 1 điểm, như vậy cửa hàng sẽ lỗ. Hãy giảm giá trị đổi hoặc tăng số tiền mỗi điểm") }
	return l, nil
}

func (db *DB) SaveLoyalty(l Loyalty) error {
	return db.Do(func(s *State, t *Tx) error {
		c, err := cleanLoyalty(l)
		if err != nil { return err }
		s.Settings.Loyalty = c
		t.put("settings", "", s.Settings)
		return nil
	})
}

// AdjustPoints cộng/trừ điểm thủ công, có ghi lịch sử.
func (db *DB) AdjustPoints(custID string, delta int, note string, u *User) (*PtsEntry, error) {
	var out *PtsEntry
	err := db.Do(func(s *State, t *Tx) error {
		c := s.Customers[custID]
		if c == nil { return uerr("Không tìm thấy khách hàng") }
		if delta == 0 { return uerr("Nhập số điểm cần cộng hoặc trừ") }
		if c.Points+delta < 0 { return uerr("Khách chỉ có %d điểm, không trừ được %d", c.Points, -delta) }
		c.Points += delta
		e := &PtsEntry{ID: newID(), Time: nowMs(), CustID: c.ID, CustName: c.Name, Delta: delta, Bal: c.Points, Note: strings.TrimSpace(note), Staff: u.Name}
		s.PtsLog[e.ID] = e
		t.put("ptslog", e.ID, e)
		t.put("customer", c.ID, c)
		out = e
		return nil
	})
	return out, err
}
