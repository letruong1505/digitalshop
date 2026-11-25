/* Digital Shop — đơn đặt (Zalo), hàng hóa, nhập hàng, kiểm kho */
"use strict";

/* ================= ĐƠN ĐẶT / ZALO ================= */
S.zText = store.get("zText", ""); S.drafts = [];
VIEWS.orders = function () {
  const open = [...S.orders.values()].filter(o => o.status === "new");
  const td = dkey(), tm = addDays(td, 1);
  const grp = { late: [], today: [], tomorrow: [], later: [] };
  open.forEach(o => { const d = o.due || td; (d < td ? grp.late : d === td ? grp.today : d === tm ? grp.tomorrow : grp.later).push(o); });
  Object.values(grp).forEach(a => a.sort((x, y) => (x.dueTime || "99").localeCompare(y.dueTime || "99")));
  const prep = {};
  [...grp.late, ...grp.today, ...grp.tomorrow].forEach(o => o.lines.forEach(l => {
    const k = l.pid + "|" + l.unit; prep[k] = prep[k] || { name: l.name, unit: l.unit, qty: 0, pid: l.pid, ratio: l.ratio }; prep[k].qty += l.qty;
  }));
  const prepList = Object.values(prep).sort((a, b) => a.name.localeCompare(b.name, "vi"));
  const card = o => {
    const total = o.lines.reduce((a, l) => a + l.qty * l.price, 0);
    return `<div class="ocard"><div class="row"><b class="grow">${esc(o.custName)}</b><span class="ps num">${esc(o.code)}</span>${o.dueTime ? `<span class="pill blue">${esc(o.dueTime)}</span>` : ""}</div>
      <div class="ol">${o.lines.map(l => `${fq(l.qty)} ${esc(l.unit)} ${esc(l.name)}`).join(" · ")}</div>
      ${o.note ? `<div class="ps">${esc(o.note)}</div>` : ""}
      <div class="row"><b class="num grow">${money(total)}</b><button class="btn sm danger" data-act="orderCancel" data-id="${esc(o.id)}" data-confirm="Bấm lần nữa để hủy">Hủy</button><button class="btn sm pri" data-act="orderSell" data-id="${esc(o.id)}">Bán / giao</button></div></div>`;
  };
  const sec = (t, a, cls) => a.length ? `<div class="stack"><h3 style="${cls || ""}">${t} (${a.length})</h3>${a.map(card).join("")}</div>` : "";
  $("#main").innerHTML = `<div class="page">
    <div class="head"><h1>Đơn đặt hàng</h1><button class="btn" data-act="draftNew">+ Đơn mới</button></div>
    <div class="orders-grid">
      <div class="stack">
        <div class="card stack">
          <div><h2>Dán tin nhắn từ nhóm Zalo</h2><p class="muted small" style="margin:4px 0 0">Chọn các tin nhắn đặt hàng trong Zalo → Sao chép → dán vào đây. Phần mềm tự tách tên khách, mặt hàng, số lượng và ngày lấy.</p></div>
          <textarea id="zText" class="in" rows="6" placeholder="Ví dụ:&#10;Anh Tuấn: lấy 2 thùng bia sg với 1 cây thăng long, chiều 5h qua lấy&#10;Chị Hoa xóm 3: mai cho chị 1 vỉ trứng, 10 gói hảo hảo">${esc(S.zText)}</textarea>
          <div class="row"><button class="btn pri" data-act="zParse">Tách đơn</button>${S.hasKey ? `<button class="btn" data-act="zParseAI">Tách bằng AI</button>` : ""}<span class="grow"></span>${S.zText ? `<button class="btn ghost sm" data-act="zClear">Xóa chữ</button>` : ""}</div>
          ${!S.hasKey && isOwner() ? `<div class="ps">Tin nhắn viết quá tắt? Có thể bật <b>tách bằng AI</b> trong Cài đặt (cần khóa API Claude, có tính phí).</div>` : ""}
        </div>
        <div id="drafts" class="stack"></div>
      </div>
      <div class="stack">
        ${prepList.length ? `<div class="card"><h3>Hàng cần chuẩn bị (quá hạn, hôm nay, ngày mai)</h3><div class="tbl" style="margin-top:8px;border:0"><table><tbody>${prepList.map(x => { const p = S.products.get(x.pid); const need = x.qty * (x.ratio || 1); const short = p && !p.isCombo && p.stock < need; return `<tr><td class="pn">${esc(x.name)}</td><td class="r"><b>${fq(x.qty)}</b> ${esc(x.unit)}</td><td class="r ps">${p ? (short ? `<span class="pill red">kho còn ${esc(stockText(p))}</span>` : "kho " + esc(stockText(p))) : ""}</td></tr>`; }).join("")}</tbody></table></div></div>` : ""}
        ${open.length ? sec("Quá hạn chưa lấy", grp.late, "color:var(--red)") + sec("Hôm nay", grp.today) + sec("Ngày mai", grp.tomorrow) + sec("Những ngày sau", grp.later)
          : `<div class="empty"><b>Chưa có đơn đặt nào đang chờ</b>Dán tin nhắn Zalo bên cạnh để tạo đơn, hoặc bấm + Đơn mới.</div>`}
      </div>
    </div></div>`;
  renderDrafts();
};
function renderDrafts() {
  const box = $("#drafts"); if (!box) return;
  box.innerHTML = S.drafts.map((d, di) => {
    const nc = norm(d.custName);
    const custs = [...S.customers.values()].map(c => ({ c, s: nc && norm(c.name).includes(nc.split(" ").slice(-1)[0]) ? 1 : 0 })).sort((a, b) => b.s - a.s || a.c.name.localeCompare(b.c.name, "vi")).slice(0, 40);
    const total = d.lines.reduce((a, l) => a + l.qty * l.price, 0);
    return `<div class="draft">
      <div class="row"><h3 class="grow">Đơn nháp ${di + 1}</h3><button class="x" data-act="draftRm" data-i="${di}" aria-label="Bỏ đơn nháp">×</button></div>
      <div class="grid2">
        <div class="field"><label for="dn-${di}">Tên khách</label><input id="dn-${di}" class="in" data-in="dr" data-i="${di}" data-f="custName" value="${esc(d.custName)}"></div>
        <div class="field"><label for="dc-${di}">Khách trong sổ</label><select id="dc-${di}" class="in" data-ch="drCust" data-i="${di}"><option value="">— không liên kết —</option>${custs.map(x => `<option value="${esc(x.c.id)}" ${x.c.id === d.custId ? "selected" : ""}>${esc(x.c.name)}${x.c.phone ? " · " + esc(x.c.phone) : ""}</option>`).join("")}</select></div>
        <div class="field"><label for="dd-${di}">Ngày lấy</label><input id="dd-${di}" type="date" class="in" data-in="dr" data-i="${di}" data-f="due" value="${esc(d.due || dkey())}"></div>
        <div class="field"><label for="dt-${di}">Giờ lấy</label><input id="dt-${di}" type="time" class="in" data-in="dr" data-i="${di}" data-f="dueTime" value="${esc(d.dueTime || "")}"></div>
      </div>
      ${d.raw ? `<div class="raw">${esc(d.raw)}</div>` : ""}
      <div class="stack" style="gap:8px">${d.lines.map((l, li) => {
        const p = S.products.get(l.pid);
        const cands = [...(l.cands || [])]; if (p && !cands.some(c => c.pid === p.id)) cands.unshift({ pid: p.id, name: p.name });
        const units = p ? unitOptions(p) : [];
        return `<div class="dline"><select class="in sm" data-ch="drProd" data-i="${di}" data-l="${li}" aria-label="Mặt hàng">${cands.map(c => `<option value="${esc(c.pid)}" ${c.pid === l.pid ? "selected" : ""}>${esc(c.name)}</option>`).join("")}<option value="__pick">Chọn mặt hàng khác…</option></select>
          <input class="in sm r num" data-ch="drQty" data-i="${di}" data-l="${li}" value="${fq(l.qty)}" inputmode="decimal" aria-label="Số lượng">
          <select class="in sm u" data-ch="drUnit" data-i="${di}" data-l="${li}" aria-label="Đơn vị">${units.map(u => `<option ${u.name === l.unit ? "selected" : ""}>${esc(u.name)}</option>`).join("")}</select>
          <button class="x" data-act="drLineRm" data-i="${di}" data-l="${li}" aria-label="Bỏ dòng">×</button>
          ${l.text ? `<div class="src">từ: “${esc(l.text)}”${!l.sure && (l.cands || []).length > 1 ? ' <span class="pill amber">kiểm tra lại</span>' : ""}</div>` : ""}</div>`;
      }).join("")}</div>
      ${(d.unknown || []).length ? `<div class="banner" style="font-size:13px">Không tìm thấy hàng cho: ${d.unknown.map(u => `<b>“${esc(u)}”</b>`).join(", ")}. Bấm + Thêm dòng để chọn tay.</div>` : ""}
      <div class="row"><button class="btn sm" data-act="drLineAdd" data-i="${di}">+ Thêm dòng</button><span class="grow"></span><b class="num">${money(total)}</b><button class="btn pri" data-act="draftSave" data-i="${di}">Lưu đơn</button></div>
    </div>`;
  }).join("") + (S.drafts.length > 1 ? `<div class="row"><span class="grow"></span><button class="btn pri" data-act="draftSaveAll">Lưu tất cả ${S.drafts.length} đơn</button></div>` : "");
}
ON_INPUT.zText = el => { S.zText = el.value; store.set("zText", el.value); };
ACT.zClear = () => { S.zText = ""; store.set("zText", ""); VIEWS.orders(); };
async function zParse(ai) {
  if (!S.zText.trim()) return toast("Dán tin nhắn vào ô trước", true);
  const r = await run(() => api("POST", "/api/zalo/parse", { text: S.zText, ai }));
  if (!r) return;
  if (!r.length) return toast("Không tìm thấy mặt hàng nào trong tin nhắn. Kiểm tra tên hàng hoặc thêm “tên gọi khác” cho mặt hàng", true);
  S.drafts = S.drafts.concat(r);
  toast(`Đã tách ${r.length} đơn. Kiểm tra lại rồi bấm Lưu`);
  renderDrafts();
}
ACT.zParse = () => zParse(false);
ACT.zParseAI = () => { toast("Đang gửi cho AI phân tích…"); zParse(true); };
ACT.draftNew = () => { S.drafts.push({ custName: "", custId: "", due: dkey(), dueTime: "", lines: [], raw: "" }); renderDrafts(); };
ACT.draftRm = b => { S.drafts.splice(+b.dataset.i, 1); renderDrafts(); };
ON_INPUT.dr = el => { S.drafts[+el.dataset.i][el.dataset.f] = el.value; };
ON_CHANGE.drCust = el => { const d = S.drafts[+el.dataset.i]; d.custId = el.value; const c = S.customers.get(el.value); if (c) d.custName = c.name; renderDrafts(); };
ON_CHANGE.drProd = el => {
  const d = S.drafts[+el.dataset.i], l = d.lines[+el.dataset.l];
  if (el.value === "__pick") return pickProduct(pid => { setDraftProd(l, pid); renderDrafts(); });
  setDraftProd(l, el.value); renderDrafts();
};
function setDraftProd(l, pid) { const p = S.products.get(pid); if (!p) return; l.pid = pid; l.name = p.name; l.unit = p.unit; l.ratio = 1; l.price = p.price; l.sure = true; }
ON_CHANGE.drQty = el => { S.drafts[+el.dataset.i].lines[+el.dataset.l].qty = parseQty(el.value); renderDrafts(); };
ON_CHANGE.drUnit = el => { const l = S.drafts[+el.dataset.i].lines[+el.dataset.l], p = S.products.get(l.pid); const u = unitOptions(p).find(x => x.name === el.value); if (u) { l.unit = u.name; l.ratio = u.ratio; l.price = u.price; } renderDrafts(); };
ACT.drLineRm = b => { S.drafts[+b.dataset.i].lines.splice(+b.dataset.l, 1); renderDrafts(); };
ACT.drLineAdd = b => { const d = S.drafts[+b.dataset.i]; pickProduct(pid => { const p = S.products.get(pid); d.lines.push({ pid, name: p.name, unit: p.unit, ratio: 1, qty: 1, price: p.price, sure: true }); renderDrafts(); }); };
async function saveDraft(i) {
  const d = S.drafts[i];
  if (!d.lines.length) { toast("Đơn chưa có mặt hàng", true); return false; }
  const r = await run(() => api("POST", "/api/orders", { custId: d.custId, custName: d.custName, phone: d.phone || "", due: d.due || dkey(), dueTime: d.dueTime || "", raw: d.raw || "", lines: d.lines.map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio, qty: l.qty, price: l.price })) }));
  if (r) { S.drafts.splice(i, 1); return true; }
  return false;
}
ACT.draftSave = async b => { if (await saveDraft(+b.dataset.i)) { toast("Đã lưu đơn đặt"); await refresh(true); } };
ACT.draftSaveAll = async () => { let n = 0; while (S.drafts.length) { if (!(await saveDraft(0))) break; n++; } if (n) toast(`Đã lưu ${n} đơn`); await refresh(true); };
ACT.orderSell = b => { const o = S.orders.get(b.dataset.id); if (o) loadOrderToPOS(o); };
ACT.orderCancel = async b => { if (await run(() => api("POST", `/api/orders/${b.dataset.id}/status`, { status: "cancel" }), "Đã hủy đơn")) refresh(true); };

