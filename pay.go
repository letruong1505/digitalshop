package main

// Tự xác nhận tiền chuyển khoản và màn hình hiển thị cho khách.
//
// Xác nhận chuyển khoản có 2 cách, chủ tiệm chọn một trong Cài đặt:
//   - SePay: tiệm liên kết tài khoản ngân hàng với SePay. Mỗi đơn có một mã nội dung (vd TXK3P9QA)
//     nằm trong mã VietQR; phần mềm đọc lịch sử giao dịch qua API SePay để tìm khoản tiền có mã đó.
//   - payOS: phần mềm tạo yêu cầu thanh toán trên payOS (mã QR do payOS cấp) và hỏi trạng thái đơn.
// Cả hai đều chỉ cần máy tiệm có internet đi ra, không cần mở cổng hay địa chỉ công khai.

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type PaySession struct {
	Code      string    `json:"code"`
	Amount    float64   `json:"amount"`
	Provider  string    `json:"provider"`
	OrderCode int64     `json:"orderCode,omitempty"`
	QR        string    `json:"qr,omitempty"`
	Status    string    `json:"status"` // pending | paid | expired | error
	Msg       string    `json:"msg,omitempty"`
	Created   time.Time `json:"-"`
	PaidAt    time.Time `json:"-"`
	TxRef     string    `json:"txRef,omitempty"`
	lastCheck time.Time
}

type payConf struct {
	Provider, SepayToken, Acct, PayosID, PayosKey, PayosSum string
}

// Có thể đổi địa chỉ khi thử nghiệm (máy chủ giả).
var (
	sepayBase = envOr("TAPHOAX_SEPAY_URL", "https://my.sepay.vn")
	payosBase = envOr("TAPHOAX_PAYOS_URL", "https://api-merchant.payos.vn")
)

func envOr(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

var (
	payMu     sync.Mutex
	paySess   = map[string]*PaySession{}
	usedTx    = map[string]bool{}
	sepayLast time.Time
	payHTTP   = &http.Client{Timeout: 15 * time.Second}
)

func (db *DB) payConf() payConf {
	var c payConf
	db.Read(func(s *State) {
		st := s.Settings
		c = payConf{st.PayProvider, st.SepayToken, st.Acct, st.PayosClientID, st.PayosAPIKey, st.PayosChecksum}
	})
	return c
}

func payCode() string {
	const al = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(al))))
		b[i] = al[n.Int64()]
	}
	return "TX" + string(b)
}

func (db *DB) PayStart(amount float64) (*PaySession, error) {
	c := db.payConf()
	if c.Provider == "" { return nil, uerr("Chưa bật tự xác nhận chuyển khoản trong Cài đặt") }
	amount = r0(amount)
	if amount < 1000 { return nil, uerr("Số tiền chuyển khoản tối thiểu 1.000đ") }
	ps := &PaySession{Code: payCode(), Amount: amount, Provider: c.Provider, Status: "pending", Created: time.Now()}
	switch c.Provider {
	case "sepay":
		if c.SepayToken == "" || c.Acct == "" { return nil, uerr("Thiếu API token SePay hoặc số tài khoản trong Cài đặt") }
	case "payos":
		if c.PayosID == "" || c.PayosKey == "" || c.PayosSum == "" { return nil, uerr("Thiếu Client ID / API Key / Checksum Key của payOS trong Cài đặt") }
		ps.OrderCode = time.Now().UnixMilli()%1e12*10 + int64(time.Now().Nanosecond()%10)
		qr, err := payosCreate(c, ps)
		if err != nil { return nil, err }
		ps.QR = qr
	default:
		return nil, uerr("Cách xác nhận chuyển khoản không hợp lệ")
	}
	payMu.Lock()
	for k, v := range paySess { if time.Since(v.Created) > 2*time.Hour { delete(paySess, k) } }
	paySess[ps.Code] = ps
	payMu.Unlock()
	return ps, nil
}

