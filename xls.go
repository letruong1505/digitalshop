package main

// Đọc file Excel đời cũ (.xls, Excel 97–2003, định dạng BIFF8 trong vỏ OLE2).
// MISA SME, MISA eShop bản cũ và nhiều phần mềm kế toán vẫn xuất ra loại file này.
// Chỉ lấy giá trị ô (chữ, số, kết quả công thức), không cần định dạng.

import (
	"encoding/binary"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	cfbFree   = 0xFFFFFFFF
	cfbEnd    = 0xFFFFFFFE
	xlsMaxRow = 300000
	xlsMaxCol = 300
)

type cfb struct {
	data      []byte
	ss        int // kích thước sector
	fat       []uint32
	miniFat   []uint32
	miniData  []byte
	cutoff    uint32
	entries   []cfbEntry
}

type cfbEntry struct {
	name  string
	typ   byte
	start uint32
	size  uint32
}

func (c *cfb) sector(n uint32) []byte {
	off := (int(n) + 1) * c.ss
	if n >= 0xFFFFFFFA || off < 0 || off+c.ss > len(c.data) {
		if off >= 0 && off < len(c.data) { return c.data[off:] } // sector cuối bị cắt ngắn
		return nil
	}
	return c.data[off : off+c.ss]
}

func (c *cfb) chain(start uint32, fat []uint32, get func(uint32) []byte, limit int) []byte {
	var out []byte
	seen := map[uint32]bool{}
	for n := start; n != cfbEnd && n != cfbFree && int(n) < len(fat) && !seen[n]; n = fat[n] {
		seen[n] = true
		s := get(n)
		if s == nil { break }
		out = append(out, s...)
		if limit > 0 && len(out) >= limit { break }
	}
	return out
}

func openCFB(b []byte) (*cfb, error) {
	if len(b) < 512 { return nil, uerr("File .xls bị hỏng (quá ngắn)") }
	le := binary.LittleEndian
	c := &cfb{data: b}
	shift := le.Uint16(b[0x1E:])
	if shift != 9 && shift != 12 { return nil, uerr("File .xls không đúng định dạng") }
	c.ss = 1 << shift
	nFat := int(le.Uint32(b[0x2C:]))
	dirStart := le.Uint32(b[0x30:])
	c.cutoff = le.Uint32(b[0x38:])
	miniFatStart := le.Uint32(b[0x3C:])
	difatNext := le.Uint32(b[0x44:])
	// Danh sách sector chứa bảng FAT
	var fatSecs []uint32
	for i := 0; i < 109; i++ {
		v := le.Uint32(b[0x4C+4*i:])
		if v != cfbFree { fatSecs = append(fatSecs, v) }
	}
	for guard := 0; difatNext != cfbEnd && difatNext != cfbFree && guard < 10000; guard++ {
		s := c.sector(difatNext)
		if len(s) < c.ss { break }
		for i := 0; i < c.ss/4-1; i++ {
			v := le.Uint32(s[4*i:])
			if v != cfbFree { fatSecs = append(fatSecs, v) }
		}
		difatNext = le.Uint32(s[c.ss-4:])
	}
	if nFat > 0 && len(fatSecs) > nFat { fatSecs = fatSecs[:nFat] }
	for _, fs := range fatSecs {
		s := c.sector(fs)
		for i := 0; i+4 <= len(s); i += 4 { c.fat = append(c.fat, le.Uint32(s[i:])) }
	}
	dir := c.chain(dirStart, c.fat, c.sector, 0)
	for i := 0; i+128 <= len(dir); i += 128 {
		e := dir[i : i+128]
		nl := int(le.Uint16(e[64:]))
		if nl > 64 { nl = 64 }
		u := make([]uint16, 0, 32)
		for j := 0; j+1 < nl-2 && j+1 < 64; j += 2 { u = append(u, le.Uint16(e[j:])) }
		c.entries = append(c.entries, cfbEntry{name: string(utf16.Decode(u)), typ: e[66], start: le.Uint32(e[116:]), size: le.Uint32(e[120:])})
	}
	if len(c.entries) == 0 || c.entries[0].typ != 5 { return nil, uerr("File .xls bị hỏng (không có thư mục gốc)") }
	root := c.entries[0]
	c.miniData = c.chain(root.start, c.fat, c.sector, 0)
	mf := c.chain(miniFatStart, c.fat, c.sector, 0)
	for i := 0; i+4 <= len(mf); i += 4 { c.miniFat = append(c.miniFat, le.Uint32(mf[i:])) }
	return c, nil
}

func (c *cfb) stream(names ...string) []byte {
	for _, want := range names {
		for _, e := range c.entries {
			if e.typ != 2 || !strings.EqualFold(e.name, want) { continue }
			var b []byte
			if e.size < c.cutoff {
				b = c.chain(e.start, c.miniFat, func(n uint32) []byte {
					off := int(n) * 64
					if off+64 > len(c.miniData) { return nil }
					return c.miniData[off : off+64]
				}, int(e.size))
			} else {
				b = c.chain(e.start, c.fat, c.sector, int(e.size))
			}
			if len(b) > int(e.size) { b = b[:e.size] }
			return b
		}
	}
	return nil
}

