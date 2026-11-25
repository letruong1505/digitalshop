// Nam Linh Group - mua bản quyền online (Digital Shop / Digital FnB)
// Luồng: khách chọn gói -> /api/order tạo đơn + mã QR -> khách chuyển khoản -> SePay gọi /sepay
// -> Worker ký mã kích hoạt (cùng định dạng công cụ CapBanQuyen) -> trang mua / phần mềm nhận mã.
import { BUY_PAGE, ADMIN_PAGE } from "./pages.js";

const PRODUCTS = { shop: "Digital Shop", fnb: "Digital FnB" };
const PLANS = { "1y": { label: "1 năm", years: 1 }, "2y": { label: "2 năm", years: 2 }, life: { label: "Vĩnh viễn", years: 0 } };
const CODE_ALPHABET = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ";
const CODE_RE = /NLBQ[2-9A-HJ-NP-Z]{6}/g;

const json = (o, status = 200) => new Response(JSON.stringify(o), {
  status, headers: { "content-type": "application/json; charset=utf-8", "cache-control": "no-store" },
});
const html = (s) => new Response(s, {
  headers: {
    "content-type": "text/html; charset=utf-8", "cache-control": "no-store",
    "x-content-type-options": "nosniff", "referrer-policy": "no-referrer",
    "content-security-policy": "default-src 'self'; img-src 'self' https://qr.sepay.vn data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'",
  },
});
const fail = (msg, status = 400) => json({ error: msg }, status);

