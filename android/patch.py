"""Chỉnh dự án Android do Capacitor sinh ra: cho phép http trong mạng nội bộ, giữ màn hình sáng,
đặt số phiên bản, tên ứng dụng và khóa ký. Chạy sau `cap sync`."""
import re, sys, pathlib, os

ver, code = sys.argv[1], sys.argv[2]
JAVA_UPDATER = r'''package vn.namlinh.digitalshop;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import android.webkit.JavascriptInterface;
import android.webkit.WebView;
import androidx.core.content.FileProvider;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;

/** Tải APK mới ngay trong ứng dụng rồi mở trình cài đặt của Android (không nhảy ra trình duyệt). */
public class DsUpdater {
    private final Activity act;
    private final WebView web;
    private volatile boolean busy = false;
    private File pending = null;

    public DsUpdater(Activity a, WebView w) { act = a; web = w; }

    private void js(final String code) { act.runOnUiThread(new Runnable() { public void run() { web.evaluateJavascript(code, null); } }); }
    private void state(String st, String msg) { js("window.dsApkState&&window.dsApkState(" + q(st) + "," + q(msg) + ")"); }
    private static String q(String s) { return "'" + (s == null ? "" : s.replace("\\", "\\\\").replace("'", "\\'")) + "'"; }

    @JavascriptInterface
    public boolean available() { return true; }

    /** Tải và cài. Lần đầu Android hỏi cho phép "cài ứng dụng không rõ nguồn" cho Digital Shop. */
    @JavascriptInterface
    public void install(final String url) {
        if (busy) return;
        busy = true;
        new Thread(new Runnable() { public void run() {
            try {
                File dir = new File(act.getCacheDir(), "apk");
                dir.mkdirs();
                File[] old = dir.listFiles();
                if (old != null) for (File f : old) f.delete();
                File out = new File(dir, "DigitalShop-update.apk");
                HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
                c.setConnectTimeout(15000); c.setReadTimeout(30000); c.setInstanceFollowRedirects(true);
                int code = c.getResponseCode();
                if (code != 200) { state("error", "Máy chủ tải trả về mã " + code); return; }
                long total = c.getContentLengthLong(), got = 0; int last = -1;
                InputStream in = c.getInputStream();
                FileOutputStream fo = new FileOutputStream(out);
                byte[] buf = new byte[32768]; int n;
                while ((n = in.read(buf)) > 0) {
                    fo.write(buf, 0, n); got += n;
                    if (total > 0) { int pct = (int) (got * 100 / total); if (pct != last) { last = pct; js("window.dsApkProgress&&window.dsApkProgress(" + pct + ")"); } }
                }
                fo.close(); in.close();
                if (got < 100000) { state("error", "File tải về bị thiếu"); return; }
                pending = out;
                launch();
            } catch (Exception e) {
                state("error", "Không tải được: " + e.getMessage());
            } finally { busy = false; }
        } }).start();
    }

    /** Gọi lại sau khi người dùng đã cho phép cài từ nguồn này. */
    @JavascriptInterface
    public void retryInstall() { if (pending != null && pending.exists()) launch(); else state("error", "Chưa có file, hãy bấm tải lại"); }

    private void launch() {
        try {
            if (Build.VERSION.SDK_INT >= 26 && !act.getPackageManager().canRequestPackageInstalls()) {
                state("permission", "Cần cho phép Digital Shop cài ứng dụng");
                Intent s = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:" + act.getPackageName()));
                s.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                act.startActivity(s);
                return;
            }
            Uri u = FileProvider.getUriForFile(act, act.getPackageName() + ".fileprovider", pending);
            Intent i = new Intent(Intent.ACTION_VIEW);
            i.setDataAndType(u, "application/vnd.android.package-archive");
            i.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
            act.startActivity(i);
            state("installing", "Đang mở trình cài đặt");
        } catch (Exception e) { state("error", "Không mở được trình cài đặt: " + e.getMessage()); }
    }
}
'''
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
                   "public class MainActivity extends BridgeActivity {\n    @Override\n    public void onCreate(Bundle savedInstanceState) {\n        super.onCreate(savedInstanceState);\n        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);\n        // cập nhật ứng dụng ngay trong app: tải APK và mở trình cài đặt của Android\n        this.bridge.getWebView().addJavascriptInterface(new DsUpdater(this, this.bridge.getWebView()), \"DSUpdater\");\n        // bấm liên kết tải .apk: chuyển cho trình duyệt hệ thống tải về (WebView mặc định không tải file)\n        this.bridge.getWebView().setDownloadListener(new DownloadListener() {\n            @Override\n            public void onDownloadStart(String url, String userAgent, String contentDisposition, String mimeType, long contentLength) {\n                try {\n                    Intent i = new Intent(Intent.ACTION_VIEW, Uri.parse(url));\n                    i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);\n                    startActivity(i);\n                } catch (Exception e) { }\n            }\n        });\n    }\n}", t, count=1)
        j.write_text(t, encoding="utf-8")

# 2b. Cập nhật trong ứng dụng: quyền cài gói, lớp DsUpdater, và cho FileProvider của Capacitor đọc thư mục cache
if "REQUEST_INSTALL_PACKAGES" not in mf.read_text(encoding="utf-8"):
    m = mf.read_text(encoding="utf-8")
    m = m.replace("<application", '<uses-permission android:name="android.permission.REQUEST_INSTALL_PACKAGES" />\n    <application', 1)
    mf.write_text(m, encoding="utf-8")
for j in (root / "app/src/main/java").rglob("MainActivity.java"):
    (j.parent / "DsUpdater.java").write_text(JAVA_UPDATER, encoding="utf-8")
fp = root / "app/src/main/res/xml/file_paths.xml"
fp.parent.mkdir(parents=True, exist_ok=True)
fp.write_text('<?xml version="1.0" encoding="utf-8"?>\n<paths xmlns:android="http://schemas.android.com/apk/res/android">\n    <external-path name="my_images" path="." />\n    <cache-path name="my_cache_images" path="." />\n</paths>\n', encoding="utf-8")

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