// segReader đọc dữ liệu trải qua nhiều bản ghi CONTINUE (chuỗi trong SST có thể bị cắt ngang).
type segReader struct {
	segs [][]byte
	i, p int
}

func (r *segReader) eof() bool {
	for r.i < len(r.segs) && r.p >= len(r.segs[r.i]) { r.i++; r.p = 0 }
	return r.i >= len(r.segs)
}
func (r *segReader) bytes(n int) []byte {
	out := make([]byte, 0, n)
	for n > 0 && !r.eof() {
		s := r.segs[r.i][r.p:]
		k := min(n, len(s))
		out = append(out, s[:k]...)
		r.p += k
		n -= k
	}
	return out
}
func (r *segReader) u8() byte { b := r.bytes(1); if len(b) < 1 { return 0 }; return b[0] }
func (r *segReader) u16() uint16 { b := r.bytes(2); if len(b) < 2 { return 0 }; return binary.LittleEndian.Uint16(b) }
func (r *segReader) u32() uint32 { b := r.bytes(4); if len(b) < 4 { return 0 }; return binary.LittleEndian.Uint32(b) }

// chars đọc cch ký tự; khi sang bản ghi CONTINUE mới, byte đầu cho biết độ rộng ký tự.
func (r *segReader) chars(cch int, high bool) string {
	u := make([]uint16, 0, cch)
	for cch > 0 {
		if r.i < len(r.segs) && r.p >= len(r.segs[r.i]) {
			r.i++; r.p = 0
			if r.i >= len(r.segs) { break }
			high = r.segs[r.i][0]&1 == 1
			r.p = 1
		}
		if r.i >= len(r.segs) { break }
		s := r.segs[r.i]
		if high {
			if r.p+2 > len(s) { r.p = len(s); continue }
			u = append(u, binary.LittleEndian.Uint16(s[r.p:]))
			r.p += 2
		} else {
			u = append(u, uint16(s[r.p]))
			r.p++
		}
		cch--
	}
	return string(utf16.Decode(u))
}

// richString đọc XLUnicodeRichExtendedString (dùng trong SST).
func (r *segReader) richString() string {
	cch := int(r.u16())
	fl := r.u8()
	runs, ext := 0, 0
	if fl&0x08 != 0 { runs = int(r.u16()) }
	if fl&0x04 != 0 { ext = int(r.u32()) }
	s := r.chars(cch, fl&1 == 1)
	if runs > 0 { r.bytes(4 * runs) }
	if ext > 0 { r.bytes(ext) }
	return s
}

func xlsString(b []byte, lenBytes int) string {
	if len(b) < lenBytes+1 { return "" }
	var cch int
	if lenBytes == 1 { cch = int(b[0]) } else { cch = int(binary.LittleEndian.Uint16(b)) }
	r := &segReader{segs: [][]byte{b[lenBytes:]}}
	fl := r.u8()
	runs, ext := 0, 0
	if fl&0x08 != 0 { runs = int(r.u16()) }
	if fl&0x04 != 0 { ext = int(r.u32()) }
	_ = runs; _ = ext
	return r.chars(cch, fl&1 == 1)
}

func rkValue(rk uint32) float64 {
	var f float64
	if rk&2 != 0 {
		f = float64(int32(rk) >> 2)
	} else {
		f = math.Float64frombits(uint64(rk&0xFFFFFFFC) << 32)
	}
	if rk&1 != 0 { f /= 100 }
	return f
}

func numCell(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 { return NumMark + strconv.FormatInt(int64(f), 10) }
	return NumMark + strconv.FormatFloat(f, 'f', -1, 64)
}

type xlsRec struct {
	typ  uint16
	data []byte
	pos  int
}

