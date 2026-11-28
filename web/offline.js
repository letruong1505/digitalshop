/* Digital Shop — ứng dụng Android: kết nối máy chủ trong mạng wifi, bán hàng ngoại tuyến, đẩy hóa đơn về máy chủ khi có mạng.
   Chỉ hoạt động khi chạy trong ứng dụng Android (window.Capacitor); bản chạy trên máy tính không bị ảnh hưởng. */
"use strict";
const APP = !!window.Capacitor;
const srvBase = () => APP ? (store.get("server", "") || "") : "";
S.offline = false;
if (APP) document.documentElement.classList.add("app");

/* ---------- IndexedDB nhỏ gọn: kv (bản chụp dữ liệu), outbox (hóa đơn chờ đẩy) ---------- */
let _db = null;
function idb() {
  if (_db) return _db;
  _db = new Promise((ok, no) => {
    const r = indexedDB.open("digitalshop", 1);
    r.onupgradeneeded = () => { r.result.createObjectStore("kv"); r.result.createObjectStore("outbox", { keyPath: "cid" }); };
    r.onsuccess = () => ok(r.result); r.onerror = () => no(r.error);
  });
  return _db;
}
async function idbDo(store_, mode, fn) {
  const db = await idb();
  return new Promise((ok, no) => { const tx = db.transaction(store_, mode); const rq = fn(tx.objectStore(store_)); tx.oncomplete = () => ok(rq && rq.result); tx.onerror = () => no(tx.error); });
}
/* ---------- đăng nhập ngoại tuyến cho nhân viên bán hàng ---------- */
function sha256hex(str) {
  const msg = new TextEncoder().encode(str);
  if (window.crypto && crypto.subtle) return crypto.subtle.digest("SHA-256", msg).then(d => [...new Uint8Array(d)].map(x => x.toString(16).padStart(2, "0")).join(""));
  const K = [], H = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
  for (let n = 2, c = 0; c < 64; n++) { let p = true; for (let i = 2; i * i <= n; i++) if (n % i === 0) { p = false; break; } if (p) { K[c++] = (Math.cbrt(n) % 1) * 4294967296 | 0; } }
  const l = msg.length, w = new Uint32Array(((l + 9 + 63) >> 6) << 4), v = new Uint8Array(w.buffer);
  v.set(msg); v[l] = 0x80; const dv = new DataView(w.buffer); dv.setUint32(w.length * 4 - 4, l * 8, false); dv.setUint32(w.length * 4 - 8, Math.floor(l / 536870912), false);
  const rr = (x, n) => (x >>> n) | (x << (32 - n));
  for (let i = 0; i < w.length; i += 16) {
    const x = new Uint32Array(64); for (let j = 0; j < 16; j++) x[j] = dv.getUint32((i + j) * 4, false);
    for (let j = 16; j < 64; j++) { const s0 = rr(x[j - 15], 7) ^ rr(x[j - 15], 18) ^ (x[j - 15] >>> 3), s1 = rr(x[j - 2], 17) ^ rr(x[j - 2], 19) ^ (x[j - 2] >>> 10); x[j] = (x[j - 16] + s0 + x[j - 7] + s1) | 0; }
    let [a, b, c, d, e, f, g, h] = H;
    for (let j = 0; j < 64; j++) { const t1 = (h + (rr(e, 6) ^ rr(e, 11) ^ rr(e, 25)) + ((e & f) ^ (~e & g)) + K[j] + x[j]) | 0, t2 = ((rr(a, 2) ^ rr(a, 13) ^ rr(a, 22)) + ((a & b) ^ (a & c) ^ (b & c))) | 0; h = g; g = f; f = e; e = (d + t1) | 0; d = c; c = b; b = a; a = (t1 + t2) | 0; }
    H[0] = (H[0] + a) | 0; H[1] = (H[1] + b) | 0; H[2] = (H[2] + c) | 0; H[3] = (H[3] + d) | 0; H[4] = (H[4] + e) | 0; H[5] = (H[5] + f) | 0; H[6] = (H[6] + g) | 0; H[7] = (H[7] + h) | 0;
  }
  return Promise.resolve(H.map(x => (x >>> 0).toString(16).padStart(8, "0")).join(""));
}
async function offBootGet() { const c = await kvGet("boot:" + srvBase()); if (c) return c; return srvList().length <= 1 ? kvGet("boot") : null; } // bản chụp riêng cho từng máy chủ
async function offUsers() { try { const c = await offBootGet(); return (c && c.b && c.b.offUsers) || []; } catch (e) { return []; } }
const offHasUsers = async () => (await offUsers()).length > 0;
async function offLogin(pin) {
  for (const u of await offUsers()) {
    if (await sha256hex(u.salt + ":" + pin) === u.hash) {
      S.offMe = { id: u.id, name: u.name, role: u.role, perms: u.perms };
      store.set("offMe", JSON.stringify(S.offMe));
      await enterApp(); toast("Đã vào chế độ ngoại tuyến: " + u.name);
      return true;
    }
  }
  return false;
}

