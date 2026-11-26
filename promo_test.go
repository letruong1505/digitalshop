package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRolesCan(t *testing.T) {
	cases := []struct{ role, perm string; want bool }{
		{"owner", "admin", true}, {"manager", "admin", false}, {"manager", "manage", true},
		{"cashier", "sell", true}, {"cashier", "stock", false}, {"cashier", "report", false}, {"staff", "sell", true}, {"staff", "money", false},
		{"warehouse", "stock", true}, {"warehouse", "sell", false}, {"accountant", "report", true}, {"accountant", "sell", false},
		{"cashier", "sell|money", true}, {"cashier", "stock|money", false}, {"cashier", "", true},
	}
	for _, c := range cases {
		if got := (&User{Role: c.role}).can(c.perm); got != c.want { t.Fatalf("%s/%s = %v", c.role, c.perm, got) }
	}
}

func TestPromoAndTier(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Chủ", Role: "owner"}
	a, err := db.SaveProduct(ProductReq{Product: Product{Name: "Nước ngọt", Unit: "chai", Price: 10000, Cost: 6000, Group: "Đồ uống"}, SetStock: f100()}, u)
	if err != nil { t.Fatal(err) }
	b, err := db.SaveProduct(ProductReq{Product: Product{Name: "Bánh", Unit: "gói", Price: 20000, Cost: 12000, Group: "Bánh kẹo"}, SetStock: f100()}, u)
	if err != nil { t.Fatal(err) }
	c, _ := db.SaveParty("cust", Party{Name: "Cô Lan"})

	now := time.Date(2026, 11, 26, 20, 0, 0, 0, time.Local) // thứ năm 20:00
	err = db.SavePromos([]Promo{
		{Name: "Giờ vàng đồ uống", Active: true, Kind: "pct", Value: 20, Scope: "group", Target: "Đồ uống", TStart: "18:00", TEnd: "22:00"},
		{Name: "Khuya", Active: true, Kind: "pct", Value: 50, Scope: "all", TStart: "22:00", TEnd: "02:00"},
		{Name: "VIP bánh", Active: true, Kind: "amt", Value: 3000, Scope: "product", Target: b.ID, MinTier: 1},
		{Name: "Tắt", Active: false, Kind: "pct", Value: 90, Scope: "all"},
	}, []Tier{{"Vàng", 1000000}, {"Bạc", 100000}})
	if err != nil { t.Fatal(err) }
	var s0 State
	db.Read(func(s *State) { s0 = *s })
	if s0.Settings.Tiers[0].Name != "Bạc" { t.Fatal("tier phải xếp tăng dần") }

	lines := []Line{{PID: a.ID, Qty: 5, Price: 10000}, {PID: b.ID, Qty: 2, Price: 20000}}
	d, hits := promoDisc(&s0, lines, "", now)
	if d != 10000 || len(hits) != 1 { t.Fatalf("giờ vàng 20%% của 50.000 = 10.000, được %v %v", d, hits) }
	d, _ = promoDisc(&s0, lines, "", time.Date(2026, 11, 26, 15, 0, 0, 0, time.Local))
	if d != 0 { t.Fatalf("ngoài giờ không giảm, được %v", d) }
	d, _ = promoDisc(&s0, lines, "", time.Date(2026, 11, 26, 23, 30, 0, 0, time.Local))
	if d != 45000 { t.Fatalf("khuya qua nửa đêm giảm 50%% của 90.000, được %v", d) }
	d, _ = promoDisc(&s0, lines, "", time.Date(2026, 11, 27, 1, 0, 0, 0, time.Local))
	if d != 45000 { t.Fatalf("1h sáng vẫn trong khung khuya, được %v", d) }

	// khách chưa đạt hạng: không có KM VIP; mua đủ 100.000 thì đạt hạng 1
	d, _ = promoDisc(&s0, lines, c.ID, now)
	if d != 10000 { t.Fatalf("chưa đạt hạng, được %v", d) }
	if _, err := db.CreateInvoice(SaleReq{CID: "x1", Lines: []Line{{PID: a.ID, Qty: 30, Price: 10000, Ratio: 1}}, Method: "cash", CustID: c.ID}, u); err != nil { t.Fatal(err) }
	db.Read(func(s *State) { s0 = *s })
	if tierIndex(tiersOf(&s0), custSpendAll(&s0)[c.ID]) != 1 { t.Fatal("phải đạt hạng Bạc") }
	d, _ = promoDisc(&s0, lines, c.ID, now)
	if d != 10000+6000 { t.Fatalf("hạng Bạc được thêm giảm 3.000 x 2 gói, được %v", d) }

	// hóa đơn thật áp dụng khuyến mãi tự động và ghi nhận
	rep := db.Report("2000-01-01", "2100-01-01", true)
	if len(rep.Custs) != 1 || rep.Custs[0]["tier"] != "Bạc" { t.Fatalf("xếp hạng khách: %v", rep.Custs) }
}

