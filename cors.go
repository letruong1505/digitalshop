package main

import "net/http"

// Ứng dụng Android (Capacitor) chạy giao diện từ http://localhost trong điện thoại và gọi API của máy chủ
// trong mạng wifi cửa hàng. Chỉ cho phép đúng các nguồn này; đăng nhập bằng token (header Authorization),
// không dùng cookie nên không mở thêm đường tấn công từ trang web lạ.
var corsOrigins = map[string]bool{"http://localhost": true, "https://localhost": true, "capacitor://localhost": true}

func corsHandle(w http.ResponseWriter, r *http.Request) (done bool) {
	o := r.Header.Get("Origin")
	if o == "" || !corsOrigins[o] { return false }
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", o)
	h.Set("Vary", "Origin")
	h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-DS-Client")
	h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	h.Set("Access-Control-Max-Age", "600")
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return true }
	return false
}

func bearerToken(r *http.Request) string {
	a := r.Header.Get("Authorization")
	if len(a) > 7 && a[:7] == "Bearer " { return a[7:] }
	return ""
}
