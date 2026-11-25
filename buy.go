package main

// Mua bản quyền trực tuyến: mở trang mua của Nam Linh Group (thanh toán QR/SePay), sau đó
// phần mềm tự lấy mã đã mua về và kích hoạt. Chữ ký mã vẫn được kiểm tra tại máy (parseLicense).

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	buyProduct     = "shop"
	buyBaseDefault = "https://namlinh-banquyen.namlinh.workers.dev"
)

func buyBase() string {
	if u := strings.TrimRight(os.Getenv("DS_BUY_URL"), "/"); u != "" { return u }
	return buyBaseDefault
}

func buyURL(machine string) string {
	return buyBase() + "/mua?sp=" + buyProduct + "&may=" + url.QueryEscape(machine)
}

// licClaim hỏi máy chủ bản quyền xem máy này đã mua chưa; nếu có thì kích hoạt luôn.
func licClaim() (LicStatus, bool, error) {
	lic.mu.Lock()
	machine, cur := lic.machine, lic.st.Key
	lic.mu.Unlock()
	cl := &http.Client{Timeout: 15 * time.Second}
	resp, err := cl.Get(buyBase() + "/api/claim?sp=" + buyProduct + "&m=" + url.QueryEscape(machine))
	if err != nil { return LicStatus{}, false, uerr("Không kết nối được máy chủ bản quyền. Kiểm tra internet rồi thử lại.") }
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var r struct {
		OK  bool   `json:"ok"`
		Key string `json:"key"`
	}
	if resp.StatusCode != 200 || json.Unmarshal(b, &r) != nil { return LicStatus{}, false, uerr("Máy chủ bản quyền chưa trả lời đúng, thử lại sau ít phút.") }
	if !r.OK || r.Key == "" { return licStatus(), false, nil }
	if strings.Join(strings.Fields(r.Key), "") == cur { return licStatus(), false, nil }
	st, err := licActivate(r.Key)
	if err != nil { return LicStatus{}, false, err }
	return st, true, nil
}

// licInfoFn do main đặt: trả về tên, điện thoại, địa chỉ cửa hàng đang khai trong Cài đặt.
var licInfoFn func() (shop, phone, addr string)

// licRevokeCheck báo thông tin liên hệ của cửa hàng cho Nam Linh Group (để quản lý bản quyền, hỗ trợ) và hỏi
// máy chủ xem bản quyền đang dùng có bị thu hồi không. Chỉ khi máy chủ trả lời rõ ràng mới đổi trạng thái;
// mất mạng hoặc máy chủ lỗi thì giữ nguyên (tiệm mất internet vẫn bán bình thường).
// Chỉ gửi: số bản quyền, mã máy, phiên bản, tên/điện thoại/địa chỉ cửa hàng. Không gửi hàng hóa, hóa đơn, khách, công nợ.
func licRevokeCheck() {
	lic.mu.Lock()
	key, machine := lic.st.Key, lic.machine
	lic.mu.Unlock()
	if key == "" { return }
	l, err := parseLicense(key)
	if err != nil || l.ID == "" { return }
	info := map[string]string{"id": l.ID, "machine": machine, "ver": AppVersion}
	if licInfoFn != nil { info["shop"], info["phone"], info["addr"] = licInfoFn() }
	body, _ := json.Marshal(info)
	cl := &http.Client{Timeout: 15 * time.Second}
	resp, err := cl.Post(buyBase()+"/api/check", "application/json", bytes.NewReader(body))
	if err != nil { return }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return }
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<12))
	var r struct {
		OK        bool  `json:"ok"`
		Revoked   bool  `json:"revoked"`
		Effective int64 `json:"effective"` // mili giây; trống thì tính 24 giờ kể từ lúc nhận tin
	}
	if json.Unmarshal(b, &r) != nil || !r.OK { return }
	lic.mu.Lock()
	defer lic.mu.Unlock()
	if lic.st.Key != key { return }
	switch {
	case r.Revoked && lic.st.Revoked != l.ID:
		eff := r.Effective / 1000
		if eff <= 0 { eff = time.Now().Add(24 * time.Hour).Unix() }
		lic.st.Revoked, lic.st.RevokedEff = l.ID, eff
		licSave()
		say("Bản quyền " + l.ID + " đã bị thu hồi, chuyển sang chỉ xem lúc " + time.Unix(eff, 0).Format("02/01/2006 15:04"))
	case !r.Revoked && lic.st.Revoked == l.ID:
		lic.st.Revoked, lic.st.RevokedEff = "", 0
		licSave()
		say("Bản quyền " + l.ID + " đã được khôi phục")
	}
	lic.cur = licCompute()
}

func licRevokeLoop() {
	time.Sleep(30 * time.Second)
	for {
		licRevokeCheck()
		time.Sleep(2 * time.Hour)
	}
}

// buyProxy chuyển yêu cầu mua bản quyền tới máy chủ bản quyền (để hộp mua trong phần mềm không bị chặn CORS
// và địa chỉ máy chủ do phần mềm quyết định, không nhận từ trình duyệt).
func buyProxy(w http.ResponseWriter, method, path, rawQuery string, body []byte) {
	u := buyBase() + path
	if rawQuery != "" { u += "?" + rawQuery }
	req, err := http.NewRequest(method, u, bytes.NewReader(body))
	if err != nil { fail(w, 500, "Lỗi nội bộ"); return }
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil { fail(w, 502, "Không kết nối được máy chủ bản quyền. Kiểm tra internet rồi thử lại."); return }
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<18))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	w.Write(b)
}

func (s *Server) buyRoutes(m *http.ServeMux) {
	// Thông tin để dựng hộp mua: bảng giá, mã máy, thông tin cửa hàng đã khai.
	m.HandleFunc("GET /api/license/buy/info", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		lic.mu.Lock(); machine := lic.machine; lic.mu.Unlock()
		var shop, phone string
		if licInfoFn != nil { shop, phone, _ = licInfoFn() }
		cl := &http.Client{Timeout: 15 * time.Second}
		resp, err := cl.Get(buyBase() + "/api/config")
		if err != nil { fail(w, 502, "Không kết nối được máy chủ bản quyền. Kiểm tra internet rồi thử lại."); return }
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		var cfg map[string]any
		if resp.StatusCode != 200 || json.Unmarshal(b, &cfg) != nil { fail(w, 502, "Máy chủ bản quyền chưa trả lời đúng, thử lại sau ít phút."); return }
		writeJSON(w, map[string]any{"config": cfg, "machine": machine, "shop": shop, "phone": phone, "product": buyProduct})
	}))
	m.HandleFunc("GET /api/license/buy/partner", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		q := url.Values{"sp": {buyProduct}, "plan": {r.URL.Query().Get("plan")}, "code": {r.URL.Query().Get("code")}}
		buyProxy(w, "GET", "/api/partner", q.Encode(), nil)
	}))
	m.HandleFunc("POST /api/license/buy/order", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		var in map[string]any
		if json.NewDecoder(io.LimitReader(r.Body, 1<<14)).Decode(&in) != nil { fail(w, 400, "Dữ liệu không hợp lệ"); return }
		lic.mu.Lock(); machine := lic.machine; lic.mu.Unlock()
		in["sp"], in["machine"] = buyProduct, machine
		b, _ := json.Marshal(in)
		buyProxy(w, "POST", "/api/order", "", b)
	}))
	m.HandleFunc("GET /api/license/buy/order", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		q := url.Values{"code": {r.URL.Query().Get("code")}, "t": {r.URL.Query().Get("t")}}
		buyProxy(w, "GET", "/api/order", q.Encode(), nil)
	}))
}