func f100() *float64 { x := 100.0; return &x }

func TestOfflineInvoice(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "u", Name: "Thu ngân", Role: "cashier"}
	a, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Kẹo", Unit: "gói", Price: 5000, Cost: 3000}, SetStock: f100()}, &User{Role: "owner"})
	when := time.Now().Add(-3 * time.Hour).UnixMilli()
	req := SaleReq{CID: "off1", Lines: []Line{{PID: a.ID, Qty: 4, Price: 5000, Ratio: 1}}, Method: "cash", Given: 20000, Offline: true, Time: when, PromoDisc: 2000, PromoNote: "Giờ vàng"}
	inv, err := db.CreateInvoice(req, u)
	if err != nil { t.Fatal(err) }
	if inv.Time != when || inv.Promo != 2000 || inv.Total != 18000 { t.Fatalf("hóa đơn ngoại tuyến: time=%v promo=%v total=%v", inv.Time, inv.Promo, inv.Total) }
	again, _ := db.CreateInvoice(req, u) // đồng bộ lặp lại không tạo trùng
	if again.ID != inv.ID || again.Code != inv.Code { t.Fatal("gửi lại phải trả đúng hóa đơn cũ") }
	req2 := req; req2.CID = "off2"; req2.Time = time.Now().Add(-90 * 24 * time.Hour).UnixMilli()
	inv2, _ := db.CreateInvoice(req2, u)
	if inv2.Time < time.Now().Add(-time.Minute).UnixMilli() { t.Fatal("giờ quá cũ phải bị bỏ qua, dùng giờ hiện tại") }
}

func TestCustomPermsAndLAN(t *testing.T) {
	u := &User{Role: "cashier", Perms: cleanPerms("cashier", []string{"sell", "stock", "admin", "zzz"})}
	if !u.can("sell") || !u.can("stock") || u.can("admin") || u.can("money") { t.Fatalf("perms=%v", u.perms()) }
	if cleanPerms("cashier", []string{"sell"}) != nil { t.Fatal("giong mac dinh phai tra nil") }
	z := &User{Role: "cashier", Perms: cleanPerms("cashier", []string{})}
	if z.Perms == nil || z.can("sell") { t.Fatal("khong quyen nao phai giu rong, khong ve mac dinh") }
	if (&User{Role: "owner", Perms: []string{}}).can("admin") == false { t.Fatal("chu luon day du") }
	mk := func(addr string, h map[string]string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil); r.RemoteAddr = addr
		for k, v := range h { r.Header.Set(k, v) }
		return r
	}
	cases := []struct{ addr string; h map[string]string; want bool }{
		{"192.168.1.20:5000", nil, true}, {"127.0.0.1:5000", nil, true}, {"10.0.0.5:1", nil, true},
		{"8.8.8.8:1", nil, false}, {"100.64.1.2:1", nil, false}, {"[2001:db8::1]:1", nil, false},
		{"127.0.0.1:1", map[string]string{"Cf-Connecting-Ip": "113.1.2.3"}, false},
		{"127.0.0.1:1", map[string]string{"X-Forwarded-For": "113.1.2.3, 10.0.0.1"}, false},
		{"127.0.0.1:1", map[string]string{"X-Forwarded-For": "192.168.1.9"}, true},
		{"8.8.8.8:1", map[string]string{"X-Forwarded-For": "192.168.1.9"}, false},
	}
	for _, c := range cases { if got := isLANReq(mk(c.addr, c.h)); got != c.want { t.Fatalf("%s %v = %v", c.addr, c.h, got) } }
}