const kvGet = k => idbDo("kv", "readonly", s => s.get(k));
const kvSet = (k, v) => idbDo("kv", "readwrite", s => s.put(v, k));
const obAll = () => idbDo("outbox", "readonly", s => s.getAll()).then(a => (a || []).sort((x, y) => x.at - y.at));
const obPut = it => idbDo("outbox", "readwrite", s => s.put(it));
const obDel = cid => idbDo("outbox", "readwrite", s => s.delete(cid));

/* ---------- bản chụp dữ liệu máy chủ để dùng khi mất mạng ---------- */
let lastCacheV = null;
function offCacheBoot(b) {
  if (!APP || (b.version === lastCacheV && lastCacheV != null)) return; // chỉ lưu lại khi dữ liệu máy chủ đã đổi
  lastCacheV = b.version;
  kvSet("boot:" + srvBase(), { b, at: Date.now() }).catch(() => {});
}
async function offLoadBoot() {
  try {
    const c = await offBootGet(); if (!c || !c.b) return null;
    const b = c.b;
    try { const m = S.offMe || JSON.parse(store.get("offMe", "") || "null"); if (m) { S.offMe = m; b.me = m; } } catch (e) {} // đang đăng nhập ngoại tuyến: dùng đúng người đó
    for (const it of (await obAll()).filter(x => !x.srv || x.srv === srvBase())) offApplyLocal(b.products, b.customers, it.local);
    S.cachedAt = c.at;
    return b;
  } catch (e) { return null; }
}
// Trừ tồn kho / cộng nợ theo các hóa đơn đang chờ đẩy, để màn hình bán hàng không bán quá tồn
function offApplyLocal(products, customers, loc) {
  if (!loc) return;
  const pm = Array.isArray(products) ? new Map(products.map(p => [p.id, p])) : products;
  const cm = Array.isArray(customers) ? new Map(customers.map(c => [c.id, c])) : customers;
  for (const l of loc.lines || []) { const p = pm.get(l.pid); if (p && !p.isCombo) p.stock = r3((p.stock || 0) - l.qty * (l.ratio || 1)); }
  if (loc.custId && loc.debt) { const c = cm.get(loc.custId); if (c) c.debt = (c.debt || 0) + loc.debt; }
}

