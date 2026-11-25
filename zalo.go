package main

// Tách đơn đặt hàng từ tin nhắn nhóm Zalo. Mặc định dùng quy tắc (không cần
// internet); nếu chủ tiệm nhập khóa API Claude thì có thể tách bằng AI.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type DraftLine struct {
	PID   string     `json:"pid"`
	Name  string     `json:"name"`
	Unit  string     `json:"unit"`
	Ratio float64    `json:"ratio"`
	Qty   float64    `json:"qty"`
	Price float64    `json:"price"`
	Text  string     `json:"text"`
	Cands []DraftCnd `json:"cands"`
	Sure  bool       `json:"sure"`
}

type DraftCnd struct {
	PID  string `json:"pid"`
	Name string `json:"name"`
}

type Draft struct {
	CustName string      `json:"custName"`
	CustID   string      `json:"custId"`
	Phone    string      `json:"phone"`
	Due      string      `json:"due"`
	DueTime  string      `json:"dueTime"`
	Lines    []DraftLine `json:"lines"`
	Raw      string      `json:"raw"`
	Unknown  []string    `json:"unknown"`
}

var honorifics = map[string]bool{"anh": true, "chị": true, "chi": true, "cô": true, "chú": true, "bác": true, "em": true, "ông": true, "bà": true, "thím": true, "dì": true, "cậu": true, "mợ": true, "a": true, "c": true, "e": true}

var numWords = map[string]float64{"một": 1, "hai": 2, "ba": 3, "bốn": 4, "tư": 4, "năm": 5, "sáu": 6, "bảy": 7, "bẩy": 7, "tám": 8, "chín": 9, "mười": 10, "nửa": 0.5, "chục": 10}

var unitWords = map[string]string{
	"thung": "thùng", "ket": "thùng", "loc": "lốc", "goi": "gói", "chai": "chai", "lon": "lon", "hop": "hộp", "vi": "vỉ",
	"kg": "kg", "can": "kg", "ky": "kg", "ki": "kg", "bao": "bao", "cay": "cây", "bich": "bịch", "qua": "quả", "trai": "quả",
	"cuon": "cuộn", "tui": "túi", "hu": "hũ", "lo": "lọ", "bo": "bó", "cai": "cái", "chiec": "cái", "tep": "tép", "bin": "bình", "binh": "bình",
	"lit": "lít", "l": "lít", "g": "g", "gam": "g", "combo": "combo", "phan": "phần", "suat": "suất", "doi": "đôi", "cap": "cặp",
}

var fillers = map[string]bool{
	"lay": true, "dat": true, "mua": true, "cho": true, "minh": true, "toi": true, "tui": false, "nhe": true, "nha": true, "nhen": true, "a": true, "ah": true,
	"giup": true, "voi": true, "them": true, "ship": true, "giao": true, "mai": true, "nay": true, "chieu": true, "sang": true, "trua": true,
	"luc": true, "gio": true, "h": true, "di": true, "ve": true, "nua": false, "con": true, "co": true, "khong": true, "ko": true, "k": true, "nhu": true,
	"moi": true, "lan": true, "e": true, "em": true, "anh": true, "chi": true, "oi": true, "ak": true, "nhi": true, "de": true, "san": true, "dum": true,
	"hen": true, "ngay": true, "kia": true, "mot": false, "it": true, "o": true, "cua": true, "gui": true, "order": true, "chot": true, "don": true, "loai": true,
}

var (
	reTS      = regexp.MustCompile(`^\[?\s*(\d{1,2}[:h]\d{2}(\s*(AM|PM|SA|CH))?)?\s*,?\s*(\d{1,2}/\d{1,2}(/\d{2,4})?)?\s*\]?\s*`)
	reSender  = regexp.MustCompile(`^(\pL[^:]{1,39}):\s*(.+)$`)
	reDate    = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})(/(\d{2,4}))?\b`)
	reHour    = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(h|giờ|gio)\s*(\d{0,2})\b`)
	reNumTok  = regexp.MustCompile(`^\d+([.,]\d+)?$`)
	reNumUnit = regexp.MustCompile(`^(\d+(?:[.,]\d+)?)([a-zA-Zđ]+)$`)
)

type pIndex struct {
	p      *Product
	toks   map[string]bool
	weight float64
}

type parser struct {
	s     *State
	prods []pIndex
	idf   map[string]float64
	pop   map[string]int
}

