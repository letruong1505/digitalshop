package main

// Tự cập nhật phần mềm (Windows): tải bản mới từ địa chỉ do Nam Linh Group công bố, kiểm tra chữ ký
// Ed25519 + sha256 + kích thước, rồi thay file exe và mở lại. Khóa ký riêng, khác khóa bản quyền.
//
// File latest-shop.json: {product, version, url, sha256, size, notes, sig}
// sig = chữ ký Ed25519 của chuỗi  product|version|sha256|url

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var updPubKey = "0eE4Fe+t2pDrJ52nY5A97y78AUMEGJtijFi8pSEBC3Y="

const (
	updProduct    = "shop"
	updURLDefault = "https://pub-ccfe83f2b7834443814836e1985185b3.r2.dev/digitalshop/latest-shop.json"
	updMaxSize    = 200 << 20
)

var updAllowHTTP bool // chỉ bật khi thử nghiệm

type updManifest struct {
	Product string `json:"product"`
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Notes   string `json:"notes"`
	Sig     string `json:"sig"`
}

type UpdInfo struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Newer   bool   `json:"newer"`
	Notes   string `json:"notes,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Auto    bool   `json:"auto"`
	Can     bool   `json:"can"` // hệ điều hành này tự thay được file không
}

var upd struct {
	mu   sync.Mutex
	last *updManifest
}

func updFeedURL(st Settings) string {
	if u := strings.TrimSpace(os.Getenv("DS_UPDATE_URL")); u != "" { return u }
	return updURLDefault
}

func httpsOK(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Host == "" { return false }
	return p.Scheme == "https" || (updAllowHTTP && p.Scheme == "http")
}

// verVal đổi "3.1.10" thành các số để so sánh đúng (3.1.10 > 3.1.9).
func verVal(v string) [4]int {
	var out [4]int
	for i, p := range strings.SplitN(strings.TrimSpace(v), ".", 4) {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		out[i] = n
	}
	return out
}

func verNewer(a, b string) bool {
	x, y := verVal(a), verVal(b)
	for i := range x {
		if x[i] != y[i] { return x[i] > y[i] }
	}
	return false
}

func decodeSig(s string) []byte {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil { return b }
	}
	return nil
}

func updVerify(m *updManifest) error {
	pub, _ := base64.StdEncoding.DecodeString(updPubKey)
	sig := decodeSig(m.Sig)
	msg := m.Product + "|" + m.Version + "|" + m.SHA256 + "|" + m.URL
	if len(sig) != ed25519.SignatureSize || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(pub), []byte(msg), sig) {
		return uerr("Thông tin bản cập nhật không có chữ ký hợp lệ của Nam Linh Group, đã bỏ qua.")
	}
	if m.Product != updProduct { return uerr("Bản cập nhật này không dành cho Digital Shop.") }
	if !httpsOK(m.URL) { return uerr("Địa chỉ tải bản cập nhật không an toàn (không phải https).") }
	if len(m.SHA256) != 64 || m.Size <= 0 || m.Size > updMaxSize { return uerr("Thông tin bản cập nhật không đầy đủ.") }
	return nil
}

func updCheck(feed string) (*UpdInfo, error) {
	if !httpsOK(feed) { return nil, uerr("Địa chỉ cập nhật phải bắt đầu bằng https://") }
	cl := &http.Client{Timeout: 20 * time.Second}
	resp, err := cl.Get(feed)
	if err != nil { return nil, uerr("Không kết nối được máy chủ cập nhật. Kiểm tra internet rồi thử lại.") }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, uerr("Máy chủ cập nhật chưa có thông tin bản mới (mã %d).", resp.StatusCode) }
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var m updManifest
	if json.Unmarshal(b, &m) != nil { return nil, uerr("Thông tin cập nhật bị lỗi.") }
	if err := updVerify(&m); err != nil { return nil, err }
	upd.mu.Lock()
	upd.last = &m
	upd.mu.Unlock()
	return &UpdInfo{Current: AppVersion, Latest: m.Version, Newer: verNewer(m.Version, AppVersion), Notes: m.Notes, Size: m.Size, Can: updSupported}, nil
}

// updDownload tải bản mới cạnh file exe hiện tại và kiểm tra; trả về đường dẫn file .new.
func updDownload(m *updManifest, exe string) (string, error) {
	tmp := exe + ".new"
	os.Remove(tmp)
	cl := &http.Client{Timeout: 10 * time.Minute}
	resp, err := cl.Get(m.URL)
	if err != nil { return "", uerr("Không tải được bản cập nhật. Kiểm tra internet rồi thử lại.") }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return "", uerr("Máy chủ không có file cập nhật (mã %d).", resp.StatusCode) }
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil { return "", uerr("Không ghi được file vào thư mục cài đặt (%v). Hãy cài lại bằng bộ cài mới nhất.", err) }
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, m.Size+1))
	f.Close()
	bad := func(msg string) (string, error) { os.Remove(tmp); return "", uerr("%s", msg) }
	if err != nil { return bad("Tải bản cập nhật bị gián đoạn, thử lại.") }
	if n != m.Size { return bad("File tải về sai kích thước, đã hủy cập nhật.") }
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), m.SHA256) { return bad("File tải về không khớp mã kiểm tra, đã hủy cập nhật.") }
	if updSupported {
		hd := make([]byte, 2)
		if g, err := os.Open(tmp); err == nil { g.Read(hd); g.Close() }
		if string(hd) != "MZ" { return bad("File tải về không phải chương trình Windows, đã hủy cập nhật.") }
	}
	return tmp, nil
}

// updApply tải và thay file exe; khi trả về nil, bên gọi phải tắt tiến trình hiện tại.
func updApply(feed string) (string, error) {
	if !updSupported { return "", uerr("Tự cập nhật chỉ dùng trên Windows. Hãy cài bản mới bằng tay.") }
	info, err := updCheck(feed)
	if err != nil { return "", err }
	if !info.Newer { return "", uerr("Bạn đang dùng bản mới nhất (%s).", AppVersion) }
	upd.mu.Lock()
	m := upd.last
	upd.mu.Unlock()
	exe, err := os.Executable()
	if err != nil { return "", uerr("Không xác định được vị trí chương trình.") }
	if r, err := filepath.EvalSymlinks(exe); err == nil { exe = r }
	tmp, err := updDownload(m, exe)
	if err != nil { return "", err }
	if err := updSwap(exe, tmp); err != nil { os.Remove(tmp); return "", err }
	return m.Version, nil
}

// updCleanup xóa file .old/.new còn sót lại sau lần cập nhật trước.
func updCleanup() {
	exe, err := os.Executable()
	if err != nil { return }
	go func() {
		if os.Getenv("DS_UPDATED") != "" { time.Sleep(15 * time.Second) }
		os.Remove(exe + ".old")
		os.Remove(exe + ".new")
	}()
}

func (s *Server) updRoutes(m *http.ServeMux) {
	feed := func() (string, bool) {
		var st Settings
		s.db.Read(func(x *State) { st = x.Settings })
		return updFeedURL(st), !st.UpdateOff
	}
	m.HandleFunc("POST /api/update/check", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		f, auto := feed()
		info, err := updCheck(f)
		if err != nil { failErr(w, err); return }
		info.Auto = auto
		writeJSON(w, info)
	}))
	m.HandleFunc("POST /api/update/apply", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		f, _ := feed()
		ver, err := updApply(f)
		if err != nil { failErr(w, err); return }
		say("Cập nhật lên " + ver + ", đang mở lại")
		writeJSON(w, map[string]any{"ok": true, "version": ver})
		if fl, ok := w.(http.Flusher); ok { fl.Flush() }
		go func() { time.Sleep(600 * time.Millisecond); updRestart(); s.Quit() }()
	}))
	m.HandleFunc("POST /api/update/config", s.auth(true, func(w http.ResponseWriter, r *http.Request, u *User) {
		var in struct{ Auto bool `json:"auto"` }
		if err := readJSON(r, &in); err != nil { failErr(w, err); return }
		err := s.db.Do(func(st *State, t *Tx) error {
			st.Settings.UpdateOff = !in.Auto
			t.put("settings", "", st.Settings)
			return nil
		})
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
}
