package main

import (
	"fmt"
	"strconv"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Server struct {
	db       *DB
	mux      *http.ServeMux
	sessMu   sync.Mutex
	sessions map[string]session
	fails    map[string][]time.Time
	quit     chan struct{}
	quitOnce sync.Once
	port     int
}

// Quit yêu cầu tắt phần mềm (gọi được nhiều lần, từ web hoặc từ biểu tượng khay hệ thống).
func (s *Server) Quit() { s.quitOnce.Do(func() { close(s.quit) }) }

type session struct {
	uid string
	exp time.Time
}

func NewServer(db *DB, port int) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), sessions: map[string]session{}, fails: map[string][]time.Time{}, quit: make(chan struct{}), port: port}
	sub, _ := fs.Sub(webFS, "web")
	static := http.FileServer(http.FS(sub))
	s.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".woff2") { w.Header().Set("Cache-Control", "max-age=31536000") } else { w.Header().Set("Cache-Control", "no-cache") }
		static.ServeHTTP(w, r)
	}))
	s.routes()
	db.Do(func(st *State, t *Tx) error {
		if st.Settings.DisplayKey == "" { st.Settings.DisplayKey = randHex(6); t.put("settings", "", st.Settings) }
		return nil
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if msg := licGate(r.Method, r.URL.Path); msg != "" { fail(w, 402, msg); return }
	s.mux.ServeHTTP(w, r)
}

// ---------- tiện ích HTTP ----------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func failErr(w http.ResponseWriter, err error) {
	var ue userErr
	if errors.As(err, &ue) { fail(w, 400, ue.msg); return }
	fail(w, 500, "Lỗi hệ thống: "+err.Error())
}

func readJSON(r *http.Request, v any) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil { return uerr("Không đọc được dữ liệu gửi lên") }
	if err := json.Unmarshal(b, v); err != nil { return uerr("Dữ liệu gửi lên không hợp lệ") }
	return nil
}

func hashPin(salt, pin string) string {
	h := sha256.Sum256([]byte(salt + ":" + pin))
	return hex.EncodeToString(h[:])
}

func randHex(n int) string { b := make([]byte, n); rand.Read(b); return hex.EncodeToString(b) }

func clientIP(r *http.Request) string { h, _, _ := net.SplitHostPort(r.RemoteAddr); return h }

func (s *Server) user(r *http.Request) *User {
	c, err := r.Cookie("thx")
	if err != nil { return nil }
	s.sessMu.Lock()
	ss, ok := s.sessions[c.Value]
	s.sessMu.Unlock()
	if !ok || time.Now().After(ss.exp) { return nil }
	var u *User
	s.db.Read(func(st *State) { if x := st.Users[ss.uid]; x != nil { cp := *x; u = &cp } })
	return u
}

type handler func(w http.ResponseWriter, r *http.Request, u *User)

func (s *Server) auth(perm string, h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := s.user(r)
		if u == nil { fail(w, 401, "Phiên đăng nhập đã hết, hãy nhập lại mã PIN"); return }
		if !u.can(perm) { fail(w, 403, "Vai trò "+strings.ToLower(roleNames[normRole(u.Role)])+" không có quyền làm thao tác này. Nhờ chủ cửa hàng hỗ trợ."); return }
		h(w, r, u)
	}
}

func (s *Server) login(w http.ResponseWriter, uid string) {
	tok := randHex(24)
	s.sessMu.Lock()
	s.sessions[tok] = session{uid: uid, exp: time.Now().Add(30 * 24 * time.Hour)}
	s.sessMu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "thx", Value: tok, Path: "/", HttpOnly: true, MaxAge: 30 * 86400, SameSite: http.SameSiteLaxMode})
}

func lanAddrs(port int) []string {
	out := []string{}
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 { continue }
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil { continue }
			ip := ipn.IP.String()
			if strings.HasPrefix(ip, "169.254.") { continue }
			out = append(out, "http://"+ip+":"+itoa(int64(port)))
		}
	}
	sort.Slice(out, func(a, b int) bool { return strings.Contains(out[a], "192.168.") && !strings.Contains(out[b], "192.168.") })
	return out
}

// ---------- định tuyến ----------

