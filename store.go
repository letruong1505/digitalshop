package main

// Lưu trữ: toàn bộ dữ liệu nằm trong bộ nhớ, mỗi thay đổi được ghi nối vào
// nhật ký (journal.jsonl) và fsync ngay. Khi khởi động: đọc ảnh chụp
// (data.json) + phát lại nhật ký, rồi gộp thành ảnh chụp mới.

import (
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type UnitConv struct {
	Name    string  `json:"name"`
	Ratio   float64 `json:"ratio"`
	Price   float64 `json:"price"`
	Barcode string  `json:"barcode,omitempty"`
}

type Lot struct {
	ID  string  `json:"id"`
	Qty float64 `json:"qty"`
	Exp string  `json:"exp"`
	In  string  `json:"in"`
	Ref string  `json:"ref,omitempty"`
}

type ComboItem struct {
	PID string  `json:"pid"`
	Qty float64 `json:"qty"`
}

type Product struct {
	ID       string      `json:"id"`
	Code     string      `json:"code"`
	Name     string      `json:"name"`
	Group    string      `json:"group"`
	Unit     string      `json:"unit"`
	Barcode  string      `json:"barcode"`
	Location string      `json:"location"`
	Alias    string      `json:"alias"`
	Price    float64     `json:"price"`
	Cost     float64     `json:"cost"`
	Stock    float64     `json:"stock"`
	Min      float64     `json:"min"`
	Units    []UnitConv  `json:"units"`
	Lots     []Lot       `json:"lots"`
	IsCombo  bool        `json:"isCombo"`
	Items    []ComboItem `json:"items"`
	Hidden   bool        `json:"hidden"`
	Created  int64       `json:"created"`
	Updated  int64       `json:"updated"`
}

type Party struct {
	ID      string  `json:"id"`
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Addr    string  `json:"addr"`
	Note    string  `json:"note"`
	TaxCode string  `json:"taxCode,omitempty"` // mã số thuế (NCC lấy từ hóa đơn đầu vào)
	Debt    float64 `json:"debt"`
	Created int64   `json:"created"`
	Last    int64   `json:"last"`
}

type Line struct {
	PID   string  `json:"pid"`
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Ratio float64 `json:"ratio"`
	Qty   float64 `json:"qty"`
	Price float64 `json:"price"`
	Cost  float64 `json:"cost"` // giá vốn cho 1 đơn vị (theo Unit)
	Exp   string  `json:"exp,omitempty"`
}

type Invoice struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Time     int64   `json:"time"`
	Lines    []Line  `json:"lines"`
	Sub      float64 `json:"sub"`
	Disc     float64 `json:"disc"`
	Total    float64 `json:"total"`
	Cost     float64 `json:"cost"`
	Cash     float64 `json:"cash"`
	Bank     float64 `json:"bank"`
	DebtAmt  float64 `json:"debt"`
	Given    float64 `json:"given"`
	CustID   string  `json:"custId"`
	CustName string  `json:"custName"`
	Staff    string  `json:"staff"`
	Note     string  `json:"note"`
	OrderID  string  `json:"orderId,omitempty"`
	Src      string  `json:"src,omitempty"` // "import" = lịch sử nhập từ phần mềm khác
	Void     bool    `json:"void"`
	VoidAt   int64   `json:"voidAt,omitempty"`
}

type Receipt struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Time     int64   `json:"time"`
	SuppID   string  `json:"suppId"`
	SuppName string  `json:"suppName"`
	Lines    []Line  `json:"lines"`
	Sub      float64 `json:"sub"`
	Disc     float64 `json:"disc"`
	Total    float64 `json:"total"`
	Paid     float64 `json:"paid"`
	Method   string  `json:"method"`
	DebtAmt  float64 `json:"debt"`
	Staff    string  `json:"staff"`
	Note     string  `json:"note"`
	Void     bool    `json:"void"`
}

type CountLine struct {
	PID    string  `json:"pid"`
	Name   string  `json:"name"`
	Unit   string  `json:"unit"`
	Before float64 `json:"before"`
	After  float64 `json:"after"`
	Cost   float64 `json:"cost"`
}

type Count struct {
	ID    string      `json:"id"`
	Code  string      `json:"code"`
	Time  int64       `json:"time"`
	Lines []CountLine `json:"lines"`
	Note  string      `json:"note"`
	Kind  string      `json:"kind"` // "count" | "dispose"
	Staff string      `json:"staff"`
}