/* ---------- trạng thái mạng + thanh thông báo ---------- */
function setOffline(v) {
  if (!APP || S.offline === v) { offBar(); return; }
  S.offline = v; offBar();
  if (S.me && S.view === "pos") { try { renderCart(); } catch (e) {} }
}
let pending = [];
async function offBar() {
  if (!APP) return;
  try { pending = (await obAll()).filter(x => !x.srv || x.srv === srvBase()); } catch (e) { pending = []; }
  let el = $("#offBar");
  const bad = pending.filter(x => x.err).length;
  if (!S.offline && !pending.length) { if (el) el.remove(); return; }
  if (!el) { el = document.createElement("div"); el.id = "offBar"; el.setAttribute("data-act", "offList"); document.body.appendChild(el); }
  const n = pending.length;
  const st = S.offline ? "off" : bad ? "bad" : "sync";
  const full = S.offline ? `Mất kết nối máy chủ · bán hàng vẫn lưu trên máy${n ? ` · ${n} hóa đơn chờ đẩy` : ""}`
    : bad ? `${bad} hóa đơn chưa đẩy được lên máy chủ, bấm để xem` : `Đang đẩy ${n} hóa đơn về máy chủ…`;
  const small = S.offline ? "Ngoại tuyến" + (n ? " · " + n : "") : bad ? bad + " lỗi" : "Đang đẩy " + n;
  // Thông báo đầy đủ chỉ hiện vài giây khi trạng thái đổi, sau đó thu thành nhãn nhỏ ở trên cùng để không che thanh điều hướng
  const key = st + ":" + (bad || n);
  if (el._key !== key) {
    el._key = key; el.classList.add("full");
    clearTimeout(el._t); el._t = setTimeout(() => { el.classList.remove("full"); el.textContent = small; }, 5000);
  }
  el.className = "offbar " + st + (el.classList.contains("full") ? " full" : "");
  el.textContent = el.classList.contains("full") ? full : small;
  el.title = full;
}

/* ---------- bán hàng ngoại tuyến ---------- */
function offSale(body, t, T) {
  const promo = T.promo || 0;
  const lines = t.lines.map(l => { const p = S.products.get(l.pid) || {}; return { pid: l.pid, code: p.code || "", name: p.name || "", unit: l.unit, ratio: l.ratio || 1, qty: l.qty, price: l.price }; });
  const total = T.total, given = body.given || 0;
  let cash = 0, bank = 0, debt = 0;
  if (body.method === "bank") bank = total;
  else if (body.method === "debt") debt = total;
  else if (given > 0 && given < total && body.custId) { cash = given; debt = total - given; } else cash = total;
  const cust = body.custId ? S.customers.get(body.custId) : null;
  const time = Date.now();
  const q = { ...body, offline: true, offStaff: S.me ? S.me.name : "", time, promoDisc: promo, promoNote: ((t.promo && t.promo.hits) || []).map(h => h.name).join(", ") };
  const inv = { id: body.cid, code: "CHỜ-" + body.cid.slice(-5).toUpperCase(), time, lines, sub: T.sub, disc: T.disc + promo, promo, promoNote: q.promoNote, total, given: cash ? given : 0, cash, bank, debt,
    custId: body.custId || "", custName: cust ? cust.name : "", staff: S.me ? S.me.name : "", note: body.note || "", offlineSale: true };
  const local = { lines: lines.map(l => ({ pid: l.pid, qty: l.qty, ratio: l.ratio })), custId: body.custId || "", debt };
  return obPut({ cid: body.cid, body: q, at: time, inv, local, srv: srvBase() }).then(() => { offApplyLocal(S.products, S.customers, local); offBar(); return inv; });
}

