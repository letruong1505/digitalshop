package main

// Nhập dữ liệu từ file xuất của KiotViet, MISA (SME/AMIS, eShop), Sổ Tạp Hóa V1.0
// hoặc file Excel tự soạn. Tự dò dòng tiêu đề, tự ghép cột theo tên gọi quen thuộc,
// người dùng có thể sửa lại cách ghép trước khi nhập.

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type Field struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Need  bool     `json:"need"`
	Syn   []string `json:"-"`
}

var importKinds = map[string][]Field{
	"products": {
		{Key: "code", Label: "Mã hàng", Syn: []string{"Mã hàng", "Mã SKU", "Mã hàng hóa", "Mã sản phẩm", "Mã SP", "Mã VTHH", "SKU", "Mã"}},
		{Key: "name", Label: "Tên hàng", Need: true, Syn: []string{"Tên hàng", "Tên hàng hóa", "Tên sản phẩm", "Tên VTHH", "Tên vật tư hàng hóa", "Mặt hàng", "Sản phẩm", "Tên"}},
		{Key: "group", Label: "Nhóm hàng", Syn: []string{"Nhóm hàng(3 Cấp)", "Nhóm hàng (3 cấp)", "Nhóm hàng", "Nhóm hàng hóa", "Nhóm VTHH", "Nhóm", "Danh mục", "Loại sản phẩm"}},
		{Key: "unit", Label: "Đơn vị tính", Syn: []string{"ĐVT", "Đơn vị tính", "Đơn vị tính chính", "Đơn vị", "Đơn vị cơ bản"}},
		{Key: "barcode", Label: "Mã vạch", Syn: []string{"Mã vạch", "Barcode", "Mã code"}},
		{Key: "price", Label: "Giá bán", Syn: []string{"Giá bán", "Giá bán lẻ", "Đơn giá bán", "Đơn giá bán 1", "Đơn giá bán cố định", "Giá niêm yết"}},
		{Key: "cost", Label: "Giá vốn", Syn: []string{"Giá vốn", "Giá nhập", "Giá mua", "Giá nhập cuối", "Đơn giá mua gần nhất", "Đơn giá mua cố định", "Giá vốn trung bình"}},
		{Key: "stock", Label: "Tồn kho", Syn: []string{"Tồn kho", "Số lượng tồn", "SL tồn", "Tồn", "Tồn cuối kỳ", "Số lượng"}},
		{Key: "value", Label: "Giá trị tồn", Syn: []string{"Giá trị tồn", "Giá trị tồn kho", "Giá trị tồn cuối kỳ"}},
		{Key: "min", Label: "Tồn tối thiểu", Syn: []string{"Tồn nhỏ nhất", "Số lượng tồn tối thiểu", "Tồn tối thiểu", "Định mức tồn tối thiểu"}},
		{Key: "baseCode", Label: "Mã ĐVT cơ bản", Syn: []string{"Mã ĐVT Cơ bản", "Mã đơn vị cơ bản", "Mã hàng cơ bản"}},
		{Key: "ratio", Label: "Quy đổi", Syn: []string{"Quy đổi", "Tỷ lệ quy đổi", "Giá trị quy đổi"}},
		{Key: "location", Label: "Vị trí", Syn: []string{"Vị trí", "Vị trí kệ", "Vị trí lưu kho"}},
		{Key: "exp", Label: "Hạn sử dụng", Syn: []string{"HSD", "Hạn sử dụng", "Ngày hết hạn", "Hạn dùng"}},
		{Key: "type", Label: "Loại hàng", Syn: []string{"Loại hàng", "Tính chất", "Loại"}},
		{Key: "active", Label: "Đang kinh doanh", Syn: []string{"Đang kinh doanh", "Trạng thái", "Ngừng kinh doanh"}},
	},
	"customers": {
		{Key: "code", Label: "Mã khách hàng", Syn: []string{"Mã khách hàng", "Mã KH", "Mã đối tượng", "Mã"}},
		{Key: "name", Label: "Tên khách hàng", Need: true, Syn: []string{"Tên khách hàng", "Tên khách", "Tên KH", "Khách hàng", "Tên đối tượng", "Họ tên", "Họ và tên", "Tên"}},
		{Key: "phone", Label: "Điện thoại", Syn: []string{"Điện thoại", "Số điện thoại", "SĐT", "Di động", "ĐT"}},
		{Key: "addr", Label: "Địa chỉ", Syn: []string{"Địa chỉ"}},
		{Key: "debt", Label: "Nợ hiện tại", Syn: []string{"Nợ cần thu hiện tại", "Nợ cần thu", "Nợ hiện tại", "Công nợ", "Số dư nợ", "Dư nợ", "Dư Nợ cuối kỳ", "Số còn phải thu", "Nợ"}},
		{Key: "note", Label: "Ghi chú", Syn: []string{"Ghi chú", "Mô tả"}},
	},
	"suppliers": {
		{Key: "code", Label: "Mã nhà cung cấp", Syn: []string{"Mã nhà cung cấp", "Mã NCC", "Mã đối tượng", "Mã"}},
		{Key: "name", Label: "Tên nhà cung cấp", Need: true, Syn: []string{"Tên nhà cung cấp", "Tên NCC", "Nhà cung cấp", "Tên đối tượng", "Tên"}},
		{Key: "phone", Label: "Điện thoại", Syn: []string{"Điện thoại", "Số điện thoại", "SĐT"}},
		{Key: "addr", Label: "Địa chỉ", Syn: []string{"Địa chỉ"}},
		{Key: "debt", Label: "Nợ cần trả", Syn: []string{"Nợ cần trả hiện tại", "Nợ cần trả", "Nợ hiện tại", "Công nợ", "Số dư có", "Dư Có cuối kỳ", "Số còn phải trả", "Nợ"}},
		{Key: "note", Label: "Ghi chú", Syn: []string{"Ghi chú"}},
	},
	"invoices": {
		{Key: "invCode", Label: "Mã hóa đơn", Need: true, Syn: []string{"Mã hóa đơn", "Số hóa đơn", "Mã chứng từ", "Số chứng từ", "Mã đơn hàng", "Mã HĐ"}},
		{Key: "time", Label: "Thời gian", Syn: []string{"Thời gian", "Thời gian tạo", "Ngày bán", "Ngày hóa đơn", "Ngày chứng từ", "Ngày hạch toán", "Ngày"}},
		{Key: "custName", Label: "Khách hàng", Syn: []string{"Tên khách hàng", "Khách hàng", "Tên đối tượng"}},
		{Key: "pcode", Label: "Mã hàng", Syn: []string{"Mã hàng", "Mã SKU", "Mã VTHH", "Mã hàng hóa"}},
		{Key: "pname", Label: "Tên hàng", Need: true, Syn: []string{"Tên hàng", "Tên hàng hóa", "Tên VTHH", "Diễn giải", "Tên sản phẩm"}},
		{Key: "unit", Label: "ĐVT", Syn: []string{"ĐVT", "Đơn vị tính"}},
		{Key: "qty", Label: "Số lượng", Need: true, Syn: []string{"Số lượng", "SL", "Số lượng bán"}},
		{Key: "price", Label: "Đơn giá", Syn: []string{"Đơn giá", "Giá bán", "Đơn giá sau CK", "Giá"}},
		{Key: "amount", Label: "Thành tiền", Syn: []string{"Thành tiền", "Tiền hàng", "Doanh thu"}},
		{Key: "status", Label: "Trạng thái", Syn: []string{"Trạng thái"}},
	},
}

