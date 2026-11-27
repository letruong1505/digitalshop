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
        t = t.replace("import com.getcapacitor.BridgeActivity;", "import android.content.Intent;\nimport android.net.Uri;\nimport android.os.Bundle;\nimport android.view.WindowManager;\nimport android.webkit.DownloadListener;\nimport com.getcapacitor.BridgeActivity;", 1)
        t = re.sub(r"public class MainActivity extends BridgeActivity \{\s*\}",
                   "public class MainActivity extends BridgeActivity {\n    @Override\n    public void onCreate(Bundle savedInstanceState) {\n        super.onCreate(savedInstanceState);\n        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);\n        // bấm liên kết tải .apk: chuyển cho trình duyệt hệ thống tải về (WebView mặc định không tải file)\n        this.bridge.getWebView().setDownloadListener(new DownloadListener() {\n            @Override\n            public void onDownloadStart(String url, String userAgent, String contentDisposition, String mimeType, long contentLength) {\n                try {\n                    Intent i = new Intent(Intent.ACTION_VIEW, Uri.parse(url));\n                    i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);\n                    startActivity(i);\n                } catch (Exception e) { }\n            }\n        });\n    }\n}", t, count=1)
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
# 5. Biểu tượng ứng dụng: dùng cùng hình với bản Windows (installer/icon.png), đã dựng sẵn trong android/icons/res
import shutil
res = root / "app/src/main/res"
for d in res.glob("mipmap-*"):
    for n in ("ic_launcher.png", "ic_launcher_round.png", "ic_launcher_foreground.png", "ic_launcher.webp", "ic_launcher_round.webp", "ic_launcher_foreground.webp"):
        (d / n).unlink(missing_ok=True)
for d in res.glob("drawable*"):
    for n in ("ic_launcher_foreground.xml", "ic_launcher_background.xml"):
        (d / n).unlink(missing_ok=True)
shutil.copytree(pathlib.Path("icons/res"), res, dirs_exist_ok=True)
print("Đã chỉnh dự án Android: phiên bản", ver, "mã", code)
