package main

// Bản quyền Digital Shop
//
// Mỗi bản quyền gắn với 1 máy chủ (máy chạy phần mềm và giữ dữ liệu), điện thoại trong tiệm không tính.
//   - Mã máy: băm từ thông tin riêng của máy (Windows: MachineGuid + số sê-ri ổ hệ thống;
//     Android: ANDROID_ID do app truyền vào). Chép phần mềm/dữ liệu sang máy khác → mã máy khác.
//   - Mã kích hoạt: "DS1." + base64url(JSON) + "." + base64url(chữ ký Ed25519) do Nam Linh Group ký
//     bằng khóa bí mật (công cụ licgen). Phần mềm chỉ có khóa công khai → không tự làm mã giả được.
//   - Chưa kích hoạt: dùng thử đầy đủ 30 ngày. Hết hạn: chỉ xem và xuất dữ liệu (không khóa dữ liệu của khách).
//   - Một gói duy nhất "Đầy đủ" (mã cũ gói co-ban/hddt đều được chấp nhận và coi như Đầy đủ).
//   - Chống lùi đồng hồ: dùng mốc thời gian lớn nhất từng thấy.

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var licPubKey = "liHb941iLYtHy190MziQp1H4xUOG/JeGXEBWrK+dqk0="

const (
	trialDays  = 30
	licFile    = "banquyen.json"
	PlanBasic  = "co-ban"
	PlanEinv   = "hddt"
)

// License: nội dung mã kích hoạt (JSON gọn để mã ngắn).
type License struct {
	V       int    `json:"v"`
	ID      string `json:"id"`
	Machine string `json:"m"`
	Shop    string `json:"shop"`
	MST     string `json:"mst,omitempty"`
	Plan    string `json:"plan"`
	Exp     string `json:"exp,omitempty"` // YYYY-MM-DD, trống = vĩnh viễn
	Iss     string `json:"iss"`
}

type licState struct {
	TrialStart int64  `json:"trialStart"` // unix giây
	LastSeen   int64  `json:"lastSeen"`
	Key        string `json:"key,omitempty"`
	Revoked    string `json:"revoked,omitempty"`    // số bản quyền đã bị Nam Linh Group thu hồi (máy chủ báo)
	RevokedEff int64  `json:"revokedEff,omitempty"` // thời điểm (unix giây) bắt đầu chuyển sang chỉ xem/xuất; báo trước 1 ngày
}

type LicStatus struct {
	State    string   `json:"state"` // trial | active | trial-over | expired
	Machine  string   `json:"machine"`
	DaysLeft int      `json:"daysLeft"`
	TrialEnd string   `json:"trialEnd,omitempty"`
	CanWrite bool     `json:"canWrite"`
	CanEinv  bool     `json:"canEinv"`
	License  *License `json:"license,omitempty"`
	Note     string   `json:"note,omitempty"` // vd mã đã lưu không hợp lệ
	RevokeOn string   `json:"revokeOn,omitempty"` // đã bị thu hồi nhưng còn dùng được đến mốc này (RFC3339)
	BuyURL   string   `json:"buyUrl,omitempty"`
}

var lic struct {
	mu      sync.Mutex
	dir     string
	st      licState
	machine string
	cur     LicStatus
}

// ---------- mã máy ----------

func machineCode() string {
	h := sha256.Sum256([]byte("digitalshop-v1|" + machineRaw()))
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(h[:10])
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
}

func machineRaw() string {
	if onAndroid {
		if id := strings.TrimSpace(os.Getenv("DS_DEVID")); id != "" { return "android|" + id }
	}
	return platformMachineRaw()
}

func normMachine(s string) string {
	s = strings.ToUpper(strings.NewReplacer("-", "", " ", "", "\t", "").Replace(s))
	if len(s) != 16 { return s }
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
}

// ---------- mã kích hoạt ----------