func newParser(s *State) *parser {
	ps := &parser{s: s, idf: map[string]float64{}, pop: map[string]int{}}
	df := map[string]int{}
	since := time.Now().AddDate(0, 0, -60).UnixMilli()
	for _, inv := range s.Invoices {
		if inv.Void || inv.Time < since { continue }
		for _, l := range inv.Lines { ps.pop[l.PID]++ }
	}
	for _, p := range s.Products {
		if p.Hidden { continue }
		tk := map[string]bool{}
		words := tokens(p.Name)
		for _, w := range words { tk[w] = true }
		for i := 0; i+1 < len(words); i++ {
			tk[string(words[i][0])+string(words[i+1][0])] = true
			if i+2 < len(words) { tk[string(words[i][0])+string(words[i+1][0])+string(words[i+2][0])] = true }
		}
		for _, a := range strings.Split(p.Alias, ",") {
			for _, w := range tokens(a) { tk[w] = true }
			if k := key(a); k != "" { tk[k] = true }
		}
		for _, u := range p.Units { tk[norm(u.Name)] = false }
		for w, ok := range tk { if ok { df[w]++ } }
		ps.prods = append(ps.prods, pIndex{p: p, toks: tk})
	}
	n := float64(len(ps.prods)) + 1
	for w, c := range df { ps.idf[w] = math.Log(n/float64(c)) + 0.3 }
	return ps
}

func (ps *parser) match(words []string) []DraftCnd {
	type sc struct{ p *Product; v float64 }
	total := 0.0
	for _, w := range words { total += ps.idfOf(w) }
	if total == 0 { return nil }
	out := []sc{}
	for _, pi := range ps.prods {
		hit := 0.0
		for _, w := range words {
			if pi.toks[w] {
				hit += ps.idfOf(w)
			} else if len(w) >= 3 {
				for t, ok := range pi.toks { if ok && len(t) > len(w) && strings.HasPrefix(t, w) { hit += ps.idfOf(w) * 0.7; break } }
			}
		}
		if hit == 0 { continue }
		cov := hit / total
		if cov < 0.45 { continue }
		out = append(out, sc{pi.p, cov*10 + math.Log1p(float64(ps.pop[pi.p.ID]))*0.15})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].v > out[j].v })
	res := []DraftCnd{}
	for i := 0; i < len(out) && i < 4; i++ { res = append(res, DraftCnd{PID: out[i].p.ID, Name: out[i].p.Name}) }
	return res
}

func (ps *parser) idfOf(w string) float64 {
	if v, ok := ps.idf[w]; ok { return v }
	return 0.25 // từ không có trong tên hàng nào: trọng số thấp
}

type tok struct{ raw, low, n string }

func splitTokens(s string) []tok {
	out := []tok{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune("!?()[]\"“”'…", r) }) {
		f = strings.Trim(f, ".:")
		if f == "" { continue }
		out = append(out, tok{raw: f, low: strings.ToLower(f), n: norm(f)})
	}
	return out
}

func isCap(s string) bool {
	for _, r := range s { return unicode.IsUpper(r) }
	return false
}

func (ps *parser) ParseText(text string) []Draft {
	text = strings.ReplaceAll(text, "\r", "")
	type msg struct{ sender, body string }
	msgs := []msg{}
	sender := ""
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(reTS.ReplaceAllString(strings.TrimSpace(ln), ""))
		if ln == "" { continue }
		if m := reSender.FindStringSubmatch(ln); m != nil && len(strings.Fields(m[1])) <= 6 {
			sender = strings.TrimSpace(m[1])
			msgs = append(msgs, msg{sender, m[2]})
			continue
		}
		f := strings.Fields(ln)
		if len(f) <= 4 && !strings.ContainsAny(ln, "0123456789,") && allCap(f) && len(ps.match(tokens(ln))) == 0 {
			sender = ln // dòng chỉ có tên người gửi (kiểu chép từ Zalo PC)
			continue
		}
		msgs = append(msgs, msg{sender, ln})
	}
	drafts := []*Draft{}
	byCust := map[string]*Draft{}
	for _, m := range msgs {
		cust, custID := ps.findCustomer(m.body)
		if cust == "" { cust, custID = ps.customerFromName(m.sender) }
		if cust == "" { cust = "Khách lẻ" }
		lines, unknown := ps.parseItems(m.body)
		if len(lines) == 0 { continue }
		k := norm(cust)
		d := byCust[k]
		if d == nil {
			d = &Draft{CustName: cust, CustID: custID}
			if custID != "" { if c := ps.s.Customers[custID]; c != nil { d.Phone = c.Phone } }
			byCust[k] = d
			drafts = append(drafts, d)
		}
		due, dueT := parseDue(m.body)
		if due != "" { d.Due = due }
		if dueT != "" { d.DueTime = dueT }
		d.Lines = append(d.Lines, lines...)
		d.Unknown = append(d.Unknown, unknown...)
		if d.Raw != "" { d.Raw += "\n" }
		if m.sender != "" { d.Raw += m.sender + ": " }
		d.Raw += m.body
	}
	out := []Draft{}
	for _, d := range drafts {
		if d.Due == "" { d.Due = today() }
		out = append(out, *d)
	}
	return out
}

