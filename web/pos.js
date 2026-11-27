/* Digital Shop — màn hình bán hàng: nhiều hóa đơn chờ, mã vạch, đơn vị quy đổi, VietQR, in */
"use strict";
S.tabs = store.get("tabs", null) || [];
S.tabIdx = store.get("tabIdx", 0);
S.posQ = ""; S.posGroup = ""; S.posFilter = ""; S.lastSale = null; S.custOpen = false; S.custQ = "";
const newTab = () => ({ cid: uid(), lines: [], custId: "", method: "cash", given: "", disc: "", note: "", orderId: "" });
if (!S.tabs.length) S.tabs = [newTab()];
function curTab() { if (S.tabIdx >= S.tabs.length) S.tabIdx = S.tabs.length - 1; if (S.tabIdx < 0) S.tabIdx = 0; return S.tabs[S.tabIdx]; }
function saveTabs() { store.set("tabs", S.tabs.map(({ promo, promoSig, ...r }) => r)); store.set("tabIdx", S.tabIdx); }
const canEditPrice = () => can("price") || S.settings.staffEditPrice;

function tabTotals(t) {
  const sub = Math.round(t.lines.reduce((a, l) => a + l.qty * l.price, 0));
  const promo = Math.min((t.promo && t.promo.disc) || 0, sub);
  const disc = Math.min(parseMoney(t.disc), sub - promo);
  const pts = ptsAllowed(t, sub - promo - disc);
  const ptsDisc = Math.min(Math.round(pts.use * (S.settings.loyalty || {}).pointValue || 0), sub - promo - disc);
  return { sub, disc, promo, total: Math.max(0, sub - promo - disc - ptsDisc), n: t.lines.length, pts: pts.use, ptsDisc, ptsMax: pts.max, ptsBal: pts.bal };
}
/* Tích điểm: số điểm dùng được = min(điểm khách có, trần % hóa đơn); máy chủ kiểm tra lại lúc lập hóa đơn */
function ptsAllowed(t, payable) {
  const lo = S.settings.loyalty || {}, c = t.custId ? S.customers.get(t.custId) : null;
  const bal = c ? c.points || 0 : 0;
  if (!lo.on || !c || !lo.pointValue || (APP && S.offline) || payable <= 0 || bal <= 0) return { use: 0, max: 0, bal };
  let cap = lo.maxPct > 0 && lo.maxPct < 100 ? payable * lo.maxPct / 100 : payable;
  let max = Math.min(bal, Math.floor(cap / lo.pointValue));
  if (lo.minRedeem && max < lo.minRedeem) max = 0;
  const want = Math.max(0, Math.floor(+t.pts || 0));
  const use = want && max && want >= (lo.minRedeem || 0) ? Math.min(want, max) : 0;
  return { use, max, bal };
}
/* Khuyến mãi tự động: hỏi máy chủ (máy chủ cũng tự tính lại lúc lập hóa đơn nên luôn đúng giờ, đúng hạng khách) */
let promoTimer = null;
const promoSig = t => JSON.stringify([t.custId, t.lines.map(l => [l.pid, l.unit, l.ratio, l.qty, l.price])]);
function promoSync(t) {
  if (!(S.settings.promos || []).some(p => p.active) || !t.lines.length) { t.promo = null; t.promoSig = ""; return; }
  const sig = promoSig(t);
  if (APP && S.offline) { t.promoSig = sig; t.promo = offPromo(t); return; } // mất mạng: tính khuyến mãi ngay trên máy
  if (t.promoSig === sig) return;
  t.promoSig = sig; clearTimeout(promoTimer);
  promoTimer = setTimeout(async () => {
    try {
      const r = await api("POST", "/api/promo/quote", { lines: t.lines.map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio, qty: l.qty, price: l.price })), custId: t.custId });
      if (t.promoSig !== sig) return;
      const old = (t.promo && t.promo.disc) || 0;
      t.promo = r.disc > 0 ? { disc: r.disc, hits: r.hits || [] } : null;
      if (((t.promo && t.promo.disc) || 0) !== old && curTab() === t) {
        const a = document.activeElement;
        if (!(a && ["posGiven", "posNote", "posDisc", "custQ"].includes(a.id))) renderCart();
        else { const d = $("#posDue"); if (d) d.textContent = money(tabTotals(t).total); }
      }
    } catch (e) {}
  }, 250);
}
function addLine(pid, unit) {
  const p = S.products.get(pid); if (!p) return;
  const t = curTab();
  const u = unit || { name: p.unit, ratio: 1, price: p.price };
  const ex = t.lines.find(l => l.pid === pid && l.unit === u.name);
  if (ex) ex.qty = r3(ex.qty + 1);
  else t.lines.push({ pid, unit: u.name, ratio: u.ratio, qty: 1, price: u.price });
  const e = expInfo(p);
  if (e.expired > 0 && S.settings.blockExpired) toast(`${p.name}: có ${fq(e.expired)} ${p.unit} đã hết hạn, đừng lấy nhầm lô cũ`, true);
  saveTabs(); renderTiles(); renderCart();
}

VIEWS.pos = function () {
  $("#main").innerHTML = `<div class="pos">
    <section class="shelf page" style="max-width:none">
      <div class="shelf-top">
        <div class="search grow">${icon("search")}<input id="posQ" class="in" type="search" autocomplete="off" placeholder="Tìm tên, mã hàng hoặc quét mã vạch (F2)" value="${esc(S.posQ)}"></div>
        <button class="btn only-d" data-act="dispMenu" title="Màn hình cho khách">Màn hình khách${S.dispOn ? ' <span class="pill green">bật</span>' : ""}</button>
        <button class="btn only-d" data-act="go" data-v="orders" title="Đơn đặt">${icon("orders")} Đơn đặt <span class="pill amber" id="ordCount"></span></button>
      </div>
      <div id="posWarn"></div>
      <div class="chips" id="posChips"></div>
      <div class="gridwrap" id="posGrid"></div>
    </section>
    <aside class="cart" id="cart" aria-label="Hóa đơn"></aside>
    <div class="cartbar" id="cartBar"></div></div>`;
  renderTiles(); renderCart();
  if (matchMedia("(min-width:900px)").matches && !sheetOpen()) setTimeout(() => { const q = $("#posQ"); if (q && document.activeElement === document.body) q.focus(); }, 0);
};