type CashEntry struct {
	ID        string  `json:"id"`
	Code      string  `json:"code"`
	Time      int64   `json:"time"`
	Kind      string  `json:"kind"`   // thu | chi | adj
	Method    string  `json:"method"` // cash | bank
	Amount    float64 `json:"amount"`
	Cat       string  `json:"cat"`
	PartyType string  `json:"partyType"` // cust | supp | ""
	PartyID   string  `json:"partyId"`
	PartyName string  `json:"partyName"`
	Note      string  `json:"note"`
	Staff     string  `json:"staff"`
	Void      bool    `json:"void"`
}

type OrderLine struct {
	PID   string  `json:"pid"`
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Ratio float64 `json:"ratio"`
	Qty   float64 `json:"qty"`
	Price float64 `json:"price"`
}

type Order struct {
	ID        string      `json:"id"`
	Code      string      `json:"code"`
	Time      int64       `json:"time"`
	CustID    string      `json:"custId"`
	CustName  string      `json:"custName"`
	Phone     string      `json:"phone"`
	Lines     []OrderLine `json:"lines"`
	Raw       string      `json:"raw"`
	Due       string      `json:"due"`
	DueTime   string      `json:"dueTime"`
	Status    string      `json:"status"` // new | done | cancel
	Note      string      `json:"note"`
	InvoiceID string      `json:"invoiceId"`
	Staff     string      `json:"staff"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"` // owner | staff
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type Settings struct {
	ShopName       string  `json:"shopName"`
	Phone          string  `json:"phone"`
	Addr           string  `json:"addr"`
	BankBIN        string  `json:"bankBin"`
	BankName       string  `json:"bankName"`
	Acct           string  `json:"acct"`
	Holder         string  `json:"holder"`
	Paper          string  `json:"paper"`
	AutoPrint      bool    `json:"autoPrint"`
	PrintFeed      int     `json:"printFeed"` // số dòng trống cuối phiếu in nhiệt (1–30)
	Footer         string  `json:"footer"`
	ExpWarn        int     `json:"expWarn"`
	ExpDanger      int     `json:"expDanger"`
	MinMargin      float64 `json:"minMargin"`
	ApiKey         string  `json:"apiKey,omitempty"`
	StaffSeeCost   bool    `json:"staffSeeCost"`
	StaffEditPrice bool    `json:"staffEditPrice"`
	SellNegative   bool    `json:"sellNegative"`
	BlockExpired   bool    `json:"blockExpired"`
	BackupDir2     string  `json:"backupDir2"`
	PayProvider    string  `json:"payProvider"`
	SepayToken     string  `json:"sepayToken,omitempty"`
	PayosClientID  string  `json:"payosClientId,omitempty"`
	PayosAPIKey    string  `json:"payosApiKey,omitempty"`
	PayosChecksum  string  `json:"payosChecksum,omitempty"`
	AutoBankDone   bool    `json:"autoBankDone"`
	DisplayKey     string  `json:"displayKey"`
	TermsVersion   string  `json:"termsVersion"`
	TermsAt        int64   `json:"termsAt"`
	TermsBy        string  `json:"termsBy"`
	UpdateOff      bool    `json:"updateOff"` // tắt tự kiểm tra cập nhật hằng ngày
	GdtUser        string  `json:"gdtUser"` // tài khoản (MST) đăng nhập hoadondientu.gdt.gov.vn; không lưu mật khẩu
}

type State struct {
	Products  map[string]*Product   `json:"products"`
	Customers map[string]*Party     `json:"customers"`
	Suppliers map[string]*Party     `json:"suppliers"`
	Invoices  map[string]*Invoice   `json:"invoices"`
	Receipts  map[string]*Receipt   `json:"receipts"`
	Counts    map[string]*Count     `json:"counts"`
	Cash      map[string]*CashEntry `json:"cash"`
	Orders    map[string]*Order     `json:"orders"`
	Users     map[string]*User      `json:"users"`
	InInvs    map[string]*InInvoice `json:"inInvoices"`
	InMap     map[string]*InMap     `json:"inMap"`
	Settings  Settings              `json:"settings"`
	Counters  map[string]int        `json:"counters"`
}

func newState() *State {
	return &State{
		Products: map[string]*Product{}, Customers: map[string]*Party{}, Suppliers: map[string]*Party{},
		Invoices: map[string]*Invoice{}, Receipts: map[string]*Receipt{}, Counts: map[string]*Count{},
		Cash: map[string]*CashEntry{}, Orders: map[string]*Order{}, Users: map[string]*User{},
		InInvs: map[string]*InInvoice{}, InMap: map[string]*InMap{},
		Counters: map[string]int{},
		Settings: Settings{Paper: "80", ExpWarn: 30, ExpDanger: 7, MinMargin: 10, Footer: "Cảm ơn quý khách, hẹn gặp lại!", BlockExpired: true},
	}
}

func (s *State) fill() {
	if s.Products == nil { s.Products = map[string]*Product{} }
	if s.Customers == nil { s.Customers = map[string]*Party{} }
	if s.Suppliers == nil { s.Suppliers = map[string]*Party{} }
	if s.Invoices == nil { s.Invoices = map[string]*Invoice{} }
	if s.Receipts == nil { s.Receipts = map[string]*Receipt{} }
	if s.Counts == nil { s.Counts = map[string]*Count{} }
	if s.Cash == nil { s.Cash = map[string]*CashEntry{} }
	if s.Orders == nil { s.Orders = map[string]*Order{} }
	if s.Users == nil { s.Users = map[string]*User{} }
	if s.InInvs == nil { s.InInvs = map[string]*InInvoice{} }
	if s.InMap == nil { s.InMap = map[string]*InMap{} }
	if s.Counters == nil { s.Counters = map[string]int{} }
	if s.Settings.Paper == "" { s.Settings.Paper = "80" }
	if s.Settings.ExpWarn == 0 { s.Settings.ExpWarn = 30 }
	if s.Settings.ExpDanger == 0 { s.Settings.ExpDanger = 7 }
}

// ---------- nhật ký ----------

type rec struct {
	K   string          `json:"k"`
	ID  string          `json:"id"`
	V   json.RawMessage `json:"v,omitempty"`
	Del bool            `json:"del,omitempty"`
}

type Tx struct{ recs []rec }

func (t *Tx) put(kind, id string, v any) {
	b, err := json.Marshal(v)
	if err != nil { panic(err) }
	t.recs = append(t.recs, rec{K: kind, ID: id, V: b})
}
func (t *Tx) del(kind, id string) { t.recs = append(t.recs, rec{K: kind, ID: id, Del: true}) }

type DB struct {
	mu      sync.Mutex
	dir     string
	S       *State
	jf      *os.File
	jsize   int64
	Version int64
}

func OpenDB(dir string) (*DB, error) {
	if err := os.MkdirAll(filepath.Join(dir, "SaoLuu"), 0o755); err != nil { return nil, err }
	db := &DB{dir: dir, S: newState(), Version: time.Now().UnixMilli()}
	snap := filepath.Join(dir, "data.json")
	if b, err := os.ReadFile(snap); err == nil && len(b) > 0 {
		st := newState()
		if err := json.Unmarshal(b, st); err != nil {
			return nil, fmt.Errorf("file dữ liệu bị hỏng (%v). Hãy khôi phục từ thư mục SaoLuu", err)
		}
		st.fill()
		db.S = st
	}
	jpath := filepath.Join(dir, "journal.jsonl")
	if f, err := os.Open(jpath); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 64<<20)
		n := 0
		for sc.Scan() {
			var r rec
			if json.Unmarshal(sc.Bytes(), &r) != nil { continue } // dòng ghi dở khi mất điện
			db.apply(r)
			n++
		}
		f.Close()
	}
	db.S.fill()
	if err := db.snapshotLocked(); err != nil { return nil, err }
	return db, nil
}