var kindLabels = map[string]string{"products": "Hàng hóa", "customers": "Khách hàng", "suppliers": "Nhà cung cấp", "invoices": "Lịch sử hóa đơn bán"}

type ImportJob struct {
	Token   string
	Kind    string
	Sheets  []Sheet
	Sheet   int
	Header  int
	Created time.Time
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*ImportJob{}
)

type Preview struct {
	Token    string            `json:"token"`
	FileName string            `json:"fileName"`
	Source   string            `json:"source"`
	Kind     string            `json:"kind"`
	KindName string            `json:"kindName"`
	Sheets   []string          `json:"sheets"`
	Sheet    int               `json:"sheet"`
	Header   int               `json:"header"`
	Columns  []string          `json:"columns"`
	Fields   []Field           `json:"fields"`
	Mapping  map[string]int    `json:"mapping"`
	Sample   [][]string        `json:"sample"`
	Rows     int               `json:"rows"`
	Stats    map[string]int    `json:"stats"`
	Notes    []string          `json:"notes"`
}

func matchField(fields []Field, h string) (string, int) {
	k := key(h)
	if k == "" { return "", 0 }
	for _, f := range fields {
		for i, s := range f.Syn {
			if key(s) == k { return f.Key, 100 - i }
		}
	}
	return "", 0
}