func (s *Server) routes() {
	m := s.mux
	s.licRoutes(m)
	s.updRoutes(m)
	m.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("taphoax")) })

	m.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		need := false
		shop := ""
		s.db.Read(func(st *State) { need = len(st.Users) == 0; shop = st.Settings.ShopName })
		u := s.user(r)
		writeJSON(w, map[string]any{"needSetup": need, "loggedIn": u != nil, "shopName": shop, "app": AppName, "version": AppVersion, "termsVersion": TermsVersion})
	})

	m.HandleFunc("POST /api/setup", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ShopName, OwnerName, Pin string
			AcceptTerms             bool
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		if !req.AcceptTerms { fail(w, 400, "Cần đồng ý Điều khoản sử dụng để bắt đầu"); return }
		if len(req.Pin) < 4 || len(req.Pin) > 8 || strings.Trim(req.Pin, "0123456789") != "" { fail(w, 400, "Mã PIN gồm 4 đến 8 chữ số"); return }
		if strings.TrimSpace(req.OwnerName) == "" { req.OwnerName = "Chủ cửa hàng" }
		var uid string
		err := s.db.Do(func(st *State, t *Tx) error {
			if len(st.Users) > 0 { return uerr("Đã thiết lập rồi") }
			u := &User{ID: newID(), Name: strings.TrimSpace(req.OwnerName), Role: "owner", Salt: randHex(8)}
			u.Hash = hashPin(u.Salt, req.Pin)
			st.Users[u.ID] = u
			t.put("user", u.ID, u)
			st.Settings.ShopName = strings.TrimSpace(req.ShopName)
			st.Settings.TermsVersion, st.Settings.TermsAt, st.Settings.TermsBy = TermsVersion, nowMs(), u.Name
			t.put("settings", "", st.Settings)
			uid = u.ID
			return nil
		})
		if err != nil { failErr(w, err); return }
		s.login(w, uid)
		writeJSON(w, map[string]bool{"ok": true})
	})

	m.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Pin string }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		ip := clientIP(r)
		s.sessMu.Lock()
		recent := []time.Time{}
		for _, t := range s.fails[ip] { if time.Since(t) < time.Minute { recent = append(recent, t) } }
		s.fails[ip] = recent
		s.sessMu.Unlock()
		if len(recent) >= 5 { fail(w, 429, "Nhập sai quá nhiều lần. Đợi 1 phút rồi thử lại"); return }
		var uid string
		s.db.Read(func(st *State) {
			for _, u := range st.Users { if hashPin(u.Salt, req.Pin) == u.Hash { uid = u.ID } }
		})
		if uid == "" {
			s.sessMu.Lock(); s.fails[ip] = append(s.fails[ip], time.Now()); s.sessMu.Unlock()
			fail(w, 400, "Mã PIN không đúng")
			return
		}
		s.login(w, uid)
		writeJSON(w, map[string]bool{"ok": true})
	})

	m.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("thx"); err == nil { s.sessMu.Lock(); delete(s.sessions, c.Value); s.sessMu.Unlock() }
		http.SetCookie(w, &http.Cookie{Name: "thx", Value: "", Path: "/", MaxAge: -1})
		writeJSON(w, map[string]bool{"ok": true})
	})

	m.HandleFunc("GET /api/version", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		s.db.mu.Lock(); v := s.db.Version; s.db.mu.Unlock()
		writeJSON(w, map[string]int64{"v": v})
	}))

	m.HandleFunc("GET /api/boot", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		out := map[string]any{}
		s.db.Read(func(st *State) {
			owner := u.seeCost(&st.Settings)
			set := st.Settings
			out["hasKey"] = set.ApiKey != ""
			out["hasPayKeys"] = map[string]bool{"sepay": set.SepayToken != "", "payos": set.PayosClientID != "" && set.PayosAPIKey != "" && set.PayosChecksum != ""}
			set.ApiKey, set.SepayToken, set.PayosClientID, set.PayosAPIKey, set.PayosChecksum = "", "", "", "", ""
			out["settings"] = set
			ps := make([]Product, 0, len(st.Products))
			for _, p := range st.Products {
				cp := *p
				if cp.IsCombo { cp.Stock = comboStock(st, p); cp.Cost = r0(comboCost(st, p)) }
				if !owner { cp.Cost = 0 }
				ps = append(ps, cp)
			}
			out["products"] = ps
			cs := make([]Party, 0, len(st.Customers))
			spend := custSpendAll(st)
			tiers := tiersOf(st)
			for _, c := range st.Customers {
				cp := *c
				cp.Spend = r0(spend[c.ID])
				if n := tierIndex(tiers, spend[c.ID]); n > 0 { cp.Tier = tiers[n-1].Name }
				cs = append(cs, cp)
			}
			out["customers"] = cs
			out["suppliers"] = values(st.Suppliers)
			os := []*Order{}
			for _, o := range st.Orders { if o.Status == "new" { os = append(os, o) } }
			out["orders"] = os
			pop := map[string]int{}
			since := time.Now().AddDate(0, 0, -30).UnixMilli()
			for _, inv := range st.Invoices {
				if inv.Void || inv.Time < since { continue }
				for _, l := range inv.Lines { pop[l.PID]++ }
			}
			out["pop"] = pop
			inNew := 0
			for _, x := range st.InInvs { if x.Status == "new" { inNew++ } }
			out["inNew"] = inNew
			us := []map[string]string{}
			for _, x := range st.Users { us = append(us, map[string]string{"id": x.ID, "name": x.Name, "role": x.Role}) }
			out["users"] = us
			out["version"] = s.db.Version
		})
		out["me"] = map[string]any{"id": u.ID, "name": u.Name, "role": normRole(u.Role), "perms": u.perms()}
		out["lan"] = lanAddrs(s.port)
		out["dataDir"] = s.db.dir
		out["app"], out["appVersion"], out["termsVersion"] = AppName, AppVersion, TermsVersion
		out["today"] = today()
		writeJSON(w, out)
	}))

	// ---- mặt hàng ----
	m.HandleFunc("POST /api/products", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req ProductReq
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		p, err := s.db.SaveProduct(req, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, p)
	}))
	m.HandleFunc("DELETE /api/products/{id}", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		if err := s.db.DeleteProduct(r.PathValue("id")); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("POST /api/products/dispose", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ IDs []string `json:"ids"` }
		readJSON(r, &req)
		c, err := s.db.DisposeExpired(req.IDs, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, c)
	}))

	// ---- khách hàng / NCC ----
	for _, kind := range []string{"cust", "supp"} {
		kind := kind
		path := map[string]string{"cust": "customers", "supp": "suppliers"}[kind]
		m.HandleFunc("POST /api/"+path, s.auth(map[string]string{"cust": "sell|money", "supp": "stock|money"}[kind], func(w http.ResponseWriter, r *http.Request, u *User) {
			var p Party
			if err := readJSON(r, &p); err != nil { failErr(w, err); return }
			if !u.can("manage") { p.Debt = 0 }
			out, err := s.db.SaveParty(kind, p)
			if err != nil { failErr(w, err); return }
			writeJSON(w, out)
		}))
		m.HandleFunc("DELETE /api/"+path+"/{id}", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
			if err := s.db.DeleteParty(kind, r.PathValue("id")); err != nil { failErr(w, err); return }
			writeJSON(w, map[string]bool{"ok": true})
		}))
		m.HandleFunc("GET /api/"+path+"/{id}/history", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
			id := r.PathValue("id")
			items := []map[string]any{}
			s.db.Read(func(st *State) {
				if kind == "cust" {
					for _, inv := range st.Invoices {
						if inv.CustID != id { continue }
						items = append(items, map[string]any{"type": "invoice", "id": inv.ID, "code": inv.Code, "time": inv.Time, "total": inv.Total, "debt": inv.DebtAmt, "void": inv.Void, "lines": inv.Lines})
					}
				} else {
					for _, rc := range st.Receipts {
						if rc.SuppID != id { continue }
						items = append(items, map[string]any{"type": "receipt", "id": rc.ID, "code": rc.Code, "time": rc.Time, "total": rc.Total, "debt": rc.DebtAmt, "void": rc.Void, "lines": rc.Lines})
					}
				}
				for _, e := range st.Cash {
					if e.PartyID != id || e.PartyType != kind { continue }
					items = append(items, map[string]any{"type": "cash", "id": e.ID, "code": e.Code, "time": e.Time, "kind": e.Kind, "amount": e.Amount, "method": e.Method, "cat": e.Cat, "note": e.Note, "void": e.Void, "debtEffect": debtEffect(*e)})
				}
			})
			sort.Slice(items, func(i, j int) bool { return items[i]["time"].(int64) > items[j]["time"].(int64) })
			if len(items) > 300 { items = items[:300] }
			writeJSON(w, items)
		}))
	}

	// ---- bán hàng ----
	m.HandleFunc("POST /api/invoices", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req SaleReq
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		inv, err := s.db.CreateInvoice(req, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, inv)
	}))
	m.HandleFunc("POST /api/invoices/{id}/void", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		if err := s.db.VoidInvoice(r.PathValue("id"), u); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/invoices", s.auth("sell|money|report", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		q := norm(r.URL.Query().Get("q"))
		owner := u.can("cost")
		out := []Invoice{}
		s.db.Read(func(st *State) {
			if st.Settings.StaffSeeCost { owner = true }
			for _, inv := range st.Invoices {
				if !inRange(inv.Time, from, to) { continue }
				if q != "" && !strings.Contains(norm(inv.Code+" "+inv.CustName), q) { continue }
				cp := *inv
				if !owner { cp.Cost = 0; for i := range cp.Lines { cp.Lines[i].Cost = 0 } }
				out = append(out, cp)
			}
		})
		sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
		if len(out) > 3000 { out = out[:3000] }
		writeJSON(w, out)
	}))

	// ---- khuyến mãi, hạng khách ----
	m.HandleFunc("POST /api/promos", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Promos []Promo `json:"promos"`
			Tiers  []Tier  `json:"tiers"`
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		if err := s.db.SavePromos(req.Promos, req.Tiers); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("POST /api/promo/quote", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Lines  []Line `json:"lines"`
			CustID string `json:"custId"`
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		var disc float64
		var hits []PromoHit
		s.db.Read(func(st *State) {
			lines := make([]Line, 0, len(req.Lines))
			for _, l := range req.Lines {
				p := st.Products[l.PID]
				if p == nil || l.Qty <= 0 { continue }
				if l.Ratio <= 0 { l.Ratio = 1 }
				if !u.editPrice(&st.Settings) { l.Price = unitPrice(p, l.Unit, l.Ratio) }
				lines = append(lines, l)
			}
			disc, hits = promoDisc(st, lines, req.CustID, time.Now())
		})
		writeJSON(w, map[string]any{"disc": disc, "hits": hits})
	}))

	// ---- nhập hàng ----
	m.HandleFunc("POST /api/receipts", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req ReceiptReq
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		rc, err := s.db.CreateReceipt(req, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, rc)
	}))
	m.HandleFunc("POST /api/receipts/{id}/void", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		if err := s.db.VoidReceipt(r.PathValue("id")); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/receipts", s.auth("stock|money|report", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		out := []*Receipt{}
		s.db.Read(func(st *State) { for _, x := range st.Receipts { if inRange(x.Time, from, to) { out = append(out, x) } } })
		sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
		writeJSON(w, out)
	}))

	// ---- hóa đơn đầu vào ----
	m.HandleFunc("GET /api/inin", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		type row struct {
			*InInvoice
			Ready bool `json:"ready"`
			OK    int  `json:"ok"`
			N     int  `json:"n"`
		}
		out := []row{}
		s.db.Read(func(st *State) {
			for _, x := range st.InInvs {
				cp := *x
				ok, a, n := x.ready()
				cp.Lines = nil
				out = append(out, row{&cp, ok, a, n})
			}
		})
		sort.Slice(out, func(i, j int) bool {
			if out[i].Date != out[j].Date { return out[i].Date > out[j].Date }
			return out[i].At > out[j].At
		})
		writeJSON(w, out)
	}))
	m.HandleFunc("GET /api/inin/{id}", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		inv, err := s.db.RefreshInInvoice(r.PathValue("id"))
		if err != nil { failErr(w, err); return }
		ok, a, n := inv.ready()
		writeJSON(w, map[string]any{"inv": inv, "ready": ok, "ok": a, "n": n})
	}))
	m.HandleFunc("POST /api/inin/upload", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
		if err := r.ParseMultipartForm(100 << 20); err != nil { fail(w, 400, "File quá lớn hoặc bị lỗi khi gửi"); return }
		var all []*InInvoice
		var errs []string
		for _, fh := range r.MultipartForm.File["files"] {
			f, err := fh.Open()
			if err != nil { continue }
			b, _ := io.ReadAll(f)
			f.Close()
			l, e := ParseInvoiceFiles(fh.Filename, b)
			all = append(all, l...)
			errs = append(errs, e...)
		}
		res := s.db.AddInInvoices(all, "xml")
		res.Errs = append(errs, res.Errs...)
		writeJSON(w, res)
	}))
	m.HandleFunc("POST /api/inin/{id}/approve", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Lines []InLineSel `json:"lines"`
			Pay   string      `json:"pay"`
			Note  string      `json:"note"`
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		rc, err := s.db.ApproveInInvoice(r.PathValue("id"), req.Lines, req.Pay, req.Note, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, rc)
	}))
	m.HandleFunc("POST /api/inin/{id}/status", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Status string }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		if err := s.db.SetInStatus(r.PathValue("id"), req.Status); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	// cổng hoadondientu.gdt.gov.vn
	m.HandleFunc("GET /api/gdt/status", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		writeJSON(w, gdt.Status())
	}))
	m.HandleFunc("GET /api/gdt/captcha", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		k, svg, err := gdt.Captcha()
		if err != nil { fail(w, 502, err.Error()); return }
		writeJSON(w, map[string]string{"key": k, "svg": svg})
	}))
	m.HandleFunc("POST /api/gdt/login", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ User, Pass, Cvalue, Ckey string }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		req.User = strings.TrimSpace(req.User)
		if req.User == "" || req.Pass == "" || strings.TrimSpace(req.Cvalue) == "" { fail(w, 400, "Nhập đủ mã số thuế, mật khẩu và mã captcha"); return }
		if err := gdt.Login(req.User, req.Pass, strings.TrimSpace(req.Cvalue), req.Ckey); err != nil { fail(w, 400, err.Error()); return }
		s.db.Do(func(st *State, t *Tx) error {
			if st.Settings.GdtUser != req.User { st.Settings.GdtUser = req.User; t.put("settings", "", st.Settings) }
			return nil
		})
		writeJSON(w, gdt.Status())
	}))
	m.HandleFunc("POST /api/gdt/logout", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		gdt.Logout()
		writeJSON(w, gdt.Status())
	}))
	m.HandleFunc("POST /api/gdt/fetch", s.auth("tax", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ From, To string }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		from, e1 := time.ParseInLocation("2006-01-02", req.From, time.Local)
		to, e2 := time.ParseInLocation("2006-01-02", req.To, time.Local)
		if e1 != nil || e2 != nil || to.Before(from) { fail(w, 400, "Khoảng ngày không hợp lệ"); return }
		if to.Sub(from) > 366*24*time.Hour { fail(w, 400, "Mỗi lần lấy tối đa 12 tháng"); return }
		if err := gdt.Start(s.db, from, to); err != nil { failErr(w, err); return }
		writeJSON(w, gdt.Status())
	}))

	// ---- kiểm kho ----
	m.HandleFunc("POST /api/counts", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req CountReq
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		c, err := s.db.CreateCount(req, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, c)
	}))
	m.HandleFunc("GET /api/counts", s.auth("stock|report", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		out := []*Count{}
		s.db.Read(func(st *State) { for _, x := range st.Counts { if inRange(x.Time, from, to) { out = append(out, x) } } })
		sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
		writeJSON(w, out)
	}))

	// ---- sổ quỹ ----
	m.HandleFunc("POST /api/cash", s.auth("money", func(w http.ResponseWriter, r *http.Request, u *User) {
		var e CashEntry
		if err := readJSON(r, &e); err != nil { failErr(w, err); return }
		if e.Kind == "adj" && !u.can("manage") { fail(w, 403, "Chỉ chủ cửa hàng được điều chỉnh công nợ"); return }
		out, err := s.db.CreateCash(e, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, out)
	}))
	m.HandleFunc("POST /api/cash/{id}/void", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		if err := s.db.VoidCash(r.PathValue("id")); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/cashbook", s.auth("money|report", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		rows := []map[string]any{}
		s.db.Read(func(st *State) {
			daily := map[string]*[2]float64{}
			for _, inv := range st.Invoices {
				if inv.Void || inv.Src == "import" || !inRange(inv.Time, from, to) { continue }
				d := dayOf(inv.Time)
				if daily[d] == nil { daily[d] = &[2]float64{} }
				daily[d][0] += inv.Cash
				daily[d][1] += inv.Bank
			}
			for d, v := range daily {
				t, _ := time.ParseInLocation("2006-01-02", d, time.Local)
				end := t.Add(23*time.Hour + 59*time.Minute).UnixMilli()
				if v[0] > 0 { rows = append(rows, map[string]any{"auto": true, "time": end, "kind": "thu", "method": "cash", "amount": r0(v[0]), "cat": "Bán hàng", "note": "Tiền mặt bán hàng ngày " + vnDate(d)}) }
				if v[1] > 0 { rows = append(rows, map[string]any{"auto": true, "time": end, "kind": "thu", "method": "bank", "amount": r0(v[1]), "cat": "Bán hàng", "note": "Chuyển khoản bán hàng ngày " + vnDate(d)}) }
			}
			for _, rc := range st.Receipts {
				if rc.Void || rc.Paid <= 0 || !inRange(rc.Time, from, to) { continue }
				rows = append(rows, map[string]any{"auto": true, "time": rc.Time, "kind": "chi", "method": rc.Method, "amount": rc.Paid, "cat": "Nhập hàng", "code": rc.Code, "party": rc.SuppName, "note": "Trả tiền phiếu nhập " + rc.Code})
			}
			for _, e := range st.Cash {
				if e.Kind == "adj" || !inRange(e.Time, from, to) { continue }
				rows = append(rows, map[string]any{"id": e.ID, "time": e.Time, "kind": e.Kind, "method": e.Method, "amount": e.Amount, "cat": e.Cat, "code": e.Code, "party": e.PartyName, "note": e.Note, "staff": e.Staff, "void": e.Void})
			}
		})
		sort.Slice(rows, func(i, j int) bool { return rows[i]["time"].(int64) > rows[j]["time"].(int64) })
		writeJSON(w, rows)
	}))

	// ---- đơn đặt ----
	m.HandleFunc("POST /api/orders", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var o Order
		if err := readJSON(r, &o); err != nil { failErr(w, err); return }
		out, err := s.db.SaveOrder(o, u)
		if err != nil { failErr(w, err); return }
		writeJSON(w, out)
	}))
	m.HandleFunc("POST /api/orders/{id}/status", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Status string }
		readJSON(r, &req)
		if err := s.db.SetOrderStatus(r.PathValue("id"), req.Status); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/orders", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		out := []*Order{}
		s.db.Read(func(st *State) { for _, o := range st.Orders { if inRange(o.Time, from, to) || (o.Due >= from && o.Due <= to) { out = append(out, o) } } })
		sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
		writeJSON(w, out)
	}))
	m.HandleFunc("POST /api/zalo/parse", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Text string `json:"text"`
			AI   bool   `json:"ai"`
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		var ps *parser
		key := ""
		s.db.Read(func(st *State) { ps = newParser(st); key = st.Settings.ApiKey })
		if req.AI {
			if key == "" { fail(w, 400, "Chưa có khóa API trong Cài đặt"); return }
			var prompt string
			s.db.Read(func(st *State) { prompt = ps.aiPrompt(req.Text) })
			txt, err := aiCall(key, prompt)
			if err != nil { failErr(w, err); return }
			var out []Draft
			s.db.Read(func(st *State) { ps = newParser(st); out, err = ps.aiResult(txt) })
			if err != nil { failErr(w, err); return }
			writeJSON(w, out)
			return
		}
		var out []Draft
		s.db.Read(func(st *State) { out = ps.ParseText(req.Text) })
		writeJSON(w, out)
	}))

	// ---- báo cáo ----
	m.HandleFunc("GET /api/report", s.auth("report", func(w http.ResponseWriter, r *http.Request, u *User) {
		from, to := dates(r)
		owner := u.can("cost")
		s.db.Read(func(st *State) { if st.Settings.StaffSeeCost { owner = true } })
		writeJSON(w, s.db.Report(from, to, owner))
	}))
	m.HandleFunc("GET /api/combos", s.auth("manage", func(w http.ResponseWriter, r *http.Request, u *User) {
		days := 90
		if d := int(num(r.URL.Query().Get("days"))); d > 0 { days = d }
		writeJSON(w, s.db.SuggestCombos(days))
	}))

	// ---- nhập dữ liệu ----
	m.HandleFunc("POST /api/import/preview", s.auth("import", func(w http.ResponseWriter, r *http.Request, u *User) {
		r.Body = http.MaxBytesReader(w, r.Body, 60<<20)
		if err := r.ParseMultipartForm(60 << 20); err != nil { fail(w, 400, "File quá lớn hoặc bị lỗi khi gửi"); return }
		f, hdr, err := r.FormFile("file")
		if err != nil { fail(w, 400, "Chưa chọn file"); return }
		defer f.Close()
		b, _ := io.ReadAll(f)
		p, err := NewImport(hdr.Filename, b, r.FormValue("kind"))
		if err != nil { failErr(w, err); return }
		writeJSON(w, p)
	}))
	m.HandleFunc("POST /api/import/reconfigure", s.auth("import", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Token  string `json:"token"`
			Sheet  int    `json:"sheet"`
			Header int    `json:"header"`
			Kind   string `json:"kind"`
		}
		readJSON(r, &req)
		j := getJob(req.Token)
		if j == nil { fail(w, 400, "Phiên nhập file đã hết hạn, chọn lại file"); return }
		writeJSON(w, j.Reconfigure(req.Sheet, req.Header, req.Kind))
	}))
	m.HandleFunc("POST /api/import/commit", s.auth("import", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Token   string         `json:"token"`
			Mapping map[string]int `json:"mapping"`
			Mode    string         `json:"mode"`
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		j := getJob(req.Token)
		if j == nil { fail(w, 400, "Phiên nhập file đã hết hạn, chọn lại file"); return }
		s.db.Backup("-truoc-nhap-file")
		res, err := s.db.CommitImport(j, req.Mapping, req.Mode, u)
		if err != nil { failErr(w, err); return }
		jobsMu.Lock(); delete(jobs, req.Token); jobsMu.Unlock()
		writeJSON(w, res)
	}))

	// ---- xuất Excel ----
	m.HandleFunc("GET /api/export/{what}", s.auth("report|stock|money", func(w http.ResponseWriter, r *http.Request, u *User) {
		what := r.PathValue("what")
		var data []byte
		name := ""
		owner := u.can("cost")
		s.db.Read(func(st *State) {
			if st.Settings.StaffSeeCost { owner = true }
			data, name = exportXLSX(st, what, r, owner)
		})
		if data == nil { fail(w, 404, "Không có mẫu xuất này"); return }
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		w.Write(data)
	}))

	// ---- cài đặt, người dùng, sao lưu ----
	m.HandleFunc("POST /api/settings", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		var in struct {
			Settings
			KeepKey bool `json:"keepKey"`
		}
		if err := readJSON(r, &in); err != nil { failErr(w, err); return }
		err := s.db.Do(func(st *State, t *Tx) error {
			ns := in.Settings
			if in.KeepKey { ns.ApiKey = st.Settings.ApiKey }
			ns.ApiKey = strings.TrimSpace(ns.ApiKey)
			if ns.ExpWarn <= 0 { ns.ExpWarn = 30 }
			if ns.ExpDanger <= 0 { ns.ExpDanger = 7 }
			if ns.Paper == "" { ns.Paper = "80" }
			ns.TermsVersion, ns.TermsAt, ns.TermsBy = st.Settings.TermsVersion, st.Settings.TermsAt, st.Settings.TermsBy
			ns.BackupDir2 = st.Settings.BackupDir2
			o := st.Settings
			ns.PayProvider, ns.SepayToken, ns.PayosClientID, ns.PayosAPIKey, ns.PayosChecksum, ns.DisplayKey = o.PayProvider, o.SepayToken, o.PayosClientID, o.PayosAPIKey, o.PayosChecksum, o.DisplayKey
			ns.GdtUser = o.GdtUser
			ns.UpdateOff = o.UpdateOff
			st.Settings = ns
			t.put("settings", "", st.Settings)
			return nil
		})
		if err != nil { failErr(w, err); return }
		go licRevokeCheck() // báo tên/điện thoại/địa chỉ mới cho Nam Linh Group
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("POST /api/users", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ ID, Name, Role, Pin string }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		err := s.db.Do(func(st *State, t *Tx) error {
			req.Name = strings.TrimSpace(req.Name)
			if req.Name == "" { return uerr("Nhập tên nhân viên") }
			if _, ok := rolePerms[req.Role]; !ok { req.Role = "cashier" }
			x := st.Users[req.ID]
			if x == nil {
				if req.Pin == "" { return uerr("Nhập mã PIN") }
				x = &User{ID: newID()}
			}
			if req.Pin != "" {
				if len(req.Pin) < 4 || len(req.Pin) > 8 || strings.Trim(req.Pin, "0123456789") != "" { return uerr("Mã PIN gồm 4 đến 8 chữ số") }
				for _, o := range st.Users { if o.ID != x.ID && hashPin(o.Salt, req.Pin) == o.Hash { return uerr("Mã PIN này đã có người dùng, chọn mã khác") } }
				x.Salt = randHex(8)
				x.Hash = hashPin(x.Salt, req.Pin)
			}
			if x.Role == "owner" && req.Role != "owner" {
				n := 0
				for _, o := range st.Users { if o.Role == "owner" { n++ } }
				if n <= 1 { return uerr("Cần giữ ít nhất một tài khoản chủ cửa hàng") }
			}
			x.Name, x.Role = req.Name, req.Role
			st.Users[x.ID] = x
			t.put("user", x.ID, x)
			return nil
		})
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("DELETE /api/users/{id}", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		id := r.PathValue("id")
		err := s.db.Do(func(st *State, t *Tx) error {
			x := st.Users[id]
			if x == nil { return uerr("Không tìm thấy") }
			if x.ID == u.ID { return uerr("Không tự xóa tài khoản đang đăng nhập") }
			if x.Role == "owner" {
				n := 0
				for _, o := range st.Users { if o.Role == "owner" { n++ } }
				if n <= 1 { return uerr("Cần giữ ít nhất một tài khoản chủ cửa hàng") }
			}
			delete(st.Users, id)
			t.del("user", id)
			return nil
		})
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/backups", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) { writeJSON(w, s.db.Backups()) }))
	m.HandleFunc("POST /api/backups", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		n, err := s.db.Backup("-thu-cong")
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]string{"name": n})
	}))
	m.HandleFunc("POST /api/backups/restore", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Name string }
		readJSON(r, &req)
		if err := s.db.Restore(req.Name); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/backups/download", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		s.db.mu.Lock()
		b, _ := json.Marshal(s.db.S)
		s.db.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="taphoax-saoluu-`+time.Now().Format("2006-01-02_1504")+`.json"`)
		w.Write(b)
	}))
	m.HandleFunc("POST /api/backups/upload", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
		if err := r.ParseMultipartForm(200 << 20); err != nil { fail(w, 400, "File quá lớn"); return }
		f, hdr, err := r.FormFile("file")
		if err != nil { fail(w, 400, "Chưa chọn file"); return }
		defer f.Close()
		b, _ := io.ReadAll(f)
		if strings.HasSuffix(strings.ToLower(hdr.Filename), ".gz") || (len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b) {
			zr, err := gzip.NewReader(bytes.NewReader(b))
			if err != nil { fail(w, 400, "File sao lưu .gz bị hỏng"); return }
			b, err = io.ReadAll(zr)
			if err != nil { fail(w, 400, "File sao lưu .gz bị hỏng"); return }
		}
		if err := s.db.RestoreJSON(b); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/backups/external", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		writeJSON(w, map[string]any{"drives": listDrives(), "status": extStatus()})
	}))
	m.HandleFunc("POST /api/backups/external", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Dir string }
		readJSON(r, &req)
		req.Dir = strings.TrimSpace(req.Dir)
		if req.Dir != "" {
			if err := checkExtDir(req.Dir); err != nil { failErr(w, err); return }
		}
		err := s.db.Do(func(st *State, t *Tx) error { st.Settings.BackupDir2 = req.Dir; t.put("settings", "", st.Settings); return nil })
		if err != nil { failErr(w, err); return }
		if req.Dir == "" { extMu.Lock(); extStat = ExtStatus{Msg: "Chưa cài đặt"}; extMu.Unlock(); writeJSON(w, extStatus()); return }
		st := s.db.SyncExternal(true)
		if !st.Ok { fail(w, 400, st.Msg); return }
		writeJSON(w, st)
	}))
	m.HandleFunc("POST /api/backups/external/now", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		st := s.db.SyncExternal(true)
		if !st.Ok { fail(w, 400, st.Msg); return }
		writeJSON(w, st)
	}))
	// ---- xác nhận chuyển khoản ----
	m.HandleFunc("POST /api/pay/start", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Amount float64 }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		ps, err := s.db.PayStart(req.Amount)
		if err != nil { failErr(w, err); return }
		writeJSON(w, ps)
	}))
	m.HandleFunc("GET /api/pay/status", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		ps, err := s.db.PayStatus(r.URL.Query().Get("code"))
		if err != nil { failErr(w, err); return }
		writeJSON(w, ps)
	}))
	m.HandleFunc("POST /api/pay/config", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct {
			Provider, SepayToken, PayosClientId, PayosApiKey, PayosChecksum string
			AutoBankDone                                                  bool
		}
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		var c payConf
		s.db.Read(func(st *State) {
			o := st.Settings
			c = payConf{req.Provider, pick(req.SepayToken, o.SepayToken), o.Acct, pick(req.PayosClientId, o.PayosClientID), pick(req.PayosApiKey, o.PayosAPIKey), pick(req.PayosChecksum, o.PayosChecksum)}
		})
		switch c.Provider {
		case "":
		case "sepay":
			if c.Acct == "" { fail(w, 400, "Nhập số tài khoản ngân hàng ở mục Nhận chuyển khoản trước"); return }
			if c.SepayToken == "" { fail(w, 400, "Nhập API token SePay"); return }
			if err := sepayTest(c.SepayToken, c.Acct); err != nil { failErr(w, err); return }
		case "payos":
			if c.PayosID == "" || c.PayosKey == "" || c.PayosSum == "" { fail(w, 400, "Nhập đủ Client ID, API Key và Checksum Key của payOS"); return }
			if err := payosTest(c); err != nil { failErr(w, err); return }
		default:
			fail(w, 400, "Cách xác nhận không hợp lệ"); return
		}
		err := s.db.Do(func(st *State, t *Tx) error {
			ss := &st.Settings
			ss.PayProvider, ss.SepayToken, ss.PayosClientID, ss.PayosAPIKey, ss.PayosChecksum, ss.AutoBankDone = c.Provider, c.SepayToken, c.PayosID, c.PayosKey, c.PayosSum, req.AutoBankDone
			t.put("settings", "", st.Settings)
			return nil
		})
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))

	// ---- màn hình khách ----
	m.HandleFunc("POST /api/display", s.auth("sell", func(w http.ResponseWriter, r *http.Request, u *User) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
		if err != nil || !json.Valid(b) { fail(w, 400, "Dữ liệu không hợp lệ"); return }
		dispHub.set(b)
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/display", func(w http.ResponseWriter, r *http.Request) {
		k := r.URL.Query().Get("k")
		ok := false
		shop := ""
		s.db.Read(func(st *State) { ok = k != "" && k == st.Settings.DisplayKey; shop = st.Settings.ShopName })
		if !ok { fail(w, 403, "Đường dẫn màn hình khách không đúng. Mở lại từ Cài đặt → Màn hình khách"); return }
		since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		seq, data := dispHub.get(since, 25*time.Second)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, `{"seq":%d,"shop":%q,"data":%s}`, seq, shop, data)
	})
	m.HandleFunc("POST /api/display/key", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		k := randHex(6)
		s.db.Do(func(st *State, t *Tx) error { st.Settings.DisplayKey = k; t.put("settings", "", st.Settings); return nil })
		writeJSON(w, map[string]string{"key": k})
	}))

	m.HandleFunc("POST /api/shutdown", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		writeJSON(w, map[string]bool{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); s.Quit() }()
	}))
	m.HandleFunc("POST /api/open-folder", s.auth("admin", func(w http.ResponseWriter, r *http.Request, u *User) {
		openFolder(s.db.dir)
		writeJSON(w, map[string]bool{"ok": true})
	}))
}

func values[T any](m map[string]*T) []*T {
	out := make([]*T, 0, len(m))
	for _, v := range m { out = append(out, v) }
	return out
}

func dates(r *http.Request) (string, string) {
	q := r.URL.Query()
	from, to := parseDate(q.Get("from")), parseDate(q.Get("to"))
	if from == "" { from = today() }
	if to == "" { to = from }
	if to < from { from, to = to, from }
	return from, to
}

func exportXLSX(st *State, what string, r *http.Request, owner bool) ([]byte, string) {
	stamp := time.Now().Format("2006-01-02")
	switch what {
	case "products", "template":
		cols := []XCol{{"Mã hàng", 14, false}, {"Tên hàng", 38, false}, {"Nhóm hàng", 18, false}, {"ĐVT", 8, false}, {"Mã vạch", 16, false}, {"Giá bán", 12, true}, {"Giá vốn", 12, true}, {"Tồn kho", 10, true}, {"Tồn nhỏ nhất", 12, true}, {"Vị trí", 12, false}, {"HSD gần nhất", 13, false}, {"Giá trị tồn", 14, true}, {"Đơn vị lớn", 12, false}, {"Quy đổi", 9, true}, {"Giá đơn vị lớn", 14, true}}
		rows := [][]any{}
		if what == "template" {
			rows = append(rows, []any{"SP000001", "Mì Hảo Hảo tôm chua cay", "Mì - Bún khô", "gói", "8934563138165", 4500.0, 3800.0, 120.0, 30.0, "Kệ 1", "", nil, "thùng", 30.0, 125000.0})
			return WriteXLSX("Hàng hóa", cols, rows), "mau-nhap-hang-hoa.xlsx"
		}
		ps := values(st.Products)
		sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
		for _, p := range ps {
			if p.IsCombo { continue }
			exp := ""
			for _, l := range p.Lots { if exp == "" || l.Exp < exp { exp = l.Exp } }
			cost := any(p.Cost)
			val := any(r0(maxf(p.Stock, 0) * p.Cost))
			if !owner { cost, val = nil, nil }
			row := []any{p.Code, p.Name, p.Group, p.Unit, p.Barcode, p.Price, cost, p.Stock, p.Min, p.Location, vnDateOr(exp), val}
			if len(p.Units) > 0 { row = append(row, p.Units[0].Name, p.Units[0].Ratio, p.Units[0].Price) }
			rows = append(rows, row)
		}
		return WriteXLSX("Hàng hóa", cols, rows), "hang-hoa-" + stamp + ".xlsx"
	case "customers", "suppliers":
		m := st.Customers
		title := "Khách hàng"
		if what == "suppliers" { m, title = st.Suppliers, "Nhà cung cấp" }
		cols := []XCol{{"Mã", 12, false}, {"Tên " + strings.ToLower(title), 30, false}, {"Điện thoại", 14, false}, {"Địa chỉ", 30, false}, {"Công nợ hiện tại", 16, true}, {"Ghi chú", 24, false}}
		ps := values(m)
		sort.Slice(ps, func(i, j int) bool { return ps[i].Debt > ps[j].Debt })
		rows := [][]any{}
		for _, p := range ps { rows = append(rows, []any{p.Code, p.Name, p.Phone, p.Addr, p.Debt, p.Note}) }
		return WriteXLSX(title, cols, rows), map[string]string{"customers": "khach-hang-", "suppliers": "nha-cung-cap-"}[what] + stamp + ".xlsx"
	case "invoices":
		from, to := dates(r)
		cols := []XCol{{"Mã hóa đơn", 12, false}, {"Thời gian", 17, false}, {"Khách hàng", 22, false}, {"Mã hàng", 12, false}, {"Tên hàng", 34, false}, {"ĐVT", 8, false}, {"Số lượng", 10, true}, {"Đơn giá", 12, true}, {"Thành tiền", 13, true}, {"Giảm giá HĐ", 12, true}, {"Tổng hóa đơn", 13, true}, {"Tiền mặt", 12, true}, {"Chuyển khoản", 13, true}, {"Ghi nợ", 12, true}, {"Thu ngân", 14, false}, {"Trạng thái", 12, false}}
		if owner { cols = append(cols, XCol{"Giá vốn", 12, true}, XCol{"Lãi gộp dòng", 13, true}) }
		invs := []*Invoice{}
		for _, inv := range st.Invoices { if inRange(inv.Time, from, to) { invs = append(invs, inv) } }
		sort.Slice(invs, func(i, j int) bool { return invs[i].Time < invs[j].Time })
		rows := [][]any{}
		for _, inv := range invs {
			stt := "Hoàn thành"
			if inv.Void { stt = "Đã hủy" }
			for _, l := range inv.Lines {
				row := []any{inv.Code, time.UnixMilli(inv.Time).Format("02/01/2006 15:04"), inv.CustName, l.Code, l.Name, l.Unit, l.Qty, l.Price, r0(l.Qty * l.Price), inv.Disc, inv.Total, inv.Cash, inv.Bank, inv.DebtAmt, inv.Staff, stt}
				if owner { row = append(row, l.Cost, r0(l.Qty*(l.Price-l.Cost))) }
				rows = append(rows, row)
			}
		}
		return WriteXLSX("Hóa đơn", cols, rows), "hoa-don-" + from + "_" + to + ".xlsx"
	}
	return nil, ""
}

func maxf(a, b float64) float64 { if a > b { return a }; return b }
func vnDateOr(d string) string { if d == "" { return "" }; return vnDate(d) }

var _ = os.Getpid

func pick(n, o string) string { if strings.TrimSpace(n) != "" { return strings.TrimSpace(n) }; return o }