function renderTiles() {
  if (S.view !== "pos" || !$("#posGrid")) return;
  const n = [...S.orders.values()].filter(o => o.status === "new").length;
  const oc = $("#ordCount"); if (oc) oc.textContent = n || "";
  const all = [...S.products.values()].filter(p => !p.hidden);
  let red = 0, amber = 0, out = 0;
  for (const p of all) { const e = expInfo(p); if (e.level === "red") red++; else if (e.level === "amber") amber++; if (!p.isCombo && p.stock <= 0) out++; }
  $("#posWarn").innerHTML = (red || amber) ? `<div class="warnstrip">
      ${red ? `<button data-act="posFilter" data-v="exp" style="background:var(--red-soft);color:var(--red)">${red} mặt hàng hết hạn hoặc sắp hết hạn trong ${S.settings.expDanger || 7} ngày</button>` : ""}
      ${amber ? `<button data-act="posFilter" data-v="exp" style="background:var(--amber-soft);color:var(--amber)">${amber} mặt hàng cận date (≤ ${S.settings.expWarn || 30} ngày)</button>` : ""}</div>` : "";
  const gs = groups();
  const hasCombo = all.some(p => p.isCombo);
  $("#posChips").innerHTML = [`<button class="chip" data-act="posGroup" data-v="" aria-pressed="${!S.posGroup && !S.posFilter}">Tất cả</button>`,
    hasCombo ? `<button class="chip" data-act="posGroup" data-v="__combo" aria-pressed="${S.posGroup === "__combo"}">Combo</button>` : "",
    `<button class="chip" data-act="posFilter" data-v="exp" aria-pressed="${S.posFilter === "exp"}">Cận date</button>`,
    ...gs.map(g => `<button class="chip" data-act="posGroup" data-v="${esc(g)}" aria-pressed="${S.posGroup === g}">${esc(g)}</button>`)].join("");
  const q = S.posQ;
  let items = all.filter(p => (!S.posGroup || (S.posGroup === "__combo" ? p.isCombo : p.group === S.posGroup)) && prodMatch(p, q) && (S.posFilter !== "exp" || expInfo(p).level));
  items.sort((a, b) => S.posFilter === "exp" ? ((expInfo(a).left ?? -999) - (expInfo(b).left ?? -999)) : ((S.pop[b.id] || 0) - (S.pop[a.id] || 0) || a.name.localeCompare(b.name, "vi")));
  const inCart = {}; curTab().lines.forEach(l => inCart[l.pid] = r3((inCart[l.pid] || 0) + l.qty * l.ratio));
  let h;
  if (!S.products.size) h = `<div class="empty"><b>Chưa có mặt hàng nào</b>Bấm <b>Nhập từ file Excel</b> để lấy danh mục từ KiotViet / MISA, hoặc <b>Hàng hóa → + Mặt hàng</b>.<div class="row" style="justify-content:center;margin-top:12px">${isOwner() ? `<button class="btn pri" data-act="impGo" data-k="products">Nhập từ file Excel</button>` : ""}<button class="btn" data-act="go" data-v="products">Thêm mặt hàng</button></div></div>`;
  else if (!items.length) h = `<div class="empty"><b>Không tìm thấy “${esc(q)}”</b>Thử gõ ngắn hơn hoặc bỏ dấu.${can("stock") && q ? ` <button class="btn sm" data-act="newProduct" data-name="${esc(q)}">Thêm mặt hàng mới</button>` : ""}</div>`;
  else {
    const show = items.slice(0, 120);
    h = `<div class="tiles">` + show.map(p => {
      const e = expInfo(p), s = p.stock || 0;
      const cls = s <= 0 ? "out" : (!p.isCombo && s <= (p.min || 0) ? "low" : "");
      let st = s <= 0 ? "Hết hàng" : "Còn " + esc(stockText(p));
      let badge = "";
      if (e.expired > 0) badge = `<span class="pill red">${fq(e.expired)} hết hạn</span>`;
      else if (e.level) badge = `<span class="pill ${e.level}">HSD ${dm(e.near)}</span>`;
      return `<button class="tile ${e.level ? "exp-" + e.level : ""} ${p.isCombo ? "combo" : ""}" data-act="add" data-id="${esc(p.id)}" title="${esc(p.name)}${p.location ? " · " + esc(p.location) : ""}">
        ${inCart[p.id] ? `<span class="inq num">${fq(inCart[p.id])}</span>` : ""}
        ${p.img ? `<img class="timg" loading="lazy" alt="" src="${esc(imgUrl(p))}">` : ""}
        <span class="nm">${esc(p.name)}</span><span><span class="tag">${fmt(p.price)}</span></span>
        <span class="st ${cls} num">${st}${p.location ? ` · ${esc(p.location)}` : ""} ${badge}</span></button>`;
    }).join("") + `</div>`;
    if (items.length > show.length) h += `<div class="muted small" style="padding:8px 2px">Còn ${items.length - show.length} mặt hàng khác, gõ tên để tìm nhanh.</div>`;
  }
  $("#posGrid").innerHTML = h;
}