func scoreHeader(row []string, fields []Field) int {
	seen := map[string]bool{}
	sc := 0
	for _, c := range row {
		if f, _ := matchField(fields, cell(c)); f != "" && !seen[f] { seen[f] = true; sc++ }
	}
	return sc
}

func autoMap(cols []string, fields []Field) map[string]int {
	type cand struct{ f string; col, pr int }
	cs := []cand{}
	for i, c := range cols {
		if f, pr := matchField(fields, c); f != "" { cs = append(cs, cand{f, i, pr}) }
	}
	sort.SliceStable(cs, func(a, b int) bool { return cs[a].pr > cs[b].pr })
	m := map[string]int{}
	used := map[int]bool{}
	for _, c := range cs {
		if _, ok := m[c.f]; ok || used[c.col] { continue }
		m[c.f], used[c.col] = c.col, true
	}
	return m
}

func detectSource(cols []string) string {
	ks := map[string]bool{}
	for _, c := range cols { ks[key(c)] = true }
	switch {
	case ks[key("Nhóm hàng(3 Cấp)")] || ks[key("Mã ĐVT Cơ bản")] || ks[key("KH đặt")] || ks[key("Nợ cần thu hiện tại")] || ks[key("Nợ cần trả hiện tại")] || ks[key("Được bán trực tiếp")] || ks[key("Khách cần trả")] || ks[key("Khách đã trả")] || ks[key("Mã ĐVT")] && ks[key("Mã hóa đơn")]:
		return "KiotViet"
	case ks[key("Nhóm VTHH")] || ks[key("Đơn vị tính chính")] || ks[key("Tính chất")] || ks[key("Đơn giá mua gần nhất")]:
		return "MISA SME / AMIS"
	case ks[key("Mã SKU")] || ks[key("Nhóm hàng hóa")]:
		return "MISA eShop"
	case ks[key("HSD gần nhất")] && ks[key("Giá trị tồn")]:
		return "Digital Shop"
	case ks[key("Mặt hàng")] && ks[key("Giá trị tồn")]:
		return "Sổ Tạp Hóa V1.0"
	}
	return "File Excel khác"
}