// PayStatus trả trạng thái đơn; tự hỏi nhà cung cấp nếu đã quá 3 giây từ lần hỏi trước.
func (db *DB) PayStatus(code string) (*PaySession, error) {
	payMu.Lock()
	ps := paySess[code]
	payMu.Unlock()
	if ps == nil { return nil, uerr("Không tìm thấy phiên thanh toán") }
	if ps.Status != "pending" { cp := *ps; return &cp, nil }
	if time.Since(ps.Created) > 30*time.Minute { ps.Status = "expired"; cp := *ps; return &cp, nil }
	c := db.payConf()
	switch ps.Provider {
	case "sepay":
		sepayPoll(c)
	case "payos":
		payMu.Lock()
		due := time.Since(ps.lastCheck) > 3*time.Second
		if due { ps.lastCheck = time.Now() }
		payMu.Unlock()
		if due { payosCheck(c, ps) }
	}
	payMu.Lock()
	cp := *ps
	payMu.Unlock()
	return &cp, nil
}

// ---------- SePay ----------

func compact(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) { if unicode.IsLetter(r) || unicode.IsDigit(r) { b.WriteRune(r) } }
	return b.String()
}

// sepayPoll đọc giao dịch mới (tối đa 1 lần / 3 giây cho mọi phiên) và đánh dấu các phiên đã nhận tiền.
func sepayPoll(c payConf) {
	payMu.Lock()
	if time.Since(sepayLast) < 3*time.Second { payMu.Unlock(); return }
	sepayLast = time.Now()
	var oldest time.Time
	pending := []*PaySession{}
	for _, p := range paySess {
		if p.Provider == "sepay" && p.Status == "pending" {
			pending = append(pending, p)
			if oldest.IsZero() || p.Created.Before(oldest) { oldest = p.Created }
		}
	}
	payMu.Unlock()
	if len(pending) == 0 { return }
	q := url.Values{}
	q.Set("account_number", c.Acct)
	q.Set("transaction_date_min", oldest.Add(-3*time.Minute).Format("2006-01-02 15:04:05"))
	q.Set("limit", "100")
	req, _ := http.NewRequest("GET", sepayBase+"/userapi/transactions/list?"+q.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+c.SepayToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := payHTTP.Do(req)
	if err != nil { setMsg(pending, "Mất kết nối internet tới SePay, đang thử lại…"); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 || resp.StatusCode == 403 { setMsg(pending, "API token SePay không đúng, kiểm tra Cài đặt"); return }
	if resp.StatusCode == 429 { return }
	if resp.StatusCode != 200 { setMsg(pending, fmt.Sprintf("SePay báo lỗi %d, đang thử lại…", resp.StatusCode)); return }
	var out struct {
		Transactions []struct {
			ID      string `json:"id"`
			Content string `json:"transaction_content"`
			In      string `json:"amount_in"`
			Ref     string `json:"reference_number"`
			Code    any    `json:"code"`
		} `json:"transactions"`
	}
	if json.Unmarshal(body, &out) != nil { return }
	payMu.Lock()
	defer payMu.Unlock()
	for _, p := range pending { p.Msg = "" }
	for _, t := range out.Transactions {
		amt, _ := strconv.ParseFloat(strings.TrimSpace(t.In), 64)
		if amt <= 0 || usedTx[t.ID] { continue }
		content := compact(t.Content + " " + fmt.Sprint(t.Code))
		for _, p := range pending {
			if p.Status != "pending" || !strings.Contains(content, p.Code) { continue }
			if amt+0.5 < p.Amount {
				p.Msg = fmt.Sprintf("Khách mới chuyển %s, còn thiếu %s", money(amt), money(p.Amount-amt))
				continue
			}
			p.Status, p.PaidAt, p.TxRef = "paid", time.Now(), t.Ref
			usedTx[t.ID] = true
			break
		}
	}
}

func setMsg(ps []*PaySession, m string) { payMu.Lock(); for _, p := range ps { p.Msg = m }; payMu.Unlock() }

func sepayTest(token, acct string) error {
	req, _ := http.NewRequest("GET", sepayBase+"/userapi/transactions/list?limit=1&account_number="+url.QueryEscape(acct), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := payHTTP.Do(req)
	if err != nil { return uerr("Không kết nối được tới SePay. Kiểm tra internet của máy tính tiền") }
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 { return uerr("API token SePay không đúng") }
	if resp.StatusCode != 200 { return uerr("SePay báo lỗi %d", resp.StatusCode) }
	return nil
}

// ---------- payOS ----------

func payosDo(c payConf, method, path string, body any) (json.RawMessage, error) {
	var rd io.Reader
	if body != nil { b, _ := json.Marshal(body); rd = bytes.NewReader(b) }
	req, _ := http.NewRequest(method, payosBase+path, rd)
	req.Header.Set("x-client-id", c.PayosID)
	req.Header.Set("x-api-key", c.PayosKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := payHTTP.Do(req)
	if err != nil { return nil, uerr("Không kết nối được tới payOS. Kiểm tra internet của máy tính tiền") }
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code string          `json:"code"`
		Desc string          `json:"desc"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &out) != nil { return nil, uerr("payOS trả kết quả lạ (HTTP %d)", resp.StatusCode) }
	if out.Code != "00" { return nil, uerr("payOS: %s (mã %s)", out.Desc, out.Code) }
	return out.Data, nil
}

func payosCreate(c payConf, ps *PaySession) (string, error) {
	const back = "http://localhost/"
	amount := int64(ps.Amount)
	sigStr := fmt.Sprintf("amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s", amount, back, ps.Code, ps.OrderCode, back)
	m := hmac.New(sha256.New, []byte(c.PayosSum))
	m.Write([]byte(sigStr))
	body := map[string]any{"orderCode": ps.OrderCode, "amount": amount, "description": ps.Code, "cancelUrl": back, "returnUrl": back,
		"signature": hex.EncodeToString(m.Sum(nil)), "expiredAt": time.Now().Add(30 * time.Minute).Unix()}
	data, err := payosDo(c, "POST", "/v2/payment-requests", body)
	if err != nil { return "", err }
	var d struct{ QrCode string `json:"qrCode"` }
	json.Unmarshal(data, &d)
	if d.QrCode == "" { return "", uerr("payOS không trả mã QR") }
	return d.QrCode, nil
}

func payosCheck(c payConf, ps *PaySession) {
	data, err := payosDo(c, "GET", "/v2/payment-requests/"+strconv.FormatInt(ps.OrderCode, 10), nil)
	payMu.Lock()
	defer payMu.Unlock()
	if err != nil { ps.Msg = err.Error(); return }
	var d struct {
		Status     string  `json:"status"`
		AmountPaid float64 `json:"amountPaid"`
	}
	json.Unmarshal(data, &d)
	ps.Msg = ""
	switch d.Status {
	case "PAID":
		ps.Status, ps.PaidAt = "paid", time.Now()
	case "UNDERPAID":
		ps.Msg = fmt.Sprintf("Khách mới chuyển %s, còn thiếu %s", money(d.AmountPaid), money(ps.Amount-d.AmountPaid))
	case "CANCELLED", "EXPIRED", "FAILED":
		ps.Status = "expired"
	}
}

func payosTest(c payConf) error {
	_, err := payosDo(c, "GET", "/v2/payment-requests/1", nil)
	if err == nil { return nil }
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "kết nối") { return err }
	if strings.Contains(low, "client") || strings.Contains(low, "api key") || strings.Contains(low, "apikey") || strings.Contains(low, "unauthor") || strings.Contains(low, "xác thực") {
		return uerr("Client ID hoặc API Key payOS không đúng (%s)", msg)
	}
	return nil // đơn số 1 không tồn tại là bình thường: khóa đã được chấp nhận
}

// ---------- màn hình khách ----------

type displayHub struct {
	mu   sync.Mutex
	seq  int64
	data json.RawMessage
	wait chan struct{}
}

var dispHub = &displayHub{wait: make(chan struct{}), data: json.RawMessage(`{"status":"idle"}`)}

func (h *displayHub) set(b []byte) {
	h.mu.Lock()
	h.seq++
	h.data = append(json.RawMessage(nil), b...)
	close(h.wait)
	h.wait = make(chan struct{})
	h.mu.Unlock()
}

func (h *displayHub) get(since int64, timeout time.Duration) (int64, json.RawMessage) {
	h.mu.Lock()
	if h.seq != since { s, d := h.seq, h.data; h.mu.Unlock(); return s, d }
	w := h.wait
	h.mu.Unlock()
	select {
	case <-w:
	case <-time.After(timeout):
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq, h.data
}
