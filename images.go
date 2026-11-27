package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// Ảnh mặt hàng: ảnh đã thu nhỏ ở phía trình duyệt, lưu riêng khỏi danh sách hàng để danh sách luôn nhẹ.
// Ảnh nằm trong dữ liệu nên được sao lưu cùng; không bao giờ in lên hóa đơn.

type PImg struct {
	Mime string `json:"mime"`
	Data string `json:"data"` // base64
	Time int64  `json:"time"`
}

const maxImgBytes = 300 << 10

func decodeImgURI(uri string) (mime, b64 string, err error) {
	for _, m := range []string{"image/jpeg", "image/png", "image/webp"} {
		pre := "data:" + m + ";base64,"
		if strings.HasPrefix(uri, pre) {
			b64 = uri[len(pre):]
			raw, e := base64.StdEncoding.DecodeString(b64)
			if e != nil { return "", "", uerr("Ảnh không hợp lệ") }
			if len(raw) > maxImgBytes { return "", "", uerr("Ảnh quá lớn (tối đa 300 KB sau khi thu nhỏ)") }
			if len(raw) < 16 { return "", "", uerr("Ảnh không hợp lệ") }
			return m, b64, nil
		}
	}
	return "", "", uerr("Chỉ nhận ảnh JPG, PNG hoặc WEBP")
}

type ImgItem struct {
	PID  string `json:"pid"`
	Data string `json:"data"`
}

func (db *DB) SetImages(items []ImgItem) (int, error) {
	n := 0
	err := db.Do(func(s *State, t *Tx) error {
		type pend struct{ p *Product; im *PImg }
		var list []pend
		for _, it := range items {
			p := s.Products[it.PID]
			if p == nil { return uerr("Không tìm thấy mặt hàng") }
			m, b, err := decodeImgURI(it.Data)
			if err != nil { return err }
			list = append(list, pend{p, &PImg{Mime: m, Data: b, Time: nowMs()}})
		}
		for _, x := range list {
			s.Images[x.p.ID] = x.im
			x.p.Img = x.im.Time
			t.put("pimg", x.p.ID, x.im)
			t.put("product", x.p.ID, x.p)
			n++
		}
		return nil
	})
	return n, err
}

func (db *DB) DeleteImage(pid string) error {
	return db.Do(func(s *State, t *Tx) error {
		p := s.Products[pid]
		if p == nil { return uerr("Không tìm thấy mặt hàng") }
		if s.Images[pid] != nil { delete(s.Images, pid); t.del("pimg", pid) }
		if p.Img != 0 { p.Img = 0; t.put("product", p.ID, p) }
		return nil
	})
}

func (s *Server) imageRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/products/images", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		var req struct{ Items []ImgItem `json:"items"` }
		if err := readJSON(r, &req); err != nil { failErr(w, err); return }
		if len(req.Items) == 0 || len(req.Items) > 40 { fail(w, 400, "Mỗi lần gửi tối đa 40 ảnh"); return }
		n, err := s.db.SetImages(req.Items)
		if err != nil { failErr(w, err); return }
		writeJSON(w, map[string]int{"saved": n})
	}))
	m.HandleFunc("POST /api/products/{id}/image/delete", s.auth("stock", func(w http.ResponseWriter, r *http.Request, u *User) {
		if err := s.db.DeleteImage(r.PathValue("id")); err != nil { failErr(w, err); return }
		writeJSON(w, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/img/{id}", s.auth("", func(w http.ResponseWriter, r *http.Request, u *User) {
		var im *PImg
		s.db.Read(func(st *State) { im = st.Images[r.PathValue("id")] })
		if im == nil { http.NotFound(w, r); return }
		raw, err := base64.StdEncoding.DecodeString(im.Data)
		if err != nil { http.NotFound(w, r); return }
		w.Header().Set("Content-Type", im.Mime)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(raw)
	}))
}