func parseLicense(key string) (*License, error) {
	key = strings.Join(strings.Fields(key), "") // bỏ khoảng trắng/xuống dòng khi dán từ Zalo
	parts := strings.Split(key, ".")
	if len(parts) != 3 || parts[0] != "DS1" { return nil, uerr("Mã kích hoạt không đúng dạng. Dán đầy đủ mã bắt đầu bằng DS1.") }
	payload, err1 := base64.RawURLEncoding.DecodeString(parts[1])
	sig, err2 := base64.RawURLEncoding.DecodeString(parts[2])
	if err1 != nil || err2 != nil { return nil, uerr("Mã kích hoạt bị thiếu hoặc sai ký tự. Sao chép lại đầy đủ mã.") }
	pub, _ := base64.StdEncoding.DecodeString(licPubKey)
	if len(sig) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(pub), payload, sig) {
		return nil, uerr("Mã kích hoạt không hợp lệ (không phải do Nam Linh Group cấp hoặc đã bị sửa).")
	}
	var l License
	if err := json.Unmarshal(payload, &l); err != nil || l.V != 1 { return nil, uerr("Mã kích hoạt không đọc được") }
	return &l, nil
}

func planName(p string) string { return "Đầy đủ" }

// ---------- trạng thái ----------

func licNow() time.Time {
	t := time.Now()
	if lic.st.LastSeen > 0 && t.Unix() < lic.st.LastSeen { t = time.Unix(lic.st.LastSeen, 0) }
	return t
}

func licCompute() LicStatus {
	now := licNow()
	s := LicStatus{Machine: lic.machine}
	trialEnd := time.Unix(lic.st.TrialStart, 0).AddDate(0, 0, trialDays)
	if lic.st.Key != "" {
		l, err := parseLicense(lic.st.Key)
		switch {
		case err != nil:
			s.Note = err.Error()
		case normMachine(l.Machine) != lic.machine:
			s.Note = "Mã kích hoạt đã lưu cấp cho máy khác (" + normMachine(l.Machine) + "). Cần mã mới cho máy này."
		case lic.st.Revoked != "" && lic.st.Revoked == l.ID && now.Unix() >= lic.st.RevokedEff:
			s.License = l
			s.State = "revoked"
			return s
		default:
			s.License = l
			if lic.st.Revoked == l.ID { s.RevokeOn = time.Unix(lic.st.RevokedEff, 0).Format(time.RFC3339) }
			today := now.Format("2006-01-02")
			if l.Exp == "" || today <= l.Exp {
				s.State, s.CanWrite, s.CanEinv = "active", true, true
				if l.Exp != "" {
					if e, err := time.ParseInLocation("2006-01-02", l.Exp, time.Local); err == nil {
						s.DaysLeft = int(e.AddDate(0, 0, 1).Sub(now).Hours()/24 + 0.999)
					}
				}
			} else {
				s.State = "expired"
			}
			return s
		}
	}
	if now.Before(trialEnd) {
		s.State, s.CanWrite, s.CanEinv = "trial", true, true
		s.DaysLeft = int(trialEnd.Sub(now).Hours()/24 + 0.999)
	} else {
		s.State = "trial-over"
	}
	s.TrialEnd = trialEnd.Format("2006-01-02")
	return s
}

func licSave() {
	b, _ := json.MarshalIndent(lic.st, "", "  ")
	p := filepath.Join(lic.dir, licFile)
	tmp := p + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil { os.Rename(tmp, p) }
}

// licInit đọc trạng thái bản quyền; gọi một lần khi khởi động.
func licInit(dir string) {
	lic.mu.Lock()
	defer lic.mu.Unlock()
	lic.dir = dir
	lic.machine = machineCode()
	if b, err := os.ReadFile(filepath.Join(dir, licFile)); err == nil { json.Unmarshal(b, &lic.st) }
	now := time.Now().Unix()
	// ngày bắt đầu dùng thử: sớm nhất trong file dữ liệu và dấu lưu riêng trên máy (cài lại không làm mới được)
	if m := trialMarkGet(); m > 0 && (lic.st.TrialStart == 0 || m < lic.st.TrialStart) { lic.st.TrialStart = m }
	if lic.st.TrialStart == 0 || lic.st.TrialStart > now+86400 { lic.st.TrialStart = now }
	trialMarkSet(lic.st.TrialStart)
	if now > lic.st.LastSeen { lic.st.LastSeen = now }
	lic.cur = licCompute()
	licSave()
	go licRevokeLoop()
	go func() {
		for range time.Tick(10 * time.Minute) {
			lic.mu.Lock()
			if n := time.Now().Unix(); n > lic.st.LastSeen { lic.st.LastSeen = n; licSave() }
			lic.cur = licCompute()
			lic.mu.Unlock()
		}
	}()
}

