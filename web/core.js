/* Digital Shop — lõi giao diện: tiện ích, đăng nhập, khung, in, VietQR */
"use strict";
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const NF = new Intl.NumberFormat("vi-VN");
const fmt = n => NF.format(Math.round(n || 0));
const money = n => fmt(n) + "đ";
const fq = n => NF.format(Math.round((n || 0) * 1000) / 1000);
const r3 = n => Math.round(n * 1000) / 1000;
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const norm = s => String(s || "").normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/[đĐ]/g, "d").toLowerCase().trim();
const pad = n => String(n).padStart(2, "0");
const dkey = (d = new Date()) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const addDays = (k, n) => { const d = new Date(k + "T00:00:00"); d.setDate(d.getDate() + n); return dkey(d); };
const vnDate = k => k ? `${k.slice(8, 10)}/${k.slice(5, 7)}/${k.slice(0, 4)}` : "";
const dm = k => k ? `${k.slice(8, 10)}/${k.slice(5, 7)}` : "";
const hm = t => { const d = new Date(t); return pad(d.getHours()) + ":" + pad(d.getMinutes()); };
const dt = t => { const d = new Date(t); return `${pad(d.getDate())}/${pad(d.getMonth() + 1)} ${hm(t)}`; };
const daysTo = k => Math.round((new Date(k + "T00:00:00") - new Date(dkey() + "T00:00:00")) / 864e5);
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 9);
const WD = ["Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];
function parseMoney(s) {
  s = String(s ?? "").toLowerCase().replace(/\s|đ|vnd/g, "");
  let mul = 1;
  if (/tr$/.test(s)) { mul = 1e6; s = s.slice(0, -2); } else if (/k$/.test(s)) { mul = 1e3; s = s.slice(0, -1); }
  if (mul > 1) s = s.replace(",", "."); else s = s.replace(/[.,]/g, "");
  const n = parseFloat(s); return isNaN(n) ? 0 : Math.round(n * mul);
}
function parseQty(s) {
  s = String(s ?? "").trim().replace(/\s/g, "");
  if (/^-?\d{1,3}(\.\d{3})+(,\d+)?$/.test(s)) s = s.replace(/\./g, "").replace(",", ".");
  else s = s.replace(",", ".");
  const n = parseFloat(s); return isNaN(n) ? 0 : r3(n);
}
const store = {
  get(k, d) { try { const v = localStorage.getItem("thx_" + k); return v == null ? d : JSON.parse(v); } catch (e) { return d; } },
  set(k, v) { try { localStorage.setItem("thx_" + k, JSON.stringify(v)); } catch (e) {} }
};

const IC = {
  pos: '<path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.5L21 8H6.2"/><circle cx="10" cy="20" r="1.3"/><circle cx="17" cy="20" r="1.3"/>',
  orders: '<path d="M4 5a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7l-5 4v-4H6a2 2 0 0 1-2-2z"/><path d="M8 8h8M8 11.5h5"/>',
  products: '<path d="M21 8 12 3 3 8v8l9 5 9-5z"/><path d="m3 8 9 5 9-5M12 13v8"/>',
  receipts: '<path d="M12 3v12m0 0-4-4m4 4 4-4"/><path d="M4 15v4a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-4"/>',
  counts: '<path d="M9 4h6a1 1 0 0 1 1 1v1H8V5a1 1 0 0 1 1-1z"/><path d="M8 5H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/><path d="m9 13 2 2 4-4"/>',
  customers: '<path d="M5 3h12a2 2 0 0 1 2 2v16l-3-2-3 2-3-2-3 2-2-1.5z"/><path d="M9 8h6M9 12h6"/>',
  suppliers: '<path d="M3 7h11v10H3zM14 10h4l3 3v4h-7"/><circle cx="7" cy="18" r="1.6"/><circle cx="17" cy="18" r="1.6"/>',
  cash: '<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M6 9v.01M18 15v.01"/>',
  report: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>',
  combos: '<path d="M20 12v8H4v-8M2 7h20v5H2zM12 22V7M12 7H8a2.5 2.5 0 1 1 0-5c3 0 4 5 4 5zM12 7h4a2.5 2.5 0 1 0 0-5c-3 0-4 5-4 5z"/>',
  inin: '<path d="M6 2h9l5 5v13a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2z"/><path d="M14 2v6h6M12 11v7m0 0-3-3m3 3 3-3"/>',
  import: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6M8 13h8M8 17h5"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  more: '<circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
  print: '<path d="M6 9V3h12v6M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><path d="M6 14h12v7H6z"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
};
const icon = (k, cls = "") => `<svg class="ic ${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${IC[k] || ""}</svg>`;

const NAV = [
  ["pos", "Bán hàng"], ["orders", "Đơn đặt / Zalo"], ["products", "Hàng hóa"], ["receipts", "Nhập hàng"], ["inin", "Hóa đơn đầu vào", true], ["counts", "Kiểm kho"],
  ["customers", "Khách hàng"], ["suppliers", "Nhà cung cấp"], ["cash", "Sổ quỹ"], ["report", "Báo cáo"], ["combos", "Combo & khuyến mãi", true],
  ["import", "Nhập từ Excel", true], ["settings", "Cài đặt", true],
];

/* ---------- trạng thái chung ---------- */
const S = {
  me: null, settings: {}, products: new Map(), customers: new Map(), suppliers: new Map(), orders: new Map(),
  pop: {}, users: [], lan: [], version: 0, view: "pos", hasKey: false, dataDir: "", today: dkey(),
};
const isOwner = () => S.me && S.me.role === "owner";
const seeCost = () => isOwner() || S.settings.staffSeeCost;
const VIEWS = {};
// Nút "Nhập từ Excel" đặt ở đầu các trang danh mục; k = loại dữ liệu chọn sẵn
const impBtn = (k, cls = "") => isOwner() ? `<button class="btn sm ${cls}" data-act="impGo" data-k="${k}">${icon("import")} Nhập từ Excel</button>` : "";
const ACT = {};
const ON_INPUT = {};
const ON_CHANGE = {};

/* ---------- API ---------- */
async function api(method, path, body, opt = {}) {
  const init = { method, headers: {}, credentials: "same-origin" };
  if (body instanceof FormData) init.body = body;
  else if (body !== undefined) { init.headers["Content-Type"] = "application/json"; init.body = JSON.stringify(body); }
  let res;
  try { res = await fetch(path, init); } catch (e) { throw new Error("Mất kết nối với máy chủ Digital Shop. Kiểm tra máy tính chính còn bật phần mềm không."); }
  if (res.status === 401 && !opt.noAuth) { showLogin(); throw new Error("Cần đăng nhập lại"); }
  const ct = res.headers.get("content-type") || "";
  const data = ct.includes("json") ? await res.json() : await res.text();
  if (!res.ok) throw new Error((data && data.error) || "Lỗi " + res.status);
  return data;
}
async function run(fn, okMsg) {
  try { const r = await fn(); if (okMsg) toast(typeof okMsg === "function" ? okMsg(r) : okMsg); return r; }
  catch (e) { toast(e.message, true); return null; }
}

async function refresh(silent) {
  const b = await api("GET", "/api/boot");
  S.me = b.me; S.settings = b.settings || {}; S.hasKey = b.hasKey; S.hasPayKeys = b.hasPayKeys || {}; S.users = b.users || []; S.lan = b.lan || []; S.dataDir = b.dataDir; S.appVersion = b.appVersion || ""; S.inNew = b.inNew || 0; S.today = b.today || dkey();
  S.products = new Map((b.products || []).map(p => [p.id, p]));
  S.customers = new Map((b.customers || []).map(c => [c.id, c]));
  S.suppliers = new Map((b.suppliers || []).map(c => [c.id, c]));
  S.orders = new Map((b.orders || []).map(o => [o.id, o]));
  S.pop = b.pop || {};
  S.version = b.version;
  try { S.lic = await api("GET", "/api/license"); } catch (e) {}
  autoUpdCheck();
  if (!silent) renderShell();
  else render();
}

let pollT;
function startPoll() {
  clearInterval(pollT);
  pollT = setInterval(async () => {
    if (document.hidden) return;
    try {
      const r = await api("GET", "/api/version");
      if (r.v !== S.version) await refresh(true);
    } catch (e) {}
  }, 3500);
}

/* ---------- toast, sheet, xác nhận 2 lần ---------- */
let toastT;
function toast(msg, err) {
  const t = $("#toast"); t.textContent = msg; t.className = "toast" + (err ? " err" : ""); t.hidden = false;
  clearTimeout(toastT); toastT = setTimeout(() => { t.hidden = true; }, err ? 4500 : 2400);
}
let sheetClose = null;
function openSheet(html, opts = {}) {
  const c = $("#sheetCard"); c.className = "sheet-card" + (opts.wide ? " wide" : ""); c.innerHTML = html;
  $("#sheet").hidden = false; sheetClose = opts.onClose || null;
  if (opts.focus) setTimeout(() => { const f = $(opts.focus); if (f) f.focus(); }, 30);
}
function closeSheet() { $("#sheet").hidden = true; $("#sheetCard").innerHTML = ""; const f = sheetClose; sheetClose = null; if (f) f(); }
const sheetOpen = () => !$("#sheet").hidden;
$("#sheet").addEventListener("mousedown", e => { if (e.target.id === "sheet") closeSheet(); });

function copyText(text, ok) {
  const fb = () => openSheet(`<div class="sheet-h"><h2>Sao chép nội dung</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <p class="muted" style="margin:0">Chọn toàn bộ chữ bên dưới rồi sao chép (Ctrl+C).</p><textarea id="copyArea" class="in" rows="10" readonly>${esc(text)}</textarea>`, { focus: "#copyArea" });
  try { navigator.clipboard.writeText(text).then(() => toast(ok || "Đã sao chép"), fb); } catch (e) { fb(); }
}

/* ---------- mặt hàng: tìm, hạn dùng ---------- */
function prodMatch(p, q) {
  if (!q) return true;
  const t = q.trim();
  if (p.barcode === t || (p.code && p.code.toLowerCase() === t.toLowerCase())) return true;
  if ((p.units || []).some(u => u.barcode && u.barcode === t)) return true;
  const hay = norm(p.name + " " + (p.group || "") + " " + (p.alias || "") + " " + (p.code || "") + " " + (p.location || ""));
  return norm(t).split(/\s+/).every(w => hay.includes(w));
}
function byBarcode(code) {
  code = String(code).trim();
  for (const p of S.products.values()) {
    if (p.barcode && p.barcode === code) return { p, unit: null };
    const u = (p.units || []).find(u => u.barcode && u.barcode === code);
    if (u) return { p, unit: u };
  }
  for (const p of S.products.values()) if (p.code && p.code.toLowerCase() === code.toLowerCase()) return { p, unit: null };
  return null;
}
function expInfo(p) {
  const td = S.today, warn = S.settings.expWarn || 30, danger = S.settings.expDanger || 7;
  let expired = 0, near = null, nearQty = 0;
  for (const l of p.lots || []) {
    if (!l.exp) continue;
    if (l.exp < td) expired += l.qty;
    else if (!near || l.exp < near) { near = l.exp; }
  }
  if (near) for (const l of p.lots) if (l.exp === near) nearQty += l.qty;
  const left = near ? daysTo(near) : null;
  let level = "";
  if (expired > 0) level = "red";
  else if (left !== null && left <= danger) level = "red";
  else if (left !== null && left <= warn) level = "amber";
  return { expired: r3(expired), near, nearQty: r3(nearQty), left, level };
}
function stockText(p, q) {
  const s = q ?? (p.stock || 0), u = p.unit || "";
  const big = (p.units || []).filter(x => x.ratio > 1).sort((a, b) => b.ratio - a.ratio)[0];
  if (big && s >= big.ratio) {
    const b = Math.floor(s / big.ratio), rest = r3(s - b * big.ratio);
    return `${fq(b)} ${big.name}${rest ? " " + fq(rest) + " " + u : ""}`;
  }
  return `${fq(s)} ${u}`;
}
function prodStatus(p) {
  if (p.hidden) return { k: "mute", t: "Ngừng bán" };
  const e = expInfo(p);
  if (e.expired > 0) return { k: "red", t: "Có hàng hết hạn" };
  if ((p.stock || 0) <= 0) return { k: "red", t: "Hết hàng" };
  if (e.level === "red") return { k: "red", t: `Hết hạn sau ${e.left} ngày` };
  if (e.level === "amber") return { k: "amber", t: "Cận date" };
  if (!p.isCombo && (p.stock || 0) <= (p.min || 0)) return { k: "amber", t: "Sắp hết" };
  return { k: "green", t: "Còn hàng" };
}
function groups() {
  const set = new Set();
  for (const p of S.products.values()) if (p.group && !p.isCombo) set.add(p.group);
  return [...set].sort((a, b) => a.localeCompare(b, "vi"));
}
function unitOptions(p) { return [{ name: p.unit, ratio: 1, price: p.price }, ...(p.units || [])]; }

/* ---------- ô tìm chọn (dùng lại nhiều nơi) ---------- */
function pickList(kind, q, ctx, limit = 8) {
  let items = [];
  if (kind === "product") {
    items = [...S.products.values()].filter(p => !p.hidden && prodMatch(p, q))
      .sort((a, b) => (S.pop[b.id] || 0) - (S.pop[a.id] || 0) || a.name.localeCompare(b.name, "vi")).slice(0, limit)
      .map(p => `<button type="button" data-act="${ctx}" data-id="${esc(p.id)}"><span>${esc(p.name)}${p.isCombo ? ' <span class="pill blue">combo</span>' : ""}</span><span class="ps num">${esc(stockText(p))} · ${fmt(p.price)}</span></button>`);
  } else {
    const m = kind === "customer" ? S.customers : S.suppliers, nq = norm(q);
    items = [...m.values()].filter(c => !nq || norm(c.name + " " + c.phone + " " + c.code).includes(nq))
      .sort((a, b) => (b.last || 0) - (a.last || 0)).slice(0, limit)
      .map(c => `<button type="button" data-act="${ctx}" data-id="${esc(c.id)}"><span>${esc(c.name)} <span class="ps">${esc(c.phone || "")}</span></span><span class="ps num" style="color:${c.debt > 0 ? "var(--red)" : "inherit"}">${c.debt ? "nợ " + fmt(c.debt) : ""}</span></button>`);
    if (q.trim() && ![...m.values()].some(c => norm(c.name) === nq)) items.push(`<button type="button" data-act="${ctx}-new" data-name="${esc(q.trim())}"><span>+ Thêm mới “${esc(q.trim())}”</span></button>`);
  }
  return items.join("") || `<div class="ps" style="padding:10px 12px">Không tìm thấy</div>`;
}

/* ---------- khung ứng dụng ---------- */
function navBadge(k) {
  if (k === "orders") { const n = [...S.orders.values()].filter(o => o.status === "new").length; return n ? `<span class="badge">${n}</span>` : ""; }
  if (k === "inin") return S.inNew ? `<span class="badge">${S.inNew}</span>` : "";
  if (k === "products") { let n = 0; for (const p of S.products.values()) if (!p.hidden && expInfo(p).level === "red") n++; return n ? `<span class="badge" style="background:var(--red-soft);color:var(--red)">${n}</span>` : ""; }
  return "";
}
function renderShell() {
  const nav = NAV.filter(n => !n[2] || isOwner()).map(([k, t], i) =>
    (k === "combos" ? '<div class="navsep"></div>' : "") + `<button class="nav" data-act="go" data-v="${k}" aria-current="${S.view === k ? "page" : "false"}">${icon(k)}<span>${t}</span>${navBadge(k)}</button>`).join("");
  const d = new Date();
  $("#app").innerHTML = `<div class="shell">
    <aside class="side"><div class="brand"><small>Digital Shop</small><b id="shopName">${esc(S.settings.shopName || "Digital Shop")}</b><span class="num">${WD[d.getDay()]}, ${pad(d.getDate())}/${pad(d.getMonth() + 1)}</span></div>
      <nav id="nav" aria-label="Chức năng">${nav}</nav>
      <div class="me">${icon("user")}<span class="grow">${esc(S.me.name)}${isOwner() ? "" : " · nhân viên"}</span><button class="btn sm ghost" style="color:inherit" data-act="logout">Thoát</button></div></aside>
    <header class="mtop"><b>${esc(S.settings.shopName || "Digital Shop")}</b><button class="btn sm" data-act="go" data-v="settings" style="background:transparent;color:inherit;border-color:rgba(255,255,255,.4)">${icon("settings")}</button></header>
    <main class="main" id="main"></main>
    <nav class="bottom" id="bottom"></nav></div>`;
  render();
}
function renderNav() {
  const n = $("#nav");
  if (n) n.innerHTML = NAV.filter(n => !n[2] || isOwner()).map(([k, t]) =>
    (k === "combos" ? '<div class="navsep"></div>' : "") + `<button class="nav" data-act="go" data-v="${k}" aria-current="${S.view === k ? "page" : "false"}">${icon(k)}<span>${t}</span>${navBadge(k)}</button>`).join("");
  const b = $("#bottom");
  if (b) {
    const main = [["pos", "Bán hàng"], ["orders", "Đơn đặt"], ["products", "Hàng hóa"], ["customers", "Sổ nợ"]];
    const inMain = main.some(m => m[0] === S.view);
    b.innerHTML = main.map(([k, t]) => `<button data-act="go" data-v="${k}" aria-current="${S.view === k ? "page" : "false"}">${icon(k)}${t}${navBadge(k)}</button>`).join("") +
      `<button data-act="moreMenu" aria-current="${inMain ? "false" : "page"}">${icon("more")}Thêm</button>`;
  }
}
const SAFE_IDS = ["main"];
function render() {
  if (!$("#main")) return;
  renderNav();
  const sn = $("#shopName"); if (sn) sn.textContent = S.settings.shopName || "Digital Shop";
  const a = document.activeElement;
  const main = $("#main");
  const fn = VIEWS[S.view] || VIEWS.pos;
  let keep = null;
  if (a && main.contains(a) && a.id && /INPUT|TEXTAREA|SELECT/.test(a.tagName)) { keep = { id: a.id }; try { keep.s = a.selectionStart; keep.e = a.selectionEnd; } catch (e) {} }
  fn();
  licBar();
  if (keep) { const el = document.getElementById(keep.id); if (el && el !== document.activeElement) { el.focus(); try { if (keep.s != null) el.setSelectionRange(keep.s, keep.e); } catch (e) {} } }
}
function go(v) {
  if (!VIEWS[v]) v = "pos";
  S.view = v; store.set("view", v);
  document.body.classList.remove("cart-open");
  try { history.replaceState(null, "", "#" + v); } catch (e) {}
  const m = $("#main"); if (m) m.scrollTop = 0; window.scrollTo(0, 0);
  render();
}
ACT.go = b => { closeSheet(); go(b.dataset.v); };
ACT.close = () => closeSheet();
ACT.moreMenu = () => openSheet(`<div class="sheet-h"><h2>Chức năng</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
  <div class="grid2">${NAV.filter(n => !n[2] || isOwner()).map(([k, t]) => `<button class="btn" style="justify-content:flex-start" data-act="go" data-v="${k}">${icon(k)} ${t}</button>`).join("")}
  <button class="btn" style="justify-content:flex-start" data-act="logout">${icon("user")} Đăng xuất (${esc(S.me.name)})</button></div>`);
ACT.logout = async () => { await api("POST", "/api/logout").catch(() => {}); closeSheet(); showLogin(); };

/* ---------- đăng nhập / thiết lập lần đầu ---------- */
let pinBuf = "";
async function boot() {
  let st;
  try { st = await api("GET", "/api/status", undefined, { noAuth: true }); }
  catch (e) { $("#app").innerHTML = `<div class="splash">${esc(e.message)}</div>`; return; }
  if (st.needSetup) return showSetup();
  if (!st.loggedIn) return showLogin(st.shopName);
  await enterApp();
}
async function enterApp() {
  const h = location.hash.slice(1);
  S.view = VIEWS[h] ? h : store.get("view", "pos");
  await refresh();
  if (!VIEWS[S.view] || (NAV.find(n => n[0] === S.view) || [])[2] && !isOwner()) go("pos");
  startPoll();
}
function showSetup() {
  $("#app").innerHTML = `<div class="login"><form class="box" id="setupForm" autocomplete="off">
    <div class="sign"><small>Bắt đầu dùng</small><b>Digital Shop</b></div>
    <p class="muted" style="margin:0;font-size:14px">Đặt tên cửa hàng và mã PIN của chủ tiệm. Mã PIN dùng để đăng nhập trên máy này và trên điện thoại.</p>
    <div class="field"><label for="su-shop">Tên cửa hàng</label><input id="su-shop" class="in" required placeholder="vd Tạp hóa Nam Linh"></div>
    <div class="field"><label for="su-name">Tên chủ cửa hàng</label><input id="su-name" class="in" required placeholder="vd Anh Linh"></div>
    <div class="grid2"><div class="field"><label for="su-pin">Mã PIN (4–8 số)</label><input id="su-pin" class="in num" inputmode="numeric" type="password" required minlength="4" maxlength="8" pattern="[0-9]{4,8}"></div>
    <div class="field"><label for="su-pin2">Nhập lại PIN</label><input id="su-pin2" class="in num" inputmode="numeric" type="password" required minlength="4" maxlength="8"></div></div>
    <label class="check"><input type="checkbox" id="su-terms" required> <span>Tôi đã đọc và đồng ý <button type="button" class="btn ghost sm" style="padding:0;min-height:0;text-decoration:underline;color:var(--green-text)" data-act="legal" data-f="DieuKhoanSuDung">Điều khoản sử dụng</button> của Digital Shop</span></label>
    <button class="btn pri lg" type="submit">Bắt đầu</button></form></div>`;
  $("#su-shop").focus();
  $("#setupForm").addEventListener("submit", async e => {
    e.preventDefault();
    if ($("#su-pin").value !== $("#su-pin2").value) return toast("Hai lần nhập PIN không giống nhau", true);
    if (!$("#su-terms").checked) return toast("Cần đồng ý Điều khoản sử dụng để bắt đầu", true);
    const r = await run(() => api("POST", "/api/setup", { shopName: $("#su-shop").value, ownerName: $("#su-name").value, pin: $("#su-pin").value, acceptTerms: true }, { noAuth: true }));
    if (r) { await enterApp(); toast("Chào mừng! Bắt đầu bằng mục Nhập từ Excel (lấy dữ liệu KiotViet/MISA) hoặc Hàng hóa"); }
  });
}
function showLogin(shop) {
  clearInterval(pollT); closeSheet();
  pinBuf = "";
  $("#app").innerHTML = `<div class="login"><div class="box">
    <div class="sign"><small>Digital Shop</small><b>${esc(shop || S.settings.shopName || "Đăng nhập")}</b></div>
    <div class="muted" style="text-align:center">Nhập mã PIN của bạn</div>
    <div class="dots" id="dots"></div>
    <input id="pinIn" class="in num" type="password" inputmode="numeric" autocomplete="off" aria-label="Mã PIN" style="text-align:center;font-size:20px;letter-spacing:.3em" maxlength="8">
    <div class="pad">${[1, 2, 3, 4, 5, 6, 7, 8, 9, "C", 0, "OK"].map(k => `<button type="button" data-act="pin" data-k="${k}" ${k === "OK" ? 'style="background:var(--green);color:#fff;border-color:var(--green);font-size:17px"' : ""}>${k === "C" ? "Xóa" : k}</button>`).join("")}</div>
  </div></div>`;
  const pi = $("#pinIn"); pi.focus();
  pi.addEventListener("keydown", e => { if (e.key === "Enter") doLogin(pi.value); });
}
ACT.pin = b => {
  const pi = $("#pinIn"), k = b.dataset.k;
  if (k === "C") pi.value = "";
  else if (k === "OK") return doLogin(pi.value);
  else if (pi.value.length < 8) pi.value += k;
  if (pi.value.length >= 4 && k !== "C" && pi.value.length === 6) {}
};
async function doLogin(pin) {
  if (!pin) return;
  const r = await run(() => api("POST", "/api/login", { pin }, { noAuth: true }));
  if (r) await enterApp(); else { const pi = $("#pinIn"); if (pi) { pi.value = ""; pi.focus(); } }
}

/* ---------- VietQR ---------- */
const BANKS = [["970436", "Vietcombank"], ["970415", "VietinBank"], ["970418", "BIDV"], ["970405", "Agribank"], ["970407", "Techcombank"], ["970422", "MB Bank"],
  ["970416", "ACB"], ["970432", "VPBank"], ["970423", "TPBank"], ["970403", "Sacombank"], ["970437", "HDBank"], ["970441", "VIB"], ["970443", "SHB"],
  ["970448", "OCB"], ["970426", "MSB"], ["970440", "SeABank"], ["970449", "LPBank"], ["970431", "Eximbank"], ["970428", "Nam A Bank"], ["970409", "Bac A Bank"],
  ["970406", "DongA Bank"], ["970425", "ABBANK"], ["970427", "VietABank"], ["970412", "PVcomBank"], ["970438", "BaoViet Bank"], ["970452", "KienlongBank"],
  ["970419", "NCB"], ["970400", "Saigonbank"], ["970454", "BVBank"], ["970433", "Vietbank"], ["970430", "PGBank"], ["970408", "GPBank"]];
function crc16(s) {
  let c = 0xFFFF;
  for (let i = 0; i < s.length; i++) { c ^= s.charCodeAt(i) << 8; for (let j = 0; j < 8; j++) c = (c & 0x8000) ? ((c << 1) ^ 0x1021) & 0xFFFF : (c << 1) & 0xFFFF; }
  return c.toString(16).toUpperCase().padStart(4, "0");
}
function vietqr(amount, info) {
  const st = S.settings;
  if (!st.bankBin || !st.acct) return "";
  const tlv = (id, v) => id + String(v.length).padStart(2, "0") + v;
  const ascii = s => norm(s).replace(/[^a-z0-9 ]/gi, "").toUpperCase().slice(0, 25);
  const acc = tlv("00", "A000000727") + tlv("01", tlv("00", st.bankBin) + tlv("01", st.acct)) + tlv("02", "QRIBFTTA");
  let p = tlv("00", "01") + tlv("01", amount ? "12" : "11") + tlv("38", acc) + tlv("53", "704") + (amount ? tlv("54", String(Math.round(amount))) : "") + tlv("58", "VN") + (info ? tlv("62", tlv("08", ascii(info))) : "") + "6304";
  return p + crc16(p);
}
function qrSvg(text, cls = "") {
  if (!text || typeof qrcode === "undefined") return "";
  const q = qrcode(0, "M"); q.addData(text); q.make();
  return q.createSvgTag({ cellSize: 4, margin: 2, scalable: true }).replace("<svg ", `<svg class="${cls}" `);
}

/* ---------- in hóa đơn ---------- */
function printHtml(html) {
  const P = $("#print"), paper = (S.settings && S.settings.paper) || "80";
  const thermal = paper === "58" || paper === "80";
  const feed = Math.min(30, Math.max(1, +(S.settings && S.settings.printFeed) || 8));
  P.innerHTML = html + (thermal ? `<div style="height:${feed * 1.35}em"></div>` : "");
  fitPage(P, thermal ? (paper === "58" ? 58 : 80) : 0);
  setTimeout(() => { window.print(); }, 80);
}
/* Đặt khổ trang đúng bằng chiều cao phiếu để máy in nhiệt không nhả thêm giấy trắng và cắt đúng chỗ. */
function fitPage(P, widthMm) {
  let st = document.getElementById("pgsz");
  if (!st) { st = document.createElement("style"); st.id = "pgsz"; document.head.appendChild(st); }
  if (!widthMm) { st.textContent = ""; return; }
  const old = P.getAttribute("style");
  P.style.cssText = "display:block;position:absolute;left:-9999px;top:0;width:" + widthMm + "mm;visibility:hidden";
  const h = P.getBoundingClientRect().height;
  if (old === null) P.removeAttribute("style"); else P.setAttribute("style", old);
  const mm = Math.ceil(h * 25.4 / 96) + 2;
  st.textContent = "@page{size:" + widthMm + "mm " + mm + "mm;margin:0}";
}
function receiptHtml(inv) {
  const st = S.settings, paper = st.paper || "80";
  const cls = paper === "58" ? "rc p58" : paper === "a5" ? "rc a5" : "rc";
  const lines = (inv.lines || []).map(l => `<tr><td colspan="3">${esc(l.name)}</td></tr><tr><td>${fq(l.qty)} ${esc(l.unit || "")} x ${fmt(l.price)}</td><td></td><td class="r">${fmt(l.qty * l.price)}</td></tr>`).join("");
  let pay = "";
  if (inv.cash) pay += `<tr><td>Tiền mặt</td><td colspan="2" class="r">${fmt(inv.cash)}</td></tr>`;
  if (inv.given && inv.given > inv.total) pay += `<tr><td>Khách đưa</td><td colspan="2" class="r">${fmt(inv.given)}</td></tr><tr><td>Tiền thối</td><td colspan="2" class="r">${fmt(inv.given - inv.total)}</td></tr>`;
  if (inv.bank) pay += `<tr><td>Chuyển khoản</td><td colspan="2" class="r">${fmt(inv.bank)}</td></tr>`;
  if (inv.debt) pay += `<tr><td><b>Ghi nợ</b></td><td colspan="2" class="r"><b>${fmt(inv.debt)}</b></td></tr>`;
  const cust = inv.custId ? S.customers.get(inv.custId) : null;
  const qr = inv.debt && st.acct ? `<div class="c" style="margin-top:4px">Quét mã để chuyển khoản trả nợ</div>${qrSvg(vietqr(inv.debt, (inv.code || "") + " " + (inv.custName || "")), "qr")}` : "";
  return `<div class="${cls}">
    <div class="c shop">${esc(st.shopName || "Cửa hàng")}</div>
    ${st.addr ? `<div class="c">${esc(st.addr)}</div>` : ""}${st.phone ? `<div class="c">ĐT: ${esc(st.phone)}</div>` : ""}
    <hr><div class="c big">HÓA ĐƠN BÁN HÀNG</div>
    <div class="c">${esc(inv.code || "")} · ${dt(inv.time)}/${new Date(inv.time).getFullYear()}</div>
    ${inv.custName ? `<div>Khách: ${esc(inv.custName)}${cust && cust.phone ? " · " + esc(cust.phone) : ""}</div>` : ""}
    <div>Thu ngân: ${esc(inv.staff || "")}</div><hr>
    <table>${lines}</table><hr>
    <table><tr><td>Tiền hàng</td><td colspan="2" class="r">${fmt(inv.sub)}</td></tr>
    ${inv.disc ? `<tr><td>Giảm giá</td><td colspan="2" class="r">-${fmt(inv.disc)}</td></tr>` : ""}
    <tr><td class="big">TỔNG CỘNG</td><td colspan="2" class="r big">${fmt(inv.total)}</td></tr>${pay}
    ${cust && inv.debt ? `<tr><td>Tổng nợ hiện tại</td><td colspan="2" class="r">${fmt(cust.debt)}</td></tr>` : ""}</table>
    ${inv.note ? `<div>Ghi chú: ${esc(inv.note)}</div>` : ""}${qr}
    <hr><div class="c">${esc(st.footer || "Cảm ơn quý khách!")}</div></div>`;
}
function receiptText(inv) {
  const st = S.settings, L = [(st.shopName || "Cửa hàng").toUpperCase()];
  if (st.addr) L.push(st.addr); if (st.phone) L.push("ĐT: " + st.phone);
  L.push(`${inv.code} · ${dt(inv.time)}`, "--------------------");
  (inv.lines || []).forEach(l => L.push(l.name, `  ${fq(l.qty)} ${l.unit || ""} x ${fmt(l.price)} = ${fmt(l.qty * l.price)}`));
  L.push("--------------------");
  if (inv.disc) L.push(`Giảm giá: -${fmt(inv.disc)}`);
  L.push(`TỔNG CỘNG: ${money(inv.total)}`);
  if (inv.debt) L.push(`Ghi nợ: ${money(inv.debt)}`);
  L.push(st.footer || "Cảm ơn quý khách!");
  return L.join("\n");
}

/* ---------- sự kiện toàn cục ---------- */
document.addEventListener("click", e => {
  const b = e.target.closest("[data-act]"); if (!b || b.disabled) return;
  if (b.dataset.confirm && !b.dataset.armed) {
    e.preventDefault(); b.dataset.armed = "1"; b._t = b.innerHTML; b.textContent = b.dataset.confirm;
    setTimeout(() => { if (b.isConnected && b.dataset.armed) { delete b.dataset.armed; b.innerHTML = b._t; } }, 3500); return;
  }
  const f = ACT[b.dataset.act]; if (f) { e.preventDefault(); f(b, e); }
});
document.addEventListener("input", e => { const k = e.target.dataset.in || e.target.id; const f = ON_INPUT[k]; if (f) f(e.target, e); });
document.addEventListener("change", e => { const k = e.target.dataset.ch || e.target.dataset.in || e.target.id; const f = ON_CHANGE[k]; if (f) f(e.target, e); });
document.addEventListener("keydown", e => { if (e.key === "Escape" && sheetOpen()) closeSheet(); });
window.addEventListener("hashchange", () => { const h = location.hash.slice(1); if (VIEWS[h] && h !== S.view && S.me) go(h); });

/* ---------- khung trang dùng chung ---------- */
function periodChips(cur, act) {
  const P = [["today", "Hôm nay"], ["yday", "Hôm qua"], ["7d", "7 ngày"], ["month", "Tháng này"], ["lmonth", "Tháng trước"], ["30d", "30 ngày"], ["90d", "90 ngày"]];
  return P.map(([k, t]) => `<button class="chip" data-act="${act}" data-v="${k}" aria-pressed="${cur === k}">${t}</button>`).join("");
}
function periodRange(k, custom) {
  const t = dkey();
  if (k === "today") return [t, t];
  if (k === "yday") { const y = addDays(t, -1); return [y, y]; }
  if (k === "7d") return [addDays(t, -6), t];
  if (k === "30d") return [addDays(t, -29), t];
  if (k === "90d") return [addDays(t, -89), t];
  if (k === "month") return [t.slice(0, 8) + "01", t];
  if (k === "lmonth") { const d = new Date(t.slice(0, 8) + "01T00:00:00"); d.setDate(0); const e = dkey(d); return [e.slice(0, 8) + "01", e]; }
  if (k === "custom" && custom) return custom;
  return [t, t];
}

/* ---------- điều khoản & giấy phép ---------- */
ACT.legal = async b => {
  const f = b.dataset.f, title = f === "DieuKhoanSuDung" ? "Điều khoản sử dụng" : "Giấy phép bên thứ ba";
  let txt = "";
  try { const r = await fetch("legal/" + f + ".txt"); txt = await r.text(); } catch (e) { return toast("Không mở được tài liệu", true); }
  openSheet(`<div class="sheet-h"><h2>${title}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div style="white-space:pre-wrap;font-size:13.5px;line-height:1.55;max-height:65vh;overflow:auto;background:var(--sunk);border-radius:10px;padding:14px">${esc(txt)}</div>
    <div class="row"><span class="ps grow">Bản đầy đủ cũng có trong thư mục cài đặt.</span><button class="btn pri" data-act="close">Đóng</button></div>`, { wide: true });
};

/* tiếng "ting" báo tiền về */
function ding() {
  try {
    const A = window.AudioContext || window.webkitAudioContext; if (!A) return;
    const c = ding.c || (ding.c = new A()), t = c.currentTime;
    [[880, 0], [1320, 0.12]].forEach(([f, d]) => { const o = c.createOscillator(), g = c.createGain(); o.frequency.value = f; o.type = "sine";
      g.gain.setValueAtTime(0.0001, t + d); g.gain.exponentialRampToValueAtTime(0.35, t + d + 0.02); g.gain.exponentialRampToValueAtTime(0.0001, t + d + 0.5);
      o.connect(g); g.connect(c.destination); o.start(t + d); o.stop(t + d + 0.55); });
  } catch (e) {}
}

/* ---------- bản quyền: thanh nhắc ở đầu trang ---------- */
function licBar() {
  const main = $("#main"), l = S.lic; if (!main || !l) return;
  const old = $("#licBar"); if (old) old.remove();
  let msg = "", bad = false;
  if (l.state === "trial-over") { msg = "Đã hết thời gian dùng thử. Dữ liệu vẫn xem và xuất được; muốn bán hàng tiếp, hãy mua bản quyền."; bad = true; }
  else if (l.state === "revoked") { msg = "Bản quyền đã bị thu hồi. Dữ liệu vẫn xem và xuất được; liên hệ Nam Linh Group để được hỗ trợ."; bad = true; }
  else if (l.revokeOn) msg = "Bản quyền đã bị thu hồi. Từ " + fmtRev(l.revokeOn) + " phần mềm sẽ chuyển sang chế độ chỉ xem và xuất dữ liệu. Liên hệ Nam Linh Group (0968.646.853) để được hỗ trợ.", bad = true;
  else if (l.state === "expired") { msg = "Bản quyền đã hết hạn. Gia hạn để tiếp tục bán hàng."; bad = true; }
  else if (l.state === "trial" && l.daysLeft <= 7) msg = "Dùng thử còn " + l.daysLeft + " ngày.";
  else if (l.state === "active" && l.license && l.license.exp && l.daysLeft <= 15) msg = "Bản quyền còn " + l.daysLeft + " ngày, gia hạn trước khi hết hạn.";
  if (!msg) return;
  const d = document.createElement("div");
  d.id = "licBar";
  d.style.cssText = "margin:0 0 10px;padding:9px 14px;border-radius:8px;font-size:14px;display:flex;gap:10px;align-items:center;flex-wrap:wrap;background:" + (bad ? "#fde2e0;color:#8a1c14" : "#fff4d6;color:#6b4a00");
  d.innerHTML = "<span style='flex:1'>" + esc(msg) + "</span>" + (isOwner() ? "<button class='btn sm' data-act='go' data-v='settings'>Bản quyền</button>" : "<span class='ps'>Nhờ chủ cửa hàng xử lý</span>");
  main.prepend(d);
}

/* ---------- tự kiểm tra bản mới mỗi ngày (chỉ chủ cửa hàng, chỉ báo, không tự cài) ---------- */
let updChecked = false;
async function autoUpdCheck() {
  if (updChecked || !isOwner() || (S.settings && S.settings.updateOff)) return;
  updChecked = true;
  try {
    const last = +store.get("updAt", 0) || 0;
    if (Date.now() - last < 20 * 3600 * 1000) return;
    const r = await api("POST", "/api/update/check");
    store.set("updAt", Date.now());
    if (r && r.newer) toast("Có bản mới " + r.latest + ". Vào Cài đặt → Cập nhật phần mềm để cài.");
  } catch (e) {}
}

function fmtRev(iso) {
  const d = new Date(iso), p = n => String(n).padStart(2, "0");
  return p(d.getHours()) + ":" + p(d.getMinutes()) + " ngày " + p(d.getDate()) + "/" + p(d.getMonth() + 1) + "/" + d.getFullYear();
}
