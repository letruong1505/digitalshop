package main

// Đọc/ghi Excel (.xlsx) chỉ bằng thư viện chuẩn, cùng CSV và bảng HTML
// (nhiều phần mềm xuất "file .xls" thực chất là trang HTML). File .xls thật: xem xls.go.

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// NumMark đánh dấu ô số thật trong Excel (dấu chấm là phần thập phân).
const NumMark = "\x00"

// cell bỏ dấu đánh số để dùng như chữ.
func cell(s string) string { return strings.TrimPrefix(s, NumMark) }

type Sheet struct {
	Name string
	Rows [][]string
}

func ReadTable(name string, data []byte) ([]Sheet, error) {
	low := strings.ToLower(name)
	switch {
	case bytes.HasPrefix(data, []byte("PK")):
		return readXLSX(data)
	case bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0}):
		return readXLS(data)
	}
	text := decodeText(data)
	trim := strings.TrimSpace(text)
	if strings.HasPrefix(trim, "<") && strings.Contains(strings.ToLower(trim[:min(len(trim), 20000)]), "<table") {
		return readHTMLTables(text), nil
	}
	if strings.HasSuffix(low, ".xlsx") || strings.HasSuffix(low, ".xls") {
		if !strings.Contains(trim, ",") && !strings.Contains(trim, ";") && !strings.Contains(trim, "\t") {
			return nil, uerr("Không đọc được file Excel này")
		}
	}
	return []Sheet{{Name: "CSV", Rows: readCSV(text)}}, nil
}

func decodeText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		u := make([]uint16, (len(b)-2)/2)
		for i := range u { u[i] = uint16(b[2+2*i]) | uint16(b[3+2*i])<<8 }
		return string(utf16.Decode(u))
	}
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		u := make([]uint16, (len(b)-2)/2)
		for i := range u { u[i] = uint16(b[3+2*i]) | uint16(b[2+2*i])<<8 }
		return string(utf16.Decode(u))
	}
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(b) { return string(b) }
	r := make([]rune, len(b)) // Windows-1252 dự phòng
	for i, c := range b { r[i] = rune(c) }
	return string(r)
}

func readCSV(text string) [][]string {
	first := text
	if i := strings.IndexByte(text, '\n'); i > 0 { first = text[:i] }
	delim := ','
	if strings.Count(first, ";") > strings.Count(first, ",") { delim = ';' }
	if strings.Count(first, "\t") > strings.Count(first, string(delim)) { delim = '\t' }
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, _ := r.ReadAll()
	return rows
}

var (
	reTR   = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	reTD   = regexp.MustCompile(`(?is)<t[dh]([^>]*)>(.*?)</t[dh]>`)
	reTag  = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpan = regexp.MustCompile(`(?i)colspan\s*=\s*["']?(\d+)`)
	reTbl  = regexp.MustCompile(`(?is)<table[^>]*>(.*?)</table>`)
)

func readHTMLTables(text string) []Sheet {
	out := []Sheet{}
	for i, tm := range reTbl.FindAllStringSubmatch(text, -1) {
		sh := Sheet{Name: fmt.Sprintf("Bảng %d", i+1)}
		for _, tr := range reTR.FindAllStringSubmatch(tm[1], -1) {
			row := []string{}
			for _, td := range reTD.FindAllStringSubmatch(tr[1], -1) {
				v := strings.TrimSpace(html.UnescapeString(reTag.ReplaceAllString(strings.ReplaceAll(td[2], "<br", "\n<br"), "")))
				v = strings.Join(strings.Fields(v), " ")
				row = append(row, v)
				if m := reSpan.FindStringSubmatch(td[1]); m != nil {
					n, _ := strconv.Atoi(m[1])
					for k := 1; k < n && k < 50; k++ { row = append(row, "") }
				}
			}
			sh.Rows = append(sh.Rows, row)
		}
		if len(sh.Rows) > 0 { out = append(out, sh) }
	}
	return out
}

// ---------- XLSX ----------

func readXLSX(data []byte) ([]Sheet, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil { return nil, uerr("File Excel bị hỏng hoặc không đúng định dạng") }
	files := map[string]*zip.File{}
	for _, f := range zr.File { files[strings.TrimPrefix(f.Name, "/")] = f }
	readF := func(name string) []byte {
		f := files[name]
		if f == nil { return nil }
		rc, err := f.Open()
		if err != nil { return nil }
		defer rc.Close()
		b, _ := io.ReadAll(rc)
		return b
	}
	shared := parseShared(readF("xl/sharedStrings.xml"))
	type wbSheet struct {
		Name string `xml:"name,attr"`
		RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	}
	var wb struct {
		Sheets []wbSheet `xml:"sheets>sheet"`
	}
	xml.Unmarshal(readF("xl/workbook.xml"), &wb)
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	xml.Unmarshal(readF("xl/_rels/workbook.xml.rels"), &rels)
	target := map[string]string{}
	for _, r := range rels.Rel {
		t := r.Target
		if strings.HasPrefix(t, "/") { t = strings.TrimPrefix(t, "/") } else { t = path.Join("xl", t) }
		target[r.ID] = t
	}
	out := []Sheet{}
	for _, s := range wb.Sheets {
		p := target[s.RID]
		if p == "" { continue }
		rows := parseSheet(readF(p), shared)
		out = append(out, Sheet{Name: s.Name, Rows: rows})
	}
	if len(out) == 0 { // workbook không chuẩn: thử mọi sheet
		for n := range files {
			if strings.HasPrefix(n, "xl/worksheets/") && strings.HasSuffix(n, ".xml") {
				out = append(out, Sheet{Name: path.Base(n), Rows: parseSheet(readF(n), shared)})
			}
		}
	}
	if len(out) == 0 { return nil, uerr("Không tìm thấy trang tính nào trong file") }
	return out, nil
}

