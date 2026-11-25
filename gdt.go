package main

// Lấy hóa đơn mua vào từ cổng hoadondientu.gdt.gov.vn của Cục Thuế.
// Cổng không có API công khai: đây là các lời gọi mà chính trang web của cổng dùng
// (kiểm tra lại 26/10/2026: máy chủ dữ liệu ở https://hoadondientu.gdt.gov.vn/api, không còn cổng :30000).
// Người dùng tự gõ mã captcha; mật khẩu chỉ gửi thẳng tới cổng thuế, không lưu, không ghi nhật ký.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var gdtBase = strings.TrimRight(envOr("TAPHOAX_GDT_URL", "https://hoadondientu.gdt.gov.vn/api"), "/")

type GdtJob struct {
	Running  bool     `json:"running"`
	Phase    string   `json:"phase"`
	Found    int      `json:"found"`
	Todo     int      `json:"todo"`
	Done     int      `json:"done"`
	Added    int      `json:"added"`
	Dup      int      `json:"dup"`
	Fail     int      `json:"fail"`
	Errs     []string `json:"errs"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Started  int64    `json:"started"`
	Finished int64    `json:"finished"`
	Expired  bool     `json:"expired"`
}

type gdtClient struct {
	mu      sync.Mutex
	hc      *http.Client
	token   string
	user    string
	loginAt time.Time
	job     GdtJob
	paceMu  sync.Mutex
	last    time.Time
	dir     string    // thư mục dữ liệu: lưu phiên đăng nhập (đã mã hóa) để mở lại phần mềm không phải đăng nhập lại
	exp     time.Time // phiên hết hạn lúc
	shape   bool // đã ghi cấu trúc dữ liệu mẫu vào nhật ký
	shapeList bool
}

var gdt = &gdtClient{hc: &http.Client{Timeout: 40 * time.Second}}

type gdtErr struct {
	code int
	msg  string
}

func (e *gdtErr) Error() string { return e.msg }

func snippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 300 { s = s[:300] + "…" }
	return s
}

// portalMsg lấy câu báo lỗi trong phản hồi của cổng (thường ở trường message).
func portalMsg(b []byte) string {
	var m map[string]any
	if json.Unmarshal(b, &m) == nil {
		for _, k := range []string{"message", "msg", "error_description", "error", "detail"} {
			if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" { return v }
		}
	}
	return ""
}

// pace giãn cách các lần gọi: cổng thuế trả 429 (Too Many Requests) nếu gọi dồn dập.
func (g *gdtClient) pace(gap time.Duration) {
	g.paceMu.Lock()
	defer g.paceMu.Unlock()
	if w := gap - time.Since(g.last); w > 0 { time.Sleep(w) }
	g.last = time.Now()
}

func retryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if n, err := strconv.Atoi(h); err == nil && n > 0 { return time.Duration(n) * time.Second }
	if t, err := http.ParseTime(h); err == nil { if d := time.Until(t); d > 0 { return d } }
	return 0
}

func waitText(d time.Duration) string {
	if d < 90*time.Second { return fmt.Sprintf("%d giây", int(d.Seconds()+0.5)) }
	return fmt.Sprintf("%d phút", int(d.Minutes()+0.5))
}

func (g *gdtClient) do(method, p string, body any, auth bool) ([]byte, error) {
	var lastErr error
	backoff := []time.Duration{20 * time.Second, 60 * time.Second, 120 * time.Second}
	for attempt := 0; attempt < 4; attempt++ {
		var rd io.Reader
		if body != nil { b, _ := json.Marshal(body); rd = bytes.NewReader(b) }
		if auth { g.pace(1500 * time.Millisecond) } else { g.pace(500 * time.Millisecond) }
		req, _ := http.NewRequest(method, gdtBase+p, rd)
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) TapHoaX/"+AppVersion)
		if body != nil { req.Header.Set("Content-Type", "application/json") }
		req.Header.Set("Origin", "https://hoadondientu.gdt.gov.vn")
		req.Header.Set("Referer", "https://hoadondientu.gdt.gov.vn/")
		req.Header.Set("Accept-Language", "vi")
		req.Header.Set("request-id", randHex(16))
		if strings.Contains(p, "/invoices/") {
			req.Header.Set("End-Point", "/tra-cuu/tra-cuu-hoa-don")
			if strings.Contains(p, "/purchase") { req.Header.Set("Action", url.PathEscape("Tìm kiếm (hóa đơn mua vào)")) }
		}
		if auth {
			g.mu.Lock()
			tk := g.token
			g.mu.Unlock()
			req.Header.Set("Authorization", "Bearer "+tk)
		}
		resp, err := g.hc.Do(req)
		if err != nil {
			say("[cổng thuế] " + method + " " + p + " → lỗi mạng: " + err.Error())
			lastErr = &gdtErr{0, "Không kết nối được máy chủ dữ liệu của cổng hóa đơn điện tử (" + shortNetErr(err) + "). Nếu mở được trang hoadondientu.gdt.gov.vn bằng trình duyệt, hãy gửi file TapHoaX.log cho Nam Linh Group."}
			if attempt >= 1 { break }
			time.Sleep(2 * time.Second)
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			say(fmt.Sprintf("[cổng thuế] %s %s → %d (%d byte)", method, p, resp.StatusCode, len(b)))
			return b, nil
		}
		ra := retryAfter(resp.Header.Get("Retry-After"))
		extra := ""
		if ra > 0 { extra = " (Retry-After " + resp.Header.Get("Retry-After") + ")" }
		say(fmt.Sprintf("[cổng thuế] %s %s → %d%s %s", method, p, resp.StatusCode, extra, snippet(b)))
		msg := portalMsg(b)
		switch {
		case resp.StatusCode == 429:
			wait := ra
			if wait <= 0 { wait = backoff[min(attempt, len(backoff)-1)] }
			if !auth || attempt >= 3 || wait > 5*time.Minute {
				// Đăng nhập/captcha: người dùng đang chờ trước màn hình, không treo máy mà báo thời gian chờ.
				if ra <= 0 { ra = 2 * time.Minute }
				return nil, &gdtErr{429, "Cổng thuế đang tạm chặn vì truy cập nhiều lần liên tiếp (mã 429). Chờ khoảng " + waitText(ra) + " rồi thử lại. Hóa đơn đã lấy được vẫn được giữ, lần sau chỉ tải phần còn thiếu."}
			}
			g.upd(func(j *GdtJob) { if j.Running { j.Phase = "Cổng thuế yêu cầu chờ " + waitText(wait) + ", sẽ tự tiếp tục…" } })
			time.Sleep(wait)
			g.upd(func(j *GdtJob) { if j.Running { j.Phase = "Đang tiếp tục…" } })
			lastErr = &gdtErr{429, "Cổng thuế tạm chặn (mã 429)"}
			continue
		case resp.StatusCode == 401 || resp.StatusCode == 403:
			return nil, &gdtErr{401, "Phiên đăng nhập cổng thuế đã hết hạn. Hãy đăng nhập lại."}
		case resp.StatusCode >= 500:
			lastErr = &gdtErr{resp.StatusCode, "Cổng thuế đang lỗi hoặc quá tải (mã " + strconv.Itoa(resp.StatusCode) + "). Thử lại sau ít phút."}
			if attempt >= 1 { return nil, lastErr }
			time.Sleep(3 * time.Second)
			continue
		default:
			if msg == "" { msg = "Cổng thuế từ chối yêu cầu (mã " + strconv.Itoa(resp.StatusCode) + ")" }
			return nil, &gdtErr{resp.StatusCode, msg}
		}
	}
	return nil, lastErr
}

func shortNetErr(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "certificate"):
		return "lỗi chứng chỉ bảo mật, kiểm tra ngày giờ máy tính"
	case strings.Contains(s, "timeout"), strings.Contains(s, "deadline"):
		return "quá thời gian chờ"
	case strings.Contains(s, "no such host"):
		return "không có mạng hoặc không phân giải được tên miền"
	}
	return "lỗi kết nối"
}

func (g *gdtClient) Captcha() (string, string, error) {
	b, err := g.do("GET", "/captcha", nil, false)
	if err != nil { return "", "", err }
	var r struct{ Key, Content string }
	if json.Unmarshal(b, &r) != nil || r.Key == "" || r.Content == "" { return "", "", &gdtErr{0, "Cổng thuế trả về captcha không đúng dạng"} }
	return r.Key, r.Content, nil
}

func (g *gdtClient) Login(user, pass, cvalue, ckey string) error {
	b, err := g.do("POST", "/security-taxpayer/authenticate", map[string]string{"username": user, "password": pass, "cvalue": cvalue, "ckey": ckey}, false)
	if err != nil { return err }
	var r struct{ Token string }
	if json.Unmarshal(b, &r) != nil || r.Token == "" {
		if m := portalMsg(b); m != "" { return &gdtErr{400, m} }
		return &gdtErr{0, "Đăng nhập không thành công (cổng thuế không trả mã phiên)"}
	}
	g.mu.Lock()
	g.token, g.user, g.loginAt, g.exp = r.Token, user, time.Now(), jwtExp(r.Token)
	g.mu.Unlock()
	g.saveSession()
	return nil
}

// jwtExp đọc hạn của mã phiên (JWT); không đọc được thì coi như còn 2 giờ.
func jwtExp(tok string) time.Time {
	parts := strings.Split(tok, ".")
	if len(parts) == 3 {
		if b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "=")); err == nil {
			var c struct{ Exp int64 `json:"exp"` }
			if json.Unmarshal(b, &c) == nil && c.Exp > 0 { return time.Unix(c.Exp, 0) }
		}
	}
	return time.Now().Add(2 * time.Hour)
}

type gdtSess struct {
	User  string `json:"user"`
	Token string `json:"token"`
	At    int64  `json:"at"`
	Exp   int64  `json:"exp"`
}

func (g *gdtClient) sessPath() string { if g.dir == "" { return "" }; return filepath.Join(g.dir, "phien-cong-thue.bin") }

func (g *gdtClient) saveSession() {
	p := g.sessPath()
	if p == "" { return }
	g.mu.Lock()
	sv := gdtSess{g.user, g.token, g.loginAt.UnixMilli(), g.exp.Unix()}
	g.mu.Unlock()
	if sv.Token == "" { os.Remove(p); return }
	b, _ := json.Marshal(sv)
	if enc, err := protectData(b); err == nil { os.WriteFile(p, enc, 0o600) }
}

// Init nạp lại phiên đăng nhập cổng thuế còn hạn (mã hóa bằng tài khoản Windows, máy khác không đọc được).
func (g *gdtClient) Init(dir string) {
	g.dir = dir
	b, err := os.ReadFile(g.sessPath())
	if err != nil { return }
	plain, err := unprotectData(b)
	var sv gdtSess
	if err != nil || json.Unmarshal(plain, &sv) != nil || sv.Token == "" { os.Remove(g.sessPath()); return }
	if time.Until(time.Unix(sv.Exp, 0)) < 5*time.Minute { os.Remove(g.sessPath()); return }
	g.mu.Lock()
	g.token, g.user, g.loginAt, g.exp = sv.Token, sv.User, time.UnixMilli(sv.At), time.Unix(sv.Exp, 0)
	g.mu.Unlock()
	say("[cổng thuế] dùng lại phiên đăng nhập " + sv.User + ", còn hạn đến " + g.exp.Format("15:04 02/01"))
}

func (g *gdtClient) Logout() { g.mu.Lock(); g.token = ""; g.mu.Unlock(); g.saveSession() }

func (g *gdtClient) Status() map[string]any {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.token != "" && !g.exp.IsZero() && time.Now().After(g.exp) { g.token = "" }
	return map[string]any{"loggedIn": g.token != "", "user": g.user, "loginAt": g.loginAt.UnixMilli(), "exp": g.exp.UnixMilli(), "job": g.job}
}

// list lấy danh sách hóa đơn mua vào trong khoảng ngày (≤ 31 ngày), có phân trang bằng "state".
// Giống trang "Tra cứu hóa đơn mua vào": chỉ lọc theo ngày lập, lấy mọi kết quả xử lý (có mã, không mã).
func (g *gdtClient) list(prefix string, from, to time.Time) ([]map[string]any, error) {
	search := fmt.Sprintf("tdlap=ge=%sT00:00:00;tdlap=le=%sT23:59:59", from.Format("02/01/2006"), to.Format("02/01/2006"))
	var out []map[string]any
	state := ""
	for page := 0; page < 200; page++ {
		q := "/" + prefix + "/invoices/purchase?sort=tdlap:desc&size=50"
		if state != "" { q += "&state=" + url.QueryEscape(state) }
		q += "&search=" + search
		b, err := g.do("GET", q, nil, true)
		if err != nil { return out, err }
		var r struct {
			Datas []map[string]any `json:"datas"`
			Total int              `json:"total"`
			State string           `json:"state"`
		}
		if err := json.Unmarshal(b, &r); err != nil { return out, &gdtErr{0, "Danh sách hóa đơn từ cổng thuế không đúng dạng"} }
		if len(r.Datas) > 0 && !g.shapeList {
			g.shapeList = true
			ks := []string{}
			for k := range r.Datas[0] { ks = append(ks, k) }
			sort.Strings(ks)
			say(fmt.Sprintf("[cổng thuế] danh sách: total=%d, trường: %s", r.Total, strings.Join(ks, ",")))
		}
		out = append(out, r.Datas...)
		if len(r.Datas) == 0 || r.State == "" || r.State == state || (r.Total > 0 && len(out) >= r.Total) { break }
		state = r.State
	}
	return out, nil
}

func (g *gdtClient) detail(prefix string, it map[string]any) (*InInvoice, error) {
	q := fmt.Sprintf("/%s/invoices/detail?nbmst=%s&khhdon=%s&shdon=%s&khmshdon=%s", prefix,
		url.QueryEscape(js(it, "nbmst")), url.QueryEscape(js(it, "khhdon")), url.QueryEscape(js(it, "shdon")), url.QueryEscape(js(it, "khmshdon")))
	b, err := g.do("GET", q, nil, true)
	if err == nil {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			g.logShape(m)
			inv := invFromPortal(m)
			if inv.SellerMST == "" { inv = invFromPortal(it) }
			if len(inv.Lines) > 0 { return inv, inv.check() }
		}
	} else if ge, ok := err.(*gdtErr); ok && ge.code == 401 {
		return nil, err
	}
	// Dự phòng: tải file XML gốc (zip) như nút "Xuất XML" trên cổng
	q = strings.Replace(q, "/invoices/detail?", "/invoices/export-xml?", 1)
	xb, xerr := g.do("GET", q, nil, true)
	if xerr != nil {
		if err != nil { return nil, err }
		return nil, xerr
	}
	list, errs := ParseInvoiceFiles("cong-thue.zip", xb)
	if len(list) == 0 {
		if len(errs) > 0 { return nil, &gdtErr{0, errs[0]} }
		return nil, &gdtErr{0, "Không đọc được hóa đơn tải về"}
	}
	return list[0], nil
}

// logShape ghi tên các trường (không ghi giá trị) của hóa đơn đầu tiên, để sửa nhanh khi cổng đổi cấu trúc.
func (g *gdtClient) logShape(m map[string]any) {
	if g.shape { return }
	g.shape = true
	keys := make([]string, 0, len(m))
	for k := range m { keys = append(keys, k) }
	sort.Strings(keys)
	line := ""
	if arr, ok := m["hdhhdvu"].([]any); ok && len(arr) > 0 {
		if l, ok := arr[0].(map[string]any); ok {
			lk := []string{}
			for k := range l { lk = append(lk, k) }
			sort.Strings(lk)
			line = strings.Join(lk, ",")
		}
	}
	say("[cổng thuế] cấu trúc hóa đơn: " + strings.Join(keys, ",") + " | dòng hàng: " + line)
}

// ---------- đọc JSON của cổng ----------

func jv(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil { return v }
		for kk, v := range m { if strings.EqualFold(kk, k) && v != nil { return v } }
	}
	return nil
}

func js(m map[string]any, keys ...string) string {
	switch v := jv(m, keys...).(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == math.Trunc(v) { return strconv.FormatInt(int64(v), 10) }
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

func jf(m map[string]any, keys ...string) float64 {
	switch v := jv(m, keys...).(type) {
	case float64:
		return v
	case string:
		return xnum(v)
	}
	return 0
}

var vnTZ = time.FixedZone("ICT", 7*3600)

func jdate(s string) string {
	if s == "" { return "" }
	for _, l := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z0700", "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(l, s); err == nil {
			if l == "2006-01-02T15:04:05" { return t.Format("2006-01-02") }
			return t.In(vnTZ).Format("2006-01-02")
		}
	}
	if len(s) >= 10 { return s[:10] }
	return s
}

func invFromPortal(m map[string]any) *InInvoice {
	inv := &InInvoice{
		Title: js(m, "thdon"), Form: js(m, "khmshdon"), Serial: js(m, "khhdon"), No: js(m, "shdon"), Date: jdate(js(m, "tdlap", "nlap")),
		SellerMST: js(m, "nbmst"), SellerName: js(m, "nbten"), SellerAddr: js(m, "nbdchi"),
		BuyerMST: js(m, "nmmst"), BuyerName: firstNonEmpty(js(m, "nmten"), js(m, "nmtnmua")),
		Pre: jf(m, "tgtcthue"), VAT: jf(m, "tgtthue"), Total: jf(m, "tgtttbso"), Pay: js(m, "thtttoan", "htttoan"),
	}
	if arr, ok := jv(m, "hdhhdvu", "dshhdvu").([]any); ok {
		for _, x := range arr {
			l, ok := x.(map[string]any)
			if !ok { continue }
			k := int(jf(l, "tchat"))
			if k == 0 { k = 1 }
			rate := 0.0
			rtx := js(l, "ltsuat")
			if rtx != "" { rate = parseRate(rtx) } else if v := jv(l, "tsuat"); v != nil { rtx = js(l, "tsuat"); rate = parseRate(rtx) }
			inv.Lines = append(inv.Lines, InLine{Kind: k, Code: js(l, "mhhdvu"), Name: cleanSpaces(js(l, "ten", "thhdvu")), Unit: js(l, "dvtinh"),
				Qty: jf(l, "sluong"), Price: jf(l, "dgia"), Amount: jf(l, "thtien"), Rate: rate, RateTx: rtx})
		}
	}
	return inv
}

// ---------- chạy nền ----------

func (g *gdtClient) Start(db *DB, from, to time.Time) error {
	g.mu.Lock()
	if g.token == "" { g.mu.Unlock(); return uerr("Chưa đăng nhập cổng thuế") }
	if g.job.Running { g.mu.Unlock(); return uerr("Đang lấy hóa đơn, chờ xong lượt này") }
	g.job = GdtJob{Running: true, Phase: "Đang lấy danh sách hóa đơn…", From: from.Format("2006-01-02"), To: to.Format("2006-01-02"), Started: nowMs()}
	g.mu.Unlock()
	go g.run(db, from, to)
	return nil
}

func (g *gdtClient) upd(f func(j *GdtJob)) { g.mu.Lock(); f(&g.job); g.mu.Unlock() }

func (g *gdtClient) run(db *DB, from, to time.Time) {
	defer g.upd(func(j *GdtJob) { j.Running = false; j.Finished = nowMs(); if j.Phase == "" || strings.HasPrefix(j.Phase, "Đang") { j.Phase = "Xong" } })
	say(fmt.Sprintf("[cổng thuế] bắt đầu lấy hóa đơn mua vào %s → %s (máy chủ %s)", from.Format("02/01/2006"), to.Format("02/01/2006"), gdtBase))
	have := map[string]bool{}
	db.Read(func(s *State) { for _, x := range s.InInvs { have[x.Key] = true } })
	type item struct {
		prefix string
		m      map[string]any
	}
	var items []item
	seen := map[string]bool{}
	fail := func(err error) bool {
		var ge *gdtErr
		if errors.As(err, &ge) && ge.code == 401 {
			g.Logout()
			g.upd(func(j *GdtJob) { j.Expired = true; j.Phase = "Phiên đăng nhập hết hạn"; j.Errs = append(j.Errs, ge.msg) })
			return true
		}
		if errors.As(err, &ge) && ge.code == 429 {
			g.upd(func(j *GdtJob) { j.Phase = "Tạm dừng: cổng thuế giới hạn số lần truy cập"; j.Errs = append(j.Errs, ge.msg) })
			return true
		}
		g.upd(func(j *GdtJob) { if len(j.Errs) < 20 { j.Errs = append(j.Errs, err.Error()) } })
		return false
	}
	// Cổng giới hạn mỗi lần tìm tối đa khoảng một tháng: chia thành đợt 30 ngày
	for a := from; !a.After(to); a = a.AddDate(0, 0, 30) {
		b := a.AddDate(0, 0, 29)
		if b.After(to) { b = to }
		g.upd(func(j *GdtJob) { j.Phase = fmt.Sprintf("Đang lấy danh sách %s – %s…", a.Format("02/01"), b.Format("02/01/2006")) })
		// Hóa đơn máy tính tiền mua vào (sco-query) cổng bắt gõ thêm captcha cho mỗi lần tìm: chưa lấy ở bản này.
		for _, q := range [][1]string{{"query"}} {
			list, err := g.list(q[0], a, b)
			if err != nil {
				if fail(err) { return }
				continue
			}
			for _, m := range list {
				k := invKey(js(m, "nbmst"), js(m, "khmshdon"), js(m, "khhdon"), js(m, "shdon"))
				if seen[k] { continue }
				seen[k] = true
				if have[k] { g.upd(func(j *GdtJob) { j.Found++; j.Dup++ }); continue }
				items = append(items, item{q[0], m})
				g.upd(func(j *GdtJob) { j.Found++ })
			}
		}
	}
	g.upd(func(j *GdtJob) { j.Phase = "Đang tải chi tiết hóa đơn…"; j.Todo = len(items) })
	var batch []*InInvoice
	flush := func() {
		if len(batch) == 0 { return }
		r := db.AddInInvoices(batch, "portal")
		g.upd(func(j *GdtJob) { j.Added += len(r.Added); j.Dup += r.Dup })
		batch = nil
	}
	for _, it := range items {
		inv, err := g.detail(it.prefix, it.m)
		g.upd(func(j *GdtJob) { j.Done++ })
		if err != nil {
			g.upd(func(j *GdtJob) { j.Fail++ })
			if fail(fmt.Errorf("Hóa đơn %s số %s của %s: %w", js(it.m, "khhdon"), js(it.m, "shdon"), js(it.m, "nbten"), err)) { flush(); return }
			continue
		}
		batch = append(batch, inv)
		if len(batch) >= 10 { flush() }
	}
	flush()
	g.upd(func(j *GdtJob) {
		j.Phase = "Xong"
		say(fmt.Sprintf("[cổng thuế] xong: tìm thấy %d, thêm %d, đã có %d, lỗi %d", j.Found, j.Added, j.Dup, j.Fail))
	})
}