func NewImport(fileName string, data []byte, kind string) (*Preview, error) {
	sheets, err := ReadTable(fileName, data)
	if err != nil { return nil, err }
	bestKind, bestSheet, bestRow, bestScore := "", 0, 0, -1
	kinds := []string{"products", "customers", "suppliers", "invoices"}
	if kind != "" && kind != "auto" { kinds = []string{kind} }
	for si, sh := range sheets {
		for ri := 0; ri < len(sh.Rows) && ri < 30; ri++ {
			for _, k := range kinds {
				sc := scoreHeader(sh.Rows[ri], importKinds[k])
				if k == "invoices" && sc > 0 { // phải có mã hóa đơn + số lượng mới là chi tiết hóa đơn
					m := autoMap(sh.Rows[ri], importKinds[k])
					_, a := m["invCode"]
					_, b := m["qty"]
					if !a || !b { sc = 0 } else { sc += 2 }
				}
				if k == "suppliers" {
					if !hasKeyLike(sh.Rows[ri], "nhacungcap", "ncc") { sc -= 3 }
				}
				if k == "customers" && hasKeyLike(sh.Rows[ri], "nhacungcap", "ncc") { sc -= 3 }
				if k == "products" && hasKeyLike(sh.Rows[ri], "khachhang") && !hasKeyLike(sh.Rows[ri], "giaban", "dvt", "donvitinh") { sc -= 3 }
				if sc > bestScore { bestKind, bestSheet, bestRow, bestScore = k, si, ri, sc }
			}
		}
	}
	if bestScore < 2 {
		return nil, uerr("Không nhận ra dòng tiêu đề cột trong file. File cần có dòng tiêu đề như \"Mã hàng, Tên hàng, ĐVT, Giá bán, Tồn kho\"")
	}
	job := &ImportJob{Token: newID(), Kind: bestKind, Sheets: sheets, Sheet: bestSheet, Header: bestRow, Created: time.Now()}
	jobsMu.Lock()
	for t, j := range jobs { if time.Since(j.Created) > time.Hour { delete(jobs, t) } }
	jobs[job.Token] = job
	jobsMu.Unlock()
	p := job.preview(nil)
	p.FileName = fileName
	return p, nil
}

func hasKeyLike(row []string, subs ...string) bool {
	for _, c := range row {
		k := key(cell(c))
		for _, s := range subs { if strings.Contains(k, s) { return true } }
	}
	return false
}

func getJob(tok string) *ImportJob { jobsMu.Lock(); defer jobsMu.Unlock(); return jobs[tok] }

func (j *ImportJob) rows() ([]string, [][]string) {
	sh := j.Sheets[j.Sheet]
	if j.Header >= len(sh.Rows) { return nil, nil }
	cols := make([]string, len(sh.Rows[j.Header]))
	for i, c := range sh.Rows[j.Header] { cols[i] = strings.TrimSpace(cell(c)) }
	data := [][]string{}
	for _, r := range sh.Rows[j.Header+1:] {
		empty := true
		for _, c := range r { if strings.TrimSpace(cell(c)) != "" { empty = false; break } }
		if !empty { data = append(data, r) }
	}
	return cols, data
}

func (j *ImportJob) preview(mapping map[string]int) *Preview {
	cols, data := j.rows()
	fields := importKinds[j.Kind]
	if mapping == nil { mapping = autoMap(cols, fields) }
	p := &Preview{Token: j.Token, Kind: j.Kind, KindName: kindLabels[j.Kind], Sheet: j.Sheet, Header: j.Header, Columns: cols, Fields: fields, Mapping: mapping, Rows: len(data), Source: detectSource(cols)}
	for _, s := range j.Sheets { p.Sheets = append(p.Sheets, s.Name) }
	for i := 0; i < len(data) && i < 8; i++ {
		r := make([]string, len(cols))
		for c := range cols { if c < len(data[i]) { r[c] = cell(data[i][c]) } }
		p.Sample = append(p.Sample, r)
	}
	for _, f := range fields {
		if _, ok := mapping[f.Key]; f.Need && !ok { p.Notes = append(p.Notes, "Chưa chọn cột cho \""+f.Label+"\" (bắt buộc)") }
	}
	if j.Kind == "products" {
		if _, ok := mapping["baseCode"]; ok { p.Notes = append(p.Notes, "File có đơn vị quy đổi (thùng, lốc…): các dòng đơn vị lớn sẽ được gộp vào mặt hàng gốc") }
		if _, ok := mapping["price"]; !ok { p.Notes = append(p.Notes, "Không thấy cột giá bán: mặt hàng sẽ có giá 0, nhớ sửa lại") }
	}
	if j.Kind == "invoices" { p.Notes = append(p.Notes, "Hóa đơn cũ chỉ dùng cho báo cáo và gợi ý combo, không trừ tồn kho") }
	return p
}