func parseShared(b []byte) []string {
	if b == nil { return nil }
	d := xml.NewDecoder(bytes.NewReader(b))
	out := []string{}
	var cur strings.Builder
	inSI, inT, inRPh := false, false, false
	for {
		tok, err := d.Token()
		if err != nil { break }
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				inT = true
			case "rPh":
				inRPh = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "si":
				inSI = false
				out = append(out, cur.String())
			case "t":
				inT = false
			case "rPh":
				inRPh = false
			}
		case xml.CharData:
			if inSI && inT && !inRPh { cur.Write(t) }
		}
	}
	return out
}

func colIndex(ref string) int {
	n := 0
	for _, c := range ref {
		if c >= 'A' && c <= 'Z' { n = n*26 + int(c-'A'+1) } else if c >= 'a' && c <= 'z' { n = n*26 + int(c-'a'+1) } else { break }
	}
	return n - 1
}

func parseSheet(b []byte, shared []string) [][]string {
	if b == nil { return nil }
	d := xml.NewDecoder(bytes.NewReader(b))
	rows := [][]string{}
	var row []string
	rowIdx := -1
	var cType, cRef string
	var val strings.Builder
	inV, inIS, inT := false, false, false
	col := -1
	for {
		tok, err := d.Token()
		if err != nil { break }
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = []string{}
				col = -1
				for _, a := range t.Attr {
					if a.Name.Local == "r" {
						if n, err := strconv.Atoi(a.Value); err == nil {
							for len(rows) < n-1 { rows = append(rows, nil) }
							rowIdx = n - 1
						}
					}
				}
			case "c":
				cType, cRef = "", ""
				for _, a := range t.Attr {
					if a.Name.Local == "t" { cType = a.Value }
					if a.Name.Local == "r" { cRef = a.Value }
				}
				val.Reset()
				if cRef != "" { col = colIndex(cRef) } else { col++ }
			case "v":
				inV = true
			case "is":
				inIS = true
			case "t":
				inT = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
			case "is":
				inIS = false
			case "t":
				inT = false
			case "c":
				v := val.String()
				if cType == "s" {
					if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i >= 0 && i < len(shared) { v = shared[i] }
				} else if cType == "b" {
					if v == "1" { v = "TRUE" } else { v = "FALSE" }
				} else if cType == "" || cType == "n" {
					if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
						if f == math.Trunc(f) && math.Abs(f) < 1e15 { v = NumMark + strconv.FormatInt(int64(f), 10) } else { v = NumMark + strconv.FormatFloat(f, 'f', -1, 64) }
					}
				}
				if col >= 0 && col < 500 {
					for len(row) <= col { row = append(row, "") }
					row[col] = strings.TrimSpace(v)
				}
			case "row":
				if rowIdx >= 0 && rowIdx == len(rows) { rows = append(rows, row) } else { rows = append(rows, row) }
				rowIdx = -1
			}
		case xml.CharData:
			if inV || (inIS && inT) { val.Write(t) }
		}
	}
	return rows
}

// ---------- ghi XLSX ----------

type XCol struct {
	Title string
	Width float64
	Num   bool
}

func WriteXLSX(sheetName string, cols []XCol, rows [][]any) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w := func(name, s string) { f, _ := zw.Create(name); f.Write([]byte(s)) }
	w("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`)
	w("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)
	w("xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="`+xmlEsc(sheetName)+`" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	w("xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`)
	w("xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="#,##0.###"/></numFmts><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFD8EDE1"/></patternFill></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="3"><xf/><xf fontId="1" fillId="2" applyFont="1" applyFill="1"/><xf numFmtId="164" applyNumberFormat="1"/></cellXfs></styleSheet>`)
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols>`)
	for i, c := range cols {
		wd := c.Width
		if wd == 0 { wd = 14 }
		sb.WriteString(fmt.Sprintf(`<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, wd))
	}
	sb.WriteString(`</cols><sheetData><row r="1">`)
	for i, c := range cols {
		sb.WriteString(fmt.Sprintf(`<c r="%s1" t="inlineStr" s="1"><is><t>%s</t></is></c>`, colName(i), xmlEsc(c.Title)))
	}
	sb.WriteString(`</row>`)
	for ri, r := range rows {
		sb.WriteString(fmt.Sprintf(`<row r="%d">`, ri+2))
		for ci, v := range r {
			ref := fmt.Sprintf("%s%d", colName(ci), ri+2)
			switch x := v.(type) {
			case float64:
				sb.WriteString(fmt.Sprintf(`<c r="%s" s="2"><v>%s</v></c>`, ref, strconv.FormatFloat(x, 'f', -1, 64)))
			case int:
				sb.WriteString(fmt.Sprintf(`<c r="%s" s="2"><v>%d</v></c>`, ref, x))
			case int64:
				sb.WriteString(fmt.Sprintf(`<c r="%s" s="2"><v>%d</v></c>`, ref, x))
			case nil:
			default:
				s := fmt.Sprint(x)
				if s == "" { continue }
				sb.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, xmlEsc(s)))
			}
		}
		sb.WriteString(`</row>`)
	}
	sb.WriteString(`</sheetData></worksheet>`)
	w("xl/worksheets/sheet1.xml", sb.String())
	zw.Close()
	return buf.Bytes()
}

func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 { s = string(rune('A'+(i-1)%26)) + s }
	return s
}

func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' { continue }
		b.WriteRune(r)
	}
	var o bytes.Buffer
	xml.EscapeText(&o, []byte(b.String()))
	return o.String()
}
