package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuyProxy(t *testing.T) {
	var gotPath, gotQuery, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotQuery, gotBody = r.URL.Path, r.URL.RawQuery, string(b)
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"x"}`))
	}))
	defer ts.Close()
	t.Setenv("DS_BUY_URL", ts.URL)
	rec := httptest.NewRecorder()
	buyProxy(rec, "POST", "/api/order", "", []byte(`{"a":1}`))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), `"error"`) { t.Fatal("pass-through", rec.Code, rec.Body.String()) }
	if gotPath != "/api/order" || gotBody != `{"a":1}` { t.Fatal(gotPath, gotBody) }
	rec = httptest.NewRecorder()
	buyProxy(rec, "GET", "/api/partner", "sp=shop&code=A", nil)
	if gotQuery != "sp=shop&code=A" { t.Fatal(gotQuery) }
}
