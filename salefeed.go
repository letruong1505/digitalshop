package main

import "sync"

// Thông báo bán hàng: máy chủ nhớ vài chục hóa đơn gần nhất để màn hình của chủ/quản lý hiện
// "Nhân viên … đã bán …" khi hóa đơn được tạo từ máy khác (điện thoại, máy phụ, hóa đơn ngoại tuyến đẩy lên).

type SaleEvt struct {
	Seq   int64   `json:"seq"`
	UID   string  `json:"uid"`
	Staff string  `json:"staff"`
	Code  string  `json:"code"`
	Total float64 `json:"total"`
	Time  int64   `json:"time"`
}

var saleFeed struct {
	mu   sync.Mutex
	seq  int64
	evts []SaleEvt
}

func pushSale(u *User, inv *Invoice) {
	saleFeed.mu.Lock()
	defer saleFeed.mu.Unlock()
	for _, e := range saleFeed.evts { if e.Code == inv.Code { return } } // gửi lại do mạng chập chờn: không báo hai lần
	saleFeed.seq++
	saleFeed.evts = append(saleFeed.evts, SaleEvt{Seq: saleFeed.seq, UID: u.ID, Staff: u.Name, Code: inv.Code, Total: inv.Total, Time: nowMs()})
	if len(saleFeed.evts) > 50 { saleFeed.evts = saleFeed.evts[len(saleFeed.evts)-50:] }
}

func saleFeedAfter(after int64) (int64, []SaleEvt) {
	saleFeed.mu.Lock()
	defer saleFeed.mu.Unlock()
	var out []SaleEvt
	for _, e := range saleFeed.evts { if e.Seq > after { out = append(out, e) } }
	return saleFeed.seq, out
}
