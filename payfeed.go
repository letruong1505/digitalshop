package main

// Báo tiền về: khi tiệm đã nối SePay (API token) và bật "Đọc to khi tiền về", phần mềm tự hỏi SePay vài giây một lần,
// thấy khoản tiền vào mới thì đưa vào danh sách; màn hình bán hàng (máy tính hoặc điện thoại) đọc to "Đã nhận … đồng".
// Chỉ đọc lịch sử giao dịch của chính tiệm, không gửi gì đi. Lần đầu bật chỉ ghi nhớ giao dịch cũ, không đọc lại.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PayEvt struct {
	Seq     int     `json:"seq"`
	Amount  float64 `json:"amount"`
	Content string  `json:"content"`
	At      int64   `json:"at"`
}

var (
	feedMu   sync.Mutex
	feedEvts []PayEvt
	feedSeq  int
	feedSeen = map[string]bool{}
	feedInit bool
)

// feedIngest nhận danh sách giao dịch từ SePay; lần đầu chỉ nhớ, các lần sau thêm khoản tiền vào mới.
func feedIngest(txs []sepayTx) {
	feedMu.Lock()
	defer feedMu.Unlock()
	if len(feedSeen) > 3000 { feedSeen = map[string]bool{}; feedInit = false }
	// SePay trả mới nhất trước; đọc ngược để phát theo thứ tự thời gian.
	for i := len(txs) - 1; i >= 0; i-- {
		t := txs[i]
		if t.ID == "" || feedSeen[t.ID] { continue }
		feedSeen[t.ID] = true
		amt, _ := strconv.ParseFloat(strings.TrimSpace(t.In), 64)
		if !feedInit || amt <= 0 { continue }
		feedSeq++
		feedEvts = append(feedEvts, PayEvt{Seq: feedSeq, Amount: amt, Content: strings.TrimSpace(t.Content), At: time.Now().UnixMilli()})
		if len(feedEvts) > 50 { feedEvts = feedEvts[len(feedEvts)-50:] }
	}
	feedInit = true
}

func feedReset() {
	feedMu.Lock()
	feedInit = false
	feedSeen = map[string]bool{}
	feedMu.Unlock()
}

type sepayTx struct {
	ID      string `json:"id"`
	Content string `json:"transaction_content"`
	In      string `json:"amount_in"`
}

func sepayList(c payConf, minTime time.Time) ([]sepayTx, int, error) {
	q := url.Values{}
	q.Set("account_number", c.Acct)
	q.Set("transaction_date_min", minTime.Format("2006-01-02 15:04:05"))
	q.Set("limit", "30")
	req, _ := http.NewRequest("GET", sepayBase+"/userapi/transactions/list?"+q.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+c.SepayToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := payHTTP.Do(req)
	if err != nil { return nil, 0, err }
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 { return nil, resp.StatusCode, nil }
	var out struct{ Transactions []sepayTx `json:"transactions"` }
	if json.Unmarshal(body, &out) != nil { return nil, 200, nil }
	return out.Transactions, 200, nil
}

// payFeedLoop chạy nền suốt thời gian phần mềm mở.
func (s *Server) payFeedLoop() {
	for range time.Tick(feedEvery) {
		var c payConf
		var on bool
		s.db.Read(func(st *State) {
			c = payConf{st.Settings.PayProvider, st.Settings.SepayToken, st.Settings.Acct, "", "", ""}
			on = st.Settings.PayVoice
		})
		if !on || c.Provider != "sepay" || c.SepayToken == "" || c.Acct == "" { feedReset(); continue }
		txs, _, err := sepayList(c, time.Now().Add(-24*time.Hour))
		if err != nil || txs == nil { continue }
		feedIngest(txs)
	}
}

var feedEvery = 5 * time.Second

func (s *Server) payFeedRoutes(m *http.ServeMux) {
	// after không có: chỉ trả mốc hiện tại để máy khách không đọc lại tiền cũ.
	m.HandleFunc("GET /api/pay/feed", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		after, has := 0, r.URL.Query().Get("after") != ""
		after, _ = strconv.Atoi(r.URL.Query().Get("after"))
		feedMu.Lock()
		items := []PayEvt{}
		if has { for _, e := range feedEvts { if e.Seq > after { items = append(items, e) } } }
		seq := feedSeq
		feedMu.Unlock()
		writeJSON(w, map[string]any{"seq": seq, "items": items})
	}))
}