let syncing = false;
async function syncOutbox() {
  if (!APP || syncing) return;
  syncing = true;
  let did = 0;
  try {
    for (const it of (await obAll()).filter(x => !x.srv || x.srv === srvBase())) { // chỉ đẩy hóa đơn của đúng máy chủ đang kết nối
      try {
        await api("POST", "/api/invoices", it.body);
        await obDel(it.cid); did++;
      } catch (e) {
        if (e.net) { setOffline(true); break; }
        if (/Cần đăng nhập/.test(e.message)) break;
        it.err = e.message; await obPut(it); // lỗi nghiệp vụ (hàng đã xóa, hết quyền…): giữ lại để chủ xử lý
      }
    }
  } finally { syncing = false; }
  await offBar();
  if (did) { toast(`Đã cập nhật ${did} đơn hàng offline lên máy chủ`); try { await refresh(true); } catch (e) {} }
}
ACT.offList = async () => {
  const list = await obAll();
  openSheet(`<div class="sheet-h"><h2>Hóa đơn chờ đẩy lên máy chủ</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <p class="ps" style="margin:0 0 8px">${S.offline ? "Đang mất kết nối. Khi có wifi và máy chủ bật, phần mềm tự đẩy lên." : "Phần mềm đang tự đẩy lên máy chủ."}</p>
    ${list.length ? list.map(it => `<div class="card" style="margin-bottom:8px"><div class="row"><b class="grow">${esc(it.inv.code)} · ${hm(it.inv.time)}</b><b class="num">${money(it.inv.total)}</b></div>
      <div class="ps">${esc(it.inv.lines.map(l => l.name + " ×" + fq(l.qty)).join(", "))}</div>
      ${it.err ? `<div class="ps" style="color:var(--red)">Lỗi: ${esc(it.err)}</div>` : ""}
      <div class="row" style="margin-top:6px"><button class="btn sm" data-act="offRetry" data-cid="${esc(it.cid)}">Thử lại</button><button class="btn sm danger" data-act="offDrop" data-cid="${esc(it.cid)}" data-confirm="Bấm lần nữa để bỏ hóa đơn này">Bỏ hóa đơn</button></div></div>`).join("") : '<div class="empty">Không còn hóa đơn nào chờ.</div>'}`);
};
ACT.offRetry = async b => { const all = await obAll(); const it = all.find(x => x.cid === b.dataset.cid); if (it) { delete it.err; await obPut(it); } closeSheet(); syncOutbox(); };
ACT.offDrop = async b => { await obDel(b.dataset.cid); closeSheet(); offBar(); toast("Đã bỏ hóa đơn khỏi hàng chờ. Số hàng trong kho sẽ khớp lại khi tải dữ liệu mới."); try { await refresh(true); } catch (e) {} };

/* ---------- khuyến mãi tính tại máy khi mất mạng (cùng quy tắc với máy chủ) ---------- */
function offPromo(t) {
  const now = new Date(), d = dkey(now), cur = now.getHours() * 60 + now.getMinutes();
  const hm_ = s => /^\d\d:\d\d$/.test(s || "") ? +s.slice(0, 2) * 60 + +s.slice(3) : -1;
  const camps = [];
  for (const c of (S.settings.campaigns || [])) (c.lines || []).forEach((l, i) => camps.push({ name: c.name, active: c.active, kind: l.kind, value: l.value, scope: l.scope, target: l.target, from: c.from, to: c.to }));
  const act = (S.settings.promos || []).concat(camps).filter(p => {
    if (!p.active || !(p.value > 0)) return false;
    if (p.from && d < p.from) return false; if (p.to && d > p.to) return false;
    if (p.days && p.days.length && !p.days.includes(now.getDay())) return false;
    const a = hm_(p.tStart), b = hm_(p.tEnd);
    if (a >= 0 && b >= 0 && a !== b) { if (a < b ? (cur < a || cur >= b) : (cur < a && cur >= b)) return false; }
    return true;
  });
  if (!act.length) return null;
  const tiers = (S.settings.tiers && S.settings.tiers.length ? S.settings.tiers : [{ min: 2000000 }, { min: 10000000 }, { min: 30000000 }]).slice().sort((x, y) => x.min - y.min);
  const cust = t.custId ? S.customers.get(t.custId) : null;
  const spend = cust ? cust.spend || 0 : 0;
  let tier = 0; tiers.forEach((x, i) => { if (spend >= x.min) tier = i + 1; });
  const by = {}, per = []; let total = 0;
  const norm_ = s => String(s || "").trim().toLowerCase();
  for (const l of t.lines) {
    const amount = l.qty * l.price; if (amount <= 0) { per.push({ name: "", disc: 0 }); continue; }
    const pr = S.products.get(l.pid); let best = 0, name = "";
    if (pr) { const cu = l.ratio > 1 && l.unit !== pr.unit ? (((pr.units || []).find(c => c.name === l.unit && c.ratio === l.ratio) || {}).price ?? pr.price * l.ratio) : pr.price * (l.ratio || 1); if (l.price < cu - 0.5) { per.push({ name: "", disc: 0 }); continue; } }
    for (const p of act) {
      if (p.minTier > 0 && tier < p.minTier) continue;
      if (p.scope === "product" && p.target !== l.pid) continue;
      if (p.scope === "group" && (!pr || !p.target || norm_(pr.group) !== norm_(p.target))) continue;
      let x = p.kind === "amt" ? p.value * l.qty : amount * p.value / 100; if (x > amount) x = amount;
      if (x > best) { best = x; name = p.name; }
    }
    if (best > 0) { total += best; by[name] = (by[name] || 0) + best; }
    per.push({ name: best > 0 ? name : "", disc: Math.round(best) });
  }
  total = Math.round(total);
  return total > 0 ? { disc: total, hits: Object.entries(by).map(([name, a]) => ({ name, amount: Math.round(a) })), lines: per } : null;
}

