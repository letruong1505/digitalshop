// Hai trang HTML (viết bằng String.raw; trong trang không dùng dấu ` và ${ ).

export const BUY_PAGE = String.raw`<!doctype html>
<html lang="vi">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Mua bản quyền · Nam Linh Group</title>
<style>
:root{--bg:#f4f6f8;--card:#fff;--ink:#14202b;--mute:#5d6b78;--line:#dbe2e8;--pri:#0b6b5c;--pri2:#0e8a77;--bad:#b3261e;--ok:#12724a;--sel:#e6f4f1}
@media (prefers-color-scheme:dark){:root{--bg:#10171d;--card:#18222b;--ink:#e8eef2;--mute:#9aa9b6;--line:#2b3946;--pri:#2bb39b;--pri2:#3cc9b0;--bad:#ff8a80;--ok:#5fd6a0;--sel:#16342f}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
.wrap{max-width:640px;margin:0 auto;padding:20px 16px 48px}
.brand{font-size:13px;letter-spacing:.08em;text-transform:uppercase;color:var(--mute);margin:4px 0 2px}
h1{font-size:24px;margin:0 0 14px}
.card{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:16px;margin:0 0 14px}
h2{font-size:15px;margin:0 0 10px}
label{display:block;font-size:14px;color:var(--mute);margin:10px 0 4px}
input[type=text],input[type=tel],input[type=email],select{width:100%;padding:11px 12px;border:1px solid var(--line);border-radius:10px;background:var(--bg);color:var(--ink);font:inherit}
input:focus,select:focus{outline:2px solid var(--pri2);outline-offset:1px}
.mono{font-family:ui-monospace,Consolas,monospace;letter-spacing:.04em}
.plans{display:grid;grid-template-columns:repeat(3,1fr);gap:8px}
.plan{border:2px solid var(--line);border-radius:12px;padding:12px 8px;text-align:center;cursor:pointer;background:var(--bg);color:inherit;font:inherit}
.plan b{display:block;font-size:15px}
.plan span{display:block;font-size:17px;font-weight:700;margin-top:4px}
.plan small{display:block;color:var(--mute);font-size:12px;margin-top:2px}
.plan[aria-pressed=true]{border-color:var(--pri);background:var(--sel)}
.hint{font-size:13px;color:var(--mute);margin:4px 0 0}
.err{color:var(--bad);font-size:14px;margin:10px 0 0;min-height:1em}
.btn{width:100%;padding:13px;border:0;border-radius:12px;background:var(--pri);color:#fff;font:inherit;font-weight:700;cursor:pointer}
.btn:disabled{opacity:.55;cursor:default}
.btn.ghost{background:transparent;color:var(--pri);border:1px solid var(--pri);font-weight:600}
.chk{display:flex;gap:10px;align-items:flex-start;margin:14px 0 4px;font-size:14px;color:var(--ink)}
.chk input{margin-top:4px;width:18px;height:18px;flex:none}
details{font-size:13px;color:var(--mute);margin:6px 0 0}
details ol{margin:6px 0 0;padding-left:20px}
.qr{display:grid;place-items:center}
.qr img{width:min(300px,100%);background:#fff;border-radius:12px;padding:6px}
.kv{display:grid;grid-template-columns:auto 1fr auto;gap:6px 10px;align-items:center;font-size:15px;margin-top:12px}
.kv .k{color:var(--mute)}
.kv .v{font-weight:600;word-break:break-all}
.cp{border:1px solid var(--line);background:transparent;color:var(--pri);border-radius:8px;padding:3px 9px;font:inherit;font-size:13px;cursor:pointer}
.wait{display:flex;gap:10px;align-items:center;justify-content:center;color:var(--mute);margin:14px 0 2px}
.spin{width:16px;height:16px;border:3px solid var(--line);border-top-color:var(--pri);border-radius:50%;animation:s 1s linear infinite}
@keyframes s{to{transform:rotate(360deg)}}
.warn{background:var(--sel);border-radius:10px;padding:10px 12px;font-size:14px;margin-top:12px}
.done h2{color:var(--ok);font-size:18px}
textarea{width:100%;min-height:96px;padding:10px;border:1px solid var(--line);border-radius:10px;background:var(--bg);color:var(--ink);font:13px/1.4 ui-monospace,Consolas,monospace;word-break:break-all}
.hide{display:none}
@media (prefers-reduced-motion:reduce){.spin{animation:none}}
</style>
</head>
<body>
<div class="wrap">
  <div class="brand">Nam Linh Group</div>
  <h1 id="title">Mua bản quyền phần mềm</h1>

  <div id="viewForm">
    <div class="card">
      <h2>1. Phần mềm và thời hạn</h2>
      <div id="spRow"><label for="sp">Phần mềm</label><select id="sp"></select></div>
      <label>Thời hạn bản quyền</label>
      <div class="plans" id="plans"></div>
      <p class="hint" id="planHint">Giá cho 1 máy chủ, gồm cập nhật và hỗ trợ trong thời hạn. Gia hạn trước khi hết hạn được cộng dồn.</p>
    </div>
    <div class="card">
      <h2>2. Thông tin</h2>
      <label for="machine">Mã máy (lấy trong phần mềm: Cài đặt → Bản quyền)</label>
      <input id="machine" class="mono" type="text" autocomplete="off" spellcheck="false" placeholder="XXXX-XXXX-XXXX-XXXX" maxlength="24">
      <label for="shop">Tên cửa hàng</label>
      <input id="shop" type="text" autocomplete="organization" maxlength="80">
      <label for="phone">Số điện thoại</label>
      <input id="phone" type="tel" autocomplete="tel" maxlength="14">
      <label for="email">Email (không bắt buộc)</label>
      <input id="email" type="email" autocomplete="email" maxlength="100">
      <label for="mst">Mã số thuế (không bắt buộc)</label>
      <input id="mst" type="text" inputmode="numeric" maxlength="14">
      <label for="partner">Mã đối tác (nếu có)</label>
      <div style="display:flex;gap:8px"><input id="partner" class="mono" type="text" autocomplete="off" spellcheck="false" maxlength="24" style="flex:1"><button class="btn ghost" id="applyPartner" type="button" style="width:auto;padding:0 16px">Áp dụng</button></div>
      <p class="hint" id="partnerMsg"></p>
      <label class="chk"><input id="agree" type="checkbox"><span>Tôi đã đọc và đồng ý điều khoản sử dụng bản quyền của Nam Linh Group.</span></label>
      <details><summary>Xem điều khoản</summary>
        <ol>
          <li>Mỗi bản quyền dùng cho 1 máy chủ, đúng mã máy đã nhập. Điện thoại và máy khác kết nối vào máy chủ không tính thêm. Đổi máy chủ cần liên hệ Nam Linh Group để được cấp lại.</li>
          <li>Gói có thời hạn: khi hết hạn vẫn xem và xuất được dữ liệu, chưa bán hàng tiếp được cho đến khi gia hạn. Gia hạn trước hạn được cộng dồn.</li>
          <li>Thanh toán bằng chuyển khoản. Mã kích hoạt được cấp tự động sau khi hệ thống nhận đủ tiền. Cần nhập đúng mã máy và giữ nguyên nội dung chuyển khoản.</li>
          <li>Không sao chép, chia sẻ hoặc bán lại mã kích hoạt.</li>
        </ol>
      </details>
      <p class="err" id="err" role="alert"></p>
      <button class="btn" id="go" type="button">Tạo mã QR thanh toán</button>
    </div>
  </div>

  <div id="viewPay" class="hide">
    <div class="card">
      <h2 id="payTitle">Quét mã QR để thanh toán</h2>
      <div class="qr"><img id="qr" alt="Mã QR chuyển khoản" width="300" height="300"><p class="hint hide" id="qrErr">Không tải được mã QR. Hãy chuyển khoản thủ công theo thông tin bên dưới.</p></div>
      <div class="kv">
        <span class="k">Số tiền</span><span class="v" id="amount"></span><button class="cp" data-cp="amount" type="button">Chép</button>
        <span class="k">Ngân hàng</span><span class="v" id="bank"></span><span></span>
        <span class="k">Số tài khoản</span><span class="v mono" id="acc"></span><button class="cp" data-cp="acc" type="button">Chép</button>
        <span class="k">Chủ tài khoản</span><span class="v" id="accName"></span><span></span>
        <span class="k">Nội dung</span><span class="v mono" id="content"></span><button class="cp" data-cp="content" type="button">Chép</button>
      </div>
      <div class="warn">Giữ nguyên <b>số tiền</b> và <b>nội dung chuyển khoản</b>. Hệ thống tự nhận tiền và hiện mã kích hoạt ngay trên trang này, thường trong vài chục giây.</div>
      <div class="wait" id="wait"><span class="spin"></span><span id="waitTxt">Đang chờ thanh toán…</span></div>
      <p class="err" id="payErr" role="alert"></p>
      <button class="btn ghost" id="back" type="button">Quay lại sửa thông tin</button>
    </div>
  </div>

  <div id="viewDone" class="hide">
    <div class="card done">
      <h2>Thanh toán thành công</h2>
      <p id="doneInfo" style="margin:0 0 10px"></p>
      <label for="key" style="margin-top:0">Mã kích hoạt</label>
      <textarea id="key" readonly spellcheck="false"></textarea>
      <div style="display:grid;gap:8px;margin-top:10px">
        <button class="btn" id="copyKey" type="button">Sao chép mã</button>
      </div>
      <div class="warn">Trong phần mềm: <b>Cài đặt → Bản quyền</b>, bấm <b>Nhận mã đã mua</b> (tự kích hoạt) hoặc dán mã vào ô rồi bấm <b>Kích hoạt</b>. Lưu lại trang này hoặc chụp màn hình mã để dùng khi cài lại.</div>
      <p class="hint" id="doneNote"></p>
    </div>
  </div>
  <p class="hint" id="contact" style="text-align:center"></p>
</div>
<script>
(function () {
  "use strict";
  var $ = function (id) { return document.getElementById(id); };
  var cfg = null, sel = { plan: "1y" }, poll = null, cur = null;
  var qs = new URLSearchParams(location.search);
  var money = function (n) { return Number(n).toLocaleString("vi-VN") + " đ"; };
  function store(k, v) { try { if (v == null) localStorage.removeItem(k); else localStorage.setItem(k, JSON.stringify(v)); } catch (e) {} }
  function load(k) { try { return JSON.parse(localStorage.getItem(k)); } catch (e) { return null; } }
  function show(v) { ["viewForm", "viewPay", "viewDone"].forEach(function (x) { $(x).classList.toggle("hide", x !== v); }); window.scrollTo(0, 0); }

  function renderPlans() {
    var sp = $("sp").value, box = $("plans"), p = cfg.prices[sp] || {};
    box.textContent = "";
    Object.keys(cfg.plans).forEach(function (k) {
      var b = document.createElement("button");
      b.type = "button"; b.className = "plan"; b.setAttribute("aria-pressed", String(sel.plan === k));
      var t = document.createElement("b"); t.textContent = cfg.plans[k];
      var pr = document.createElement("span"); pr.textContent = p[k] ? money(p[k]) : "—";
      b.appendChild(t); b.appendChild(pr);
      if (k === "life") { var s = document.createElement("small"); s.textContent = "Mua một lần"; b.appendChild(s); }
      b.onclick = function () { sel.plan = k; renderPlans(); };
      box.appendChild(b);
    });
    $("title").textContent = "Mua bản quyền " + cfg.products[sp];
    document.title = "Mua bản quyền " + cfg.products[sp] + " · Nam Linh Group";
  }

  function fillPay(o) {
    cur = o;
    $("qr").style.display = ""; $("qrErr").classList.add("hide");
    $("qr").onerror = function () { $("qr").style.display = "none"; $("qrErr").classList.remove("hide"); };
    $("qr").src = o.qr;
    $("amount").textContent = money(o.amount);
    $("amount").dataset.raw = String(o.amount);
    $("bank").textContent = o.bank.bank;
    $("acc").textContent = o.bank.acc;
    $("accName").textContent = o.bank.name;
    $("content").textContent = o.code;
    $("payErr").textContent = "";
    show("viewPay");
    startPoll();
  }

  function showDone(o) {
    stopPoll();
    $("key").value = o.key;
    $("doneInfo").textContent = (o.product || "") + " · " + (o.plan || "") + (o.exp ? " · dùng đến hết ngày " + o.exp.split("-").reverse().join("/") : " · vĩnh viễn") + " · số bản quyền " + o.id;
    $("doneNote").textContent = o.note || "";
    show("viewDone");
  }

  function stopPoll() { if (poll) { clearInterval(poll); poll = null; } }
  function startPoll() {
    stopPoll();
    var n = 0;
    function tick() {
      if (!cur) return;
      n++;
      fetch("/api/order?code=" + encodeURIComponent(cur.code) + "&t=" + encodeURIComponent(cur.token), { cache: "no-store" })
        .then(function (r) { return r.json(); })
        .then(function (o) {
          if (o.error) { $("payErr").textContent = "Không tìm thấy đơn. Hãy tạo đơn mới."; stopPoll(); store("nlbq", null); return; }
          if (o.status === "paid") { showDone(o); return; }
          $("payErr").textContent = o.note || "";
          $("waitTxt").textContent = n > 40 ? "Vẫn đang chờ… nếu đã chuyển khoản, đợi thêm hoặc liên hệ Nam Linh Group kèm nội dung " + cur.code : "Đang chờ thanh toán…";
        }).catch(function () {});
    }
    tick(); poll = setInterval(tick, 3000);
  }

  function submit() {
    $("err").textContent = "";
    var body = {
      sp: $("sp").value, plan: sel.plan, machine: $("machine").value, shop: $("shop").value,
      phone: $("phone").value, email: $("email").value, mst: $("mst").value, agree: $("agree").checked,
      partner: $("partner").value
    };
    $("go").disabled = true;
    fetch("/api/order", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) })
      .then(function (r) { return r.json(); })
      .then(function (o) {
        $("go").disabled = false;
        if (o.error) { $("err").textContent = o.error; return; }
        store("nlbq", { code: o.code, token: o.token });
        store("nlbq-form", { shop: body.shop, phone: body.phone, email: body.email, mst: body.mst });
        fillPay(o);
      }).catch(function () { $("go").disabled = false; $("err").textContent = "Không kết nối được, thử lại"; });
  }

  function applyPartner() {
    var c = $("partner").value.trim(), m = $("partnerMsg");
    m.textContent = ""; m.style.color = "";
    if (!c) return;
    fetch("/api/partner?code=" + encodeURIComponent(c) + "&sp=" + encodeURIComponent($("sp").value) + "&plan=" + encodeURIComponent(sel.plan))
      .then(function (r) { return r.json(); })
      .then(function (o) {
        if (o.error) { m.textContent = o.error; m.style.color = "#c0392b"; return; }
        m.textContent = "Giảm " + o.percent + "%: " + money(o.list) + " còn " + money(o.amount);
        m.style.color = "#1a7f4b";
      }).catch(function () { m.textContent = "Không kiểm tra được mã, thử lại"; });
  }
  $("applyPartner").onclick = applyPartner;

  document.addEventListener("click", function (e) {
    var b = e.target.closest ? e.target.closest("[data-cp]") : null;
    if (!b) return;
    var el = $(b.getAttribute("data-cp")), txt = el.dataset.raw || el.textContent;
    (navigator.clipboard ? navigator.clipboard.writeText(txt) : Promise.reject()).then(function () { b.textContent = "Đã chép"; setTimeout(function () { b.textContent = "Chép"; }, 1500); }, function () {});
  });
  $("copyKey").onclick = function () {
    var k = $("key"); k.select();
    (navigator.clipboard ? navigator.clipboard.writeText(k.value) : Promise.reject()).then(function () { $("copyKey").textContent = "Đã sao chép"; }, function () { document.execCommand("copy"); });
  };
  $("go").onclick = submit;
  $("back").onclick = function () { stopPoll(); cur = null; store("nlbq", null); show("viewForm"); };
  $("sp").onchange = renderPlans;

  fetch("/api/config", { cache: "no-store" }).then(function (r) { return r.json(); }).then(function (c) {
    cfg = c;
    Object.keys(c.products).forEach(function (k) { var o = document.createElement("option"); o.value = k; o.textContent = c.products[k]; $("sp").appendChild(o); });
    var want = qs.get("sp");
    if (want && c.products[want]) { $("sp").value = want; $("spRow").classList.add("hide"); }
    var m = qs.get("may"); if (m) $("machine").value = m.slice(0, 24);
    var f = load("nlbq-form"); if (f) { $("shop").value = f.shop || ""; $("phone").value = f.phone || ""; $("email").value = f.email || ""; $("mst").value = f.mst || ""; }
    if (c.bank.contact) $("contact").textContent = "Cần hỗ trợ: " + c.bank.contact;
    renderPlans();
    var code = qs.get("don"), t = qs.get("t"), s = load("nlbq");
    if (code && t) s = { code: code, token: t };
    if (s && s.code) {
      cur = { code: s.code, token: s.token };
      fetch("/api/order?code=" + encodeURIComponent(s.code) + "&t=" + encodeURIComponent(s.token), { cache: "no-store" }).then(function (r) { return r.json(); }).then(function (o) {
        if (o.error) { cur = null; store("nlbq", null); return; }
        o.token = s.token;
        if (o.status === "paid") showDone(o); else if (o.qr) fillPay(o);
      }).catch(function () { cur = null; });
    }
  });
})();
</script>
</body>
</html>`;

