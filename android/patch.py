"""Chỉnh dự án Android do Capacitor sinh ra: cho phép http trong mạng nội bộ, giữ màn hình sáng,
đặt số phiên bản, tên ứng dụng và khóa ký. Chạy sau `cap sync`."""
import re, sys, pathlib, os

ver, code = sys.argv[1], sys.argv[2]
root = pathlib.Path("android")

# 1. Manifest: http trong mạng nội bộ (máy chủ Digital Shop chạy http), hướng màn hình theo máy
mf = root / "app/src/main/AndroidManifest.xml"
s = mf.read_text(encoding="utf-8")
if "usesCleartextTraffic" not in s:
    s = s.replace("<application", '<application\n        android:usesCleartextTraffic="true"', 1)
mf.write_text(s, encoding="utf-8")

# 2. Giữ màn hình sáng khi đang mở (quầy bán hàng và màn hình khách không được tự tắt)
for j in (root / "app/src/main/java").rglob("MainActivity.java"):
    t = j.read_text(encoding="utf-8")
    if "FLAG_KEEP_SCREEN_ON" not in t:
        t = t.replace("import com.getcapacitor.BridgeActivity;", "import android.os.Bundle;\nimport android.view.WindowManager;\nimport com.getcapacitor.BridgeActivity;", 1)
        t = re.sub(r"public class MainActivity extends BridgeActivity \{\s*\}",
                   "public class MainActivity extends BridgeActivity {\n    @Override\n    public void onCreate(Bundle savedInstanceState) {\n        super.onCreate(savedInstanceState);\n        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);\n    }\n}", t, count=1)
        j.write_text(t, encoding="utf-8")

# 3. build.gradle: phiên bản + khóa ký
g = root / "app/build.gradle"
t = g.read_text(encoding="utf-8")
t = re.sub(r"versionCode \d+", "versionCode " + code, t, count=1)
t = re.sub(r'versionName "[^"]*"', 'versionName "' + ver + '"', t, count=1)
if os.environ.get("KEYSTORE_FILE") and "signingConfigs" not in t:
    sign = '''    signingConfigs {
        release {
            storeFile file(System.getenv("KEYSTORE_FILE"))
            storePassword System.getenv("KEYSTORE_PASSWORD")
            keyAlias System.getenv("KEY_ALIAS")
            keyPassword System.getenv("KEY_PASSWORD")
        }
    }
'''
    t = t.replace("    buildTypes {", sign + "    buildTypes {", 1)
    t = re.sub(r"(buildTypes \{\s*release \{)", r"\1\n            signingConfig signingConfigs.release", t, count=1)
g.write_text(t, encoding="utf-8")

# 4. Tên hiển thị
sx = root / "app/src/main/res/values/strings.xml"
x = sx.read_text(encoding="utf-8")
x = re.sub(r"(<string name=\"app_name\">)[^<]*(</string>)", r"\1Digital Shop\2", x)
x = re.sub(r"(<string name=\"title_activity_main\">)[^<]*(</string>)", r"\1Digital Shop\2", x)
sx.write_text(x, encoding="utf-8")
print("Đã chỉnh dự án Android: phiên bản", ver, "mã", code)
