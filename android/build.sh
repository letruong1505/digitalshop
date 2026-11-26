#!/bin/sh
# Dựng APK Digital Shop (Android). Chạy trong thư mục android/. Cần Node 20+, JDK 17 và Android SDK 34.
# Biến môi trường:
#   APP_VERSION   số phiên bản, vd 3.1.5 (mặc định đọc từ main.go)
#   KEYSTORE_FILE, KEYSTORE_PASSWORD, KEY_ALIAS, KEY_PASSWORD   khóa ký (bắt buộc cho bản phát hành)
set -e
cd "$(dirname "$0")"
VER="${APP_VERSION:-$(grep -oE 'AppVersion *= *"[0-9.]+"' ../main.go | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')}"
[ -n "$VER" ] || { echo "Không đọc được số phiên bản"; exit 1; }
CODE=$(echo "$VER" | awk -F. '{printf "%d", $1*10000+$2*100+$3}')
echo "== Digital Shop Android $VER (versionCode $CODE)"

rm -rf www && mkdir www && cp -r ../web/. www/
# ghi số phiên bản vào ứng dụng để so với bản mới nhất trên máy chủ
sed -i "s|<head>|<head><script>window.DS_APK='$VER'</script>|" www/index.html
grep -q "DS_APK" www/index.html || { echo "Không ghi được số phiên bản vào index.html"; exit 1; }
npm install --no-audit --no-fund
[ -d android ] || npx cap add android
npx cap sync android
python3 patch.py "$VER" "$CODE"

cd android
chmod +x gradlew
if [ -n "$KEYSTORE_FILE" ]; then
  ./gradlew --no-daemon assembleRelease
  APK=$(ls app/build/outputs/apk/release/*.apk | head -1)
else
  echo "!! Không có khóa ký: dựng bản thử (debug), không dùng được để cập nhật đè bản cũ"
  ./gradlew --no-daemon assembleDebug
  APK=$(ls app/build/outputs/apk/debug/*.apk | head -1)
fi
mkdir -p ../../dist
cp "$APK" "../../dist/DigitalShop-Android-$VER.apk"
echo "== Xong: dist/DigitalShop-Android-$VER.apk"