func allCap(f []string) bool {
	for _, w := range f { if !isCap(w) { return false } }
	return len(f) > 0
}

func (ps *parser) customerFromName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" { return "", "" }
	n := norm(name)
	for _, c := range ps.s.Customers { if norm(c.Name) == n { return c.Name, c.ID } }
	for _, c := range ps.s.Customers { if strings.Contains(norm(c.Name), n) || strings.Contains(n, norm(c.Name)) { return c.Name, c.ID } }
	return name, ""
}

func (ps *parser) findCustomer(body string) (string, string) {
	tks := splitTokens(body)
	for i, t := range tks {
		if !honorifics[t.low] || i+1 >= len(tks) { continue }
		if (t.low == "a" || t.low == "c" || t.low == "e") && !isCap(tks[i+1].raw) { continue }
		nx := tks[i+1]
		if reNumTok.MatchString(nx.raw) || fillers[nx.n] && !isCap(nx.raw) { continue }
		name := []string{nx.raw}
		if i+2 < len(tks) && isCap(tks[i+2].raw) && !reNumTok.MatchString(tks[i+2].raw) { name = append(name, tks[i+2].raw) }
		hon := map[string]string{"a": "Anh", "c": "Chị", "e": "Em"}[t.low]
		if hon == "" { hon = strings.ToUpper(t.raw[:1]) + t.raw[1:] }
		full := hon + " " + strings.Join(name, " ")
		nfull := norm(full)
		nname := norm(strings.Join(name, " "))
		var best *Party
		for _, c := range ps.s.Customers {
			cn := norm(c.Name)
			if strings.HasPrefix(cn, nfull) { return c.Name, c.ID }
			if best == nil && (strings.Contains(" "+cn+" ", " "+nname+" ")) && strings.Contains(cn, norm(hon)) { best = c }
		}
		if best != nil { return best.Name, best.ID }
		for i := range name { name[i] = strings.ToUpper(name[i][:1]) + name[i][1:] }
		return hon + " " + strings.Join(name, " "), ""
	}
	return "", ""
}

func parseDue(body string) (string, string) {
	n := " " + norm(body) + " "
	due := ""
	switch {
	case strings.Contains(n, " ngay kia ") || strings.Contains(n, " mot ngay ") || strings.Contains(n, " ngay mot "):
		due = addDays(today(), 2)
	case strings.Contains(n, " mai ") || strings.Contains(n, " sang mai") || strings.Contains(n, " ngay mai"):
		due = addDays(today(), 1)
	case strings.Contains(n, " hom nay ") || strings.Contains(n, " chieu nay ") || strings.Contains(n, " toi nay ") || strings.Contains(n, " sang nay ") || strings.Contains(n, " trua nay "):
		due = today()
	}
	if m := reDate.FindStringSubmatch(body); m != nil {
		d, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		y := time.Now().Year()
		if m[4] != "" { y, _ = strconv.Atoi(m[4]); if y < 100 { y += 2000 } }
		if d >= 1 && d <= 31 && mo >= 1 && mo <= 12 { due = fmt.Sprintf("%04d-%02d-%02d", y, mo, d) }
	}
	tm := ""
	if m := reHour.FindStringSubmatch(body); m != nil {
		h, _ := strconv.Atoi(m[1])
		if h <= 24 {
			if (strings.Contains(n, "chieu") || strings.Contains(n, "toi")) && h < 12 { h += 12 }
			mm := 0
			if m[3] != "" { mm, _ = strconv.Atoi(m[3]) }
			tm = fmt.Sprintf("%02d:%02d", h%24, mm)
		}
	}
	return due, tm
}

