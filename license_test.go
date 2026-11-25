package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func signLic(priv ed25519.PrivateKey, l License) string {
	b, _ := json.Marshal(l)
	return "DS1." + base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, b))
}

func licTestSetup(t *testing.T) ed25519.PrivateKey {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := licPubKey
	licPubKey = base64.StdEncoding.EncodeToString(pub)
	t.Cleanup(func() { licPubKey = old; lic.machine = "" })
	lic.dir = t.TempDir()
	lic.machine = "AAAA-BBBB-CCCC-DDDD"
	lic.st = licState{TrialStart: time.Now().Unix(), LastSeen: time.Now().Unix()}
	return priv
}

func TestLicenseTrialAndActivate(t *testing.T) {
	priv := licTestSetup(t)
	if s := licStatus(); s.State != "trial" || !s.CanWrite || s.DaysLeft != 30 { t.Fatalf("trial: %+v", s) }
	if licGate("POST", "/api/invoices") != "" { t.Fatal("trial must allow writes") }

	// hết dùng thử: chặn ghi, vẫn cho xem
	lic.st.TrialStart = time.Now().AddDate(0, 0, -31).Unix()
	if s := licStatus(); s.State != "trial-over" || s.CanWrite { t.Fatalf("over: %+v", s) }
	if licGate("POST", "/api/invoices") == "" { t.Fatal("must block write") }
	if licGate("GET", "/api/boot") != "" { t.Fatal("must allow read") }
	if licGate("POST", "/api/license/activate") != "" || licGate("POST", "/api/login") != "" { t.Fatal("activation/login must stay open") }

	good := signLic(priv, License{V: 1, ID: "NL-1", Machine: "AAAA-BBBB-CCCC-DDDD", Shop: "Tiệm A", Plan: "hddt", Exp: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), Iss: "x"})
	// dán từ Zalo có xuống dòng
	st, err := licActivate(good[:40] + "\n " + good[40:])
	if err != nil || st.State != "active" || !st.CanWrite { t.Fatalf("activate: %v %+v", err, st) }
	if licGate("POST", "/api/invoices") != "" { t.Fatal("active must allow") }

	// sai máy
	other := signLic(priv, License{V: 1, ID: "NL-2", Machine: "ZZZZ-BBBB-CCCC-DDDD", Shop: "B", Plan: "hddt", Iss: "x"})
	if _, err := licActivate(other); err == nil || !strings.Contains(err.Error(), "máy") { t.Fatalf("wrong machine: %v", err) }
	// sửa payload
	parts := strings.Split(good, ".")
	pl, _ := base64.RawURLEncoding.DecodeString(parts[1])
	pl = []byte(strings.Replace(string(pl), "Tiệm A", "Tiệm Z", 1))
	if _, err := licActivate("DS1." + base64.RawURLEncoding.EncodeToString(pl) + "." + parts[2]); err == nil { t.Fatal("tamper accepted") }
	// khóa khác
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := licActivate(signLic(priv2, License{V: 1, ID: "NL-3", Machine: "AAAA-BBBB-CCCC-DDDD", Shop: "C", Plan: "hddt", Iss: "x"})); err == nil { t.Fatal("foreign key accepted") }
	// hết hạn
	lic.st.Key = signLic(priv, License{V: 1, ID: "NL-4", Machine: "AAAA-BBBB-CCCC-DDDD", Shop: "D", Plan: "hddt", Exp: "2020-01-01", Iss: "x"})
	if s := licStatus(); s.State != "trial-over" && s.State != "expired" || s.CanWrite { t.Fatalf("expired: %+v", s) }
	// vĩnh viễn
	st, err = licActivate(signLic(priv, License{V: 1, ID: "NL-5", Machine: "aaaa bbbb cccc dddd", Shop: "E", Plan: "co-ban", Iss: "x"}))
	if err != nil || st.State != "active" || st.DaysLeft != 0 { t.Fatalf("lifetime: %v %+v", err, st) }
}

func TestLicenseClockRollback(t *testing.T) {
	licTestSetup(t)
	lic.st.TrialStart = time.Now().AddDate(0, 0, -29).Unix()
	lic.st.LastSeen = time.Now().AddDate(0, 0, 5).Unix() // đã từng thấy giờ ở tương lai
	if s := licStatus(); s.State != "trial-over" { t.Fatalf("rollback must not extend trial: %+v", s) }
}

func TestLicenseClaim(t *testing.T) {
	priv := licTestSetup(t)
	key := signLic(priv, License{V: 1, ID: "NL-9", Machine: "AAAA-BBBB-CCCC-DDDD", Shop: "F", Plan: "hddt", Iss: "x"})
	paid := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/claim" || r.URL.Query().Get("sp") != "shop" || r.URL.Query().Get("m") != "AAAA-BBBB-CCCC-DDDD" { w.WriteHeader(404); return }
		if !paid { w.Write([]byte(`{"ok":false}`)); return }
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "key": key})
	}))
	defer ts.Close()
	os.Setenv("DS_BUY_URL", ts.URL)
	defer os.Unsetenv("DS_BUY_URL")
	if !strings.HasPrefix(buyURL("AAAA-BBBB-CCCC-DDDD"), ts.URL+"/mua?sp=shop&may=AAAA-BBBB-CCCC-DDDD") { t.Fatal("buyURL") }
	if _, got, err := licClaim(); err != nil || got { t.Fatalf("unpaid: %v %v", err, got) }
	paid = true
	st, got, err := licClaim()
	if err != nil || !got || st.State != "active" { t.Fatalf("paid: %v %v %+v", err, got, st) }
	if _, got, _ = licClaim(); got { t.Fatal("second claim must be no-op") }
	// máy chủ trả mã ký sai khóa → từ chối
	key = signLic(func() ed25519.PrivateKey { _, p, _ := ed25519.GenerateKey(rand.Reader); return p }(), License{V: 1, ID: "NL-X", Machine: "AAAA-BBBB-CCCC-DDDD", Shop: "F", Plan: "hddt", Iss: "x"})
	if _, _, err := licClaim(); err == nil { t.Fatal("bad signature accepted") }
}
