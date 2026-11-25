package main

// Hóa đơn đầu vào: đọc hóa đơn điện tử (XML chuẩn Thông tư 78, hoặc dữ liệu từ cổng
// hoadondientu.gdt.gov.vn), khớp từng dòng với mặt hàng trong tiệm, rồi tạo phiếu nhập khi chủ tiệm duyệt.

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type InLine struct {
	Kind   int        `json:"kind"` // 1 hàng hóa, 2 khuyến mại, 3 chiết khấu, 4 ghi chú
	Code   string     `json:"code,omitempty"`
	Name   string     `json:"name"`
	Unit   string     `json:"unit"`
	Qty    float64    `json:"qty"`
	Price  float64    `json:"price"`
	Amount float64    `json:"amount"`
	Rate   float64    `json:"rate"`
	RateTx string     `json:"rateTx,omitempty"`
	Cost   float64    `json:"cost"` // giá vốn 1 đơn vị trên hóa đơn: gồm thuế, đã trừ chiết khấu
	PID    string     `json:"pid,omitempty"`
	PUnit  string     `json:"punit,omitempty"` // tên đơn vị trong tiệm
	Ratio  float64    `json:"ratio,omitempty"` // 1 đơn vị hóa đơn = Ratio đơn vị cơ bản
	Skip   bool       `json:"skip,omitempty"`
	New    bool       `json:"new,omitempty"` // tạo mặt hàng mới khi duyệt
	Auto   string     `json:"auto,omitempty"` // map | code | guess | user
	Cands  []DraftCnd `json:"cands,omitempty"`
}

type InInvoice struct {
	ID          string   `json:"id"`
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Form        string   `json:"form"`   // ký hiệu mẫu số (1 = GTGT, 2 = bán hàng)
	Serial      string   `json:"serial"` // ký hiệu hóa đơn
	No          string   `json:"no"`
	Date        string   `json:"date"`
	SellerMST   string   `json:"sellerMst"`
	SellerName  string   `json:"sellerName"`
	SellerAddr  string   `json:"sellerAddr"`
	BuyerMST    string   `json:"buyerMst"`
	BuyerName   string   `json:"buyerName"`
	Pre         float64  `json:"pre"`
	VAT         float64  `json:"vat"`
	Total       float64  `json:"total"`
	Pay         string   `json:"pay"`
	Lines       []InLine `json:"lines"`
	Src         string   `json:"src"`    // xml | portal | email
	Status      string   `json:"status"` // new | done | expense | ignored
	SuppID      string   `json:"suppId,omitempty"`
	ReceiptID   string   `json:"receiptId,omitempty"`
	ReceiptCode string   `json:"receiptCode,omitempty"`
	At          int64    `json:"at"`
	Done        int64    `json:"doneAt,omitempty"`
}

// InMap ghi nhớ: tên hàng trên hóa đơn của một NCC ↔ mặt hàng trong tiệm.
type InMap struct {
	PID   string  `json:"pid"`
	Unit  string  `json:"unit"`
	Ratio float64 `json:"ratio"`
	At    int64   `json:"at"`
}

func mapKey(mst, name, unit string) string { return strings.TrimSpace(mst) + "|" + key(name) + "|" + key(unit) }
func invKey(mst, form, serial, no string) string {
	n := strings.TrimLeft(strings.TrimSpace(no), "0")
	return strings.TrimSpace(mst) + "|" + strings.TrimSpace(form) + "|" + strings.ToUpper(strings.TrimSpace(serial)) + "|" + n
}

// ---------- đọc XML ----------

type xLine struct {
	TChat, STT, MHHDVu, THHDVu, DVTinh, SLuong, DGia, ThTien, TSuat string
}

type xHDon struct {
	DLHDon struct {
		TTChung struct{ THDon, KHMSHDon, KHHDon, SHDon, NLap, HTTToan string }
		NDHDon  struct {
			NBan    struct{ Ten, MST, DChi string }
			NMua    struct{ Ten, MST, DChi, HVTNMHang string }
			DSHHDVu struct{ HHDVu []xLine }
			TToan   struct{ TgTCThue, TgTThue, TgTTTBSo, TTCKTMai string }
		}
	}
	MCCQT string
}

func xnum(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseRate: "8%", "10%", "KCT", "KKKNT", 0.08, "KHAC:3.5%" → tỷ lệ.
func parseRate(s string) float64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "KHAC:")
	if s == "" || strings.HasPrefix(s, "K") { return 0 }
	pct := strings.HasSuffix(s, "%")
	f := xnum(strings.TrimSuffix(s, "%"))
	if pct || f > 1 { f /= 100 }
	if f < 0 || f > 0.5 { return 0 }
	return f
}