function ptsRowHtml(t, T, cust) {
  const lo = S.settings.loyalty || {};
  if (!lo.on || !cust) return "";
  const earn = lo.earnPer > 0 ? Math.floor(T.total / lo.earnPer) : 0;
  const bal = cust.points || 0;
  let h = `<div class="sumrow" style="align-items:center"><span class="muted">Điểm: <b>${fmt(bal)}</b>${T.ptsMax ? ` <span class="ps">(dùng tối đa ${fmt(T.ptsMax)})</span>` : ""}</span>`;
  if (T.ptsMax) h += `<span class="row" style="gap:6px"><input id="posPts" class="in sm r num" style="width:80px" inputmode="numeric" placeholder="0" value="${t.pts ? esc(String(T.pts)) : ""}" aria-label="Số điểm dùng"><button class="btn sm" data-act="ptsAll">Dùng hết</button></span>`;
  h += `</div>`;
  if (T.ptsDisc > 0) h += `<div class="sumrow" style="color:var(--green-text)"><span>Dùng ${fmt(T.pts)} điểm</span><span class="num">-${money(T.ptsDisc)}</span></div>`;
  if (earn > 0) h += `<div class="sumrow"><span class="ps">Đơn này được cộng</span><span class="ps num">+${fmt(earn)} điểm</span></div>`;
  return h;
}
function renderCart() {
  if (S.view !== "pos" || !$("#cart")) return;
  const t = curTab(); promoSync(t);
  const T = tabTotals(t);
  const cust = t.custId ? S.customers.get(t.custId) : null;
  let h = `<div class="tabs" role="tablist">${S.tabs.map((x, i) => { const n = x.lines.length; return `<button role="tab" data-act="tab" data-i="${i}" aria-current="${i === S.tabIdx}">HĐ ${i + 1}${n ? ` <span class="pill mute">${n}</span>` : ""}${x.orderId ? ' <span class="pill amber">đặt</span>' : ""}${S.tabs.length > 1 ? `<span class="tx" data-act="closeTab" data-i="${i}" title="Đóng">×</span>` : ""}</button>`; }).join("")}
    <button data-act="newTab" title="Hóa đơn mới (F4)">+</button><span class="grow"></span><button class="x only-m" data-act="closeCart" aria-label="Đóng hóa đơn">×</button></div>`;
  if (t.editId) h += `<div class="editbar"><span class="grow">Đang sửa hóa đơn <b>${esc(t.editCode || "")}</b>: hóa đơn cũ sẽ bị thay bằng hóa đơn này khi thanh toán</span><button class="btn sm" data-act="editCancel">Bỏ sửa</button></div>`;
  h += `<div class="cust">`;
  if (S.custOpen) {
    h += `<div class="search">${icon("search")}<input id="custQ" class="in" autocomplete="off" placeholder="Tìm khách theo tên, số điện thoại (F8)" value="${esc(S.custQ)}"></div><div class="dd" id="custDD">${pickList("customer", S.custQ, "posCust")}</div>`;
  } else if (cust) {
    h += `<div class="row"><button class="custbtn grow" data-act="custOpen">${icon("user")}<span>${esc(cust.name)} <span class="ps">${esc(cust.phone || "")}</span>${cust.tier ? ` <span class="pill amber">${esc(cust.tier)}</span>` : ""}</span>${cust.debt ? `<span class="d num">nợ ${fmt(cust.debt)}</span>` : ""}</button><button class="x" data-act="custClear" title="Bỏ chọn khách">×</button></div>`;
  } else {
    h += `<button class="custbtn" data-act="custOpen">${icon("user")}<span class="muted">Khách lẻ · bấm để chọn khách (F8)</span></button>`;
  }
  h += `</div>`;
  if (!T.n) {
    h += `<div class="lines"><div class="empty" style="margin:14px 0">Bấm vào mặt hàng hoặc quét mã vạch để thêm.<div style="margin-top:6px"><span class="kbd">F2</span> tìm · <span class="kbd">F4</span> hóa đơn mới · <span class="kbd">F8</span> khách · <span class="kbd">F9</span> thanh toán</div></div></div>`;
    if (S.lastSale) {
      const s = S.lastSale;
      h += `<div class="lastsale"><div class="sumrow"><b>${esc(s.code)} · ${hm(s.time)}</b><b class="num">${money(s.total)}</b></div>
        <div class="muted small">${esc(s.lines.map(l => `${l.name} ×${fq(l.qty)}`).join(", "))}</div>
        ${s.given > s.total ? `<div class="num">Khách đưa ${money(s.given)} · Thối lại <b>${money(s.given - s.total)}</b></div>` : ""}
        ${s.debt ? `<div class="num" style="color:var(--red)">Ghi nợ ${money(s.debt)} cho ${esc(s.custName)}</div>` : ""}
        <div class="row"><button class="btn sm" data-act="printLast">${icon("print")} In hóa đơn</button><button class="btn sm" data-act="copyLast">Sao chép</button>${canEditInv(s) ? `<button class="btn sm" data-act="editInv" data-id="${esc(s.id)}">Sửa đơn / đổi thanh toán</button>` : ""}</div></div>`;
    }
    $("#cart").innerHTML = h; renderCartBar(); return;
  }
  const ep = canEditPrice();
  h += `<div class="lines">` + t.lines.map((l, i) => {
    const p = S.products.get(l.pid) || { name: "(đã xóa)", unit: l.unit, units: [] };
    const units = unitOptions(p);
    const sw = units.length > 1 ? `<span class="unit">${units.map(u => `<button data-act="lineUnit" data-i="${i}" data-u="${esc(u.name)}" aria-pressed="${u.name === l.unit}">${esc(u.name)}</button>`).join("")}</span>` : `<span class="ps">${esc(l.unit)}</span>`;
    const e = expInfo(p);
    return `<div class="line"><div class="ln">${esc(p.name)}${e.level ? `<small style="color:var(--${e.level === "red" ? "red" : "amber"})">HSD gần nhất ${vnDate(e.near) || "—"}${e.expired ? ` · ${fq(e.expired)} ${esc(p.unit)} đã hết hạn` : ""}</small>` : ""}</div>
      <button class="x" data-act="lineRm" data-i="${i}" aria-label="Bỏ">×</button>
      <div class="row" style="gap:6px"><span class="stepper"><button data-act="lineDec" data-i="${i}" aria-label="Bớt">−</button><input id="lq-${i}" data-ch="lineQty" data-i="${i}" inputmode="decimal" value="${fq(l.qty)}" aria-label="Số lượng"><button data-act="lineInc" data-i="${i}" aria-label="Thêm">+</button></span>
        ${sw}<span class="ps">×</span><input id="lp-${i}" class="lp" data-ch="linePrice" data-i="${i}" inputmode="numeric" value="${fmt(l.price)}" aria-label="Đơn giá" ${ep ? "" : "readonly"}></div>
      <div class="lt">${fmt(l.qty * l.price)}</div></div>`;
  }).join("") + `<div class="field" style="padding:8px 0"><input id="posNote" class="in sm" placeholder="Ghi chú hóa đơn" value="${esc(t.note)}"></div></div>`;
  h += `<div class="pay">
    <div class="sumrow"><span class="muted">Tiền hàng (${T.n} món)</span><span class="num">${money(T.sub)}</span></div>
    ${T.promo > 0 ? `<div class="sumrow" style="color:var(--green-text)"><span>Khuyến mãi${t.promo && t.promo.hits.length ? ": " + esc(t.promo.hits.map(x => x.name).join(", ")) : ""}</span><span class="num">-${money(T.promo)}</span></div>` : ""}
    <div class="sumrow"><label for="posDisc" class="muted">Giảm giá</label><input id="posDisc" class="in sm r num" style="width:130px" inputmode="numeric" placeholder="0" value="${esc(t.disc)}"></div>
    ${ptsRowHtml(t, T, cust)}
    <div class="sumrow"><span class="lbl">Khách cần trả</span><span class="due" id="posDue">${money(T.total)}</span></div>
    <div class="seg" role="group" aria-label="Hình thức thanh toán">
      <button data-act="payM" data-v="cash" aria-pressed="${t.method === "cash"}">Tiền mặt</button>
      <button data-act="payM" data-v="bank" aria-pressed="${t.method === "bank"}">Chuyển khoản</button>
      <button data-act="payM" data-v="debt" aria-pressed="${t.method === "debt"}">Ghi nợ</button></div>`;
  if (t.method === "cash") {
    const sugg = [...new Set([T.total, ...[10000, 20000, 50000, 100000, 200000, 500000].map(u => Math.ceil(T.total / u) * u)])].filter(v => v >= T.total && v > 0).slice(0, 5);
    h += `<div class="sumrow"><label for="posGiven" class="muted">Khách đưa</label><input id="posGiven" class="in r num" style="width:150px" inputmode="numeric" placeholder="vd 200k" value="${esc(t.given)}"></div>
      <div class="quick">${sugg.map(v => `<button data-act="given" data-v="${v}">${fmt(v)}</button>`).join("")}</div><div id="changeOut">${changeHtml(parseMoney(t.given), T.total, cust)}</div>`;
  } else if (t.method === "bank") {
    h += bankBoxHtml(t, T);
  } else {
    h += cust ? `<div class="banner red"><span>Ghi nợ <b>${money(T.total)}</b> cho <b>${esc(cust.name)}</b>. Tổng nợ sau đơn: <b>${money((cust.debt || 0) + T.total)}</b></span></div>`
      : `<div class="banner red">Chọn khách hàng ở phía trên để ghi nợ.</div>`;
  }
  const autoBank = t.method === "bank" && S.settings.payProvider && !(t.pay && t.pay.status === "paid");
  h += `<button class="btn ${autoBank ? "" : "pri"} lg" data-act="checkout" id="checkoutBtn">${t.method === "debt" ? "Ghi nợ" : autoBank ? "Khách đã chuyển, hoàn tất tay" : "Thanh toán"} ${money(T.total)} <span class="kbd" style="color:inherit;border-color:currentColor">F9</span></button></div>`;
  $("#cart").innerHTML = h;
  renderCartBar();
  postDisplay();
}

