package main

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

func (u *User) perms() []string { return rolePerms[normRole(u.Role)] }

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
