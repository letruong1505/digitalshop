package main

import (
	"net"
	"net/http"
	"strings"
)

// Phân quyền theo vai trò. Quyền: admin (cài đặt, người dùng, bản quyền, sao lưu), manage (xóa/hủy, điều chỉnh công nợ),
// sell (bán hàng, đơn đặt), stock (hàng hóa, nhập hàng, kiểm kho), money (sổ quỹ, công nợ), report (báo cáo, xuất Excel),
// tax (hóa đơn đầu vào), import (nhập từ Excel), cost (xem giá vốn, lợi nhuận), price (sửa giá khi bán).
var rolePerms = map[string][]string{
	"owner":      {"admin", "manage", "sell", "stock", "money", "report", "tax", "import", "cost", "price"},
	"manager":    {"manage", "sell", "stock", "money", "report", "tax", "import", "cost", "price"},
	"cashier":    {"sell"},
	"warehouse":  {"stock", "cost"},
	"accountant": {"money", "report", "tax", "cost"},
}

var roleNames = map[string]string{"owner": "Chủ cửa hàng", "manager": "Quản lý", "cashier": "Thu ngân", "warehouse": "Thủ kho", "accountant": "Kế toán"}

// normRole đưa vai trò cũ ("staff") về thu ngân.
func normRole(r string) string {
	if _, ok := rolePerms[r]; ok { return r }
	return "cashier"
}

// allPerms: các quyền có thể gán riêng cho nhân viên. Quyền "admin" chỉ thuộc chủ cửa hàng.
var allPerms = []string{"manage", "sell", "stock", "money", "report", "tax", "import", "cost", "price"}

func (u *User) perms() []string {
	r := normRole(u.Role)
	if r == "owner" || u.Perms == nil { return rolePerms[r] }
	out := []string{}
	for _, p := range allPerms { for _, h := range u.Perms { if h == p { out = append(out, p); break } } }
	return out
}

// cleanPerms giữ lại quyền hợp lệ; trả nil nếu giống hệt quyền mặc định của vai trò (để đổi vai trò sau này vẫn có tác dụng).
func cleanPerms(role string, in []string) []string {
	have := map[string]bool{}
	for _, p := range in { have[p] = true }
	out := []string{}
	for _, p := range allPerms { if have[p] { out = append(out, p) } }
	def := map[string]bool{}
	for _, p := range rolePerms[normRole(role)] { if p != "admin" { def[p] = true } }
	same := len(def) == len(out)
	for _, p := range out { if !def[p] { same = false } }
	if same { return nil }
	return out
}

// lanOnlyReq: yêu cầu có đến từ mạng nội bộ không. Máy chạy sau proxy/tunnel (Cloudflare Tunnel, nginx…) sẽ đến từ
// 127.0.0.1 kèm địa chỉ thật trong header, nên đọc header đó. Dải 100.64.0.0/10 (Tailscale, CGNAT) coi là từ xa.
func isLANReq(r *http.Request) bool {
	h, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(h)
	if ip == nil { return false }
	if !privIP(ip) { return false }
	for _, k := range []string{"Cf-Connecting-Ip", "X-Real-Ip", "X-Forwarded-For", "Forwarded"} {
		v := strings.TrimSpace(r.Header.Get(k))
		if v == "" { continue }
		if k == "X-Forwarded-For" { v = strings.TrimSpace(strings.Split(v, ",")[0]) }
		if k == "Forwarded" { return false }
		x := net.ParseIP(v)
		return x != nil && privIP(x)
	}
	return true
}

func privIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xC0 == 64 { return false }
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// can nhận một quyền, hoặc nhiều quyền cách nhau bằng "|" (có một trong các quyền là được). Chuỗi rỗng = ai đăng nhập cũng được.
func (u *User) can(perm string) bool {
	if perm == "" { return true }
	for _, want := range splitPerm(perm) {
		for _, have := range u.perms() { if have == want { return true } }
	}
	return false
}

func splitPerm(p string) []string {
	out := []string{}
	cur := ""
	for _, c := range p {
		if c == '|' { out = append(out, cur); cur = ""; continue }
		cur += string(c)
	}
	return append(out, cur)
}

func (u *User) seeCost(set *Settings) bool { return u.can("cost") || (set != nil && set.StaffSeeCost) }
func (u *User) editPrice(set *Settings) bool { return u.can("price") || (set != nil && set.StaffEditPrice) }
