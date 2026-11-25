// Ký bản cập nhật Digital Shop cho quy trình tự động (GitHub Actions).
// Dùng: kyupdate -exe dist/TapHoaX.exe -version 3.1.4 -url https://.../DigitalShop-3.1.4.exe -notes "..." -out latest-shop.json
// Khóa ký lấy từ biến môi trường UPDATE_SIGN_KEY (nội dung file khoa-ky-capnhat.key), không bao giờ ghi ra log.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	exe := flag.String("exe", "", "file exe")
	ver := flag.String("version", "", "phiên bản")
	url := flag.String("url", "", "địa chỉ tải exe (https)")
	notes := flag.String("notes", "", "ghi chú")
	out := flag.String("out", "latest-shop.json", "file kết quả")
	wantPub := flag.String("pub", "", "khóa công khai mà bản chạy hiện hành tin (nếu có, kiểm tra khớp)")
	flag.Parse()
	key := strings.TrimSpace(os.Getenv("UPDATE_SIGN_KEY"))
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || (len(raw) != ed25519.SeedSize && len(raw) != ed25519.PrivateKeySize) { die("UPDATE_SIGN_KEY không đúng định dạng") }
	var priv ed25519.PrivateKey
	if len(raw) == ed25519.SeedSize { priv = ed25519.NewKeyFromSeed(raw) } else { priv = ed25519.PrivateKey(raw) }
	pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	if *wantPub != "" && pub != *wantPub { die("Khóa ký không khớp khóa công khai trong phần mềm (" + pub + "); khách sẽ không cập nhật được") }
	if *exe == "" || *ver == "" || !strings.HasPrefix(*url, "https://") { die("thiếu -exe, -version hoặc -url https") }
	f, err := os.Open(*exe)
	if err != nil { die(err.Error()) }
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil { die(err.Error()) }
	sum := hex.EncodeToString(h.Sum(nil))
	msg := "shop|" + *ver + "|" + sum + "|" + *url
	m := map[string]any{"product": "shop", "version": *ver, "url": *url, "sha256": sum, "size": n, "notes": *notes, "sig": base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(msg)))}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(*out, b, 0o644); err != nil { die(err.Error()) }
	fmt.Println("Đã ký", *ver, "sha256", sum[:12]+"…", n, "byte; khóa công khai", pub)
}

func die(s string) { fmt.Fprintln(os.Stderr, "LỖI:", s); os.Exit(1) }