func licStatus() LicStatus {
	lic.mu.Lock()
	defer lic.mu.Unlock()
	if lic.machine == "" { return LicStatus{State: "trial", CanWrite: true, CanEinv: true} } // chưa khởi tạo (thử nghiệm đơn vị)
	lic.cur = licCompute()
	return lic.cur
}

func licActivate(key string) (LicStatus, error) {
	l, err := parseLicense(key)
	if err != nil { return LicStatus{}, err }
	lic.mu.Lock()
	defer lic.mu.Unlock()
	if normMachine(l.Machine) != lic.machine {
		return LicStatus{}, uerr("Mã này cấp cho máy %s, không phải máy này (%s). Gửi đúng mã máy cho Nam Linh Group để được cấp lại.", normMachine(l.Machine), lic.machine)
	}
	if l.Exp != "" && licNow().Format("2006-01-02") > l.Exp {
		return LicStatus{}, uerr("Mã kích hoạt đã hết hạn ngày %s. Liên hệ Nam Linh Group để gia hạn.", vnDateStr(l.Exp))
	}
	lic.st.Key = strings.Join(strings.Fields(key), "")
	licSave()
	lic.cur = licCompute()
	go licRevokeCheck()
	return lic.cur, nil
}

func vnDateStr(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil { return t.Format("02/01/2006") }
	return iso
}

// ---------- chặn theo bản quyền ----------

// licGate trả về thông báo lỗi nếu yêu cầu bị chặn ("" = cho qua).
func licGate(method, path string) string {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions { return "" }
	if !strings.HasPrefix(path, "/api/") { return "" }
	if licStatus().CanWrite { return "" }
	for _, p := range []string{"/api/setup", "/api/login", "/api/logout", "/api/license", "/api/backups", "/api/shutdown", "/api/open-folder", "/api/display", "/api/update"} {
		if path == p || strings.HasPrefix(path, p+"/") { return "" }
	}
	return licBlockedMsg(licStatus())
}

func licBlockedMsg(s LicStatus) string {
	if s.State == "revoked" {
		return "Bản quyền đã bị thu hồi. Dữ liệu vẫn xem và xuất được; liên hệ Nam Linh Group để được hỗ trợ (Cài đặt → Bản quyền)."
	}
	if s.State == "expired" && s.License != nil {
		return "Bản quyền Digital Shop đã hết hạn ngày " + vnDateStr(s.License.Exp) + ". Dữ liệu vẫn xem và xuất được; muốn bán hàng tiếp, liên hệ Nam Linh Group để gia hạn (Cài đặt → Bản quyền)."
	}
	return "Đã hết " + itoa(int64(trialDays)) + " ngày dùng thử Digital Shop. Dữ liệu vẫn xem và xuất được; muốn bán hàng tiếp, vào Cài đặt → Bản quyền để kích hoạt."
}

func (s *Server) licRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/license", s.auth(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		st := licStatus()
		st.BuyURL = buyURL(st.Machine)
		writeJSON(w, st)
	}))
	m.HandleFunc("POST /api/license/check", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		licRevokeCheck()
		st := licStatus()
		st.BuyURL = buyURL(st.Machine)
		writeJSON(w, st)
	}))
	s.buyRoutes(m)
	m.HandleFunc("POST /api/license/claim", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		st, got, err := licClaim()
		if err != nil { failErr(w, err); return }
		if got { say("Nhận bản quyền đã mua: " + st.License.ID) }
		st.BuyURL = buyURL(st.Machine)
		writeJSON(w, map[string]any{"found": got || st.License != nil, "new": got, "status": st})
	}))
	m.HandleFunc("POST /api/license/activate", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Key string `json:"key"` }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		st, err := licActivate(req.Key)
		if err != nil { failErr(w, err); return }
		say("Kích hoạt bản quyền: " + st.License.ID + " · " + st.License.Shop + " · gói " + planName(st.License.Plan))
		writeJSON(w, st)
	}))
}