/* ---- chuyển khoản: mã QR + tự xác nhận tiền về (SePay / payOS) ---- */
function payQR(t, T) {
  if (S.settings.payProvider && t.pay && t.pay.amount === T.total) return t.pay.provider === "payos" ? t.pay.qr : vietqr(T.total, t.pay.code);
  if (S.settings.payProvider) return "";
  return vietqr(T.total, (S.settings.shopName || "TAP HOA") + " " + fmt(T.total).replace(/\./g, ""));
}
function bankBoxHtml(t, T) {
  const st = S.settings;
  if (!st.payProvider && !st.acct) return `<div class="banner">Chưa có số tài khoản nhận tiền.${isAdmin() ? ` <button class="btn sm" data-act="go" data-v="settings">Thêm trong Cài đặt</button>` : ""}</div>`;
  if (st.payProvider && (!t.pay || t.pay.amount !== T.total || t.pay.status === "expired")) { ensurePay(); return `<div class="qrbox"><div class="grow">Đang tạo mã QR cho ${money(T.total)}…</div></div>`; }
  const qr = payQR(t, T), p = t.pay;
  let status = `<div class="small" style="margin-top:4px">Khách quét mã, số tiền <b>${money(T.total)}</b> đã điền sẵn.</div>`;
  if (p && st.payProvider) {
    status = p.status === "paid"
      ? `<div class="change" style="margin-top:6px"><span>Đã nhận ${money(p.amount)}</span><span>✓</span></div>`
      : `<div class="small" style="margin-top:6px"><span class="pill amber">Đang chờ tiền về…</span> nội dung <b class="num">${esc(p.code)}</b></div>${p.msg ? `<div class="small" style="color:var(--red)">${esc(p.msg)}</div>` : ""}`;
  }
  return `<div class="qrbox">${qrSvg(qr)}<div class="grow" style="min-width:0"><div class="ps">${esc(st.bankName || (p && p.provider === "payos" ? "payOS" : ""))}</div><div class="acct num">${esc(p && p.provider === "payos" ? "" : st.acct)}</div><div class="small">${esc(p && p.provider === "payos" ? "" : st.holder || "")}</div>${status}</div></div>`;
}
let payBusy = false;
async function ensurePay() {
  if (payBusy) return; payBusy = true;
  const t = curTab(), T = tabTotals(t);
  try {
    const r = await api("POST", "/api/pay/start", { amount: T.total });
    if (curTab() === t && tabTotals(t).total === r.amount) { t.pay = { code: r.code, amount: r.amount, qr: r.qr || "", provider: r.provider, status: "pending" }; saveTabs(); }
  } catch (e) { toast(e.message, true); t.method = "cash"; saveTabs(); }
  finally { payBusy = false; if (S.view === "pos") renderCart(); }
}
setInterval(async () => {
  if (!S.me || S.view !== "pos" || document.hidden) return;
  const t = curTab();
  if (t.method !== "bank" || !t.pay || t.pay.status !== "pending" || t.pay.amount !== tabTotals(t).total) return;
  try {
    const r = await api("GET", "/api/pay/status?code=" + encodeURIComponent(t.pay.code));
    if (r.status !== t.pay.status || r.msg !== t.pay.msg) {
      t.pay.status = r.status; t.pay.msg = r.msg || ""; saveTabs();
      if (r.status === "paid") {
        toast(`Đã nhận ${money(r.amount)} chuyển khoản`);
        ding();
        if (S.settings.autoBankDone) return checkout();
      }
      renderCart();
    }
  } catch (e) {}
}, 3000);

