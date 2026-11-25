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
