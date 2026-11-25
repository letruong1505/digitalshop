#!/bin/sh
# Build Digital Shop cho Windows (không cửa sổ đen) và tạo bộ cài. Chạy từ thư mục gốc mã nguồn.
set -e
cd "$(dirname "$0")/.."
(cd installer && x86_64-w64-mingw32-windres --preprocessor=cat -i app.rc -O coff -o ../rsrc_windows_amd64.syso \
  && x86_64-w64-mingw32-windres --preprocessor=cat -F pe-i386 -i app.rc -O coff -o ../rsrc_windows_386.syso)
mkdir -p dist
# Văn bản kèm bộ cài: UTF-8 có BOM + xuống dòng Windows; điều khoản cho trang cài đặt cần UTF-16
for f in installer/HuongDan.txt web/legal/DieuKhoanSuDung.txt web/legal/GiayPhepBenThuBa.txt; do
  { printf '\357\273\277'; sed 's/\r$//; s/$/\r/' "$f"; } > "dist/$(basename "$f")"
done
{ printf '\377\376'; sed 's/\r$//; s/$/\r/' web/legal/DieuKhoanSuDung.txt | iconv -f UTF-8 -t UTF-16LE; } > installer/DieuKhoanSuDung_utf16.txt
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/TapHoaX.exe .
GOOS=windows GOARCH=386   CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/TapHoaX-32bit.exe .
(cd installer && makensis -V2 setup.nsi)
ls -la dist