func (j *ImportJob) Reconfigure(sheet, header int, kind string) *Preview {
	if sheet >= 0 && sheet < len(j.Sheets) { j.Sheet = sheet }
	if kind != "" { if _, ok := importKinds[kind]; ok { j.Kind = kind } }
	if header >= 0 && header < len(j.Sheets[j.Sheet].Rows) { j.Header = header }
	return j.preview(nil)
}

type ImportResult struct {
	Added   int      `json:"added"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Units   int      `json:"units"`
	Errors  []string `json:"errors"`
}

func (db *DB) CommitImport(j *ImportJob, mapping map[string]int, mode string, u *User) (*ImportResult, error) {
	cols, data := j.rows()
	get := func(r []string, f string) string {
		i, ok := mapping[f]
		if !ok || i < 0 || i >= len(r) || i >= len(cols) { return "" }
		return strings.TrimSpace(r[i])
	}
	res := &ImportResult{}
	addErr := func(row int, msg string) {
		if len(res.Errors) < 30 { res.Errors = append(res.Errors, "Dòng "+itoa(int64(row+j.Header+2))+": "+msg) }
	}
	switch j.Kind {
	case "products":
		return res, db.Do(func(s *State, t *Tx) error {
			byCode, byBar, byName := map[string]*Product{}, map[string]*Product{}, map[string]*Product{}
			for _, p := range s.Products {
				if p.Code != "" { byCode[strings.ToLower(p.Code)] = p }
				if p.Barcode != "" { byBar[p.Barcode] = p }
				byName[norm(p.Name)] = p
			}
			type unitRow struct{ base, name, bar string; ratio, price float64; row int }
			pending := []unitRow{}
			touched := map[string]*Product{}
			for ri, r := range data {
				name := cell(get(r, "name"))
				code := cell(get(r, "code"))
				if name == "" { res.Skipped++; continue }
				typ := norm(cell(get(r, "type")))
				if strings.Contains(typ, "dich vu") || strings.Contains(typ, "combo") { res.Skipped++; continue }
				ratio := num(get(r, "ratio"))
				base := cell(get(r, "baseCode"))
				if base != "" && !strings.EqualFold(base, code) && ratio > 1 {
					pending = append(pending, unitRow{base: base, name: cell(get(r, "unit")), bar: cell(get(r, "barcode")), ratio: ratio, price: num(get(r, "price")), row: ri})
					continue
				}
				bar := cell(get(r, "barcode"))
				var p *Product
				if code != "" { p = byCode[strings.ToLower(code)] }
				if p == nil && bar != "" { p = byBar[bar] }
				if p == nil && code == "" { p = byName[norm(name)] }
				if p != nil && mode == "skip" { res.Skipped++; continue }
				isNew := p == nil
				if isNew {
					p = &Product{ID: newID(), Created: nowMs()}
					res.Added++
				} else {
					res.Updated++
				}
				p.Name = name
				if code != "" { p.Code = code }
				if g := cell(get(r, "group")); g != "" {
					g = strings.ReplaceAll(g, ">>", ">")
					parts := strings.Split(g, ">")
					p.Group = strings.TrimSpace(parts[len(parts)-1])
				}
				if v := cell(get(r, "unit")); v != "" { p.Unit = v } else if p.Unit == "" { p.Unit = "cái" }
				if bar != "" { p.Barcode = bar }
				if v, ok := parseNum(get(r, "price")); ok { p.Price = r0(v) }
				stock, hasStock := parseNum(get(r, "stock"))
				if v, ok := parseNum(get(r, "cost")); ok && v > 0 {
					p.Cost = r0(v)
				} else if val, ok := parseNum(get(r, "value")); ok && hasStock && stock > 0 {
					p.Cost = r0(val / stock)
				}
				if v, ok := parseNum(get(r, "min")); ok { p.Min = v }
				if v := cell(get(r, "location")); v != "" { p.Location = v }
				act := norm(cell(get(r, "active")))
				if act != "" {
					p.Hidden = act == "0" || act == "false" || strings.Contains(act, "ngung") || strings.Contains(act, "khong")
					if _, ok := mapping["active"]; ok && key(cols[mapping["active"]]) == key("Ngừng kinh doanh") { p.Hidden = act == "1" || act == "true" || act == "co" || act == "x" }
				}
				if hasStock && (isNew || mode != "keepStock") {
					p.Stock = r3(stock)
					p.Lots = nil
					if exp := parseDate(get(r, "exp")); exp != "" && stock > 0 {
						p.Lots = []Lot{{ID: newID(), Qty: r3(stock), Exp: exp, In: today()}}
					}
				}
				if p.Code == "" { p.Code = s.nextCode("SP", 6) }
				p.Updated = nowMs()
				s.Products[p.ID] = p
				byCode[strings.ToLower(p.Code)] = p
				if p.Barcode != "" { byBar[p.Barcode] = p }
				byName[norm(p.Name)] = p
				touched[p.ID] = p
			}
			for _, ur := range pending {
				p := byCode[strings.ToLower(ur.base)]
				if p == nil { addErr(ur.row, "không thấy mặt hàng gốc "+ur.base+" cho đơn vị "+ur.name); res.Skipped++; continue }
				conv := UnitConv{Name: ur.name, Ratio: ur.ratio, Price: r0(ur.price), Barcode: ur.bar}
				if conv.Price <= 0 { conv.Price = r0(p.Price * ur.ratio) }
				replaced := false
				for i := range p.Units { if norm(p.Units[i].Name) == norm(conv.Name) { p.Units[i] = conv; replaced = true } }
				if !replaced { p.Units = append(p.Units, conv) }
				touched[p.ID] = p
				res.Units++
			}
			for _, p := range touched { reconcileLots(p); t.put("product", p.ID, p) }
			t.put("counters", "", s.Counters)
			return nil
		})
	case "customers", "suppliers":
		return res, db.Do(func(s *State, t *Tx) error {
			m, prefix, rk, pt := s.Customers, "KH", "customer", "cust"
			if j.Kind == "suppliers" { m, prefix, rk, pt = s.Suppliers, "NCC", "supplier", "supp" }
			byCode, byPhone, byName := map[string]*Party{}, map[string]*Party{}, map[string]*Party{}
			for _, p := range m {
				if p.Code != "" { byCode[strings.ToLower(p.Code)] = p }
				if p.Phone != "" { byPhone[p.Phone] = p }
				byName[norm(p.Name)] = p
			}
			for _, r := range data {
				name := cell(get(r, "name"))
				if name == "" { res.Skipped++; continue }
				code, phone := cell(get(r, "code")), cell(get(r, "phone"))
				if phone != "" && !strings.HasPrefix(phone, "0") && len(phone) == 9 { phone = "0" + phone }
				var p *Party
				if code != "" { p = byCode[strings.ToLower(code)] }
				if p == nil && phone != "" { p = byPhone[phone] }
				if p == nil && code == "" { p = byName[norm(name)] }
				if p != nil && mode == "skip" { res.Skipped++; continue }
				if p == nil {
					p = &Party{ID: newID(), Created: nowMs(), Last: nowMs()}
					res.Added++
				} else {
					res.Updated++
				}
				p.Name = name
				if code != "" { p.Code = code } else if p.Code == "" { p.Code = s.nextCode(prefix, 6) }
				if phone != "" { p.Phone = phone }
				if v := cell(get(r, "addr")); v != "" { p.Addr = v }
				if v := cell(get(r, "note")); v != "" { p.Note = v }
				if d, ok := parseNum(get(r, "debt")); ok && r0(d) != p.Debt {
					delta := r0(d) - p.Debt
					e := &CashEntry{ID: newID(), Time: nowMs(), Kind: "adj", Amount: delta, Cat: "Nợ chuyển từ phần mềm cũ", PartyType: pt, PartyID: p.ID, PartyName: p.Name, Staff: u.Name}
					e.Code = s.nextCode("DC", 6)
					s.Cash[e.ID] = e
					t.put("cash", e.ID, e)
					p.Debt = r0(d)
				}
				m[p.ID] = p
				byCode[strings.ToLower(p.Code)] = p
				t.put(rk, p.ID, p)
			}
			t.put("counters", "", s.Counters)
			return nil
		})
	case "invoices":
		return res, db.Do(func(s *State, t *Tx) error {
			byCode, byName := map[string]*Product{}, map[string]*Product{}
			for _, p := range s.Products {
				if p.Code != "" { byCode[strings.ToLower(p.Code)] = p }
				byName[norm(p.Name)] = p
			}
			exist := map[string]bool{}
			for _, inv := range s.Invoices { if inv.Src == "import" { exist[inv.Code] = true } }
			order := []string{}
			invs := map[string]*Invoice{}
			for ri, r := range data {
				code := cell(get(r, "invCode"))
				pname := cell(get(r, "pname"))
				if code == "" || pname == "" { res.Skipped++; continue }
				if st := norm(cell(get(r, "status"))); strings.Contains(st, "huy") { res.Skipped++; continue }
				if exist[code] { res.Skipped++; continue }
				inv := invs[code]
				if inv == nil {
					tm := parseTime(get(r, "time"))
					if tm.IsZero() { addErr(ri, "không đọc được thời gian, bỏ qua hóa đơn "+code); res.Skipped++; continue }
					inv = &Invoice{ID: newID(), Code: code, Time: tm.UnixMilli(), CustName: cell(get(r, "custName")), Staff: "nhập từ file", Src: "import"}
					invs[code] = inv
					order = append(order, code)
				}
				qty := num(get(r, "qty"))
				if qty == 0 { continue }
				price := num(get(r, "price"))
				amount, hasAmt := parseNum(get(r, "amount"))
				if price == 0 && hasAmt && qty != 0 { price = amount / qty }
				var p *Product
				if c := cell(get(r, "pcode")); c != "" { p = byCode[strings.ToLower(c)] }
				if p == nil { p = byName[norm(pname)] }
				l := Line{Name: pname, Unit: cell(get(r, "unit")), Ratio: 1, Qty: r3(qty), Price: r0(price)}
				if p != nil {
					l.PID, l.Code, l.Name = p.ID, p.Code, p.Name
					if l.Unit == "" { l.Unit = p.Unit }
					for _, c := range p.Units { if norm(c.Name) == norm(l.Unit) { l.Ratio = c.Ratio } }
					l.Cost = r0(p.Cost * l.Ratio)
				}
				inv.Lines = append(inv.Lines, l)
			}
			for _, c := range order {
				inv := invs[c]
				if len(inv.Lines) == 0 { continue }
				for _, l := range inv.Lines { inv.Sub += l.Qty * l.Price; inv.Cost += l.Qty * l.Cost }
				inv.Sub, inv.Cost = r0(inv.Sub), r0(inv.Cost)
				inv.Total = inv.Sub
				inv.Cash = inv.Total
				s.Invoices[inv.ID] = inv
				t.put("invoice", inv.ID, inv)
				res.Added++
			}
			return nil
		})
	}
	return nil, uerr("Loại dữ liệu không hợp lệ")
}