export const ADMIN_PAGE = String.raw`<!doctype html>
<html lang="vi">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Quản trị bản quyền</title>
<style>
:root{--bg:#f4f6f8;--card:#fff;--ink:#14202b;--mute:#5d6b78;--line:#dbe2e8;--pri:#0b6b5c;--bad:#b3261e;--ok:#12724a}
@media (prefers-color-scheme:dark){:root{--bg:#10171d;--card:#18222b;--ink:#e8eef2;--mute:#9aa9b6;--line:#2b3946;--pri:#2bb39b;--bad:#ff8a80;--ok:#5fd6a0}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.45 system-ui,"Segoe UI",sans-serif}
.wrap{max-width:1100px;margin:0 auto;padding:16px}
h1{font-size:20px;margin:0 0 10px}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:12px;margin:0 0 12px;overflow:auto}
input{padding:9px 10px;border:1px solid var(--line);border-radius:8px;background:var(--bg);color:var(--ink);font:inherit}
button{padding:7px 12px;border:0;border-radius:8px;background:var(--pri);color:#fff;font:inherit;cursor:pointer}
button.l{background:transparent;color:var(--pri);border:1px solid var(--pri)}
table{border-collapse:collapse;width:100%;min-width:760px}
th,td{padding:6px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
th{color:var(--mute);font-weight:600;white-space:nowrap}
.mono{font-family:ui-monospace,Consolas,monospace}
.paid{color:var(--ok);font-weight:600}.pending{color:var(--mute)}.bad{color:var(--bad)}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:center}
.hide{display:none}
</style>
</head>
<body>
<div class="wrap">
  <h1>Quản trị bản quyền online</h1>
  <div class="card" id="login">
    <div class="row"><input id="tk" type="password" placeholder="Mật khẩu quản trị (ADMIN_TOKEN)" autocomplete="current-password" style="min-width:260px"><button id="in" type="button">Vào</button><span id="msg" class="bad"></span></div>
  </div>
  <div id="main" class="hide">
    <div class="card" id="st"></div>
    <div class="card">
      <div class="row" style="margin-bottom:8px"><b>Đơn hàng</b><button class="l" id="ref" type="button">Tải lại</button><button class="l" id="rs" type="button">Ký lại mã cũ bằng khóa hiện tại</button><button class="l" id="rv" type="button">Thu hồi theo số bản quyền…</button></div>
      <table><thead><tr><th>Tạo lúc</th><th>Mã đơn</th><th>Sản phẩm</th><th>Gói</th><th>Số tiền</th><th>Đối tác</th><th>Cửa hàng</th><th>Điện thoại</th><th>Mã máy</th><th>Trạng thái</th><th></th></tr></thead><tbody id="rows"></tbody></table>
    </div>
    <div class="card">
      <div class="row" style="margin-bottom:8px"><b>Mã đối tác &amp; % giảm giá</b></div>
      <div class="row" style="margin-bottom:8px;flex-wrap:wrap;gap:6px"><input id="pc" placeholder="Mã (vd DAILY01)" style="width:140px"><input id="pn" placeholder="Tên đối tác" style="width:180px"><input id="pp" type="number" min="1" max="90" placeholder="% giảm" style="width:90px"><button class="l" id="padd" type="button">Lưu / cập nhật</button></div>
      <table><thead><tr><th>Mã</th><th>Đối tác</th><th>Giảm</th><th>Đơn đã cấp</th><th>Doanh thu</th><th>Trạng thái</th><th></th></tr></thead><tbody id="ptn"></tbody></table>
    </div>
    <div class="card">
      <div class="row" style="margin-bottom:8px"><b>Bản quyền đã thu hồi</b></div>
      <table><thead><tr><th>Số bản quyền</th><th>Khách hàng</th><th>Điện thoại</th><th>Mã máy</th><th>Thu hồi lúc</th><th>Khóa từ</th><th>Lý do</th><th></th></tr></thead><tbody id="rvl"></tbody></table>
    </div>
    <div class="card"><b>Giao dịch cần xem (không khớp đơn, chuyển thiếu, lỗi)</b><table><thead><tr><th>Lúc</th><th>Số tiền</th><th>Nội dung</th><th>Đơn</th><th>Kết quả</th></tr></thead><tbody id="odd"></tbody></table></div>
  </div>
</div>
<script>
(function () {
  "use strict";
  var $ = function (id) { return document.getElementById(id); };
  var tok = "";
  try { tok = sessionStorage.getItem("nlbq-admin") || ""; } catch (e) {}
  var dt = function (t) { return t ? new Date(t).toLocaleString("vi-VN") : ""; };
  var money = function (n) { return Number(n || 0).toLocaleString("vi-VN"); };
  var SP = { shop: "Digital Shop", fnb: "Digital FnB" }, PL = { "1y": "1 năm", "2y": "2 năm", life: "Vĩnh viễn" };
  function api(path, opt) {
    opt = opt || {}; opt.headers = Object.assign({ Authorization: "Bearer " + tok, "content-type": "application/json" }, opt.headers || {});
    return fetch(path, opt).then(function (r) { return r.json().then(function (j) { if (r.status === 401) throw new Error("Sai mật khẩu"); if (j.error) throw new Error(j.error); return j; }); });
  }
  function cell(tr, text, cls) { var td = document.createElement("td"); td.textContent = text == null ? "" : text; if (cls) td.className = cls; tr.appendChild(td); return td; }
  function load() {
    api("/admin/api/status").then(function (s) {
      $("login").classList.add("hide"); $("main").classList.remove("hide");
      var st = $("st"); st.textContent = "";
      var d = document.createElement("div");
      d.textContent = "Khóa công khai của Worker: " + (s.pub || "LỖI " + s.keyError) + " · SePay key: " + (s.sepay ? "đã đặt" : "CHƯA ĐẶT") + " · Gói cấp: " + s.plan;
      st.appendChild(d);
      var h = document.createElement("div"); h.style.color = "var(--mute)"; h.textContent = "Khóa công khai phải khớp: liHb941iLYtHy190MziQp1H4xUOG/JeGXEBWrK+dqk0="; st.appendChild(h);
      return api("/admin/api/orders");
    }).then(function (j) {
      var rows = $("rows"); rows.textContent = "";
      j.orders.forEach(function (o) {
        var tr = document.createElement("tr");
        cell(tr, dt(o.created_at)); cell(tr, o.code, "mono"); cell(tr, SP[o.product] || o.product); cell(tr, PL[o.plan] || o.plan);
        cell(tr, money(o.amount) + (o.discount > 0 ? " (giảm " + money(o.discount) + ")" : "")); cell(tr, o.partner || ""); cell(tr, o.shop); cell(tr, o.phone); cell(tr, o.machine, "mono");
        var t = cell(tr, o.status === "revoked" ? "Đã thu hồi" : o.status === "paid" ? "Đã cấp " + dt(o.paid_at) + (o.exp ? " · hết hạn " + o.exp : " · vĩnh viễn") : "Chờ thanh toán", o.status === "paid" ? "paid" : o.status === "revoked" ? "bad" : "pending");
        if (o.note) { var n = document.createElement("div"); n.className = "bad"; n.textContent = o.note; t.appendChild(n); }
        var a = cell(tr, "");
        if (o.status === "revoked") {
          var u = document.createElement("button"); u.className = "l"; u.type = "button"; u.textContent = "Khôi phục";
          u.onclick = function () {
            if (!confirm("Khôi phục bản quyền của đơn " + o.code + "?")) return;
            api("/admin/api/unrevoke", { method: "POST", body: JSON.stringify({ code: o.code }) }).then(load).catch(function (e) { alert(e.message); });
          };
          a.appendChild(u);
        } else if (o.status !== "paid") {
          var b = document.createElement("button"); b.className = "l"; b.type = "button"; b.textContent = "Cấp tay";
          b.onclick = function () {
            var why = prompt("Xác nhận đã nhận " + money(o.amount) + " đ cho đơn " + o.code + "? Ghi chú (vd: nhận tiền mặt):", "");
            if (why === null) return;
            api("/admin/api/approve", { method: "POST", body: JSON.stringify({ code: o.code, note: why }) }).then(load).catch(function (e) { alert(e.message); });
          };
          a.appendChild(b);
        } else {
          var c = document.createElement("button"); c.className = "l"; c.type = "button"; c.textContent = "Chép mã";
          c.onclick = function () { navigator.clipboard.writeText(o.license_key).then(function () { c.textContent = "Đã chép"; }); };
          a.appendChild(c);
          var r = document.createElement("button"); r.className = "l"; r.type = "button"; r.textContent = "Thu hồi";
          r.onclick = function () {
            openRevoke({ code: o.code, id: o.license_id, shop: o.shop, phone: o.phone, machine: o.machine, exp: o.exp, plan: PL[o.plan] || o.plan });
          };
          a.appendChild(r);
        }
        rows.appendChild(tr);
      });
      loadPartners(); loadRevoked();
      var odd = $("odd"); odd.textContent = "";
      j.odd.forEach(function (x) {
        var tr = document.createElement("tr");
        cell(tr, dt(x.at)); cell(tr, money(x.amount)); cell(tr, x.content); cell(tr, x.order_code || "", "mono"); cell(tr, x.result, x.result === "error" ? "bad" : "");
        odd.appendChild(tr);
      });
    }).catch(function (e) { $("msg").textContent = e.message; $("login").classList.remove("hide"); $("main").classList.add("hide"); });
  }
  function loadPartners() {
    api("/admin/api/partners").then(function (r) {
      var b = $("ptn"); b.textContent = "";
      r.list.forEach(function (p) {
        var tr = document.createElement("tr");
        cell(tr, p.code, "mono"); cell(tr, p.name); cell(tr, p.percent + "%"); cell(tr, p.paid_orders); cell(tr, money(p.revenue) + " đ");
        cell(tr, p.active ? "Đang áp dụng" : "Đã tắt", p.active ? "paid" : "bad");
        var a = cell(tr, ""), t = document.createElement("button"); t.className = "l"; t.type = "button"; t.textContent = p.active ? "Tắt" : "Bật lại";
        t.onclick = function () { savePartner({ code: p.code, name: p.name, percent: p.percent, active: !p.active }); };
        a.appendChild(t); b.appendChild(tr);
      });
    }).catch(function () {});
  }
  function savePartner(b) {
    api("/admin/api/partner", { method: "POST", body: JSON.stringify(b) }).then(function () { loadPartners(); }).catch(function (e) { alert(e.message); });
  }
  $("padd").onclick = function () {
    savePartner({ code: $("pc").value, name: $("pn").value, percent: Number($("pp").value), active: true });
    $("pc").value = ""; $("pn").value = ""; $("pp").value = "";
  };
  $("in").onclick = function () { tok = $("tk").value; try { sessionStorage.setItem("nlbq-admin", tok); } catch (e) {} load(); };
  $("tk").onkeydown = function (e) { if (e.key === "Enter") $("in").onclick(); };
  $("ref").onclick = load;
  $("rs").onclick = function () {
    if (!confirm("Ký lại các mã đã cấp mà chữ ký chưa khớp khóa hiện tại? Nội dung mã giữ nguyên.")) return;
    api("/admin/api/resign", { method: "POST", body: "{}" }).then(function (r) { alert("Đã sửa " + r.fixed + " mã; đúng sẵn " + r.ok + "; lỗi định dạng " + r.bad); load(); }).catch(function (e) { alert(e.message); });
  };
  function decodeKey(k) {
    try {
      var p = k.replace(/\s+/g, "").split(".");
      if (p.length !== 3 || p[0] !== "DS1") return null;
      var x = p[1].replace(/-/g, "+").replace(/_/g, "/"); while (x.length % 4) x += "=";
      var bin = atob(x), u = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i++) u[i] = bin.charCodeAt(i);
      var j = JSON.parse(new TextDecoder().decode(u));
      return { id: j.id, shop: j.shop, machine: j.m, exp: j.exp || "" };
    } catch (e) { return null; }
  }
  function el(tag, text, css) { var e = document.createElement(tag); if (text != null) e.textContent = text; if (css) e.style.cssText = css; return e; }
  function openRevoke(info) {
    var old = $("mdl"); if (old) old.remove();
    var bg = el("div", null, "position:fixed;left:0;top:0;right:0;bottom:0;background:rgba(0,0,0,.65);display:flex;align-items:center;justify-content:center;z-index:9;padding:12px");
    bg.id = "mdl";
    var box = el("div", null, "background:var(--card,#1c2128);color:inherit;border:1px solid var(--line,#444);border-radius:12px;padding:18px;max-width:460px;width:100%");
    box.appendChild(el("h3", "Xác nhận thu hồi bản quyền", "margin:0 0 10px"));
    var rows = [["Khách hàng", info.shop], ["Điện thoại", info.phone], ["Mã máy", info.machine], ["Số bản quyền", info.id], ["Gói / hạn", (info.plan ? info.plan + " · " : "") + (info.exp ? "đến " + info.exp : "vĩnh viễn")]];
    var tb = el("div", null, "background:rgba(255,255,255,.06);border-radius:8px;padding:10px;margin-bottom:10px;font-size:14px;line-height:1.7");
    rows.forEach(function (r) { if (!r[1] && r[0] !== "Khách hàng") return; var d = el("div"); d.appendChild(el("span", r[0] + ": ", "opacity:.7")); d.appendChild(el("b", r[1] || "(chưa biết)")); tb.appendChild(d); });
    box.appendChild(tb);
    var shopIn = null;
    if (!info.shop) {
      box.appendChild(el("div", "Không biết tên khách của mã này. Nhập tên cửa hàng để ghi nhớ (bắt buộc):", "font-size:13px;margin-bottom:4px"));
      shopIn = el("input", null, "width:100%;margin-bottom:8px"); shopIn.placeholder = "Tên cửa hàng của khách"; box.appendChild(shopIn);
    }
    box.appendChild(el("div", "Lý do:", "font-size:13px;margin-bottom:4px"));
    var why = el("input", null, "width:100%;margin-bottom:8px"); why.placeholder = "vd: hoàn tiền"; box.appendChild(why);
    box.appendChild(el("div", "Khóa sau bao nhiêu phút kể từ bây giờ?", "font-size:13px;margin-bottom:4px"));
    var mins = el("input", null, "width:100%;margin-bottom:4px"); mins.type = "number"; mins.min = "0"; mins.max = "10080"; mins.value = "1440"; box.appendChild(mins);
    box.appendChild(el("div", "0 = khóa ngay · 5 = sau 5 phút · 60 = 1 giờ · 1440 = 24 giờ (mặc định)", "font-size:12px;opacity:.7;margin-bottom:12px"));
    var err = el("div", "", "color:#e5534b;font-size:13px;min-height:16px;margin-bottom:6px"); box.appendChild(err);
    var row = el("div", null, "display:flex;gap:8px;justify-content:flex-end");
    var no = el("button", "Hủy"); no.className = "l"; no.type = "button"; no.onclick = function () { bg.remove(); };
    var yes = el("button", "Thu hồi"); yes.type = "button"; yes.style.cssText = "background:#c0392b;color:#fff;border:0;border-radius:8px;padding:8px 16px;cursor:pointer";
    yes.onclick = function () {
      var shop = info.shop || (shopIn ? shopIn.value.trim() : "");
      if (!shop) { err.textContent = "Nhập tên khách hàng để tránh thu hồi nhầm"; return; }
      var m = mins.value === "" ? 1440 : Number(mins.value);
      if (!(m >= 0)) { err.textContent = "Số phút không đúng"; return; }
      yes.disabled = true;
      api("/admin/api/revoke", { method: "POST", body: JSON.stringify({ code: info.code, id: info.id, shop: shop, machine: info.machine, phone: info.phone, note: why.value, minutes: m }) })
        .then(function (r) { bg.remove(); alert("Đã thu hồi " + r.id + " (" + r.shop + "). Khóa từ " + dt(r.effective) + "."); load(); })
        .catch(function (e) { yes.disabled = false; err.textContent = e.message; });
    };
    row.appendChild(no); row.appendChild(yes); box.appendChild(row); bg.appendChild(box); document.body.appendChild(bg);
  }
  $("rv").onclick = function () {
    var v = prompt("Dán MÃ KÍCH HOẠT của khách (bắt đầu DS1.) để hiện đúng tên khách, hoặc nhập số bản quyền NL-… :", "");
    if (!v) return;
    v = v.trim();
    var d = decodeKey(v);
    if (d) { openRevoke({ id: d.id, shop: d.shop, machine: d.machine, exp: d.exp }); return; }
    if (/^NL-/i.test(v)) { openRevoke({ id: v.toUpperCase(), shop: "" }); return; }
    alert("Không đọc được. Dán đủ mã DS1… hoặc nhập số bản quyền dạng NL-261112-Y5RS.");
  };
  function loadRevoked() {
    api("/admin/api/revoked").then(function (r) {
      var b = $("rvl"); b.textContent = "";
      if (!r.list.length) { var tr0 = document.createElement("tr"); cell(tr0, "Chưa thu hồi bản quyền nào."); b.appendChild(tr0); return; }
      r.list.forEach(function (x) {
        var tr = document.createElement("tr");
        cell(tr, x.license_id, "mono"); cell(tr, x.shop || "(không rõ)"); cell(tr, x.phone || ""); cell(tr, x.machine || "", "mono");
        cell(tr, dt(x.at)); cell(tr, dt(x.eff || x.at + 86400000)); cell(tr, x.note || "");
        var a = cell(tr, ""), u = document.createElement("button"); u.className = "l"; u.type = "button"; u.textContent = "Khôi phục";
        u.onclick = function () {
          if (!confirm("Khôi phục bản quyền " + x.license_id + " của " + (x.shop || "khách này") + "?")) return;
          api("/admin/api/unrevoke", { method: "POST", body: JSON.stringify({ id: x.license_id, code: "" }) }).then(load).catch(function (e) { alert(e.message); });
        };
        a.appendChild(u); b.appendChild(tr);
      });
    }).catch(function () {});
  }
  if (tok) load();
})();
</script>
</body>
</html>`;