/* ---------- màn hình kết nối máy chủ (lần đầu và khi đổi) ---------- */
function normServer(s) {
  s = String(s || "").trim().replace(/\/+$/, "");
  if (!s) return "";
  if (!/^https?:\/\//i.test(s)) s = "http://" + s;
  if (!/:\d+$/.test(s.replace(/^https?:\/\//i, "")) && /^http:\/\//i.test(s)) s += ":8686";
  return s;
}
/* ---------- nhiều máy chủ (nhân viên làm 2 địa điểm / 2 ca): lưu tên gọi, chuyển nhanh ---------- */
function srvList() {
  let l = store.get("servers", []); if (!Array.isArray(l)) l = [];
  const cur = store.get("server", "");
  if (cur && !l.some(x => x.url === cur)) l = [{ name: "Máy chủ chính", url: cur }, ...l];
  return l;
}
const srvLabel = u => { const x = srvList().find(y => y.url === u); return x ? x.name : (u || "").replace(/^https?:\/\//, ""); };
function srvSave(name, url) {
  const l = srvList().filter(x => x.url !== url); l.push({ name: (name || "").trim() || url.replace(/^https?:\/\//, ""), url });
  store.set("servers", l);
}
ACT.srvPick = b => { const x = srvList()[+b.dataset.i]; if (!x) return; $("#cn-srv").value = x.url.replace(/^https?:\/\//, ""); $("#cn-name").value = x.name; };
ACT.srvDel = b => { const l = srvList(); l.splice(+b.dataset.i, 1); store.set("servers", l); closeSheet(); srvSheet(); };
async function srvSwitch(url) {
  if (url === store.get("server", "")) return closeSheet();
  const left = (await obAll()).filter(x => !x.srv || x.srv === srvBase()).length;
  if (left) return toast(`Còn ${left} hóa đơn chờ đẩy lên máy chủ "${srvLabel(srvBase())}". Hãy kết nối lại máy chủ đó để đẩy xong rồi mới đổi`, true);
  try {
    const ctl = new AbortController(), tm = setTimeout(() => ctl.abort(), 4000);
    const r = await fetch(url + "/api/ping", { signal: ctl.signal }); clearTimeout(tm);
    if ((await r.text()).trim() !== "taphoax") throw new Error("x");
  } catch (e) { return toast(`Không kết nối được "${srvLabel(url)}". Kiểm tra wifi và máy tính đó`, true); }
  store.set("tok", ""); S.offMe = null; store.set("offMe", ""); store.set("server", url); lastCacheV = null; S.offline = false;
  closeSheet(); clearInterval(pollT); boot();
}
ACT.srvSwitch = b => srvSwitch(srvList()[+b.dataset.i].url);
function srvSheet() {
  const l = srvList(), cur = store.get("server", "");
  openSheet(`<div class="sheet-h"><h2>Máy chủ kết nối</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="ps">Chọn máy chủ của địa điểm / ca bạn đang làm. Cần cùng wifi với máy tính đó.</div>
    <div class="stack" style="gap:8px">${l.map((x, i) => `<div class="row card" style="padding:10px 12px"><div class="grow"><b>${esc(x.name)}</b>${x.url === cur ? ' <span class="pill green">đang dùng</span>' : ""}<div class="ps">${esc(x.url.replace(/^https?:\/\//, ""))}</div></div>${x.url === cur ? "" : `<button class="btn sm pri" data-act="srvSwitch" data-i="${i}">Chuyển</button><button class="btn sm ghost danger" data-act="srvDel" data-i="${i}" data-confirm="Bấm lần nữa để xóa">Xóa</button>`}</div>`).join("")}</div>
    <div class="row"><button class="btn" data-act="connectEdit">Thêm / sửa máy chủ</button></div>`);
}
ACT.srvSheet = () => srvSheet();

function showConnect(msg) {
  $("#app").innerHTML = `<div class="login"><form class="box" id="connForm" autocomplete="off">
    <div class="sign"><small>Digital Shop · Android</small><b>Kết nối máy chủ</b></div>
    <p class="muted" style="margin:0;font-size:14px">Điện thoại cần cùng wifi với máy tính đang chạy Digital Shop. Trên máy tính, mở biểu tượng Digital Shop ở góc phải thanh tác vụ để xem địa chỉ (dạng 192.168.1.10:8686).</p>
    ${srvList().length ? `<div class="field"><label>Máy chủ đã lưu</label><div class="row" style="flex-wrap:wrap">${srvList().map((x, i) => `<button type="button" class="btn sm" data-act="srvPick" data-i="${i}">${esc(x.name)}</button>`).join("")}</div></div>` : ""}
    <div class="field"><label for="cn-name">Tên gọi (để nhớ, vd Cửa hàng 1)</label><input id="cn-name" class="in" placeholder="Cửa hàng chính" value="${esc(srvLabel(store.get("server", "")) === store.get("server", "").replace(/^https?:\/\//, "") ? "" : srvLabel(store.get("server", "")))}"></div>
    <div class="field"><label for="cn-srv">Địa chỉ máy chủ</label><input id="cn-srv" class="in" inputmode="url" placeholder="192.168.1.10:8686" value="${esc(store.get("server", ""))}" required></div>
    <div class="field"><label>Máy này dùng để</label>
      <label class="check"><input type="radio" name="cn-mode" value="sell" ${store.get("mode", "sell") !== "display" ? "checked" : ""}> Bán hàng (có bán ngoại tuyến)</label>
      <label class="check"><input type="radio" name="cn-mode" value="display" ${store.get("mode", "sell") === "display" ? "checked" : ""}> Làm màn hình cho khách</label></div>
    <div class="field" id="cn-durow" ${store.get("mode", "sell") === "display" ? "" : "hidden"}><label for="cn-du">Địa chỉ màn hình khách</label><input id="cn-du" class="in" inputmode="url" placeholder="Dán địa chỉ từ Bán hàng → Màn hình khách" value="${esc(store.get("displayUrl", ""))}"></div>
    <div id="cn-msg" class="ps" style="color:var(--red);min-height:18px">${esc(msg || "")}</div>
    <button class="btn pri lg" type="submit">Kết nối</button></form></div>`;
  document.querySelectorAll('input[name="cn-mode"]').forEach(r => r.onchange = () => { $("#cn-durow").hidden = r.value !== "display" || !r.checked; });
  $("#connForm").onsubmit = async ev => {
    ev.preventDefault();
    const srv = normServer($("#cn-srv").value), mode = document.querySelector('input[name="cn-mode"]:checked').value;
    const msgEl = $("#cn-msg"); msgEl.textContent = "Đang kết nối…"; msgEl.style.color = "";
    try {
      const ctl = new AbortController(); const tm = setTimeout(() => ctl.abort(), 7000);
      const r = await fetch(srv + "/api/ping", { signal: ctl.signal }); clearTimeout(tm);
      if ((await r.text()).trim() !== "taphoax") throw new Error("x");
    } catch (e) { msgEl.style.color = "var(--red)"; msgEl.textContent = "Không kết nối được. Kiểm tra: điện thoại cùng wifi, máy tính đang bật Digital Shop, địa chỉ đúng."; return; }
    if (mode === "display") {
      let du = ($("#cn-du").value || "").trim();
      if (!/khach\.html\?k=/.test(du)) { msgEl.style.color = "var(--red)"; msgEl.textContent = "Dán đúng địa chỉ màn hình khách (có chữ khach.html?k=…) lấy ở Bán hàng → Màn hình khách."; return; }
      store.set("displayUrl", du);
    }
    if (srv !== store.get("server", "")) {
      const left = (await obAll()).filter(x => !x.srv || x.srv === srvBase()).length;
      if (left) { msgEl.style.color = "var(--red)"; msgEl.textContent = `Còn ${left} hóa đơn chờ đẩy lên máy chủ "${srvLabel(srvBase())}". Kết nối lại máy chủ đó để đẩy xong rồi mới đổi.`; return; }
      store.set("tok", ""); S.offMe = null; store.set("offMe", ""); lastCacheV = null;
    }
    srvSave($("#cn-name").value, srv);
    store.set("server", srv); store.set("mode", mode);
    boot();
  };
}
ACT.connectEdit = () => { closeSheet(); showConnect(); };
function openDisplayMode() { const u = store.get("displayUrl", ""); if (u) location.replace(u); else showConnect(); }

/* ---------- thông báo có bản Android mới ---------- */
let apkAt = 0;
const verNum = v => String(v || "0").split(".").reduce((a, x) => a * 1000 + (parseInt(x, 10) || 0), 0);
/* Cập nhật Android ngay trong ứng dụng (cần bản có DSUpdater; bản cũ hơn tải qua trình duyệt) */
const apkInApp = () => !!(window.DSUpdater && window.DSUpdater.available);
const apkBtn = r => apkInApp() ? `<button type="button" class="btn pri" data-act="apkInstall" data-url="${esc(r.url)}">Tải và cập nhật bản ${esc(r.version)}</button>` : `<a class="btn pri" href="${esc(r.url)}">Tải bản ${esc(r.version)}</a>`;
function apkMsg(t, err) { document.querySelectorAll(".apkmsg").forEach(e => { e.textContent = t; e.style.color = err ? "var(--red)" : ""; }); }
window.dsApkProgress = p => apkMsg("Đang tải bản cập nhật… " + p + "%");
window.dsApkState = (st, m) => {
  if (st === "permission") apkMsg("Android hỏi cho phép: bật “Cho phép từ nguồn này” cho Digital Shop, rồi quay lại và bấm Tiếp tục cài.");
  else if (st === "installing") apkMsg("Đã tải xong. Bấm Cài đặt / Cập nhật ở cửa sổ Android vừa hiện. Dữ liệu được giữ nguyên.");
  else apkMsg(m || "Không cập nhật được", true);
  document.querySelectorAll('[data-act="apkInstall"]').forEach(b => { b.disabled = false; b.textContent = st === "permission" ? "Tiếp tục cài" : b.textContent; b.dataset.retry = st === "permission" ? "1" : ""; });
};
ACT.apkInstall = b => {
  if (b.dataset.retry) { b.dataset.retry = ""; apkMsg("Đang mở trình cài đặt…"); return window.DSUpdater.retryInstall(); }
  b.disabled = true; apkMsg("Đang tải bản cập nhật…"); window.DSUpdater.install(b.dataset.url);
};
async function apkCheck(force) {
  if (!APP || !window.DS_APK || S.offline || (!force && Date.now() - apkAt < 30 * 60 * 1000)) return;
  apkAt = Date.now();
  try {
    const r = await api("GET", "/api/android/latest");
    if (!r || verNum(r.version) <= verNum(window.DS_APK) || (!force && store.get("apkSkip", "") === r.version)) return;
    let el = $("#apkBar"); if (el) el.remove();
    el = document.createElement("div"); el.id = "apkBar"; el.className = "apkbar";
    el.innerHTML = `<div class="apkin"><span>Có bản Digital Shop ${esc(r.version)} mới (đang dùng ${esc(window.DS_APK)}). Dữ liệu được giữ nguyên.</span> ${apkBtn(r)} <span class="apkmsg ps"></span></div><button type="button" aria-label="Để sau" data-act="apkSkip" data-v="${esc(r.version)}">×</button>`;
    document.body.appendChild(el);
  } catch (e) { /* không có mạng hoặc chưa có bản: bỏ qua */ }
}
ACT.apkSkip = b => { store.set("apkSkip", b.dataset.v); const el = $("#apkBar"); if (el) el.remove(); };