/* chọn mặt hàng trong bảng trượt */
let pickCb = null;
function pickProduct(cb, title) {
  pickCb = cb;
  openSheet(`<div class="sheet-h"><h2>${esc(title || "Chọn mặt hàng")}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="search">${icon("search")}<input id="ppQ" class="in" autocomplete="off" placeholder="Gõ tên hoặc quét mã vạch"></div>
    <div class="dd" style="position:static;max-height:420px" id="ppDD">${pickList("product", "", "ppPick", 30)}</div>`, { focus: "#ppQ" });
}
ON_INPUT.ppQ = el => { $("#ppDD").innerHTML = pickList("product", el.value, "ppPick", 30); };
ACT.ppPick = b => { const cb = pickCb; pickCb = null; closeSheet(); if (cb) cb(b.dataset.id); };

/* ================= HÀNG HÓA ================= */
S.pQ = ""; S.pGroup = ""; S.pFilter = "all";
VIEWS.products = function () {
  const all = [...S.products.values()];
  const owner = seeCost();
  let val = 0, low = 0, out = 0, near = 0, exp = 0, combos = 0, hidden = 0;
  for (const p of all) {
    if (p.isCombo) { combos++; continue; }
    if (p.hidden) { hidden++; continue; }
    val += Math.max(0, p.stock) * (p.cost || 0);
    if (p.stock <= 0) out++; else if (p.stock <= (p.min || 0)) low++;
    const e = expInfo(p); if (e.expired > 0) exp++; else if (e.level) near++;
  }
  const F = S.pFilter;
  const stat = (k, t, v, cls) => `<button class="stat ${cls || ""}" data-act="pFilter" data-v="${k}" aria-pressed="${F === k}"><div class="k">${t}</div><div class="v">${v}</div></button>`;
  const gs = groups();
  let items = all.filter(p => (!S.pGroup || p.group === S.pGroup) && prodMatch(p, S.pQ) && (
    F === "all" ? !p.hidden : F === "low" ? !p.isCombo && !p.hidden && p.stock > 0 && p.stock <= (p.min || 0) : F === "out" ? !p.isCombo && !p.hidden && p.stock <= 0 :
    F === "near" ? !p.hidden && expInfo(p).level && !expInfo(p).expired : F === "exp" ? expInfo(p).expired > 0 : F === "combo" ? p.isCombo : F === "hidden" ? p.hidden : true));
  const rank = p => ({ red: 0, amber: 1, green: 2, mute: 3 }[prodStatus(p).k]);
  items.sort((a, b) => F === "near" || F === "exp" ? ((expInfo(a).near || "9") .localeCompare(expInfo(b).near || "9")) : rank(a) - rank(b) || a.name.localeCompare(b.name, "vi"));
  const show = items.slice(0, 400);
  $("#main").innerHTML = `<div class="page">
    <div class="head"><h1>Hàng hóa</h1>${impBtn("products")}
      <a class="btn sm" href="/api/export/products" download>Xuất Excel</a>
      ${exp && isOwner() ? `<button class="btn sm danger" data-act="dispose" data-confirm="Bấm lần nữa: hủy toàn bộ lô hết hạn">Xuất hủy hàng hết hạn (${exp})</button>` : ""}
      ${isOwner() ? `<button class="btn sm" data-act="newCombo">+ Combo</button><button class="btn sm pri" data-act="newProduct">+ Mặt hàng</button>` : ""}</div>
    <div class="stats">${stat("all", "Mặt hàng đang bán", fmt(all.length - combos - hidden))}
      ${owner ? `<div class="stat"><div class="k">Giá trị tồn (giá vốn)</div><div class="v">${money(val)}</div></div>` : ""}
      ${stat("low", "Sắp hết", low, "warn")}${stat("out", "Hết hàng", out, "bad")}${stat("near", `Cận date (≤${S.settings.expWarn || 30} ngày)`, near, "warn")}${stat("exp", "Có lô hết hạn", exp, "bad")}${stat("combo", "Combo", combos)}${hidden ? stat("hidden", "Ngừng bán", hidden) : ""}</div>
    <div class="row" style="margin:12px 0 8px"><div class="search grow" style="min-width:220px">${icon("search")}<input id="pQ" class="in" type="search" autocomplete="off" placeholder="Tìm tên, mã hàng, mã vạch, vị trí" value="${esc(S.pQ)}"></div>
      <select id="pGroup" class="in" style="width:auto"><option value="">Tất cả nhóm</option>${gs.map(g => `<option ${g === S.pGroup ? "selected" : ""}>${esc(g)}</option>`).join("")}</select></div>
    ${!all.length ? `<div class="empty"><b>Kho đang trống</b>Lấy danh mục hàng từ file Excel xuất ra từ KiotViet / MISA, hoặc thêm từng mặt hàng.<div class="row" style="justify-content:center;margin-top:12px">${impBtn("products", "pri")}</div></div>` : !items.length ? `<div class="empty">Không có mặt hàng nào khớp bộ lọc.</div>` :
    `<div class="tbl"><table><thead><tr><th>Mặt hàng</th><th>Nhóm</th><th class="r">Tồn kho</th>${owner ? '<th class="r">Giá vốn</th>' : ""}<th class="r">Giá bán</th><th>Hạn dùng gần nhất</th><th>Tình trạng</th></tr></thead><tbody>${show.map(p => {
      const st = prodStatus(p), e = expInfo(p);
      return `<tr class="click" data-act="${isOwner() ? "editProduct" : "viewProduct"}" data-id="${esc(p.id)}"><td><div class="pn">${p.isCombo ? '<span class="pill blue">combo</span> ' : ""}${esc(p.name)}</div><div class="ps">${esc(p.code)}${p.barcode ? " · " + esc(p.barcode) : ""}${p.location ? " · " + esc(p.location) : ""}${(p.units || []).length ? " · " + p.units.map(u => `1 ${esc(u.name)} = ${fq(u.ratio)} ${esc(p.unit)}`).join(", ") : ""}</div></td>
        <td class="ps">${esc(p.group || "")}</td><td class="r"><b>${esc(stockText(p))}</b></td>${owner ? `<td class="r">${fmt(p.cost)}</td>` : ""}<td class="r">${fmt(p.price)}</td>
        <td class="num">${e.near ? `${vnDate(e.near)} <span class="ps">(${e.left} ngày${e.nearQty ? ", " + fq(e.nearQty) + " " + esc(p.unit) : ""})</span>` : '<span class="ps">—</span>'}${e.expired ? `<div class="pill red">${fq(e.expired)} ${esc(p.unit)} đã hết hạn</div>` : ""}</td>
        <td><span class="pill ${st.k}">${st.t}</span></td></tr>`;
    }).join("")}</tbody></table></div>${items.length > show.length ? `<div class="muted small" style="margin-top:8px">Đang hiện 400/${items.length} mặt hàng, gõ để tìm.</div>` : ""}`}
  </div>`;
};
ON_INPUT.pQ = el => { S.pQ = el.value; VIEWS.products(); };
ON_CHANGE.pGroup = el => { S.pGroup = el.value; VIEWS.products(); };
ACT.pFilter = b => { S.pFilter = S.pFilter === b.dataset.v ? "all" : b.dataset.v; VIEWS.products(); };
ACT.dispose = async () => { const r = await run(() => api("POST", "/api/products/dispose", { ids: [] }), c => `Đã xuất hủy ${c.lines.length} mặt hàng (${c.code})`); if (r) refresh(true); };
ACT.viewProduct = b => {
  const p = S.products.get(b.dataset.id); if (!p) return;
  const e = expInfo(p);
  openSheet(`<div class="sheet-h"><h2>${esc(p.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="grid2"><div><div class="lbl">Giá bán</div><span class="tag">${fmt(p.price)}</span> / ${esc(p.unit)}</div><div><div class="lbl">Tồn kho</div><b>${esc(stockText(p))}</b></div>
    <div><div class="lbl">Vị trí</div>${esc(p.location || "—")}</div><div><div class="lbl">Hạn gần nhất</div>${e.near ? vnDate(e.near) : "—"}</div></div>
    ${(p.units || []).map(u => `<div>1 ${esc(u.name)} = ${fq(u.ratio)} ${esc(p.unit)} · giá ${money(u.price)}</div>`).join("")}`);
};

