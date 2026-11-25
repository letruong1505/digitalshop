package main

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var viMap = map[rune]rune{}

func init() {
	groups := map[rune]string{
		'a': "àáảãạăằắẳẵặâầấẩẫậ", 'e': "èéẻẽẹêềếểễệ", 'i': "ìíỉĩị",
		'o': "òóỏõọôồốổỗộơờớởỡợ", 'u': "ùúủũụưừứửữự", 'y': "ỳýỷỹỵ", 'd': "đ",
	}
	for base, s := range groups { for _, r := range s { viMap[r] = base } }
}

// norm: chữ thường, bỏ dấu tiếng Việt.
func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.Is(unicode.Mn, r) { continue }
		if m, ok := viMap[r]; ok { r = m }
		b.WriteRune(r)
	}
	return b.String()
}

// key: bỏ dấu, chỉ giữ chữ và số (dùng so khớp tiêu đề cột).
func key(s string) string {
	var b strings.Builder
	for _, r := range norm(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) { b.WriteRune(r) }
	}
	return b.String()
}

func tokens(s string) []string {
	return strings.FieldsFunc(norm(s), func(r rune) bool { return !(unicode.IsLetter(r) || unicode.IsDigit(r)) })
}

// parseNum đọc số kiểu Việt Nam: "12.500", "12,5", "1.234.567,89", "12.5", "20k", "1tr".
func parseNum(s string) (float64, bool) {
	if strings.HasPrefix(s, NumMark) {
		f, err := strconv.ParseFloat(s[len(NumMark):], 64)
		return f, err == nil
	}
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.NewReplacer(" ", "", "đ", "", "vnd", "", "vnđ", "", " ", "", "₫", "").Replace(s)
	if s == "" || s == "-" { return 0, false }
	mul := 1.0
	if strings.HasSuffix(s, "tr") { mul, s = 1e6, strings.TrimSuffix(s, "tr") } else if strings.HasSuffix(s, "k") { mul, s = 1e3, strings.TrimSuffix(s, "k") }
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") { neg, s = true, s[1:len(s)-1] }
	dots, commas := strings.Count(s, "."), strings.Count(s, ",")
	switch {
	case dots > 0 && commas > 0:
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case dots > 1:
		s = strings.ReplaceAll(s, ".", "")
	case commas > 1:
		s = strings.ReplaceAll(s, ",", "")
	case dots == 1:
		i := strings.Index(s, ".")
		if len(s)-i-1 == 3 && mul == 1 { s = strings.ReplaceAll(s, ".", "") } // 12.500 = mười hai nghìn rưỡi
	case commas == 1:
		i := strings.Index(s, ",")
		if len(s)-i-1 == 3 && mul == 1 { s = strings.ReplaceAll(s, ",", "") } else { s = strings.Replace(s, ",", ".", 1) }
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) { return 0, false }
	if neg { f = -f }
	return f * mul, true
}

func num(s string) float64 { f, _ := parseNum(s); return f }

// parseDate nhận dd/mm/yyyy, yyyy-mm-dd, số ngày kiểu Excel; trả "yyyy-mm-dd".
func parseDate(s string) string {
	t := parseTime(s)
	if t.IsZero() { return "" }
	return t.Format("2006-01-02")
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(cell(s))
	if s == "" { return time.Time{} }
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 80000 {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)
		days := math.Floor(f)
		secs := math.Round((f - days) * 86400)
		return base.AddDate(0, 0, int(days)).Add(time.Duration(secs) * time.Second)
	}
	layouts := []string{"02/01/2006 15:04:05", "02/01/2006 15:04", "2/1/2006 15:04:05", "2/1/2006 15:04", "02/01/2006", "2/1/2006",
		"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02", "02-01-2006", "2-1-2006", "02.01.2006", "2006/01/02"}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil { return t }
	}
	return time.Time{}
}
