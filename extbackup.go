package main

// Sao lưu ra ổ ngoài: USB, ổ cứng gắn ngoài, hoặc thư mục Google Drive/OneDrive.
// Mỗi bản sao lưu hằng ngày được chép thêm vào <ổ ngoài>\TapHoaX_SaoLuu khi ổ đang cắm.

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const extFolder = "TapHoaX_SaoLuu"

type ExtStatus struct {
	Dir      string `json:"dir"`
	Ok       bool   `json:"ok"`
	LastFile string `json:"lastFile"`
	LastAt   int64  `json:"lastAt"`
	Msg      string `json:"msg"`
	Files    int    `json:"files"`
}

var (
	extMu   sync.Mutex
	extStat ExtStatus
)

func extTarget(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" { return "" }
	if strings.EqualFold(filepath.Base(filepath.Clean(dir)), extFolder) { return filepath.Clean(dir) }
	return filepath.Join(dir, extFolder)
}

// checkExtDir thử ghi vào ổ ngoài; trả lỗi tiếng Việt dễ hiểu.
func checkExtDir(dir string) error {
	if strings.TrimSpace(dir) == "" { return uerr("Chưa chọn ổ USB hoặc thư mục") }
	root := strings.TrimSpace(dir)
	if _, err := os.Stat(root); err != nil { return uerr("Không thấy ổ hoặc thư mục %s. Kiểm tra USB đã cắm chưa", root) }
	t := extTarget(dir)
	if err := os.MkdirAll(t, 0o755); err != nil { return uerr("Không tạo được thư mục trên %s (USB bị khóa ghi hoặc đầy?)", root) }
	p := filepath.Join(t, ".ghi-thu")
	if err := os.WriteFile(p, []byte("ok"), 0o644); err != nil { return uerr("Không ghi được vào %s (USB bị khóa ghi hoặc đầy?)", root) }
	os.Remove(p)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil { return err }
	if _, err := io.Copy(out, in); err != nil { out.Close(); os.Remove(tmp); return err }
	if err := out.Sync(); err != nil { out.Close(); os.Remove(tmp); return err }
	out.Close()
	return os.Rename(tmp, dst)
}

// SyncExternal chép các bản sao lưu còn thiếu ra ổ ngoài (bản mới nhất trước), giữ 60 bản trên ổ ngoài.
func (db *DB) SyncExternal(force bool) ExtStatus {
	var dir string
	db.Read(func(s *State) { dir = s.Settings.BackupDir2 })
	extMu.Lock()
	defer extMu.Unlock()
	st := ExtStatus{Dir: dir, LastFile: extStat.LastFile, LastAt: extStat.LastAt}
	if strings.TrimSpace(dir) == "" { st.Msg = "Chưa cài đặt"; extStat = st; return st }
	if err := checkExtDir(dir); err != nil { st.Msg = err.Error(); extStat = st; return st }
	if force || !db.HasBackupToday() { db.Backup("") }
	t := extTarget(dir)
	have := map[string]bool{}
	ents, _ := os.ReadDir(t)
	for _, e := range ents { have[e.Name()] = true }
	list := db.Backups() // mới nhất trước
	copied := 0
	for i, b := range list {
		name := b["name"].(string)
		if have[name] { continue }
		if i > 0 && copied >= 10 { break } // lần đầu không chép quá nhiều
		if err := copyFile(filepath.Join(db.dir, "SaoLuu", name), filepath.Join(t, name)); err != nil {
			st.Msg = "Chép thất bại: " + err.Error()
			extStat = st
			return st
		}
		copied++
		if i == 0 { st.LastFile, st.LastAt = name, time.Now().UnixMilli() }
	}
	if st.LastFile == "" && len(list) > 0 && have[list[0]["name"].(string)] {
		st.LastFile = list[0]["name"].(string)
		if info, err := os.Stat(filepath.Join(t, st.LastFile)); err == nil { st.LastAt = info.ModTime().UnixMilli() }
	}
	// giữ 60 bản gần nhất trên ổ ngoài
	ents, _ = os.ReadDir(t)
	names := []string{}
	for _, e := range ents { if strings.HasSuffix(e.Name(), ".json.gz") { names = append(names, e.Name()) } }
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for i := 60; i < len(names); i++ { os.Remove(filepath.Join(t, names[i])) }
	if len(names) > 60 { names = names[:60] }
	st.Ok, st.Files = true, len(names)
	st.Msg = "Đã chép"
	extStat = st
	return st
}

func extStatus() ExtStatus { extMu.Lock(); defer extMu.Unlock(); return extStat }

type Drive struct {
	Path    string `json:"path"`
	Label   string `json:"label"`
	Kind    string `json:"kind"` // usb | disk | cloud
	FreeGB  float64 `json:"freeGB"`
	TotalGB float64 `json:"totalGB"`
}

// cloudFolders: thư mục đồng bộ đám mây thường gặp (Google Drive, OneDrive, Dropbox).
func cloudFolders() []Drive {
	out := []Drive{}
	home, _ := os.UserHomeDir()
	cands := []struct{ p, l string }{
		{filepath.Join(home, "OneDrive"), "OneDrive"},
		{filepath.Join(home, "Google Drive"), "Google Drive"},
		{filepath.Join(home, "My Drive"), "Google Drive"},
		{filepath.Join(home, "Dropbox"), "Dropbox"},
		{`G:\My Drive`, "Google Drive (ổ G)"},
		{`G:\Drive của tôi`, "Google Drive (ổ G)"},
	}
	for _, c := range cands {
		if info, err := os.Stat(c.p); err == nil && info.IsDir() { out = append(out, Drive{Path: c.p, Label: c.l, Kind: "cloud"}) }
	}
	return out
}