/* ---- form mặt hàng ---- */
S.pf = null;
function openProductForm(p) {
  S.pf = JSON.parse(JSON.stringify(p));
  S.pf.units = S.pf.units || []; S.pf.items = S.pf.items || []; S.pf.lots = S.pf.lots || [];
  renderProductForm();
}
ACT.newProduct = b => { closeSheet(); openProductForm({ id: "", name: b?.dataset?.name || "", barcode: b?.dataset?.barcode || "", unit: "gói", group: S.pGroup || "", price: "", cost: "", min: 5, units: [], items: [], lots: [], isCombo: false, _new: true, setStock: "", stockExp: "" }); };
ACT.newCombo = () => openProductForm({ id: "", name: "", unit: "combo", group: "Combo", price: "", isCombo: true, units: [], items: [], lots: [], _new: true });
ACT.editProduct = b => { const p = S.products.get(b.dataset.id); if (p) openProductForm(p); };
function renderProductForm() {
  const f = S.pf, isNew = !f.id;
  const gl = [...new Set([...groups(), "Mì - Bún khô", "Đồ uống", "Bia - Rượu", "Bánh kẹo", "Sữa", "Gia vị", "Gạo - Dầu ăn", "Hóa phẩm", "Thuốc lá", "Đồ dùng", "Đồ tươi"])];
  const ul = ["gói", "chai", "lon", "hộp", "kg", "cái", "bịch", "bao", "quả", "cuộn", "thùng", "lốc", "vỉ", "cây", "két", "túi", "lọ", "hũ"];
  const comboCost = f.items.reduce((a, it) => { const p = S.products.get(it.pid); return a + (p ? (p.cost || 0) * it.qty : 0); }, 0);
  const comboRetail = f.items.reduce((a, it) => { const p = S.products.get(it.pid); return a + (p ? p.price * it.qty : 0); }, 0);
  const td = dkey();
  openSheet(`<div class="sheet-h"><h2>${f.isCombo ? (isNew ? "Combo mới" : "Sửa combo") : isNew ? "Mặt hàng mới" : "Sửa mặt hàng"}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
  <form id="pForm" class="stack" autocomplete="off">
    <div class="grid2">
      <div class="field full"><label for="pf-name">Tên ${f.isCombo ? "combo" : "hàng"} *</label><input id="pf-name" class="in" data-in="pf" data-f="name" required value="${esc(f.name)}" placeholder="${f.isCombo ? "vd Combo bữa sáng" : "vd Mì Hảo Hảo tôm chua cay"}"></div>
      <div class="field"><label for="pf-code">Mã hàng</label><input id="pf-code" class="in" data-in="pf" data-f="code" value="${esc(f.code || "")}" placeholder="để trống: tự tạo"></div>
      ${f.isCombo ? "" : `<div class="field"><label for="pf-bar">Mã vạch</label><input id="pf-bar" class="in num" data-in="pf" data-f="barcode" value="${esc(f.barcode || "")}" placeholder="quét hoặc gõ"></div>`}
      <div class="field"><label for="pf-group">Nhóm hàng</label><input id="pf-group" class="in" list="dlGroup" data-in="pf" data-f="group" value="${esc(f.group || "")}"><datalist id="dlGroup">${gl.map(g => `<option value="${esc(g)}">`).join("")}</datalist></div>
      ${f.isCombo ? "" : `<div class="field"><label for="pf-unit">Đơn vị tính *</label><input id="pf-unit" class="in" list="dlUnit" data-in="pf" data-f="unit" required value="${esc(f.unit || "")}"><datalist id="dlUnit">${ul.map(u => `<option value="${u}">`).join("")}</datalist></div>`}
      <div class="field"><label for="pf-price">Giá bán *</label><input id="pf-price" class="in num" data-in="pf" data-f="price" inputmode="numeric" required value="${f.price === "" ? "" : fmt(f.price)}" placeholder="vd 4.500 hoặc 4,5k"></div>
      ${f.isCombo ? "" : `<div class="field"><label for="pf-cost">Giá vốn${isNew ? "" : " (bình quân)"}</label><input id="pf-cost" class="in num" data-in="pf" data-f="cost" inputmode="numeric" value="${f.cost === "" || f.cost == null ? "" : fmt(f.cost)}"></div>
      <div class="field"><label for="pf-stock">${isNew ? "Tồn đầu kỳ" : "Tồn thực tế (sửa = kiểm kho)"}</label><input id="pf-stock" class="in num" data-in="pf" data-f="setStock" inputmode="decimal" value="${isNew ? esc(f.setStock || "") : fq(f.stock)}"></div>
      ${isNew ? `<div class="field"><label for="pf-exp">Hạn sử dụng của số tồn này</label><input id="pf-exp" type="date" class="in" data-in="pf" data-f="stockExp" value="${esc(f.stockExp || "")}"></div>` : ""}
      <div class="field"><label for="pf-min">Báo sắp hết khi còn ≤</label><input id="pf-min" class="in num" data-in="pf" data-f="min" inputmode="decimal" value="${fq(f.min || 0)}"></div>
      <div class="field"><label for="pf-loc">Vị trí trong tiệm</label><input id="pf-loc" class="in" data-in="pf" data-f="location" value="${esc(f.location || "")}" placeholder="vd Kệ 2 tầng trên, Tủ mát"></div>`}
      <div class="field full"><label for="pf-alias">Tên gọi khác (cho tách đơn Zalo)</label><input id="pf-alias" class="in" data-in="pf" data-f="alias" value="${esc(f.alias || "")}" placeholder="vd bia sg, sài gòn xanh, ken"><span class="hint">Các cách khách hay gọi, cách nhau bằng dấu phẩy.</span></div>
    </div>
    ${f.isCombo ? `<fieldset class="card stack" style="margin:0"><legend class="lbl" style="padding:0 6px">Mặt hàng trong combo</legend>
      ${f.items.map((it, i) => { const p = S.products.get(it.pid); return `<div class="row"><span class="grow pn">${esc(p ? p.name : "(đã xóa)")}</span><input class="in sm r num" style="width:80px" data-in="pfi" data-i="${i}" value="${fq(it.qty)}" inputmode="decimal" aria-label="Số lượng"><span class="ps" style="width:44px">${esc(p ? p.unit : "")}</span><button type="button" class="x" data-act="pfItemRm" data-i="${i}" aria-label="Bỏ">×</button></div>`; }).join("") || '<div class="ps">Chưa có mặt hàng nào.</div>'}
      <div class="row"><button type="button" class="btn sm" data-act="pfItemAdd">+ Thêm mặt hàng</button><span class="grow"></span><span class="ps num">Giá lẻ cộng lại ${money(comboRetail)}${seeCost() ? ` · vốn ${money(comboCost)}` : ""}</span></div>
      <div class="ps">Bán combo sẽ tự trừ kho từng mặt hàng bên trong.</div></fieldset>` :
    `<fieldset class="card stack" style="margin:0"><legend class="lbl" style="padding:0 6px">Đơn vị lớn (thùng, lốc, cây…)</legend>
      ${f.units.map((u, i) => `<div class="grid3" style="grid-template-columns:1fr 90px 1fr 1fr 30px;align-items:end">
        <div class="field"><label>Tên</label><input class="in sm" data-in="pfu" data-i="${i}" data-f="name" value="${esc(u.name)}" list="dlUnit"></div>
        <div class="field"><label>= ? ${esc(f.unit || "")}</label><input class="in sm num" data-in="pfu" data-i="${i}" data-f="ratio" value="${u.ratio ? fq(u.ratio) : ""}" inputmode="decimal"></div>
        <div class="field"><label>Giá bán</label><input class="in sm num" data-in="pfu" data-i="${i}" data-f="price" value="${u.price ? fmt(u.price) : ""}" inputmode="numeric"></div>
        <div class="field"><label>Mã vạch</label><input class="in sm num" data-in="pfu" data-i="${i}" data-f="barcode" value="${esc(u.barcode || "")}"></div>
        <button type="button" class="x" data-act="pfUnitRm" data-i="${i}" aria-label="Bỏ">×</button></div>`).join("") || '<div class="ps">Bán theo thùng/lốc với giá riêng, quét mã vạch thùng là ra giá thùng.</div>'}
      <div><button type="button" class="btn sm" data-act="pfUnitAdd">+ Thêm đơn vị</button></div></fieldset>
    ${!isNew && f.lots.length ? `<fieldset class="card stack" style="margin:0"><legend class="lbl" style="padding:0 6px">Lô hàng theo hạn sử dụng</legend>
      ${f.lots.slice().sort((a, b) => a.exp.localeCompare(b.exp)).map(l => { const i = f.lots.indexOf(l); const left = daysTo(l.exp); return `<div class="row"><input type="date" class="in sm" style="width:auto" data-in="pfl" data-i="${i}" value="${esc(l.exp)}" aria-label="Hạn sử dụng"><span class="num"><b>${fq(l.qty)}</b> ${esc(f.unit)}</span><span class="pill ${l.exp < td ? "red" : left <= (S.settings.expDanger || 7) ? "red" : left <= (S.settings.expWarn || 30) ? "amber" : "green"}">${l.exp < td ? "đã hết hạn" : "còn " + left + " ngày"}</span><span class="ps grow">nhập ${vnDate(l.in)}</span><button type="button" class="x" data-act="pfLotRm" data-i="${i}" title="Bỏ theo dõi hạn lô này">×</button></div>`; }).join("")}
      <div class="ps">Phần tồn không thuộc lô nào: ${fq(Math.max(0, f.stock - f.lots.reduce((a, l) => a + l.qty, 0)))} ${esc(f.unit)}. Bán hàng tự trừ lô hạn gần nhất trước.</div></fieldset>` : ""}`}
    <label class="check"><input type="checkbox" id="pf-hidden" data-in="pfh" ${f.hidden ? "checked" : ""}> Ngừng bán (ẩn khỏi màn hình bán hàng)</label>
    <div class="row">${!isNew ? `<button type="button" class="btn danger" data-act="pfDelete" data-confirm="Bấm lần nữa để xóa">Xóa</button>` : ""}<span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri" type="submit">${isNew ? "Thêm vào kho" : "Lưu thay đổi"}</button></div>
  </form>`, { wide: true, focus: isNew ? "#pf-name" : null });
  $("#pForm").addEventListener("submit", e => { e.preventDefault(); saveProductForm(); });
}
ON_INPUT.pf = el => { S.pf[el.dataset.f] = el.value; };
ON_INPUT.pfu = el => { S.pf.units[+el.dataset.i][el.dataset.f] = el.value; };
ON_INPUT.pfi = el => { S.pf.items[+el.dataset.i].qty = parseQty(el.value); };
ON_INPUT.pfl = el => { S.pf.lots[+el.dataset.i].exp = el.value; };
ON_CHANGE.pfh = el => { S.pf.hidden = el.checked; };
ACT.pfUnitAdd = () => { S.pf.units.push({ name: "thùng", ratio: "", price: "", barcode: "" }); renderProductForm(); };
ACT.pfUnitRm = b => { S.pf.units.splice(+b.dataset.i, 1); renderProductForm(); };
ACT.pfLotRm = b => { S.pf.lots.splice(+b.dataset.i, 1); renderProductForm(); };
ACT.pfItemRm = b => { S.pf.items.splice(+b.dataset.i, 1); renderProductForm(); };
ACT.pfItemAdd = () => { const f = S.pf; pickProduct(pid => { const p = S.products.get(pid); if (p && !p.isCombo) f.items.push({ pid, qty: 1 }); else toast("Không thể lồng combo vào combo", true); S.pf = f; renderProductForm(); }, "Thêm vào combo"); };
async function saveProductForm() {
  const f = S.pf;
  const body = {
    id: f.id || "", code: f.code || "", name: f.name, group: f.group || "", unit: f.isCombo ? (f.unit || "combo") : f.unit, barcode: f.barcode || "", location: f.location || "", alias: f.alias || "",
    price: parseMoney(f.price), cost: f.isCombo ? 0 : parseMoney(f.cost), min: parseQty(f.min), hidden: !!f.hidden, isCombo: !!f.isCombo,
    units: (f.units || []).map(u => ({ name: String(u.name || "").trim(), ratio: parseQty(u.ratio), price: parseMoney(u.price), barcode: String(u.barcode || "").trim() })),
    items: (f.items || []).map(it => ({ pid: it.pid, qty: parseQty(it.qty) })),
  };
  if (!f.id) { if (String(f.setStock || "").trim() !== "") body.setStock = parseQty(f.setStock); body.stockExp = f.stockExp || ""; }
  else if (!f.isCombo) {
    body.lots = f.lots.map(l => ({ id: l.id, exp: l.exp, qty: l.qty }));
    const want = parseQty(f.setStock ?? f.stock);
    if (f.setStock !== undefined && String(f.setStock).trim() !== "" && want !== f.stock) body.setStock = want;
  }
  if (!body.price && !confirmZero()) return;
  const r = await run(() => api("POST", "/api/products", body), f.id ? "Đã lưu " + body.name : "Đã thêm " + body.name);
  if (r) { closeSheet(); S.pf = null; await refresh(true); }
}
function confirmZero() { toast("Nhập giá bán trước khi lưu", true); return false; }
ACT.pfDelete = async () => { const r = await run(() => api("DELETE", "/api/products/" + S.pf.id), "Đã xóa"); if (r) { closeSheet(); refresh(true); } };