func ParseEInvoiceXML(b []byte) (*InInvoice, error) {
	d := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})))
	d.CharsetReader = func(cs string, in io.Reader) (io.Reader, error) { return in, nil }
	d.Strict = false
	for {
		tk, err := d.Token()
		if err != nil { return nil, uerr("Không phải file hóa đơn điện tử (không thấy thẻ HDon)") }
		se, ok := tk.(xml.StartElement)
		if !ok || se.Name.Local != "HDon" { continue }
		var h xHDon
		if err := d.DecodeElement(&h, &se); err != nil { return nil, uerr("File hóa đơn bị lỗi: %v", err) }
		t, n := h.DLHDon.TTChung, h.DLHDon.NDHDon
		inv := &InInvoice{
			Title: strings.TrimSpace(t.THDon), Form: strings.TrimSpace(t.KHMSHDon), Serial: strings.TrimSpace(t.KHHDon), No: strings.TrimSpace(t.SHDon),
			Date: strings.TrimSpace(t.NLap), Pay: strings.TrimSpace(t.HTTToan),
			SellerMST: strings.TrimSpace(n.NBan.MST), SellerName: strings.TrimSpace(n.NBan.Ten), SellerAddr: strings.TrimSpace(n.NBan.DChi),
			BuyerMST: strings.TrimSpace(n.NMua.MST), BuyerName: strings.TrimSpace(firstNonEmpty(n.NMua.Ten, n.NMua.HVTNMHang)),
			Pre: xnum(n.TToan.TgTCThue), VAT: xnum(n.TToan.TgTThue), Total: xnum(n.TToan.TgTTTBSo),
		}
		for _, l := range n.DSHHDVu.HHDVu {
			k, _ := strconv.Atoi(strings.TrimSpace(l.TChat))
			if k == 0 { k = 1 }
			inv.Lines = append(inv.Lines, InLine{Kind: k, Code: strings.TrimSpace(l.MHHDVu), Name: cleanSpaces(l.THHDVu), Unit: strings.TrimSpace(l.DVTinh),
				Qty: xnum(l.SLuong), Price: xnum(l.DGia), Amount: xnum(l.ThTien), Rate: parseRate(l.TSuat), RateTx: strings.TrimSpace(l.TSuat)})
		}
		if len(inv.Date) > 10 { inv.Date = inv.Date[:10] }
		return inv, inv.check()
	}
}

func firstNonEmpty(a ...string) string {
	for _, s := range a { if strings.TrimSpace(s) != "" { return s } }
	return ""
}

var reSpaces = regexp.MustCompile(`\s+`)

func cleanSpaces(s string) string { return strings.TrimSpace(reSpaces.ReplaceAllString(s, " ")) }

func (inv *InInvoice) check() error {
	if inv.SellerMST == "" || inv.No == "" { return uerr("Hóa đơn thiếu mã số thuế người bán hoặc số hóa đơn") }
	inv.Key = invKey(inv.SellerMST, inv.Form, inv.Serial, inv.No)
	inv.computeCosts()
	return nil
}

// computeCosts tính giá vốn từng dòng: cộng thuế GTGT (hộ kinh doanh không được khấu trừ),
// trừ chiết khấu thương mại, rồi cân cho khớp tổng tiền thanh toán của hóa đơn.
func (inv *InInvoice) computeCosts() {
	pre, disc, pre0 := 0.0, 0.0, 0.0
	for i := range inv.Lines {
		l := &inv.Lines[i]
		if l.Amount == 0 && l.Qty > 0 && l.Price > 0 && l.Kind == 1 { l.Amount = r0(l.Qty * l.Price) }
		switch l.Kind {
		case 1:
			pre += l.Amount * (1 + l.Rate)
			pre0 += l.Amount
		case 3:
			disc += math.Abs(l.Amount) * (1 + l.Rate)
		}
	}
	target := pre - disc
	if inv.Total > 0 && pre > 0 && math.Abs(inv.Total-target) < pre*0.5 { target = inv.Total }
	if inv.Total == 0 { inv.Total = r0(target) }
	if inv.Pre == 0 { inv.Pre = r0(pre0) }
	scale := 1.0
	if pre > 0 { scale = target / pre }
	for i := range inv.Lines {
		l := &inv.Lines[i]
		l.Cost = 0
		if l.Kind == 1 && l.Qty > 0 { l.Cost = math.Round(l.Amount*(1+l.Rate)*scale/l.Qty*100) / 100 }
	}
}