/* ---- màn hình cho khách ---- */
S.dispOn = store.get("dispOn", false);
let dispT = null, dispLast = "";
function postDisplay(force) {
  if (!S.dispOn || !S.me) return;
  clearTimeout(dispT);
  dispT = setTimeout(() => {
    const t = curTab(), T = tabTotals(t), st = S.settings;
    const lines = t.lines.map(l => { const p = S.products.get(l.pid) || {}; return { n: p.name || l.name || "", q: l.qty, u: l.unit, p: l.price, t: Math.round(l.qty * l.price) }; });
    const qr = t.method === "bank" && T.total > 0 ? payQR(t, T) : "";
    const d = { status: T.n ? (qr ? "qr" : "cart") : "idle", lines: lines.slice(-10), count: lines.length, sub: T.sub, disc: T.disc + T.promo, total: T.total, method: t.method,
      qr, bank: { name: st.bankName || "", acct: st.acct || "", holder: st.holder || "" }, paid: !!(t.pay && t.pay.status === "paid"), cust: (S.customers.get(t.custId) || {}).name || "" };
    const s = JSON.stringify(d);
    if (s === dispLast && !force) return;
    dispLast = s;
    api("POST", "/api/display", d).catch(() => {});
  }, 120);
}
function postDisplayPaid(inv) {
  if (!S.dispOn) return;
  dispLast = "";
  api("POST", "/api/display", { status: "paid", total: inv.total, method: inv.bank ? "bank" : inv.debt && !inv.cash ? "debt" : "cash", change: inv.given > inv.total ? inv.given - inv.total : 0 }).catch(() => {});
  setTimeout(() => { if (!tabTotals(curTab()).n) postDisplay(true); }, 7000);
}
const displayURL = () => `${APP ? srvBase() : location.origin}/khach.html?k=${S.settings.displayKey || ""}`;
ACT.dispMenu = () => {
  const lan = (APP ? srvBase() : (S.lan[0] || location.origin)) + "/khach.html?k=" + (S.settings.displayKey || "");
  openSheet(`<div class="sheet-h"><h2>Màn hình cho khách</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <label class="check"><input type="checkbox" id="dispOn" ${S.dispOn ? "checked" : ""}> Gửi hóa đơn đang bán ở máy này lên màn hình khách</label>
    <div class="grid2" style="align-items:start">
      <div class="card stack"><h3>Màn hình thứ hai cắm vào máy này</h3><p class="ps" style="margin:0">Cắm màn hình (HDMI, VGA hoặc USB‑C) quay về phía khách. Bấm nút dưới, kéo cửa sổ mới sang màn hình đó rồi nhấn F11 để toàn màn hình.</p>
        <button class="btn pri" data-act="dispOpen">Mở cửa sổ màn hình khách</button></div>
      <div class="card stack"><h3>Máy tính bảng / điện thoại cũ đặt ở quầy</h3><p class="ps" style="margin:0">Bắt cùng wifi, quét mã này hoặc mở địa chỉ bên dưới.</p>
        <div style="width:150px;background:#fff;padding:6px;border-radius:8px">${qrSvg(lan)}</div><div class="num small" style="user-select:all;word-break:break-all">${esc(lan)}</div></div></div>
    <p class="ps" style="margin:0">Màn hình khách hiện danh sách hàng, tổng tiền, mã QR khi chọn chuyển khoản, và báo "Thanh toán thành công" khi tiền về.</p>`);
};
ON_CHANGE.dispOn = el => { S.dispOn = el.checked; store.set("dispOn", S.dispOn); if (S.dispOn) postDisplay(true); };
ACT.dispOpen = () => {
  if (!S.dispOn) { S.dispOn = true; store.set("dispOn", true); const c = $("#dispOn"); if (c) c.checked = true; }
  const w = window.open(displayURL(), "taphoax_khach", "popup,width=1024,height=640");
  if (!w) copyText(displayURL(), "Trình duyệt chặn cửa sổ mới. Đã sao chép địa chỉ, dán vào cửa sổ trình duyệt khác");
  postDisplay(true);
};
function changeHtml(g, total, cust) {
  if (!g) return "";
  const d = g - total;
  if (d >= 0) return `<div class="change"><span>Tiền thối lại</span><span class="num">${money(d)}</span></div>`;
  return `<div class="change neg"><span>Còn thiếu ${money(-d)}${cust ? ` → ghi nợ cho ${esc(cust.name)}` : " (chọn khách để ghi nợ)"}</span></div>`;
}
function renderCartBar() {
  const b = $("#cartBar"); if (!b) return;
  const T = tabTotals(curTab());
  b.innerHTML = T.n ? `<button data-act="openCart"><span class="cnt num">${T.n}</span><span class="grow" style="text-align:left">Xem hóa đơn</span><span class="num">${money(T.total)}</span></button>` : "";
  if (!T.n && !S.lastSale) document.body.classList.remove("cart-open");
}