/* ================= NHẬP HÀNG ================= */
S.rcPeriod = "30d"; S.rc = null;
VIEWS.receipts = async function () {
  const [a, b] = periodRange(S.rcPeriod);
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Nhập hàng</h1>${isOwner() ? `<button class="btn" data-act="go" data-v="inin">${icon("inin")} Từ hóa đơn điện tử${S.inNew ? ` (${S.inNew})` : ""}</button>` : ""}<button class="btn pri" data-act="rcNew">+ Phiếu nhập</button></div>
    <div class="chips" style="margin-bottom:12px">${periodChips(S.rcPeriod, "rcPeriod")}</div><div id="rcList"><div class="muted">Đang tải…</div></div></div>`;
  const list = await run(() => api("GET", `/api/receipts?from=${a}&to=${b}`)); if (!list || S.view !== "receipts") return;
  const tot = list.filter(x => !x.void).reduce((s, x) => s + x.total, 0), debt = list.filter(x => !x.void).reduce((s, x) => s + x.debt, 0);
  $("#rcList").innerHTML = list.length ? `<div class="stats" style="margin-bottom:12px"><div class="stat"><div class="k">Số phiếu</div><div class="v">${list.filter(x => !x.void).length}</div></div><div class="stat"><div class="k">Tổng tiền nhập</div><div class="v">${money(tot)}</div></div><div class="stat bad"><div class="k">Còn nợ nhà cung cấp</div><div class="v">${money(debt)}</div></div></div>
    <div class="tbl"><table><thead><tr><th>Mã phiếu</th><th>Thời gian</th><th>Nhà cung cấp</th><th>Mặt hàng</th><th class="r">Tổng tiền</th><th class="r">Đã trả</th><th class="r">Ghi nợ</th><th></th></tr></thead><tbody>${list.map(r => `<tr class="${r.void ? "void" : ""}"><td class="num">${esc(r.code)}</td><td class="num">${dt(r.time)}</td><td>${esc(r.suppName || "—")}</td><td class="ps">${esc(r.lines.map(l => `${l.name} ×${fq(l.qty)} ${l.unit}`).join(", ")).slice(0, 140)}</td><td class="r"><b>${fmt(r.total)}</b></td><td class="r">${fmt(r.paid)}</td><td class="r">${r.debt ? fmt(r.debt) : ""}</td><td class="r">${!r.void && isOwner() ? `<button class="btn sm danger" data-act="rcVoid" data-id="${esc(r.id)}" data-confirm="Hủy phiếu?">Hủy</button>` : r.void ? '<span class="pill mute">Đã hủy</span>' : ""}</td></tr>`).join("")}</tbody></table></div>`
    : `<div class="empty"><b>Chưa có phiếu nhập trong khoảng này</b>Tạo phiếu nhập khi lấy hàng từ đại lý: tồn kho, giá vốn và hạn sử dụng được cập nhật tự động.</div>`;
};
ACT.rcPeriod = b => { S.rcPeriod = b.dataset.v; VIEWS.receipts(); };
ACT.rcVoid = async b => { if (await run(() => api("POST", `/api/receipts/${b.dataset.id}/void`), "Đã hủy phiếu nhập")) { await refresh(true); } };
ACT.rcNew = () => { S.rc = S.rc && S.rc.lines.length ? S.rc : { cid: uid(), suppId: "", lines: [], disc: "", paid: "", paidTouched: false, method: "cash", note: "" }; renderReceiptForm(); };
scanHandlers.receipts = code => { if (!S.rc || !sheetOpen()) return false; const h = byBarcode(code); if (h) rcAdd(h.p.id, h.unit); else toast("Mã " + code + " chưa có trong kho", true); return true; };
function rcAdd(pid, unit) {
  const p = S.products.get(pid); if (!p || p.isCombo) return toast("Không nhập kho cho combo, hãy nhập từng mặt hàng", true);
  const u = unit || { name: p.unit, ratio: 1 };
  const ex = S.rc.lines.find(l => l.pid === pid && l.unit === u.name);
  if (ex) ex.qty = r3(ex.qty + 1);
  else S.rc.lines.push({ pid, unit: u.name, ratio: u.ratio, qty: 1, price: Math.round((p.cost || 0) * u.ratio), exp: "" });
  renderReceiptForm(true);
}
function rcTotals() { const sub = Math.round(S.rc.lines.reduce((a, l) => a + l.qty * l.price, 0)); const disc = Math.min(parseMoney(S.rc.disc), sub); const total = sub - disc; const paid = S.rc.paidTouched ? Math.min(parseMoney(S.rc.paid), total) : total; return { sub, disc, total, paid, debt: total - paid }; }
function renderReceiptForm(keepFocus) {
  const R = S.rc, T = rcTotals();
  const sups = [...S.suppliers.values()].sort((a, b) => a.name.localeCompare(b.name, "vi"));
  const html = `<div class="sheet-h"><h2>Phiếu nhập hàng</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="grid2"><div class="field"><label for="rcSup">Nhà cung cấp</label><div class="row" style="flex-wrap:nowrap"><select id="rcSup" class="in"><option value="">— không chọn —</option>${sups.map(s => `<option value="${esc(s.id)}" ${s.id === R.suppId ? "selected" : ""}>${esc(s.name)}${s.debt ? " (đang nợ " + fmt(s.debt) + ")" : ""}</option>`).join("")}</select><button class="btn" data-act="rcSupNew" title="Thêm nhà cung cấp">+</button></div></div>
      <div class="field"><label for="rcQ">Thêm mặt hàng (gõ tên hoặc quét mã)</label><div class="search" style="position:relative">${icon("search")}<input id="rcQ" class="in" autocomplete="off" placeholder="Tìm hàng cần nhập"><div class="dd" id="rcDD" hidden style="left:0;right:0;top:100%"></div></div></div></div>
    <div id="rcLines">${rcLinesHtml()}</div>
    <div class="grid3" style="align-items:end">
      <div class="field"><label for="rcDisc">Chiết khấu phiếu</label><input id="rcDisc" class="in r num" inputmode="numeric" value="${esc(R.disc)}" placeholder="0"></div>
      <div class="field"><label for="rcPaid">Đã trả nhà cung cấp</label><input id="rcPaid" class="in r num" inputmode="numeric" value="${R.paidTouched ? esc(R.paid) : fmt(T.total)}"></div>
      <div class="seg"><button data-act="rcMethod" data-v="cash" aria-pressed="${R.method === "cash"}">Tiền mặt</button><button data-act="rcMethod" data-v="bank" aria-pressed="${R.method === "bank"}">Chuyển khoản</button></div></div>
    <div id="rcSum">${rcSumHtml(T)}</div>
    <div class="field"><input id="rcNote" class="in" placeholder="Ghi chú (số hóa đơn của đại lý…)" value="${esc(R.note)}"></div>
    <div class="row"><button class="btn ghost" data-act="rcDiscard" data-confirm="Bỏ phiếu nháp?">Bỏ phiếu</button><span class="grow"></span><button class="btn" data-act="close">Để sau</button><button class="btn pri" data-act="rcSave" ${R.lines.length ? "" : "disabled"}>Lưu phiếu nhập</button></div>`;
  if (keepFocus && sheetOpen() && $("#rcLines")) { $("#rcLines").innerHTML = rcLinesHtml(); $("#rcSum").innerHTML = rcSumHtml(T); const s = $('[data-act="rcSave"]'); if (s) s.disabled = !R.lines.length; const p = $("#rcPaid"); if (p && !R.paidTouched) p.value = fmt(T.total); return; }
  openSheet(html, { wide: true, focus: "#rcQ" });
}
function rcLinesHtml() {
  const R = S.rc;
  if (!R.lines.length) return `<div class="empty">Chưa có mặt hàng. Gõ tên hoặc quét mã vạch ở ô phía trên.</div>`;
  return `<div class="tbl"><table><thead><tr><th>Mặt hàng</th><th>Đơn vị</th><th class="r">Số lượng</th><th class="r">Giá nhập</th><th>Hạn sử dụng</th><th class="r">Thành tiền</th><th></th></tr></thead><tbody>${R.lines.map((l, i) => {
    const p = S.products.get(l.pid) || { name: "?", unit: l.unit, units: [] };
    return `<tr><td><div class="pn">${esc(p.name)}</div><div class="ps">tồn ${esc(stockText(p))}${seeCost() ? " · vốn " + fmt(p.cost) : ""}</div></td>
      <td><select class="in sm" data-ch="rcUnit" data-i="${i}">${unitOptions(p).map(u => `<option ${u.name === l.unit ? "selected" : ""}>${esc(u.name)}</option>`).join("")}</select></td>
      <td class="r"><input class="in sm r num" style="width:80px" data-in="rcl" data-f="qty" data-i="${i}" value="${fq(l.qty)}" inputmode="decimal"></td>
      <td class="r"><input class="in sm r num" style="width:110px" data-in="rcl" data-f="price" data-i="${i}" value="${fmt(l.price)}" inputmode="numeric"></td>
      <td><input type="date" class="in sm" style="width:auto" data-in="rcl" data-f="exp" data-i="${i}" value="${esc(l.exp)}"></td>
      <td class="r"><b id="rct-${i}">${fmt(l.qty * l.price)}</b></td><td><button class="x" data-act="rcRm" data-i="${i}" aria-label="Bỏ">×</button></td></tr>`;
  }).join("")}</tbody></table></div>`;
}
function rcSumHtml(T) { return `<div class="sumrow"><span class="muted">Tiền hàng ${money(T.sub)}${T.disc ? " − chiết khấu " + money(T.disc) : ""}</span><span class="due" style="font-size:24px">${money(T.total)}</span></div>${T.debt > 0 ? `<div class="banner red">Còn nợ nhà cung cấp <b>${money(T.debt)}</b>${S.rc.suppId ? "" : " — chọn nhà cung cấp để ghi nợ"}</div>` : ""}`; }
ON_INPUT.rcQ = el => { const d = $("#rcDD"); d.hidden = !el.value.trim(); d.innerHTML = pickList("product", el.value, "rcPick", 8); };
ACT.rcPick = b => { rcAdd(b.dataset.id); const q = $("#rcQ"); q.value = ""; $("#rcDD").hidden = true; q.focus(); };
ON_CHANGE.rcSup = el => { S.rc.suppId = el.value; renderReceiptForm(true); };
ON_INPUT.rcl = el => {
  const l = S.rc.lines[+el.dataset.i], f = el.dataset.f;
  l[f] = f === "qty" ? parseQty(el.value) : f === "price" ? parseMoney(el.value) : el.value;
  const t = $("#rct-" + el.dataset.i); if (t) t.textContent = fmt(l.qty * l.price);
  const T = rcTotals(); $("#rcSum").innerHTML = rcSumHtml(T); const p = $("#rcPaid"); if (p && !S.rc.paidTouched) p.value = fmt(T.total);
};
ON_CHANGE.rcUnit = el => { const l = S.rc.lines[+el.dataset.i], p = S.products.get(l.pid); const u = unitOptions(p).find(x => x.name === el.value); if (u) { l.price = Math.round(l.price / (l.ratio || 1) * u.ratio); l.unit = u.name; l.ratio = u.ratio; } renderReceiptForm(true); };
ACT.rcRm = b => { S.rc.lines.splice(+b.dataset.i, 1); renderReceiptForm(true); };
ON_INPUT.rcDisc = el => { S.rc.disc = el.value; $("#rcSum").innerHTML = rcSumHtml(rcTotals()); const p = $("#rcPaid"); if (p && !S.rc.paidTouched) p.value = fmt(rcTotals().total); };
ON_INPUT.rcPaid = el => { S.rc.paid = el.value; S.rc.paidTouched = true; $("#rcSum").innerHTML = rcSumHtml(rcTotals()); };
ON_INPUT.rcNote = el => { S.rc.note = el.value; };
ACT.rcMethod = b => { S.rc.method = b.dataset.v; $$('[data-act="rcMethod"]').forEach(x => x.setAttribute("aria-pressed", x.dataset.v === b.dataset.v)); };
ACT.rcDiscard = () => { S.rc = null; closeSheet(); };
ACT.rcSupNew = () => {
  const name = $("#rcQ") ? "" : "";
  const R = S.rc;
  openSheet(`<div class="sheet-h"><h2>Nhà cung cấp mới</h2><button class="x" data-act="rcBack" aria-label="Quay lại">×</button></div>
    <div class="grid2"><div class="field"><label for="ns-name">Tên</label><input id="ns-name" class="in" value="${esc(name)}"></div><div class="field"><label for="ns-phone">Điện thoại</label><input id="ns-phone" class="in num"></div></div>
    <div class="row"><span class="grow"></span><button class="btn" data-act="rcBack">Quay lại</button><button class="btn pri" data-act="rcSupSave">Thêm</button></div>`, { focus: "#ns-name" });
  S.rc = R;
};
ACT.rcBack = () => renderReceiptForm();
ACT.rcSupSave = async () => {
  const r = await run(() => api("POST", "/api/suppliers", { name: $("#ns-name").value, phone: $("#ns-phone").value }), "Đã thêm nhà cung cấp");
  if (r) { S.suppliers.set(r.id, r); S.rc.suppId = r.id; renderReceiptForm(); }
};
ACT.rcSave = async () => {
  const R = S.rc, T = rcTotals();
  if (T.debt > 0 && !R.suppId) return toast("Chọn nhà cung cấp để ghi nợ phần chưa trả", true);
  const r = await run(() => api("POST", "/api/receipts", { cid: R.cid, suppId: R.suppId, lines: R.lines.map(l => ({ pid: l.pid, unit: l.unit, ratio: l.ratio, qty: l.qty, price: l.price, exp: l.exp })), disc: T.disc, paid: T.paid, method: R.method, note: R.note }), x => `Đã nhập kho ${x.code} · ${money(x.total)}`);
  if (r) { S.rc = null; closeSheet(); await refresh(true); }
};

/* ================= KIỂM KHO ================= */
S.ccPeriod = "30d"; S.cc = null;
VIEWS.counts = async function () {
  const [a, b] = periodRange(S.ccPeriod);
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Kiểm kho</h1><button class="btn pri" data-act="ccNew">+ Phiếu kiểm kho</button></div>
    <div class="chips" style="margin-bottom:12px">${periodChips(S.ccPeriod, "ccPeriod")}</div><div id="ccList"><div class="muted">Đang tải…</div></div></div>`;
  const list = await run(() => api("GET", `/api/counts?from=${a}&to=${b}`)); if (!list || S.view !== "counts") return;
  $("#ccList").innerHTML = list.length ? `<div class="tbl"><table><thead><tr><th>Mã phiếu</th><th>Thời gian</th><th>Loại</th><th>Chênh lệch</th>${seeCost() ? '<th class="r">Giá trị lệch</th>' : ""}<th>Người làm</th></tr></thead><tbody>${list.map(c => {
    const val = c.lines.reduce((s, l) => s + (l.after - l.before) * (l.cost || 0), 0);
    const diff = c.lines.filter(l => l.after !== l.before);
    return `<tr><td class="num">${esc(c.code)}</td><td class="num">${dt(c.time)}</td><td>${c.kind === "dispose" ? '<span class="pill red">Xuất hủy</span>' : '<span class="pill mute">Kiểm kho</span>'} <span class="ps">${esc(c.note || "")}</span></td>
      <td class="ps">${diff.length ? esc(diff.slice(0, 6).map(l => `${l.name} ${l.after - l.before > 0 ? "+" : ""}${fq(l.after - l.before)}`).join(", ")) + (diff.length > 6 ? "…" : "") : "khớp"}</td>${seeCost() ? `<td class="r" style="color:${val < 0 ? "var(--red)" : "inherit"}">${fmt(val)}</td>` : ""}<td class="ps">${esc(c.staff)}</td></tr>`;
  }).join("")}</tbody></table></div>` : `<div class="empty"><b>Chưa có phiếu kiểm kho</b>Đếm hàng thực tế trên kệ và nhập vào phiếu, phần mềm tự cân lại tồn kho.</div>`;
};
ACT.ccPeriod = b => { S.ccPeriod = b.dataset.v; VIEWS.counts(); };
ACT.ccNew = () => { S.cc = S.cc || { lines: [], note: "" }; renderCountForm(); };
scanHandlers.counts = code => { if (!S.cc || !sheetOpen()) return false; const h = byBarcode(code); if (h) ccAdd(h.p.id, true); else toast("Mã " + code + " chưa có trong kho", true); return true; };
function ccAdd(pid, inc) {
  const p = S.products.get(pid); if (!p || p.isCombo) return;
  const ex = S.cc.lines.find(l => l.pid === pid);
  if (ex) { if (inc) ex.actual = r3((parseQty(ex.actual) || 0) + 1); }
  else S.cc.lines.unshift({ pid, actual: inc ? 1 : "" });
  renderCountForm(true);
}
function renderCountForm(keep) {
  const C = S.cc;
  const locs = [...new Set([...S.products.values()].map(p => p.location).filter(Boolean))].sort();
  const linesHtml = () => C.lines.length ? `<div class="tbl" style="max-height:52vh;overflow:auto"><table><thead><tr><th>Mặt hàng</th><th class="r">Tồn trên máy</th><th class="r">Đếm thực tế</th><th class="r">Lệch</th><th></th></tr></thead><tbody>${C.lines.map((l, i) => {
    const p = S.products.get(l.pid); if (!p) return "";
    const a = String(l.actual).trim() === "" ? null : parseQty(l.actual);
    const d = a === null ? null : r3(a - p.stock);
    return `<tr><td><div class="pn">${esc(p.name)}</div><div class="ps">${esc(p.location || p.group || "")}</div></td><td class="r">${fq(p.stock)} ${esc(p.unit)}</td>
      <td class="r"><input class="in sm r num" style="width:90px" data-in="ccl" data-i="${i}" value="${esc(l.actual)}" inputmode="decimal" placeholder="?"></td>
      <td class="r" id="ccd-${i}" style="color:${d < 0 ? "var(--red)" : d > 0 ? "var(--green-text)" : "inherit"}">${d === null ? "" : (d > 0 ? "+" : "") + fq(d)}</td><td><button class="x" data-act="ccRm" data-i="${i}" aria-label="Bỏ">×</button></td></tr>`;
  }).join("")}</tbody></table></div>` : `<div class="empty">Thêm mặt hàng cần kiểm, hoặc thêm cả một nhóm / một kệ.</div>`;
  if (keep && sheetOpen() && $("#ccLines")) { $("#ccLines").innerHTML = linesHtml(); return; }
  openSheet(`<div class="sheet-h"><h2>Phiếu kiểm kho</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="grid3" style="align-items:end"><div class="field"><label for="ccQ">Thêm mặt hàng (gõ tên hoặc quét mã)</label><div class="search" style="position:relative">${icon("search")}<input id="ccQ" class="in" autocomplete="off"><div class="dd" id="ccDD" hidden style="left:0;right:0;top:100%"></div></div></div>
      <div class="field"><label for="ccGroup">Thêm cả nhóm</label><select id="ccGroup" class="in"><option value="">— chọn nhóm —</option>${groups().map(g => `<option>${esc(g)}</option>`).join("")}</select></div>
      <div class="field"><label for="ccLoc">Thêm cả kệ / vị trí</label><select id="ccLoc" class="in"><option value="">— chọn vị trí —</option>${locs.map(g => `<option>${esc(g)}</option>`).join("")}</select></div></div>
    <div id="ccLines">${linesHtml()}</div>
    <div class="field"><input id="ccNote" class="in" placeholder="Ghi chú" value="${esc(C.note)}"></div>
    <div class="row"><span class="ps grow">Dòng để trống “Đếm thực tế” sẽ không được lưu. Quét mã vạch nhiều lần = cộng dồn số đếm.</span><button class="btn" data-act="close">Để sau</button><button class="btn pri" data-act="ccSave">Cân bằng kho</button></div>`, { wide: true, focus: "#ccQ" });
}
ON_INPUT.ccQ = el => { const d = $("#ccDD"); d.hidden = !el.value.trim(); d.innerHTML = pickList("product", el.value, "ccPick", 8); };
ACT.ccPick = b => { ccAdd(b.dataset.id); const q = $("#ccQ"); q.value = ""; $("#ccDD").hidden = true; q.focus(); };
ON_CHANGE.ccGroup = el => { if (!el.value) return; [...S.products.values()].filter(p => p.group === el.value && !p.isCombo && !p.hidden).forEach(p => { if (!S.cc.lines.some(l => l.pid === p.id)) S.cc.lines.push({ pid: p.id, actual: "" }); }); el.value = ""; renderCountForm(true); };
ON_CHANGE.ccLoc = el => { if (!el.value) return; [...S.products.values()].filter(p => p.location === el.value && !p.isCombo && !p.hidden).forEach(p => { if (!S.cc.lines.some(l => l.pid === p.id)) S.cc.lines.push({ pid: p.id, actual: "" }); }); el.value = ""; renderCountForm(true); };
ON_INPUT.ccl = el => { const i = +el.dataset.i, l = S.cc.lines[i]; l.actual = el.value; const p = S.products.get(l.pid); const d = el.value.trim() === "" ? null : r3(parseQty(el.value) - p.stock); const c = $("#ccd-" + i); if (c) { c.textContent = d === null ? "" : (d > 0 ? "+" : "") + fq(d); c.style.color = d < 0 ? "var(--red)" : d > 0 ? "var(--green-text)" : "inherit"; } };
ON_INPUT.ccNote = el => { S.cc.note = el.value; };
ACT.ccRm = b => { S.cc.lines.splice(+b.dataset.i, 1); renderCountForm(true); };
ACT.ccSave = async () => {
  const lines = S.cc.lines.filter(l => String(l.actual).trim() !== "").map(l => ({ pid: l.pid, actual: parseQty(l.actual) }));
  if (!lines.length) return toast("Chưa nhập số đếm thực tế", true);
  const r = await run(() => api("POST", "/api/counts", { lines, note: S.cc.note }), c => `Đã cân bằng kho ${c.code}`);
  if (r) { S.cc = null; closeSheet(); await refresh(true); }
};