// ParseInvoiceFiles nhận .xml hoặc .zip (cổng thuế tải về dạng zip), trả các hóa đơn đọc được.
func ParseInvoiceFiles(name string, data []byte) ([]*InInvoice, []string) {
	var out []*InInvoice
	var errs []string
	if bytes.HasPrefix(data, []byte("PK")) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil { return nil, []string{name + ": file zip bị lỗi"} }
		for _, f := range zr.File {
			low := strings.ToLower(f.Name)
			if f.UncompressedSize64 > 20<<20 { continue }
			if !strings.HasSuffix(low, ".xml") && !strings.HasSuffix(low, ".zip") { continue }
			rc, err := f.Open()
			if err != nil { continue }
			b, _ := io.ReadAll(rc)
			rc.Close()
			o, e := ParseInvoiceFiles(name+"/"+path.Base(f.Name), b)
			out = append(out, o...)
			errs = append(errs, e...)
		}
		if len(out) == 0 && len(errs) == 0 { errs = append(errs, name+": trong file zip không có hóa đơn XML") }
		return out, errs
	}
	inv, err := ParseEInvoiceXML(data)
	if err != nil { return nil, []string{name + ": " + err.Error()} }
	return []*InInvoice{inv}, nil
}

// ---------- khớp mặt hàng ----------