/* ---- hành động ---- */
ACT.add = b => addLine(b.dataset.id);
ACT.posGroup = b => { S.posGroup = b.dataset.v; S.posFilter = ""; renderTiles(); };
ACT.posFilter = b => { S.posFilter = S.posFilter === b.dataset.v ? "" : b.dataset.v; S.posGroup = ""; renderTiles(); };
ACT.tab = (b, e) => { if (e.target.closest(".tx")) return; S.tabIdx = +b.dataset.i; S.custOpen = false; saveTabs(); renderTiles(); renderCart(); };
ACT.closeTab = (b, e) => { e.stopPropagation(); const i = +b.dataset.i; S.tabs.splice(i, 1); if (!S.tabs.length) S.tabs.push(newTab()); if (S.tabIdx >= i && S.tabIdx > 0) S.tabIdx--; saveTabs(); renderTiles(); renderCart(); };
ACT.newTab = () => { if (S.tabs.length >= 9) return toast("Tối đa 9 hóa đơn chờ", true); S.tabs.push(newTab()); S.tabIdx = S.tabs.length - 1; S.lastSale = null; saveTabs(); renderTiles(); renderCart(); };
ACT.lineInc = b => { const l = curTab().lines[+b.dataset.i]; l.qty = r3(l.qty + 1); saveTabs(); renderTiles(); renderCart(); };
ACT.lineDec = b => { const t = curTab(), i = +b.dataset.i, l = t.lines[i]; l.qty = r3(l.qty - 1); if (l.qty <= 0) t.lines.splice(i, 1); saveTabs(); renderTiles(); renderCart(); };
ACT.lineRm = b => { curTab().lines.splice(+b.dataset.i, 1); saveTabs(); renderTiles(); renderCart(); };
ACT.lineUnit = b => {
  const l = curTab().lines[+b.dataset.i], p = S.products.get(l.pid); if (!p) return;
  const u = unitOptions(p).find(x => x.name === b.dataset.u); if (!u) return;
  l.unit = u.name; l.ratio = u.ratio; l.price = u.price; saveTabs(); renderTiles(); renderCart();
};
ON_CHANGE.lineQty = el => { const t = curTab(), i = +el.dataset.i, q = parseQty(el.value); if (q <= 0) t.lines.splice(i, 1); else t.lines[i].qty = q; saveTabs(); el.blur(); renderTiles(); renderCart(); };
ON_CHANGE.linePrice = el => { if (!canEditPrice()) return; curTab().lines[+el.dataset.i].price = parseMoney(el.value); saveTabs(); el.blur(); renderCart(); };
ON_INPUT.posQ = el => { S.posQ = el.value; renderTiles(); };
ON_INPUT.posDisc = el => { curTab().disc = el.value; saveTabs(); const T = tabTotals(curTab()); const d = $("#posDue"); if (d) d.textContent = money(T.total); const c = $("#checkoutBtn"); if (c) c.firstChild.textContent = (curTab().method === "debt" ? "Ghi nợ " : "Thanh toán ") + money(T.total) + " "; };
ON_CHANGE.posDisc = el => { el.blur(); renderCart(); };
ON_CHANGE.posPts = el => { curTab().pts = Math.max(0, Math.floor(+String(el.value).replace(/\D/g, "") || 0)); saveTabs(); renderCart(); };
ACT.ptsAll = () => { const t = curTab(); t.pts = tabTotals({ ...t, pts: 1e9 }).pts; saveTabs(); renderCart(); };
ON_INPUT.posNote = el => { curTab().note = el.value; saveTabs(); };
ON_INPUT.posGiven = el => { curTab().given = el.value; saveTabs(); const c = curTab(); const o = $("#changeOut"); if (o) o.innerHTML = changeHtml(parseMoney(el.value), tabTotals(c).total, c.custId ? S.customers.get(c.custId) : null); };
ON_CHANGE.posGiven = el => { const g = parseMoney(el.value); curTab().given = g ? fmt(g) : ""; el.value = curTab().given; saveTabs(); };
ACT.given = b => { curTab().given = fmt(+b.dataset.v); saveTabs(); renderCart(); };
ACT.payM = b => { curTab().method = b.dataset.v; saveTabs(); if (b.dataset.v === "debt" && !curTab().custId) { S.custOpen = true; S.custQ = ""; } renderCart(); if (S.custOpen) setTimeout(() => { const q = $("#custQ"); if (q) q.focus(); }, 0); };
ACT.custOpen = () => { S.custOpen = true; S.custQ = ""; renderCart(); setTimeout(() => { const q = $("#custQ"); if (q) q.focus(); }, 0); };
ACT.custClear = () => { curTab().custId = ""; curTab().pts = 0; saveTabs(); renderCart(); };
ACT.posCust = b => { curTab().custId = b.dataset.id; S.custOpen = false; saveTabs(); renderCart(); };
ACT["posCust-new"] = async b => {
  const c = await run(() => api("POST", "/api/customers", { name: b.dataset.name }), "Đã thêm khách vào sổ");
  if (c) { S.customers.set(c.id, c); curTab().custId = c.id; S.custOpen = false; saveTabs(); renderCart(); }
};
ON_INPUT.custQ = el => { S.custQ = el.value; const d = $("#custDD"); if (d) d.innerHTML = pickList("customer", el.value, "posCust"); };
ACT.openCart = () => document.body.classList.add("cart-open");
ACT.closeCart = () => document.body.classList.remove("cart-open");
ACT.printLast = () => S.lastSale && printHtml(receiptHtml(S.lastSale));
ACT.copyLast = () => S.lastSale && copyText(receiptText(S.lastSale), "Đã sao chép hóa đơn");
/* Sửa đơn sau thanh toán: mở lại các dòng của hóa đơn trong một tab mới; thanh toán xong thì thay hóa đơn cũ */
function canEditInv(v) {
  if (!v || v.void || v.editedTo || v.src === "import" || S.offline) return false;
  return can("manage") || (can("sell") && v.staff === S.me.name && Date.now() - v.time <= 30 * 60 * 1000);
}
ACT.editInv = b => {
  const v = (S.lastSale && S.lastSale.id === b.dataset.id) ? S.lastSale : (typeof findInv === "function" ? findInv(b.dataset.id) : null);
  if (!v) return toast("Không tìm thấy hóa đơn", true);
  if (!canEditInv(v)) return toast("Chỉ sửa được hóa đơn của bạn trong 30 phút. Hóa đơn cũ hơn cần quản lý (quyền hủy hóa đơn)", true);
  if (S.tabs.some(x => x.editId === v.id)) { S.tabIdx = S.tabs.findIndex(x => x.editId === v.id); } else {
    if (S.tabs.length >= 9) return toast("Tối đa 9 hóa đơn chờ", true);
    const t = newTab();
    t.editId = v.id; t.editCode = v.code; t.custId = v.custId || ""; t.note = (v.note || "").replace(/\s*\(.*\)\s*$/, "");
    t.lines = v.lines.filter(l => S.products.has(l.pid)).map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio || 1, qty: l.qty, price: l.price }));
    t.method = v.bank > 0 ? "bank" : (v.debt > 0 && !(v.cash > 0)) ? "debt" : "cash";
    const man = Math.max(0, (v.disc || 0) - (v.promo || 0) - (v.ptsDisc || 0));
    t.disc = man ? fmt(man) : ""; t.pts = v.ptsUsed || 0;
    S.tabs.push(t); S.tabIdx = S.tabs.length - 1;
  }
  S.lastSale = null; saveTabs();
  if (S.view !== "pos") go("pos"); else { renderTiles(); renderCart(); }
  closeSheet();
  document.body.classList.add("cart-open");
};
ACT.editCancel = () => { const t = curTab(); S.tabs.splice(S.tabIdx, 1); if (!S.tabs.length) S.tabs.push(newTab()); if (S.tabIdx >= S.tabs.length) S.tabIdx = S.tabs.length - 1; saveTabs(); renderTiles(); renderCart(); };
ACT.checkout = () => checkout();