func readXLS(b []byte) ([]Sheet, error) {
	c, err := openCFB(b)
	if err != nil { return nil, err }
	wb := c.stream("Workbook", "Book")
	if len(wb) == 0 {
		if c.stream("EncryptedPackage") != nil { return nil, uerr("File Excel đang đặt mật khẩu. Hãy mở bằng Excel, bỏ mật khẩu rồi lưu lại") }
		return nil, uerr("File .xls không có dữ liệu bảng tính")
	}
	le := binary.LittleEndian
	var recs []xlsRec
	for p := 0; p+4 <= len(wb); {
		t, n := le.Uint16(wb[p:]), int(le.Uint16(wb[p+2:]))
		if p+4+n > len(wb) { n = len(wb) - p - 4 }
		recs = append(recs, xlsRec{t, wb[p+4 : p+4+n], p})
		p += 4 + n
	}
	if len(recs) == 0 || recs[0].typ != 0x0809 { return nil, uerr("File .xls không đúng định dạng") }
	if len(recs[0].data) >= 2 && le.Uint16(recs[0].data) < 0x0600 {
		return nil, uerr("Đây là file Excel 95 trở về trước. Hãy mở bằng Excel và lưu thành .xlsx rồi nhập lại")
	}
	type bs struct {
		name string
		pos  int
	}
	var sheets []bs
	var sst []string
	for i := 0; i < len(recs); i++ {
		r := recs[i]
		if r.typ == 0x000A { break } // hết phần workbook
		switch r.typ {
		case 0x002F:
			return nil, uerr("File Excel đang đặt mật khẩu. Hãy mở bằng Excel, bỏ mật khẩu rồi lưu lại")
		case 0x0085:
			if len(r.data) >= 8 && r.data[5] == 0 { // chỉ lấy trang tính thường
				sheets = append(sheets, bs{xlsString(r.data[6:], 1), int(le.Uint32(r.data))})
			}
		case 0x00FC:
			segs := [][]byte{r.data}
			for i+1 < len(recs) && recs[i+1].typ == 0x003C { i++; segs = append(segs, recs[i].data) }
			sr := &segReader{segs: segs}
			sr.u32()
			n := int(sr.u32())
			if n > 2_000_000 { n = 2_000_000 }
			sst = make([]string, 0, n)
			for k := 0; k < n && !sr.eof(); k++ { sst = append(sst, sr.richString()) }
		}
	}
	idx := map[int]int{}
	for i, r := range recs { idx[r.pos] = i }
	var out []Sheet
	for _, sh := range sheets {
		start, ok := idx[sh.pos]
		if !ok { continue }
		cells := map[int]map[int]string{}
		set := func(row, col int, v string) {
			if row >= xlsMaxRow || col >= xlsMaxCol { return }
			v = strings.TrimSpace(v)
			if v == "" { return }
			m := cells[row]
			if m == nil { m = map[int]string{}; cells[row] = m }
			m[col] = v
		}
		var pendRow, pendCol = -1, -1
		for i := start + 1; i < len(recs); i++ {
			r := recs[i]
			d := r.data
			if r.typ == 0x000A || r.typ == 0x0809 { break }
			if len(d) < 6 && r.typ != 0x0207 { continue }
			switch r.typ {
			case 0x00FD: // LABELSST
				if len(d) >= 10 {
					k := int(le.Uint32(d[6:]))
					if k < len(sst) { set(int(le.Uint16(d)), int(le.Uint16(d[2:])), sst[k]) }
				}
			case 0x0204, 0x00D6: // LABEL, RSTRING
				set(int(le.Uint16(d)), int(le.Uint16(d[2:])), xlsString(d[6:], 2))
			case 0x0203: // NUMBER
				if len(d) >= 14 { set(int(le.Uint16(d)), int(le.Uint16(d[2:])), numCell(math.Float64frombits(le.Uint64(d[6:])))) }
			case 0x027E: // RK
				if len(d) >= 10 { set(int(le.Uint16(d)), int(le.Uint16(d[2:])), numCell(rkValue(le.Uint32(d[6:])))) }
			case 0x00BD: // MULRK
				row, col := int(le.Uint16(d)), int(le.Uint16(d[2:]))
				for p := 4; p+6 <= len(d)-2; p += 6 {
					set(row, col, numCell(rkValue(le.Uint32(d[p+2:]))))
					col++
				}
			case 0x0205: // BOOLERR
				if len(d) >= 8 && d[7] == 0 {
					v := "FALSE"
					if d[6] != 0 { v = "TRUE" }
					set(int(le.Uint16(d)), int(le.Uint16(d[2:])), v)
				}
			case 0x0006: // FORMULA: lấy kết quả đã tính sẵn
				if len(d) >= 14 {
					row, col := int(le.Uint16(d)), int(le.Uint16(d[2:]))
					if le.Uint16(d[12:]) == 0xFFFF {
						switch d[6] {
						case 0:
							pendRow, pendCol = row, col
						case 1:
							v := "FALSE"
							if d[8] != 0 { v = "TRUE" }
							set(row, col, v)
						}
					} else {
						set(row, col, numCell(math.Float64frombits(le.Uint64(d[6:]))))
					}
				}
			case 0x0207: // STRING (kết quả chữ của công thức ngay trước)
				if pendRow >= 0 {
					segs := [][]byte{d}
					for i+1 < len(recs) && recs[i+1].typ == 0x003C { i++; segs = append(segs, recs[i].data) }
					sr := &segReader{segs: segs}
					cch := int(sr.u16())
					fl := sr.u8()
					set(pendRow, pendCol, sr.chars(cch, fl&1 == 1))
					pendRow, pendCol = -1, -1
				}
			}
		}
		rowsIdx := make([]int, 0, len(cells))
		for k := range cells { rowsIdx = append(rowsIdx, k) }
		sort.Ints(rowsIdx)
		rows := make([][]string, 0, len(rowsIdx))
		for _, ri := range rowsIdx {
			m := cells[ri]
			mx := -1
			for k := range m { mx = max(mx, k) }
			row := make([]string, mx+1)
			for k, v := range m { row[k] = v }
			rows = append(rows, row)
		}
		out = append(out, Sheet{Name: sh.name, Rows: rows})
	}
	if len(out) == 0 { return nil, uerr("Không tìm thấy trang tính nào trong file .xls") }
	return out, nil
}