var reRatio = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:thung|th|t|ket|k|loc|l|hop|h|bich|goi|x|bo|vi|kien|bao)\s*[x/*]?\s*(\d{1,3})\b`),
	regexp.MustCompile(`(?i)\b(\d{1,3})\s*(?:lon|chai|goi|hop|cai|vi|tui|bich|hu|lo|cuon|qua|banh|thanh|ly|coc|tuyp|chiec)\b`),
}

// guessRatio đoán 1 thùng/lốc trên hóa đơn có bao nhiêu đơn vị lẻ, dựa vào tên hàng.
func guessRatio(name string) float64 {
	n := norm(name)
	for _, re := range reRatio {
		if m := re.FindStringSubmatch(n); m != nil {
			if v, _ := strconv.Atoi(m[1]); v >= 2 && v <= 200 { return float64(v) }
		}
	}
	return 0
}

// unitFor xác định đơn vị trong tiệm tương ứng với đơn vị trên hóa đơn.
func unitFor(p *Product, invUnit, name string) (string, float64, bool) {
	ku := key(invUnit)
	if ku == "" || ku == key(p.Unit) { return p.Unit, 1, true }
	for _, u := range p.Units { if key(u.Name) == ku { return u.Name, u.Ratio, true } }
	if r := guessRatio(name); r > 0 { return strings.ToLower(strings.TrimSpace(invUnit)), r, false }
	return p.Unit, 1, false
}

type invMatcher struct {
	ps     *parser
	byCode map[string]*Product
	ptot   map[string]float64
}

func newInvMatcher(s *State) *invMatcher {
	m := &invMatcher{ps: newParser(s), byCode: map[string]*Product{}, ptot: map[string]float64{}}
	for _, p := range s.Products {
		if p.Hidden || p.IsCombo { continue }
		if p.Barcode != "" { m.byCode[p.Barcode] = p }
		for _, u := range p.Units { if u.Barcode != "" { m.byCode[u.Barcode] = p } }
	}
	for _, pi := range m.ps.prods {
		t := 0.0
		for _, w := range tokens(pi.p.Name) { t += m.ps.idfOf(w) }
		m.ptot[pi.p.ID] = t
	}
	return m
}

// cands chấm điểm theo mức độ tên mặt hàng trong tiệm nằm trong tên trên hóa đơn
// (tên hóa đơn thường dài: "NƯỚC GIẢI KHÁT COCA-COLA LON 320ML THÙNG 24").
func (m *invMatcher) cands(name string) ([]DraftCnd, float64) {
	words := tokens(name)
	set := map[string]bool{}
	for _, w := range words { set[w] = true }
	joined := strings.Join(words, "")
	type sc struct {
		p *Product
		v float64
	}
	var out []sc
	for _, pi := range m.ps.prods {
		if pi.p.IsCombo { continue }
		tot := m.ptot[pi.p.ID]
		if tot == 0 { continue }
		hit := 0.0
		for _, w := range tokens(pi.p.Name) {
			if set[w] {
				hit += m.ps.idfOf(w)
			} else if len(w) >= 3 && strings.Contains(joined, w) {
				hit += m.ps.idfOf(w) * 0.8
			}
		}
		cov := hit / tot
		if cov < 0.55 || hit < 0.8 { continue }
		out = append(out, sc{pi.p, cov + math.Min(hit, 6)*0.02})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].v > out[j].v })
	res := []DraftCnd{}
	best := 0.0
	for i := 0; i < len(out) && i < 5; i++ {
		res = append(res, DraftCnd{PID: out[i].p.ID, Name: out[i].p.Name})
		if i == 0 { best = out[i].v }
	}
	return res, best
}

// matchLines điền gợi ý cho các dòng chưa được người dùng chọn.
func matchLines(s *State, inv *InInvoice, m *invMatcher) {
	for i := range inv.Lines {
		l := &inv.Lines[i]
		if l.Kind != 1 && l.Kind != 2 { continue }
		if l.Skip || l.New || l.Auto == "user" {
			if l.PID != "" && s.Products[l.PID] == nil { l.PID, l.Auto = "", "" }
			continue
		}
		l.PID, l.PUnit, l.Ratio, l.Auto, l.Cands = "", "", 0, "", nil
		if mp := s.InMap[mapKey(inv.SellerMST, l.Name, l.Unit)]; mp != nil && s.Products[mp.PID] != nil {
			l.PID, l.PUnit, l.Ratio, l.Auto = mp.PID, mp.Unit, mp.Ratio, "map"
			continue
		}
		if p := m.byCode[l.Code]; l.Code != "" && p != nil {
			l.PID, l.Auto = p.ID, "code"
			l.PUnit, l.Ratio, _ = unitFor(p, l.Unit, l.Name)
			continue
		}
		c, _ := m.cands(l.Name)
		l.Cands = c
		if len(c) > 0 {
			p := s.Products[c[0].PID]
			l.PID, l.Auto = p.ID, "guess"
			l.PUnit, l.Ratio, _ = unitFor(p, l.Unit, l.Name)
		}
	}
}

// ready: mọi dòng hàng đã được chọn chắc chắn (nhớ từ trước, trùng mã vạch, hoặc người dùng chọn/bỏ qua/tạo mới).
func (inv *InInvoice) ready() (bool, int, int) {
	n, ok := 0, 0
	for _, l := range inv.Lines {
		if l.Kind != 1 && l.Kind != 2 { continue }
		n++
		if l.Skip || l.New || (l.PID != "" && (l.Auto == "map" || l.Auto == "code" || l.Auto == "user")) { ok++ }
	}
	return n > 0 && ok == n, ok, n
}

var reService = regexp.MustCompile(`(?i)\b(tien dien|tien nuoc|dien sinh hoat|dien nang|cuoc|phi |phi$|dich vu|thue (mat bang|nha|kho|cua hang)|internet|vien thong|truyen hinh|bao hiem|van chuyen|sua chua)`)

// isService: hóa đơn toàn dịch vụ (điện, nước, cước, thuê...) → chi phí, không nhập kho.
func (inv *InInvoice) isService() bool {
	n := 0
	for _, l := range inv.Lines {
		if l.Kind != 1 { continue }
		u := key(l.Unit)
		if !(reService.MatchString(norm(l.Name)) || u == "kwh" || u == "m3" || u == "thang" || u == "lan" || u == "chuyen") { return false }
		n++
	}
	return n > 0
}

// ---------- lưu ----------

type InAddResult struct {
	Added []string `json:"added"`
	Dup   int      `json:"dup"`
	Errs  []string `json:"errs"`
}

func (db *DB) AddInInvoices(list []*InInvoice, src string) InAddResult {
	res := InAddResult{}
	db.Do(func(s *State, t *Tx) error {
		have := map[string]bool{}
		for _, x := range s.InInvs { have[x.Key] = true }
		m := newInvMatcher(s)
		for _, inv := range list {
			if inv.Key == "" { if err := inv.check(); err != nil { res.Errs = append(res.Errs, err.Error()); continue } }
			if have[inv.Key] { res.Dup++; continue }
			have[inv.Key] = true
			inv.ID, inv.Src, inv.Status, inv.At = newID(), src, "new", nowMs()
			for _, sp := range s.Suppliers { if sp.TaxCode != "" && sp.TaxCode == inv.SellerMST { inv.SuppID = sp.ID; break } }
			// Hóa đơn không có dòng hàng hóa (điện, nước, cước mạng...) để ở mục chi phí, không tạo phiếu nhập.
			if _, _, n := inv.ready(); n == 0 || inv.isService() { inv.Status = "expense" }
			matchLines(s, inv, m)
			s.InInvs[inv.ID] = inv
			t.put("inInvoice", inv.ID, inv)
			res.Added = append(res.Added, inv.ID)
		}
		return nil
	})
	return res
}

func (db *DB) RefreshInInvoice(id string) (*InInvoice, error) {
	var out *InInvoice
	err := db.Do(func(s *State, t *Tx) error {
		inv := s.InInvs[id]
		if inv == nil { return uerr("Không tìm thấy hóa đơn") }
		if inv.Status == "new" { matchLines(s, inv, newInvMatcher(s)) }
		cp := *inv
		cp.Lines = append([]InLine(nil), inv.Lines...)
		out = &cp
		return nil
	})
	return out, err
}

type InLineSel struct {
	I     int     `json:"i"`
	PID   string  `json:"pid"`
	PUnit string  `json:"punit"`
	Ratio float64 `json:"ratio"`
	Skip  bool    `json:"skip"`
	New   bool    `json:"new"`
}

func (db *DB) SetInStatus(id, st string) error {
	return db.Do(func(s *State, t *Tx) error {
		inv := s.InInvs[id]
		if inv == nil { return uerr("Không tìm thấy hóa đơn") }
		if inv.Status == "done" { return uerr("Hóa đơn đã nhập kho (phiếu %s). Hủy phiếu nhập trước nếu muốn đổi", inv.ReceiptCode) }
		if st != "new" && st != "expense" && st != "ignored" { return uerr("Trạng thái không hợp lệ") }
		inv.Status = st
		t.put("inInvoice", inv.ID, inv)
		return nil
	})
}

// ApproveInInvoice: áp lựa chọn của người dùng, tạo mặt hàng mới nếu cần, tạo NCC, tạo phiếu nhập, ghi nhớ cách khớp.
var approveMu sync.Mutex

func (db *DB) ApproveInInvoice(id string, sels []InLineSel, pay, note string, u *User) (*Receipt, error) {
	approveMu.Lock() // bấm duyệt hai lần liền không tạo hai phiếu
	defer approveMu.Unlock()
	var inv InInvoice
	err := db.Do(func(s *State, t *Tx) error {
		x := s.InInvs[id]
		if x == nil { return uerr("Không tìm thấy hóa đơn") }
		if x.Status == "done" { return uerr("Hóa đơn này đã nhập kho (phiếu %s)", x.ReceiptCode) }
		for _, sl := range sels {
			if sl.I < 0 || sl.I >= len(x.Lines) { continue }
			l := &x.Lines[sl.I]
			l.Skip, l.New = sl.Skip, sl.New && !sl.Skip
			if l.Skip || l.New { l.PID = ""; continue }
			if sl.PID != "" {
				p := s.Products[sl.PID]
				if p == nil || p.IsCombo { return uerr("Dòng %d: mặt hàng không hợp lệ", sl.I+1) }
				l.PID, l.Auto = p.ID, "user"
				l.PUnit, l.Ratio = strings.TrimSpace(sl.PUnit), sl.Ratio
				if l.Ratio <= 0 { l.Ratio = 1 }
				if l.PUnit == "" { l.PUnit = p.Unit }
			}
		}
		for i, l := range x.Lines {
			if (l.Kind == 1 || l.Kind == 2) && !l.Skip && !l.New && l.PID == "" {
				return uerr("Dòng %d \"%s\" chưa chọn mặt hàng. Chọn mặt hàng, bấm \"Tạo mới\" hoặc \"Bỏ qua\"", i+1, l.Name)
			}
		}
		t.put("inInvoice", x.ID, x)
		inv = *x
		inv.Lines = append([]InLine(nil), x.Lines...)
		return nil
	})
	if err != nil { return nil, err }
	var margin float64
	db.Read(func(s *State) { margin = s.Settings.MinMargin })
	if margin <= 0 { margin = 10 }
	// mặt hàng mới
	for i := range inv.Lines {
		l := &inv.Lines[i]
		if !l.New || l.Skip { continue }
		unit := strings.ToLower(strings.TrimSpace(l.Unit))
		if unit == "" { unit = "cái" }
		price := math.Ceil(l.Cost*(1+margin/100)/500) * 500
		p, err := db.SaveProduct(ProductReq{Product: Product{Name: l.Name, Unit: unit, Cost: r0(l.Cost), Price: price}}, u)
		if err != nil { return nil, fmt.Errorf("tạo mặt hàng \"%s\": %w", l.Name, err) }
		l.PID, l.PUnit, l.Ratio, l.Auto = p.ID, p.Unit, 1, "user"
	}
	// thêm đơn vị quy đổi mới (vd "thùng" = 24) vào mặt hàng để lần sau bán/nhập theo thùng
	for _, l := range inv.Lines {
		if l.PID == "" || l.Skip || l.Ratio <= 1 { continue }
		db.Do(func(s *State, t *Tx) error {
			p := s.Products[l.PID]
			if p == nil || key(l.PUnit) == key(p.Unit) { return nil }
			for _, cu := range p.Units { if key(cu.Name) == key(l.PUnit) { return nil } }
			p.Units = append(p.Units, UnitConv{Name: l.PUnit, Ratio: l.Ratio, Price: r0(p.Price * l.Ratio)})
			p.Updated = nowMs()
			t.put("product", p.ID, p)
			return nil
		})
	}
	// nhà cung cấp
	suppID := inv.SuppID
	db.Read(func(s *State) {
		if suppID != "" && s.Suppliers[suppID] != nil { return }
		suppID = ""
		for _, sp := range s.Suppliers {
			if (sp.TaxCode != "" && sp.TaxCode == inv.SellerMST) || key(sp.Name) == key(inv.SellerName) { suppID = sp.ID; break }
		}
	})
	if suppID == "" {
		sp, err := db.SaveParty("supp", Party{Name: firstNonEmpty(inv.SellerName, "MST "+inv.SellerMST), Addr: inv.SellerAddr, TaxCode: inv.SellerMST, Note: "Tạo từ hóa đơn đầu vào"})
		if err != nil { return nil, err }
		suppID = sp.ID
	} else {
		db.Do(func(s *State, t *Tx) error {
			sp := s.Suppliers[suppID]
			if sp != nil && sp.TaxCode == "" { sp.TaxCode = inv.SellerMST; t.put("supplier", sp.ID, sp) }
			return nil
		})
	}
	// phiếu nhập
	req := ReceiptReq{CID: "in-" + newID(), SuppID: suppID, Method: "cash"}
	total := 0.0
	for _, l := range inv.Lines {
		if l.PID == "" || l.Skip || l.Qty <= 0 { continue }
		ratio := l.Ratio
		if ratio <= 0 { ratio = 1 }
		req.Lines = append(req.Lines, Line{PID: l.PID, Unit: l.PUnit, Ratio: ratio, Qty: l.Qty, Price: r0(l.Cost)})
		total += l.Qty * r0(l.Cost)
	}
	// giá vốn làm tròn từng dòng nên tổng có thể lệch vài đồng so với hóa đơn: ghi phần dư vào chiết khấu
	if d := total - inv.Total; inv.Total > 0 && d > 0 && d < 1000 { req.Disc = r0(d); total -= d }
	if len(req.Lines) == 0 { return nil, uerr("Không có dòng nào để nhập kho. Nếu đây là hóa đơn chi phí, bấm \"Ghi là chi phí\"") }
	switch pay {
	case "debt":
		req.Paid = 0
	case "bank":
		req.Paid, req.Method = r0(total), "bank"
	default:
		req.Paid = r0(total)
	}
	req.Note = strings.TrimSpace(fmt.Sprintf("Theo hóa đơn %s%s số %s ngày %s của %s. %s", inv.Form, inv.Serial, inv.No, fmtDateVN(inv.Date), inv.SellerName, note))
	rc, err := db.CreateReceipt(req, u)
	if err != nil { return nil, err }
	db.Do(func(s *State, t *Tx) error {
		x := s.InInvs[id]
		if x == nil { return nil }
		x.Lines = inv.Lines
		x.Status, x.SuppID, x.ReceiptID, x.ReceiptCode, x.Done = "done", suppID, rc.ID, rc.Code, nowMs()
		t.put("inInvoice", x.ID, x)
		for _, l := range x.Lines {
			if l.PID == "" || l.Skip { continue }
			k := mapKey(x.SellerMST, l.Name, l.Unit)
			mp := &InMap{PID: l.PID, Unit: l.PUnit, Ratio: l.Ratio, At: nowMs()}
			s.InMap[k] = mp
			t.put("inMap", k, mp)
		}
		return nil
	})
	return rc, nil
}

func fmtDateVN(d string) string {
	if t, err := time.Parse("2006-01-02", d); err == nil { return t.Format("02/01/2006") }
	return d
}