let busy = false;
async function checkout() {
  const t = curTab(), T = tabTotals(t);
  if (!T.n) return toast("Hóa đơn đang trống", true);
  if (t.method === "debt" && !t.custId) { S.custOpen = true; renderCart(); return toast("Chọn khách hàng để ghi nợ", true); }
  const g = parseMoney(t.given);
  if (t.method === "cash" && g && g < T.total && !t.custId) return toast(`Khách còn thiếu ${money(T.total - g)}. Chọn khách để ghi nợ phần còn lại`, true);
  if (busy) return; busy = true;
  const btn = $("#checkoutBtn"); if (btn) btn.disabled = true;
  try {
    const paidNote = t.method === "bank" && t.pay && t.pay.status === "paid" && t.pay.amount === T.total ? `CK đã xác nhận tự động (${t.pay.code})` : "";
    const body = { cid: t.cid, lines: t.lines.map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio, qty: l.qty, price: l.price })), disc: T.disc, points: T.pts || 0, method: t.method, given: g, custId: t.custId, note: [t.note, paidNote].filter(Boolean).join(" · "), orderId: t.orderId };
    let inv;
    if (t.editId) {
      if (S.offline) throw new Error("Đang mất kết nối, chưa sửa được đơn");
      body.cid = ""; inv = await api("POST", `/api/invoices/${t.editId}/edit`, body);
    } else if (APP && S.offline) inv = await offSale(body, t, T);
    else {
      try { inv = await api("POST", "/api/invoices", body); }
      catch (e) { if (APP && e.net) { setOffline(true); t.promo = offPromo(t); inv = await offSale(body, t, tabTotals(t)); } else throw e; } // gửi lỗi mạng: lưu trên máy, đẩy sau (máy chủ tự bỏ qua nếu đã nhận)
    }
    postDisplayPaid(inv);
    S.lastSale = inv;
    S.tabs.splice(S.tabIdx, 1);
    if (!S.tabs.length) S.tabs.push(newTab());
    if (S.tabIdx >= S.tabs.length) S.tabIdx = S.tabs.length - 1;
    saveTabs();
    S.posQ = "";
    toast(t.editId ? `Đã sửa đơn ${t.editCode} → ${inv.code}` : inv.debt ? `Đã ghi nợ ${money(inv.debt)} cho ${inv.custName}` : `Đã bán ${money(inv.total)}${inv.given > inv.total ? " · thối " + money(inv.given - inv.total) : ""}`);
    await refresh(true);
    if (S.settings.autoPrint) printHtml(receiptHtml(inv));
  } catch (e) { toast(e.message, true); }
  finally { busy = false; const b2 = $("#checkoutBtn"); if (b2) b2.disabled = false; }
}

