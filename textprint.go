package main

// Phương án dự phòng khi không dựng được ảnh phiếu (Edge/Chrome ẩn lỗi): in chữ ESC/POS không dấu.
// Phiếu vẫn ra đủ nội dung (tên hàng, số lượng, tiền), chỉ mất dấu tiếng Việt.

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	reSvg   = regexp.MustCompile(`(?is)<\s*(svg|img)\b.*?(</\s*svg\s*>|>)`)
	reHr    = regexp.MustCompile(`(?i)<\s*hr\b[^>]*>`)
	reBr    = regexp.MustCompile(`(?i)<\s*br\s*/?>|</\s*(div|p|h[1-6]|tr|li)\s*>`)
	reCell  = regexp.MustCompile(`(?i)</\s*(td|th)\s*>`)
	reTags  = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpace = regexp.MustCompile(`[ \t\r\f\v\x{00a0}]+`)
)

// vnASCII bỏ dấu tiếng Việt.
func vnASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'đ': b.WriteRune('d'); continue
		case 'Đ': b.WriteRune('D'); continue
		case '×': b.WriteRune('x'); continue
		case '₫': b.WriteString("d"); continue
		case '·', '•': b.WriteRune('-'); continue
		case '−', '–', '—': b.WriteRune('-'); continue
		}
		if r < 128 { b.WriteRune(r); continue }
		b.WriteRune(stripMark(r))
	}
	return b.String()
}

var vnFrom = []rune("àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹ")
var vnTo = []rune("aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyy")

func stripMark(r rune) rune {
	l := unicode.ToLower(r)
	for i, f := range vnFrom {
		if f == l {
			if l != r { return unicode.ToUpper(vnTo[i]) }
			return vnTo[i]
		}
	}
	return '?'
}

func wrapText(s string, cols int) []string {
	var out []string
	for utf8.RuneCountInString(s) > cols {
		cut := cols
		if i := strings.LastIndex(s[:cols], " "); i > cols/3 { cut = i }
		out = append(out, strings.TrimRight(s[:cut], " "))
		s = strings.TrimLeft(s[cut:], " ")
	}
	return append(out, s)
}

// htmlToLines đổi HTML phiếu thành các dòng chữ rộng `cols` ký tự.
func htmlToLines(body string, cols int) []string {
	body = reSvg.ReplaceAllString(body, "")
	body = reHr.ReplaceAllString(body, "\n\x01\n")
	body = reCell.ReplaceAllString(body, "\x00")
	body = reBr.ReplaceAllString(body, "\n")
	body = reTags.ReplaceAllString(body, "")
	body = html.UnescapeString(body)
	var out []string
	for _, raw := range strings.Split(body, "\n") {
		if strings.TrimSpace(raw) == "\x01" { out = append(out, strings.Repeat("-", cols)); continue }
		cells := []string{}
		for _, c := range strings.Split(raw, "\x00") {
			c = strings.TrimSpace(reSpace.ReplaceAllString(vnASCII(c), " "))
			if c != "" { cells = append(cells, c) }
		}
		switch len(cells) {
		case 0:
		case 1:
			out = append(out, wrapText(cells[0], cols)...)
		default:
			left, right := strings.Join(cells[:len(cells)-1], " "), cells[len(cells)-1]
			if utf8.RuneCountInString(left)+utf8.RuneCountInString(right)+1 <= cols {
				out = append(out, left+strings.Repeat(" ", cols-utf8.RuneCountInString(left)-utf8.RuneCountInString(right))+right)
			} else {
				out = append(out, wrapText(left, cols)...)
				out = append(out, strings.Repeat(" ", max(0, cols-utf8.RuneCountInString(right)))+right)
			}
		}
	}
	return out
}

func escposText(lines []string, feed int, cut, drawer bool) []byte {
	var o bytes.Buffer
	o.Write([]byte{0x1B, 0x40}) // ESC @
	for _, l := range lines { o.WriteString(l); o.WriteByte('\n') }
	if feed < 0 { feed = 0 }
	if feed > 30 { feed = 30 }
	o.Write([]byte{0x1B, 0x64, byte(feed)})
	if cut { o.Write([]byte{0x1D, 0x56, 0x42, 0x00}) }
	if drawer { o.Write([]byte{0x1B, 0x70, 0x00, 0x19, 0xFA}) }
	return o.Bytes()
}