// ---------- tiện ích ----------
const enc = new TextEncoder();
const b64u = (bytes) => btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
const hex = (bytes) => [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
async function sha256hex(s) { return hex(new Uint8Array(await crypto.subtle.digest("SHA-256", enc.encode(s)))); }
function randFrom(alphabet, n) {
  const a = crypto.getRandomValues(new Uint8Array(n));
  return [...a].map((x) => alphabet[x % alphabet.length]).join("");
}
function same(a, b) { // so sánh thời gian không đổi
  a = String(a); b = String(b);
  let d = a.length ^ b.length;
  for (let i = 0; i < Math.max(a.length, b.length); i++) d |= (a.charCodeAt(i) || 0) ^ (b.charCodeAt(i) || 0);
  return d === 0;
}
const vnToday = () => new Date(Date.now() + 7 * 3600e3).toISOString().slice(0, 10);
function addYears(iso, n) {
  const [y, m, d] = iso.split("-").map(Number);
  const t = new Date(Date.UTC(y + n, m - 1, d));
  if (t.getUTCMonth() !== (m - 1 + 12) % 12) t.setUTCDate(0); // 29/02 -> 28/02
  return t.toISOString().slice(0, 10);
}
function normMachine(s) { // giống công cụ cấp mã: khách gõ tay nhầm 0/O, 1/I, 8/B
  s = String(s || "").toUpperCase().replace(/[-\s]/g, "");
  return s.replace(/0/g, "O").replace(/1/g, "I").replace(/8/g, "B");
}
const fmtMachine = (m) => `${m.slice(0, 4)}-${m.slice(4, 8)}-${m.slice(8, 12)}-${m.slice(12, 16)}`;
const clean = (s, max) => String(s ?? "").replace(/[\u0000-\u001f\u007f]/g, " ").replace(/\s+/g, " ").trim().slice(0, max);

function prices(env) {
  try { return JSON.parse(env.PRICES); } catch { return {}; }
}
function bankInfo(env) {
  return { acc: env.BANK_ACC, name: env.BANK_NAME, bank: env.BANK_LABEL || "", contact: env.CONTACT || "" };
}
function qrUrl(env, amount, code) {
  const q = new URLSearchParams({ acc: env.BANK_ACC, bank: env.BANK_QR || "MBBank", amount: String(amount), des: code, template: "compact" });
  return "https://qr.sepay.vn/img?" + q;
}

// ---------- ký mã kích hoạt (cùng định dạng DS1 của phần mềm) ----------
let keyCache = null;
async function signKey(env) {
  if (keyCache && keyCache.seed === env.LICENSE_SEED) return keyCache.key;
  const raw = String(env.LICENSE_SEED || "").replace(/^﻿/, "").trim();
  if (!raw) throw new Error("Chưa đặt LICENSE_SEED");
  const seed = Uint8Array.from(atob(raw), (c) => c.charCodeAt(0));
  if (seed.length !== 32) throw new Error("LICENSE_SEED không phải khóa 32 byte");
  const pkcs8 = new Uint8Array([0x30, 0x2e, 0x02, 0x01, 0x00, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x04, 0x22, 0x04, 0x20, ...seed]);
  const key = await crypto.subtle.importKey("pkcs8", pkcs8, { name: "Ed25519" }, true, ["sign"]);
  keyCache = { seed: env.LICENSE_SEED, key };
  return key;
}
async function publicKeyB64(env) {
  const jwk = await crypto.subtle.exportKey("jwk", await signKey(env));
  const x = jwk.x.replace(/-/g, "+").replace(/_/g, "/");
  return x + "=".repeat((4 - (x.length % 4)) % 4);
}

const b64uDec = (s) => Uint8Array.from(atob(s.replace(/-/g, "+").replace(/_/g, "/") + "=".repeat((4 - (s.length % 4)) % 4)), (c) => c.charCodeAt(0));

// Ký lại các mã đã cấp bằng khóa hiện tại (dùng khi trước đó Worker từng ký bằng khóa sai).
// Giữ nguyên nội dung mã (số bản quyền, máy, hạn), chỉ ký lại; mã nào đã đúng chữ ký thì bỏ qua.
async function resignAll(env) {
  const priv = await signKey(env);
  const jwk = await crypto.subtle.exportKey("jwk", priv);
  const pub = await crypto.subtle.importKey("jwk", { kty: "OKP", crv: "Ed25519", x: jwk.x }, { name: "Ed25519" }, false, ["verify"]);
  const rows = await env.DB.prepare("SELECT code,license_key FROM orders WHERE status='paid' AND license_key IS NOT NULL").all();
  let fixed = 0, ok = 0, bad = 0;
  for (const r of rows.results) {
    const parts = String(r.license_key).split(".");
    if (parts.length !== 3 || parts[0] !== "DS1") { bad++; continue; }
    const payload = b64uDec(parts[1]);
    if (await crypto.subtle.verify("Ed25519", pub, b64uDec(parts[2]), payload)) { ok++; continue; }
    const sig = new Uint8Array(await crypto.subtle.sign("Ed25519", priv, payload));
    await env.DB.prepare("UPDATE orders SET license_key=?1 WHERE code=?2").bind("DS1." + parts[1] + "." + b64u(sig), r.code).run();
    fixed++;
  }
  return { fixed, ok, bad };
}

// Tạo mã cho đơn: gia hạn trước hạn thì cộng dồn từ ngày hết hạn cũ.
async function makeLicense(env, order) {
  const today = vnToday();
  let exp = "";
  const years = PLANS[order.plan].years;
  if (years > 0) {
    let base = today;
    const prev = await env.DB.prepare(
      "SELECT MAX(exp) AS e FROM orders WHERE product=?1 AND machine=?2 AND status='paid' AND exp<>'' AND exp>=?3"
    ).bind(order.product, order.machine, today).first();
    if (prev && prev.e) base = prev.e;
    exp = addYears(base, years);
  }
  const lic = { v: 1, id: `NL-${today.slice(2).replace(/-/g, "")}-${randFrom("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", 4)}`, m: fmtMachine(order.machine), shop: order.shop };
  if (order.mst) lic.mst = order.mst;
  lic.plan = env.PLAN === "co-ban" ? "co-ban" : "hddt";
  if (exp) lic.exp = exp;
  lic.iss = today;
  const payload = enc.encode(JSON.stringify(lic));
  const sig = new Uint8Array(await crypto.subtle.sign("Ed25519", await signKey(env), payload));
  return { key: "DS1." + b64u(payload) + "." + b64u(sig), id: lic.id, exp };
}

// Đánh dấu đã trả tiền + cấp mã. Trả về true nếu đơn này vừa được cấp (chống cấp trùng khi webhook gửi lặp).
async function settle(env, order, txnId, amount, note = "") {
  const lic = await makeLicense(env, order);
  const r = await env.DB.prepare(
    "UPDATE orders SET status='paid', paid_at=?1, paid_amount=?2, txn_id=?3, license_key=?4, license_id=?5, exp=?6, note=?7 WHERE code=?8 AND status<>'paid'"
  ).bind(Date.now(), amount, txnId, lic.key, lic.id, lic.exp, note, order.code).run();
  return (r.meta?.changes ?? 0) === 1;
}

// ---------- API khách ----------
async function createOrder(req, env) {
  let b;
  try { b = await req.json(); } catch { return fail("Dữ liệu không hợp lệ"); }
  const product = String(b.sp || "");
  if (!PRODUCTS[product]) return fail("Chọn phần mềm cần mua");
  const plan = String(b.plan || "");
  if (!PLANS[plan]) return fail("Chọn thời hạn bản quyền");
  const amount = Number(prices(env)[product]?.[plan]);
  if (!Number.isInteger(amount) || amount < 1000) return fail("Gói này chưa có giá, liên hệ Nam Linh Group");
  const machine = normMachine(b.machine);
  if (!/^[A-Z2-7]{16}$/.test(machine)) return fail("Mã máy phải gồm 16 ký tự dạng XXXX-XXXX-XXXX-XXXX. Lấy trong phần mềm: Cài đặt → Bản quyền.");
  const shop = clean(b.shop, 80);
  if (!shop) return fail("Nhập tên cửa hàng");
  const phone = String(b.phone || "").replace(/[\s.\-]/g, "");
  if (!/^(0|\+84)\d{9,10}$/.test(phone)) return fail("Số điện thoại không đúng");
  const email = clean(b.email, 100);
  if (email && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) return fail("Email không đúng");
  const mst = clean(b.mst, 20);
  if (!/^(\d{10}(-\d{3})?|\d{12})?$/.test(mst)) return fail("Mã số thuế không đúng dạng (10 hoặc 13 số, hoặc CCCD 12 số)");
  if (b.agree !== true) return fail("Cần đồng ý điều khoản để tiếp tục");

  const ip = req.headers.get("CF-Connecting-IP") || "";
  const iph = await sha256hex("ip|" + ip);
  const since = Date.now() - 3600e3;
  const n = await env.DB.prepare("SELECT COUNT(*) AS n FROM orders WHERE ip_hash=?1 AND created_at>?2").bind(iph, since).first();
  if (n && n.n >= 20) return fail("Tạo quá nhiều đơn, thử lại sau ít phút", 429);

  const life = await env.DB.prepare("SELECT 1 AS x FROM orders WHERE product=?1 AND machine=?2 AND status='paid' AND exp='' LIMIT 1").bind(product, machine).first();
  if (life) return fail("Máy này đã có bản quyền vĩnh viễn, không cần mua thêm. Mở phần mềm và bấm \"Nhận mã đã mua\".");

  const token = hex(crypto.getRandomValues(new Uint8Array(16)));
  for (let i = 0; i < 5; i++) {
    const code = "NLBQ" + randFrom(CODE_ALPHABET, 6);
    try {
      await env.DB.prepare(
        "INSERT INTO orders (code,token,product,plan,amount,machine,shop,phone,email,mst,status,created_at,ip_hash) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,'pending',?11,?12)"
      ).bind(code, token, product, plan, amount, machine, shop, phone, email, mst, Date.now(), iph).run();
      return json({ code, token, amount, qr: qrUrl(env, amount, code), bank: bankInfo(env), product: PRODUCTS[product], plan: PLANS[plan].label });
    } catch (e) {
      if (!/UNIQUE|constraint/i.test(String(e))) throw e;
    }
  }
  return fail("Không tạo được đơn, thử lại", 500);
}

async function getOrder(url, env) {
  const code = (url.searchParams.get("code") || "").toUpperCase();
  const t = url.searchParams.get("t") || "";
  const o = await env.DB.prepare("SELECT * FROM orders WHERE code=?1").bind(code).first();
  if (!o || !same(o.token, t)) return fail("Không tìm thấy đơn", 404);
  const out = { code: o.code, status: o.status, amount: o.amount, product: PRODUCTS[o.product], plan: PLANS[o.plan]?.label, note: o.note || "" };
  if (o.status === "pending") { out.qr = qrUrl(env, o.amount, o.code); out.bank = bankInfo(env); out.created = o.created_at; }
  if (o.status === "paid") { out.key = o.license_key; out.exp = o.exp; out.id = o.license_id; }
  return json(out);
}

// Phần mềm tự nhận mã đã mua theo mã máy (phần mềm kiểm tra lại chữ ký, nên lộ mã này cũng không ai dùng được ở máy khác).
async function claim(url, env) {
  const product = url.searchParams.get("sp") || "";
  const machine = normMachine(url.searchParams.get("m"));
  if (!PRODUCTS[product] || !/^[A-Z2-7]{16}$/.test(machine)) return fail("Thiếu mã máy");
  const today = vnToday();
  const o = await env.DB.prepare(
    "SELECT license_key,license_id,exp,shop FROM orders WHERE product=?1 AND machine=?2 AND status='paid' AND (exp='' OR exp>=?3) ORDER BY (exp='') DESC, exp DESC, paid_at DESC LIMIT 1"
  ).bind(product, machine, today).first();
  if (!o) return json({ ok: false });
  return json({ ok: true, key: o.license_key, id: o.license_id, exp: o.exp, shop: o.shop });
}

// ---------- webhook SePay ----------
async function sepay(req, env) {
  const auth = req.headers.get("Authorization") || "";
  if (!env.SEPAY_API_KEY || !same(auth, "Apikey " + env.SEPAY_API_KEY)) return json({ success: false, message: "unauthorized" }, 401);
  let b;
  try { b = await req.json(); } catch { return json({ success: false, message: "bad json" }, 400); }
  if (String(b.transferType || "in") !== "in") return json({ success: true });
  if (b.accountNumber && String(b.accountNumber) !== String(env.BANK_ACC)) return json({ success: true });

  const txnId = String(b.id ?? b.referenceCode ?? "");
  const amount = Math.round(Number(b.transferAmount) || 0);
  const content = clean(`${b.code ?? ""} ${b.content ?? ""} ${b.description ?? ""}`, 400);
  const text = content.toUpperCase().replace(/[^A-Z0-9]/g, "");
  const log = (code, result) => env.DB.prepare("INSERT OR REPLACE INTO txns (id,at,amount,content,order_code,result) VALUES (?1,?2,?3,?4,?5,?6)")
    .bind(txnId || "t" + Date.now(), Date.now(), amount, content, code, result).run();

  const prev = txnId ? await env.DB.prepare("SELECT result FROM txns WHERE id=?1").bind(txnId).first() : null;
  if (prev && prev.result !== "error") return json({ success: true });

  let order = null;
  for (const c of text.match(CODE_RE) || []) {
    order = await env.DB.prepare("SELECT * FROM orders WHERE code=?1").bind(c).first();
    if (order) break;
  }
  if (!order) { await log(null, "no-match"); return json({ success: true }); }
  if (order.status === "paid") { await log(order.code, "already-paid"); return json({ success: true }); }
  if (amount < order.amount) {
    await env.DB.prepare("UPDATE orders SET note=?1 WHERE code=?2").bind(`Chuyển thiếu: nhận ${amount}, cần ${order.amount} (giao dịch ${txnId})`, order.code).run();
    await log(order.code, "short");
    return json({ success: true });
  }
  try {
    await settle(env, order, txnId, amount, amount > order.amount ? `Chuyển dư ${amount - order.amount}đ` : "");
    await log(order.code, "paid");
    return json({ success: true });
  } catch (e) {
    await log(order.code, "error");
    return json({ success: false, message: "issue failed" }, 500); // SePay sẽ gửi lại
  }
}

// ---------- thu hồi ----------
// Bảng revoked giữ số bản quyền (NL-...) bị thu hồi; phần mềm hỏi định kỳ qua /api/check.
async function ensureRevoked(env) {
  await env.DB.prepare("CREATE TABLE IF NOT EXISTS revoked (license_id TEXT PRIMARY KEY, at INTEGER NOT NULL, note TEXT NOT NULL DEFAULT '')").run();
}

async function check(url, env) {
  const id = String(url.searchParams.get("id") || "").toUpperCase().slice(0, 40);
  if (!/^NL-[A-Z0-9-]{3,30}$/.test(id)) return fail("Thiếu số bản quyền");
  await ensureRevoked(env);
  const r = await env.DB.prepare("SELECT at FROM revoked WHERE license_id=?1").bind(id).first();
  return json({ ok: true, revoked: !!r });
}

// ---------- quản trị ----------
async function admin(req, url, env) {
  if (!env.ADMIN_TOKEN || !same(req.headers.get("Authorization") || "", "Bearer " + env.ADMIN_TOKEN)) return fail("Sai mật khẩu", 401);
  const path = url.pathname;
  if (path === "/admin/api/orders" && req.method === "GET") {
    const rows = await env.DB.prepare(
      "SELECT code,product,plan,amount,machine,shop,phone,email,mst,status,created_at,paid_at,paid_amount,txn_id,license_id,license_key,exp,note FROM orders ORDER BY created_at DESC LIMIT 300"
    ).all();
    const tx = await env.DB.prepare("SELECT id,at,amount,content,order_code,result FROM txns WHERE result IN ('no-match','short','error','already-paid') ORDER BY at DESC LIMIT 50").all();
    return json({ orders: rows.results, odd: tx.results });
  }
  if (path === "/admin/api/status" && req.method === "GET") {
    let pub = "", err = "";
    try { pub = await publicKeyB64(env); } catch (e) { err = String(e.message || e); }
    return json({ pub, keyError: err, sepay: !!env.SEPAY_API_KEY, plan: env.PLAN, prices: prices(env) });
  }
  if (path === "/admin/api/resign" && req.method === "POST") return json(await resignAll(env));
  if (path === "/admin/api/revoke" && req.method === "POST") {
    const b = await req.json().catch(() => ({}));
    await ensureRevoked(env);
    let id = String(b.id || "").toUpperCase().trim();
    if (b.code) {
      const o = await env.DB.prepare("SELECT license_id FROM orders WHERE code=?1").bind(String(b.code).toUpperCase()).first();
      if (!o) return fail("Không thấy đơn", 404);
      if (!o.license_id) return fail("Đơn này chưa cấp mã");
      id = o.license_id;
      await env.DB.prepare("UPDATE orders SET status='revoked', note=?1 WHERE code=?2").bind("Thu hồi: " + clean(b.note, 100), String(b.code).toUpperCase()).run();
    }
    if (!/^NL-[A-Z0-9-]{3,30}$/.test(id)) return fail("Số bản quyền không đúng dạng NL-…");
    await env.DB.prepare("INSERT OR REPLACE INTO revoked(license_id,at,note) VALUES(?1,?2,?3)").bind(id, Date.now(), clean(b.note, 100)).run();
    return json({ ok: true, id });
  }
  if (path === "/admin/api/unrevoke" && req.method === "POST") {
    const b = await req.json().catch(() => ({}));
    await ensureRevoked(env);
    let id = String(b.id || "").toUpperCase().trim();
    if (b.code) {
      const o = await env.DB.prepare("SELECT license_id FROM orders WHERE code=?1").bind(String(b.code).toUpperCase()).first();
      if (!o) return fail("Không thấy đơn", 404);
      id = o.license_id;
      await env.DB.prepare("UPDATE orders SET status='paid', note='' WHERE code=?1 AND status='revoked'").bind(String(b.code).toUpperCase()).run();
    }
    if (!id) return fail("Thiếu số bản quyền");
    await env.DB.prepare("DELETE FROM revoked WHERE license_id=?1").bind(id).run();
    return json({ ok: true, id });
  }
  if (path === "/admin/api/revoked" && req.method === "GET") {
    await ensureRevoked(env);
    return json({ list: (await env.DB.prepare("SELECT license_id,at,note FROM revoked ORDER BY at DESC LIMIT 200").all()).results });
  }
  if (path === "/admin/api/approve" && req.method === "POST") {
    const b = await req.json().catch(() => ({}));
    const o = await env.DB.prepare("SELECT * FROM orders WHERE code=?1").bind(String(b.code || "").toUpperCase()).first();
    if (!o) return fail("Không thấy đơn", 404);
    if (o.status === "paid") return fail("Đơn này đã cấp mã rồi");
    const ok = await settle(env, o, "manual", o.amount, "Duyệt tay: " + clean(b.note, 100));
    return json({ ok });
  }
  return fail("Không có", 404);
}

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const p = url.pathname;
    try {
      if (req.method === "GET" && (p === "/" || p === "/mua")) return html(BUY_PAGE);
      if (req.method === "GET" && p === "/admin") return html(ADMIN_PAGE);
      if (req.method === "GET" && p === "/api/config") return json({ prices: prices(env), products: PRODUCTS, plans: Object.fromEntries(Object.entries(PLANS).map(([k, v]) => [k, v.label])), bank: bankInfo(env) });
      if (req.method === "POST" && p === "/api/order") return await createOrder(req, env);
      if (req.method === "GET" && p === "/api/order") return await getOrder(url, env);
      if (req.method === "GET" && p === "/api/claim") return await claim(url, env);
      if (req.method === "GET" && p === "/api/check") return await check(url, env);
      if (req.method === "POST" && p === "/sepay") return await sepay(req, env);
      if (p.startsWith("/admin/api/")) return await admin(req, url, env);
      return fail("Không có", 404);
    } catch (e) {
      console.error(e);
      return fail("Lỗi máy chủ, thử lại sau", 500);
    }
  },
};
