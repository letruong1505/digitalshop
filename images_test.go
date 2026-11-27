package main

import (
	"encoding/base64"
	"testing"
)

func TestImages(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil { t.Fatal(err) }
	u := &User{ID: "o", Name: "Chủ", Role: "owner"}
	p, _ := db.SaveProduct(ProductReq{Product: Product{Name: "Nước", Unit: "chai", Price: 10000}}, u)
	px := make([]byte, 100)
	uri := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(px)
	if _, err := db.SetImages([]ImgItem{{PID: p.ID, Data: "data:text/html;base64,AAAA"}}); err == nil { t.Fatal("chap nhan sai loai") }
	if _, err := db.SetImages([]ImgItem{{PID: "zzz", Data: uri}}); err == nil { t.Fatal("hang khong ton tai") }
	big := make([]byte, maxImgBytes+1)
	if _, err := db.SetImages([]ImgItem{{PID: p.ID, Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(big)}}); err == nil { t.Fatal("qua lon") }
	if n, err := db.SetImages([]ImgItem{{PID: p.ID, Data: uri}}); err != nil || n != 1 { t.Fatal(err) }
	db.Read(func(s *State) { if s.Products[p.ID].Img == 0 || s.Images[p.ID] == nil { t.Fatal("chua luu") } })
	// sửa mặt hàng không làm mất ảnh
	q := *p; q.Name = "Nước suối"
	if _, err := db.SaveProduct(ProductReq{Product: q}, u); err != nil { t.Fatal(err) }
	db.Read(func(s *State) { if s.Products[p.ID].Img == 0 { t.Fatal("mat anh khi sua") } })
	// phát lại nhật ký / snapshot giữ ảnh
	dir := db.dir
	db.Close()
	db2, err := OpenDB(dir)
	if err != nil { t.Fatal(err) }
	db2.Read(func(s *State) { if s.Images[p.ID] == nil || s.Products[p.ID].Img == 0 { t.Fatal("mat anh sau khi mo lai") } })
	if err := db2.DeleteImage(p.ID); err != nil { t.Fatal(err) }
	db2.Read(func(s *State) { if s.Images[p.ID] != nil || s.Products[p.ID].Img != 0 { t.Fatal("chua xoa") } })
}