func (db *DB) apply(r rec) {
	s := db.S
	switch r.K {
	case "settings":
		var v Settings
		if json.Unmarshal(r.V, &v) == nil { s.Settings = v }
		return
	case "counters":
		var v map[string]int
		if json.Unmarshal(r.V, &v) == nil { s.Counters = v }
		return
	}
	switch r.K {
	case "product":
		applyMap(s.Products, r)
	case "customer":
		applyMap(s.Customers, r)
	case "supplier":
		applyMap(s.Suppliers, r)
	case "invoice":
		applyMap(s.Invoices, r)
	case "receipt":
		applyMap(s.Receipts, r)
	case "count":
		applyMap(s.Counts, r)
	case "cash":
		applyMap(s.Cash, r)
	case "order":
		applyMap(s.Orders, r)
	case "user":
		applyMap(s.Users, r)
	case "inInvoice":
		applyMap(s.InInvs, r)
	case "inMap":
		applyMap(s.InMap, r)
	}
}

func applyMap[T any](m map[string]*T, r rec) {
	if r.Del { delete(m, r.ID); return }
	v := new(T)
	if json.Unmarshal(r.V, v) == nil { m[r.ID] = v }
}

func (db *DB) snapshotLocked() error {
	if db.jf != nil { db.jf.Close(); db.jf = nil }
	b, err := json.Marshal(db.S)
	if err != nil { return err }
	tmp := filepath.Join(db.dir, "data.json.tmp")
	f, err := os.Create(tmp)
	if err != nil { return err }
	if _, err := f.Write(b); err != nil { f.Close(); return err }
	if err := f.Sync(); err != nil { f.Close(); return err }
	f.Close()
	if err := os.Rename(tmp, filepath.Join(db.dir, "data.json")); err != nil { return err }
	jf, err := os.OpenFile(filepath.Join(db.dir, "journal.jsonl"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil { return err }
	db.jf = jf
	db.jsize = 0
	return nil
}

var ErrUser = fmt.Errorf("")

type userErr struct{ msg string }

func (e userErr) Error() string { return e.msg }
func uerr(f string, a ...any) error { return userErr{fmt.Sprintf(f, a...)} }

// Do chạy một giao dịch: fn sửa State và khai báo bản ghi qua Tx.
func (db *DB) Do(fn func(s *State, t *Tx) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	t := &Tx{}
	if err := fn(db.S, t); err != nil { return err }
	if len(t.recs) == 0 { return nil }
	var buf strings.Builder
	for _, r := range t.recs {
		b, _ := json.Marshal(r)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	n, err := db.jf.WriteString(buf.String())
	if err != nil { return fmt.Errorf("không ghi được dữ liệu xuống ổ đĩa: %v", err) }
	db.jf.Sync()
	db.jsize += int64(n)
	db.Version++
	if db.jsize > 8<<20 { _ = db.snapshotLocked() }
	return nil
}

func (db *DB) Read(fn func(s *State)) { db.mu.Lock(); defer db.mu.Unlock(); fn(db.S) }

func (db *DB) Close() { db.mu.Lock(); defer db.mu.Unlock(); _ = db.snapshotLocked(); if db.jf != nil { db.jf.Close() } }

// ---------- sao lưu ----------

func (db *DB) Backup(tag string) (string, error) {
	db.mu.Lock()
	b, err := json.Marshal(db.S)
	db.mu.Unlock()
	if err != nil { return "", err }
	name := "saoluu-" + time.Now().Format("2006-01-02_1504") + tag + ".json.gz"
	p := filepath.Join(db.dir, "SaoLuu", name)
	f, err := os.Create(p)
	if err != nil { return "", err }
	zw := gzip.NewWriter(f)
	zw.Write(b)
	zw.Close()
	f.Sync()
	f.Close()
	db.pruneBackups(40)
	return name, nil
}

func (db *DB) Backups() []map[string]any {
	ents, _ := os.ReadDir(filepath.Join(db.dir, "SaoLuu"))
	out := []map[string]any{}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".json.gz") { continue }
		info, err := e.Info()
		if err != nil { continue }
		out = append(out, map[string]any{"name": e.Name(), "size": info.Size(), "time": info.ModTime().UnixMilli()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) > out[j]["name"].(string) })
	return out
}

func (db *DB) pruneBackups(keep int) {
	list := db.Backups()
	for i := keep; i < len(list); i++ {
		os.Remove(filepath.Join(db.dir, "SaoLuu", list[i]["name"].(string)))
	}
}

func (db *DB) HasBackupToday() bool {
	today := "saoluu-" + time.Now().Format("2006-01-02")
	for _, b := range db.Backups() {
		if strings.HasPrefix(b["name"].(string), today) { return true }
	}
	return false
}

func (db *DB) Restore(name string) error {
	if strings.ContainsAny(name, `/\`) || !strings.HasSuffix(name, ".json.gz") { return uerr("Tên bản sao lưu không hợp lệ") }
	f, err := os.Open(filepath.Join(db.dir, "SaoLuu", name))
	if err != nil { return uerr("Không mở được bản sao lưu") }
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil { return uerr("Bản sao lưu bị hỏng") }
	b, err := io.ReadAll(zr)
	if err != nil { return uerr("Bản sao lưu bị hỏng") }
	return db.RestoreJSON(b)
}

func (db *DB) RestoreJSON(b []byte) error {
	st := newState()
	if err := json.Unmarshal(b, st); err != nil { return uerr("File sao lưu không đúng định dạng") }
	st.fill()
	if len(st.Users) == 0 { return uerr("File sao lưu không có tài khoản, không dùng được") }
	db.Backup("-truoc-khoi-phuc")
	db.mu.Lock()
	defer db.mu.Unlock()
	db.S = st
	db.Version++
	return db.snapshotLocked()
}

// ---------- tiện ích ----------

func newID() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%x%s", time.Now().UnixMilli(), hex.EncodeToString(b))
}

func (s *State) nextCode(prefix string, width int) string {
	s.Counters[prefix]++
	return fmt.Sprintf("%s%0*d", prefix, width, s.Counters[prefix])
}

func r3(x float64) float64 { return math.Round(x*1000) / 1000 }
func r0(x float64) float64 { return math.Round(x) }

func nowMs() int64 { return time.Now().UnixMilli() }
func today() string { return time.Now().Format("2006-01-02") }
func dayOf(ms int64) string { return time.UnixMilli(ms).Format("2006-01-02") }
