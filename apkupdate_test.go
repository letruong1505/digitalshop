package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApkLatest(t *testing.T) {
	hit := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit++
		if r.URL.Path == "/ds/android-latest.json" { w.Write([]byte(`{"version":"3.1.10","file":"DigitalShop-Android-3.1.10.apk"}`)); return }
		w.WriteHeader(404)
	}))
	defer srv.Close()
	old := updAllowHTTP; updAllowHTTP = true; defer func() { updAllowHTTP = old }()
	apkCache.out = nil
	o, err := apkLatest(srv.URL + "/ds/latest-shop.json")
	if err != nil || o["version"] != "3.1.10" || o["url"] != srv.URL+"/ds/DigitalShop-Android-3.1.10.apk" { t.Fatalf("%v %v", err, o) }
	if _, err := apkLatest(srv.URL + "/ds/latest-shop.json"); err != nil || hit != 1 { t.Fatalf("phai dung bo nho tam: hit=%d %v", hit, err) }
	apkCache.out = nil
	if _, err := apkLatest(srv.URL + "/khac/latest-shop.json"); err == nil { t.Fatal("404 phai bao loi") }
}