func (ps *parser) parseItems(body string) ([]DraftLine, []string) {
	clean := reHour.ReplaceAllString(body, " ")
	clean = reDate.ReplaceAllString(clean, " ")
	segs := regexp.MustCompile(`(?i)[,;+\n]|\s(và|với|vs|thêm|-)\s`).Split(clean, -1)
	lines, unknown := []DraftLine{}, []string{}
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" { continue }
		tks := splitTokens(seg)
		// tách "2thùng" → "2", "thùng"
		ex := []tok{}
		for _, t := range tks {
			if m := reNumUnit.FindStringSubmatch(t.raw); m != nil && unitWords[norm(m[2])] != "" {
				ex = append(ex, tok{m[1], m[1], m[1]}, tok{m[2], strings.ToLower(m[2]), norm(m[2])})
				continue
			}
			ex = append(ex, t)
		}
		tks = ex
		qty, unit := 0.0, ""
		words := []string{}
		skipName := 0
		// ưu tiên con số đứng ngay trước đơn vị (vd "xóm 3 ... 1 vỉ trứng" → lấy 1)
		qIdx := -1
		for i := 0; i+1 < len(tks); i++ {
			if (reNumTok.MatchString(tks[i].raw) || numWords[tks[i].low] > 0) && unitWords[tks[i+1].n] != "" { qIdx = i; break }
		}
		for i := 0; i < len(tks); i++ {
			t := tks[i]
			if skipName > 0 { skipName--; continue }
			if qIdx >= 0 && i != qIdx && reNumTok.MatchString(t.raw) { continue }
			if honorifics[t.low] && i+1 < len(tks) && isCap(tks[i+1].raw) {
				skipName = 1
				if i+2 < len(tks) && isCap(tks[i+2].raw) { skipName = 2 }
				continue
			}
			if reNumTok.MatchString(t.raw) && qty == 0 {
				qty = num(strings.Replace(t.raw, ".", ",", 1))
				if i+1 < len(tks) && tks[i+1].n == "ruoi" { qty += 0.5; i++ }
				if i+1 < len(tks) && unitWords[tks[i+1].n] != "" { unit = unitWords[tks[i+1].n]; i++ }
				continue
			}
			if v, ok := numWords[t.low]; ok && qty == 0 && (i+1 < len(tks) && (unitWords[tks[i+1].n] != "" || t.low == "chục")) {
				qty = v
				if i+1 < len(tks) && tks[i+1].n == "ruoi" { qty += 0.5; i++ }
				if i+1 < len(tks) && unitWords[tks[i+1].n] != "" { unit = unitWords[tks[i+1].n]; i++ }
				continue
			}
			if unitWords[t.n] != "" && unit == "" && qty > 0 { unit = unitWords[t.n]; continue }
			if fillers[t.n] || honorifics[t.low] { continue }
			words = append(words, t.n)
		}
		if len(words) == 0 { continue }
		cands := ps.match(words)
		if len(cands) == 0 {
			if qty > 0 { unknown = append(unknown, seg) }
			continue
		}
		if qty == 0 { qty = 1 }
		p := ps.s.Products[cands[0].PID]
		l := DraftLine{PID: p.ID, Name: p.Name, Unit: p.Unit, Ratio: 1, Qty: qty, Text: seg, Cands: cands, Sure: len(cands) == 1}
		if unit == "chục" && norm(p.Unit) != "chuc" { l.Qty, unit = r3(qty*10), "" }
		if unit != "" && norm(unit) != norm(p.Unit) {
			for _, c := range p.Units {
				if norm(c.Name) == norm(unit) || (unit == "thùng" && norm(c.Name) == "ket") { l.Unit, l.Ratio = c.Name, c.Ratio; break }
			}
		}
		l.Price = unitPrice(p, l.Unit, l.Ratio)
		lines = append(lines, l)
	}
	return lines, unknown
}

// ---------- tách bằng AI (tùy chọn) ----------

