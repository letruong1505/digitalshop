package main

// Thông báo bản Android mới: máy chủ (có internet) đọc android-latest.json cùng thư mục với latest-shop.json,
// ứng dụng Android hỏi máy chủ nên không vướng CORS và không cần tự có internet.

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var apkFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.apk$`)
var apkVer = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}$`)

var apkCache struct {
	mu   sync.Mutex
	at   time.Time
	feed string
	out  map[string]string
}

func apkLatest(feed string) (map[string]string, error) {
	apkCache.mu.Lock()
	defer apkCache.mu.Unlock()
	if apkCache.out != nil && apkCache.feed == feed && time.Since(apkCache.at) < 10*time.Minute { return apkCache.out, nil }
	i := strings.LastIndex(feed, "/")
	if i < 0 || !httpsOK(feed) { return nil, uerr("Chưa cấu hình địa chỉ cập nhật") }
	base := feed[:i+1]
	cl := &http.Client{Timeout: 8 * time.Second}
	resp, err := cl.Get(base + "android-latest.json")
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, uerr("Chưa có bản Android") }
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var j struct{ Version, File string }
	if json.Unmarshal(b, &j) != nil || !apkVer.MatchString(j.Version) || !apkFile.MatchString(j.File) { return nil, uerr("Thông tin bản Android không hợp lệ") }
	out := map[string]string{"version": j.Version, "url": base + j.File}
	apkCache.at, apkCache.feed, apkCache.out = time.Now(), feed, out
	return out, nil
}

func (s *Server) apkRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/android/latest", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		var st Settings
		s.db.Read(func(x *State) { st = x.Settings })
		out, err := apkLatest(updFeedURL(st))
		if err != nil { failErr(w, err); return }
		writeJSON(w, out)
	}))
}