/* ---- phím tắt & máy quét mã vạch ---- */
document.addEventListener("keydown", e => {
  if (!S.me || sheetOpen()) return;
  if (e.key === "F2") { e.preventDefault(); if (S.view !== "pos") go("pos"); const q = $("#posQ"); if (q) { q.focus(); q.select(); } }
  if (S.view !== "pos") return;
  if (e.key === "F4") { e.preventDefault(); ACT.newTab(); }
  if (e.key === "F8") { e.preventDefault(); ACT.custOpen(); }
  if (e.key === "F9") { e.preventDefault(); checkout(); }
  if (e.key === "Escape" && S.custOpen) { S.custOpen = false; renderCart(); }
  if (e.key === "Enter" && e.target.id === "posQ") {
    e.preventDefault();
    const q = e.target.value.trim(); if (!q) { if (tabTotals(curTab()).n) checkout(); return; }
    const hit = byBarcode(q);
    if (hit) { addLine(hit.p.id, hit.unit); S.posQ = ""; e.target.value = ""; renderTiles(); return; }
    if (/^\d{6,14}$/.test(q)) { e.target.value = ""; S.posQ = ""; renderTiles(); return unknownBarcode(q); }
    const m = [...S.products.values()].filter(p => !p.hidden && prodMatch(p, q)).sort((a, b) => (S.pop[b.id] || 0) - (S.pop[a.id] || 0));
    if (m.length) { addLine(m[0].id); S.posQ = ""; e.target.value = ""; renderTiles(); }
    else toast("Không tìm thấy “" + q + "”", true);
  }
  if (e.key === "Enter" && e.target.id === "custQ") { e.preventDefault(); const f = $("#custDD button"); if (f) f.click(); }
});
document.addEventListener("click", e => {
  if (S.custOpen && !e.target.closest(".cust")) { S.custOpen = false; if (S.view === "pos") renderCart(); }
}, true);

/* ---- nhận đơn đặt vào hóa đơn ---- */
function loadOrderToPOS(o) {
  let t = curTab();
  if (t.lines.length) { S.tabs.push(newTab()); S.tabIdx = S.tabs.length - 1; t = curTab(); }
  t.orderId = o.id;
  t.custId = o.custId || "";
  t.note = `Đơn đặt ${o.code}${o.custId ? "" : " · " + o.custName}`;
  t.lines = o.lines.filter(l => S.products.has(l.pid)).map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio || 1, qty: l.qty, price: l.price }));
  saveTabs(); go("pos");
  if (!matchMedia("(min-width:900px)").matches) document.body.classList.add("cart-open");
}

/* ---- máy quét mã vạch: nhận chuỗi phím gõ rất nhanh kết thúc bằng Enter, kể cả khi chưa bấm vào ô tìm ---- */
const SCAN = { buf: "", last: 0, fast: 0 };
const scanHandlers = {}; // ngữ cảnh khác (phiếu nhập, kiểm kho) đăng ký tại đây
function unknownBarcode(code) {
  if (!can("stock")) return toast("Mã vạch " + code + " chưa có trong kho. Báo chủ tiệm thêm mặt hàng", true);
  openSheet(`<div class="sheet-h"><h2>Mã vạch chưa có trong kho</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <p style="margin:0">Mã <b class="num">${esc(code)}</b> chưa gắn với mặt hàng nào.</p>
    <div class="row"><button class="btn pri" data-act="newProduct" data-barcode="${esc(code)}">Tạo mặt hàng mới với mã này</button><button class="btn" data-act="attachBarcode" data-code="${esc(code)}">Gắn vào mặt hàng có sẵn</button></div>`);
}
ACT.attachBarcode = b => {
  const code = b.dataset.code;
  openSheet(`<div class="sheet-h"><h2>Gắn mã ${esc(code)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="search">${icon("search")}<input id="abQ" class="in" data-code="${esc(code)}" autocomplete="off" placeholder="Tìm mặt hàng cần gắn mã"></div><div class="dd" style="position:static" id="abDD">${pickList("product", "", "abPick")}</div>`, { focus: "#abQ" });
};
ON_INPUT.abQ = el => { $("#abDD").innerHTML = pickList("product", el.value, "abPick"); };
ACT.abPick = async b => {
  const code = $("#abQ").dataset.code, p = S.products.get(b.dataset.id); if (!p) return;
  const r = await run(() => api("POST", "/api/products", Object.assign({}, p, { barcode: code })), "Đã gắn mã vạch cho " + p.name);
  if (r) { closeSheet(); await refresh(true); if (S.view === "pos") addLine(r.id); }
};
function handleScan(code) {
  if (scanHandlers[S.view] && scanHandlers[S.view](code)) return;
  if (S.view !== "pos") return;
  const hit = byBarcode(code);
  if (hit) addLine(hit.p.id, hit.unit); else unknownBarcode(code);
}
document.addEventListener("keydown", e => {
  if (!S.me || e.ctrlKey || e.altKey || e.metaKey) return;
  const tg = e.target, inField = /INPUT|TEXTAREA|SELECT/.test(tg.tagName);
  const now = performance.now();
  if (e.key === "Enter") {
    const code = SCAN.buf; const fast = SCAN.fast;
    SCAN.buf = ""; SCAN.fast = 0;
    if (!inField && !sheetOpen() && code.length >= 4 && fast >= code.length - 2) { e.preventDefault(); handleScan(code); }
    return;
  }
  if (e.key.length !== 1) return;
  if (inField) { SCAN.buf = ""; return; }
  if (now - SCAN.last > 60) { SCAN.buf = ""; SCAN.fast = 0; } else SCAN.fast++;
  SCAN.last = now; SCAN.buf += e.key;
}, true);