// aiPrompt dựng câu lệnh cho AI (gọi khi đang giữ khóa dữ liệu).
func (ps *parser) aiPrompt(text string) string {
	type pp struct{ id, line string; pop int }
	list := []pp{}
	for _, pi := range ps.prods {
		u := pi.p.Unit
		for _, c := range pi.p.Units { u += fmt.Sprintf(", %s=%g %s", c.Name, c.Ratio, pi.p.Unit) }
		al := ""
		if pi.p.Alias != "" { al = " (gọi tắt: " + pi.p.Alias + ")" }
		list = append(list, pp{pi.p.ID, fmt.Sprintf("%s | %s%s | đơn vị: %s", pi.p.ID, pi.p.Name, al, u), ps.pop[pi.p.ID]})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].pop > list[j].pop })
	if len(list) > 600 { list = list[:600] }
	var cat strings.Builder
	for _, x := range list { cat.WriteString(x.line + "\n") }
	var custs strings.Builder
	n := 0
	for _, c := range ps.s.Customers { if n < 300 { custs.WriteString(c.ID + " | " + c.Name + "\n"); n++ } }
	prompt := "Bạn là trợ lý của một cửa hàng tạp hóa ở Việt Nam. Dưới đây là tin nhắn khách đặt hàng chép từ nhóm Zalo. " +
		"Hãy tách thành các đơn đặt hàng, mỗi khách một đơn. Chỉ dùng mặt hàng có trong DANH MỤC (ghi đúng id). " +
		"Hôm nay là " + today() + ".\n\nDANH MỤC (id | tên | đơn vị):\n" + cat.String() +
		"\nKHÁCH QUEN (id | tên):\n" + custs.String() +
		"\nTIN NHẮN:\n" + text +
		"\n\nTrả về DUY NHẤT một JSON dạng {\"orders\":[{\"customer\":\"tên khách\",\"customer_id\":\"id nếu là khách quen, không thì rỗng\",\"due\":\"YYYY-MM-DD\",\"due_time\":\"HH:MM hoặc rỗng\",\"items\":[{\"product_id\":\"id\",\"qty\":số,\"unit\":\"tên đơn vị trong danh mục\",\"text\":\"đoạn tin gốc\"}],\"unknown\":[\"đoạn không tìm thấy hàng\"]}]}"
	return prompt
}

// aiCall gửi câu lệnh tới Claude API (không giữ khóa dữ liệu trong lúc chờ).
func aiCall(apiKey, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": "claude-haiku-5-5", "max_tokens": 4000,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	})
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	cl := &http.Client{Timeout: 90 * time.Second}
	resp, err := cl.Do(req)
	if err != nil { return "", uerr("Không kết nối được tới máy chủ AI (kiểm tra internet)") }
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 { return "", uerr("Khóa API không đúng. Kiểm tra lại trong Cài đặt") }
	if resp.StatusCode != 200 { return "", uerr("Máy chủ AI báo lỗi %d. Thử lại sau hoặc dùng nút Tách đơn thường", resp.StatusCode) }
	var ar struct {
		Content []struct{ Type, Text string } `json:"content"`
	}
	json.Unmarshal(rb, &ar)
	txt := ""
	for _, c := range ar.Content { if c.Type == "text" { txt += c.Text } }
	return txt, nil
}

// aiResult chuyển kết quả AI thành đơn nháp (gọi khi đang giữ khóa dữ liệu).
func (ps *parser) aiResult(txt string) ([]Draft, error) {
	i, j := strings.Index(txt, "{"), strings.LastIndex(txt, "}")
	if i < 0 || j <= i { return nil, uerr("AI trả kết quả không đọc được, thử lại") }
	var out struct {
		Orders []struct {
			Customer string `json:"customer"`
			CustID   string `json:"customer_id"`
			Due      string `json:"due"`
			DueTime  string `json:"due_time"`
			Items    []struct {
				PID  string  `json:"product_id"`
				Qty  float64 `json:"qty"`
				Unit string  `json:"unit"`
				Text string  `json:"text"`
			} `json:"items"`
			Unknown []string `json:"unknown"`
		} `json:"orders"`
	}
	if err := json.Unmarshal([]byte(txt[i:j+1]), &out); err != nil { return nil, uerr("AI trả kết quả không đọc được, thử lại") }
	res := []Draft{}
	for _, o := range out.Orders {
		d := Draft{CustName: o.Customer, Due: o.Due, DueTime: o.DueTime, Unknown: o.Unknown}
		if c := ps.s.Customers[o.CustID]; c != nil { d.CustID, d.CustName, d.Phone = c.ID, c.Name, c.Phone }
		if parseDate(d.Due) == "" { d.Due = today() }
		for _, it := range o.Items {
			p := ps.s.Products[it.PID]
			if p == nil || it.Qty <= 0 { continue }
			l := DraftLine{PID: p.ID, Name: p.Name, Unit: p.Unit, Ratio: 1, Qty: r3(it.Qty), Text: it.Text, Sure: true, Cands: []DraftCnd{{p.ID, p.Name}}}
			for _, c := range p.Units { if norm(c.Name) == norm(it.Unit) { l.Unit, l.Ratio = c.Name, c.Ratio } }
			l.Price = unitPrice(p, l.Unit, l.Ratio)
			d.Lines = append(d.Lines, l)
		}
		if len(d.Lines) > 0 { res = append(res, d) }
	}
	return res, nil
}
