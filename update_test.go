package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerNewer(t *testing.T) {
	if !verNewer("3.1.10", "3.1.9") || verNewer("3.1.0", "3.1.0") || verNewer("3.0.16", "3.1.0") || !verNewer("3.2", "3.1.9") { t.Fatal("verNewer") }
}

func updTestServer(t *testing.T, priv ed25519.PrivateKey, body []byte, ver, product string, tamperSHA bool) (*httptest.Server, *string) {
	sum := sha256.Sum256(body)
	sh := hex.EncodeToString(sum[:])
	var feed string
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	exeURL := ts.URL + "/Digital.exe"
	m := updManifest{Product: product, Version: ver, URL: exeURL, SHA256: sh, Size: int64(len(body)), Notes: "n"}
	m.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(m.Product+"|"+m.Version+"|"+m.SHA256+"|"+m.URL)))
	mux.HandleFunc("/latest.json", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(m) })
	mux.HandleFunc("/Digital.exe", func(w http.ResponseWriter, r *http.Request) {
		b := body
		if tamperSHA { b = append([]byte{}, body...); b[len(b)-1] ^= 1 }
		w.Write(b)
	})
	feed = ts.URL + "/latest.json"
	return ts, &feed
}

func TestUpdateCheckAndDownload(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old, oldHTTP := updPubKey, updAllowHTTP
	updPubKey, updAllowHTTP = base64.StdEncoding.EncodeToString(pub), true
	defer func() { updPubKey, updAllowHTTP = old, oldHTTP }()
	body := append([]byte("MZ"), []byte(strings.Repeat("x", 5000))...)

	ts, feed := updTestServer(t, priv, body, "99.0.0", "shop", false)
	defer ts.Close()
	info, err := updCheck(*feed)
	if err != nil || !info.Newer || info.Latest != "99.0.0" { t.Fatalf("check: %v %+v", err, info) }
	exe := filepath.Join(t.TempDir(), "Digital.exe")
	os.WriteFile(exe, []byte("old"), 0o755)
	tmp, err := updDownload(upd.last, exe)
	if err != nil { t.Fatal(err) }
	if b, _ := os.ReadFile(tmp); len(b) != len(body) { t.Fatal("downloaded size") }

	// file bị sửa giữa đường → từ chối và xóa
	ts2, feed2 := updTestServer(t, priv, body, "99.0.0", "shop", true)
	defer ts2.Close()
	if _, err := updCheck(*feed2); err != nil { t.Fatal(err) }
	if _, err := updDownload(upd.last, exe); err == nil { t.Fatal("tampered binary accepted") }
	if _, err := os.Stat(exe + ".new"); err == nil { t.Fatal(".new must be removed") }

	// sai sản phẩm
	ts3, feed3 := updTestServer(t, priv, body, "99.0.0", "fnb", false)
	defer ts3.Close()
	if _, err := updCheck(*feed3); err == nil { t.Fatal("wrong product accepted") }
	// chữ ký của khóa khác
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	ts4, feed4 := updTestServer(t, priv2, body, "99.0.0", "shop", false)
	defer ts4.Close()
	if _, err := updCheck(*feed4); err == nil { t.Fatal("foreign signature accepted") }
	// http bị từ chối khi không ở chế độ thử
	updAllowHTTP = false
	if _, err := updCheck(*feed); err == nil { t.Fatal("http accepted") }
	// bản không mới hơn
	updAllowHTTP = true
	ts5, feed5 := updTestServer(t, priv, body, "1.0.0", "shop", false)
	defer ts5.Close()
	if i, err := updCheck(*feed5); err != nil || i.Newer { t.Fatalf("older: %v %+v", err, i) }
}
