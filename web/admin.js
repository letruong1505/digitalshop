/* Digital Shop — khách hàng, nhà cung cấp, sổ quỹ, báo cáo, combo, nhập dữ liệu, cài đặt */
"use strict";

/* ================= KHÁCH HÀNG (sổ nợ) ================= */
S.cQ = ""; S.cOnlyDebt = false;
VIEWS.customers = function () {
  const all = [...S.customers.values()];
  const nq = norm(S.cQ);
  const items = all.filter(c => (!nq || norm(c.name + " " + c.phone + " " + c.code).includes(nq)) && (!S.cOnlyDebt || c.debt > 0))
    .sort((a, b) => (b.debt || 0) - (a.debt || 0) || a.name.localeCompare(b.name, "vi"));
  const total = all.reduce((a, c) => a + Math.max(0, c.debt || 0), 0), owing = all.filter(c => c.debt > 0).length;
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Khách hàng & sổ nợ</h1>${impBtn("customers")}<a class="btn sm" href="/api/export/customers" download>Xuất Excel</a><button class="btn pri sm" data-act="custNew">+ Khách hàng</button></div>
    <div class="row" style="margin-bottom:12px"><div class="search grow" style="min-width:220px">${icon("search")}<input id="cQ" class="in" type="search" autocomplete="off" placeholder="Tìm tên, số điện thoại" value="${esc(S.cQ)}"></div>
      <label class="check"><input type="checkbox" id="cOnlyDebt" ${S.cOnlyDebt ? "checked" : ""}> Chỉ khách đang nợ</label></div>
    <div class="ledger"><div class="ledger-head"><div><div class="lbl">Tổng tiền khách đang nợ</div><div class="hand num" style="color:var(--red)">${money(total)}</div></div><div class="ps" style="padding-bottom:4px">${owing} khách đang nợ · ${all.length} khách trong sổ</div></div>
    ${!all.length ? `<div style="padding:20px 16px 20px 60px" class="muted">Sổ còn trống. Bấm <b>Nhập từ Excel</b> để lấy danh sách khách và công nợ từ KiotViet/MISA, hoặc bấm + Khách hàng.</div>` :
      !items.length ? `<div style="padding:20px 16px 20px 60px" class="muted">Không tìm thấy khách “${esc(S.cQ)}”.</div>` :
      items.slice(0, 500).map((c, i) => `<button class="debtrow" data-act="custOpenD" data-id="${esc(c.id)}"><span class="idx num">${i + 1}</span><span class="who">${esc(c.name)}${c.tier ? ` <span class="pill amber">${esc(c.tier)}</span>` : ""}</span>
        <span class="amt num ${(c.debt || 0) <= 0 ? "zero" : ""}">${(c.debt || 0) <= 0 ? (c.debt < 0 ? "trả dư " + fmt(-c.debt) : "không nợ") : money(c.debt)}</span>
        <span class="meta">${c.phone ? esc(c.phone) + " · " : ""}${esc(c.code)}${c.points ? " · " + fmt(c.points) + " điểm" : ""}${c.spend ? " · đã mua " + fmt(c.spend) : ""}${c.last ? " · giao dịch " + new Date(c.last).toLocaleDateString("vi-VN") : ""}${c.note ? " · " + esc(c.note) : ""}</span></button>`).join("")}
    </div></div>`;
};
ON_INPUT.cQ = el => { S.cQ = el.value; VIEWS.customers(); };
ON_CHANGE.cOnlyDebt = el => { S.cOnlyDebt = el.checked; VIEWS.customers(); };

function partyForm(kind, p) {
  const t = kind === "cust" ? "khách hàng" : "nhà cung cấp";
  openSheet(`<div class="sheet-h"><h2>${p ? "Sửa " + t : "Thêm " + t}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <form id="ptForm" class="grid2" autocomplete="off">
      <div class="field full"><label for="pt-name">Tên *</label><input id="pt-name" class="in" required value="${esc(p?.name || "")}" placeholder="${kind === "cust" ? "vd Chị Hoa xóm 3" : "vd Đại lý bia Minh Phát"}"></div>
      <div class="field"><label for="pt-phone">Điện thoại</label><input id="pt-phone" class="in num" inputmode="tel" value="${esc(p?.phone || "")}"></div>
      <div class="field"><label for="pt-code">Mã</label><input id="pt-code" class="in" value="${esc(p?.code || "")}" placeholder="để trống: tự tạo"></div>
      <div class="field full"><label for="pt-addr">Địa chỉ</label><input id="pt-addr" class="in" value="${esc(p?.addr || "")}"></div>
      ${!p && isOwner() ? `<div class="field"><label for="pt-debt">${kind === "cust" ? "Nợ cũ khách đang nợ" : "Nợ cũ mình đang nợ họ"}</label><input id="pt-debt" class="in num" inputmode="numeric" placeholder="0"></div>` : ""}
      <div class="field ${!p && isOwner() ? "" : "full"}"><label for="pt-note">Ghi chú</label><input id="pt-note" class="in" value="${esc(p?.note || "")}"></div>
      <div class="row full"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri" type="submit">${p ? "Lưu" : "Thêm"}</button></div></form>`, { focus: "#pt-name" });
  $("#ptForm").addEventListener("submit", async e => {
    e.preventDefault();
    const body = { id: p?.id || "", name: $("#pt-name").value, phone: $("#pt-phone").value, code: $("#pt-code").value, addr: $("#pt-addr").value, note: $("#pt-note").value, debt: $("#pt-debt") ? parseMoney($("#pt-debt").value) : 0 };
    const r = await run(() => api("POST", kind === "cust" ? "/api/customers" : "/api/suppliers", body), p ? "Đã lưu" : "Đã thêm " + body.name);
    if (r) { await refresh(true); partyDetail(kind, r.id); }
  });
}
ACT.custNew = () => partyForm("cust");
ACT.custOpenD = b => partyDetail("cust", b.dataset.id);
ACT.suppNew = () => partyForm("supp");
ACT.suppOpenD = b => partyDetail("supp", b.dataset.id);

async function partyDetail(kind, id) {
  const m = kind === "cust" ? S.customers : S.suppliers, p = m.get(id); if (!p) return;
  const isC = kind === "cust";
  const st = S.settings;
  openSheet(`<div class="sheet-h"><h2 ${isC ? 'style="font-family:var(--f-hand);font-size:28px;font-weight:400"' : ""}>${esc(p.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="sumrow"><div><div class="lbl">${isC ? "Khách đang nợ" : "Mình đang nợ"}</div><div class="due" style="color:${p.debt > 0 ? "var(--red)" : "var(--green-text)"}">${money(p.debt)}</div></div>
      <div style="text-align:right">${p.phone ? `<div class="num" style="user-select:all;font-weight:700">${esc(p.phone)}</div>` : ""}<div class="ps">${esc(p.code)}${p.addr ? " · " + esc(p.addr) : ""}</div><div class="ps">${esc(p.note || "")}</div></div></div>
    ${isC && ((S.settings.loyalty || {}).on || p.points) ? `<div class="card" style="padding:10px 12px"><div class="row"><div class="grow"><div class="lbl">Điểm tích lũy</div><div class="num" style="font-size:20px;font-weight:700">${fmt(p.points || 0)} điểm${(S.settings.loyalty || {}).pointValue ? ` <span class="ps">≈ ${money((p.points || 0) * S.settings.loyalty.pointValue)}</span>` : ""}</div></div>${can("manage") ? `<button class="btn sm" data-act="ptsAdj" data-id="${esc(id)}">Cộng / trừ điểm</button>` : ""}</div></div>` : ""}
    <div class="card stack" style="padding:12px">
      ${isC ? (!can("money|sell") ? "" : `<div class="row">${p.debt > 0 && can("money") ? `<button class="btn pri grow" data-act="debtPay" data-id="${esc(id)}">Thu nợ khách</button>` : `<span class="ps grow">${p.debt > 0 ? "" : "Khách không còn nợ."}</span>`}<button class="btn ${p.debt > 0 && can("money") ? "" : "pri"}" data-act="custReturn" data-id="${esc(id)}">Khách trả hàng</button></div>`) : `<div class="field"><label for="pdAmt">Trả nợ nhà cung cấp</label><div class="row"><input id="pdAmt" class="in num grow" inputmode="numeric" placeholder="Số tiền" style="min-width:140px">
        <select id="pdMethod" class="in" style="width:auto"><option value="cash">Tiền mặt</option><option value="bank">Chuyển khoản</option></select>
        <button class="btn" data-act="pdAll" data-v="${Math.max(0, p.debt)}">Hết nợ</button><button class="btn pri" data-act="pdPay" data-k="${kind}" data-id="${esc(id)}">Ghi nhận</button></div></div>`}
      ${isOwner() ? `<details><summary class="ps" style="cursor:pointer">Điều chỉnh công nợ (không qua tiền)</summary><div class="row" style="margin-top:8px"><input id="pdAdj" class="in num" style="width:150px" inputmode="numeric" placeholder="+ hoặc − số tiền"><input id="pdAdjNote" class="in grow" placeholder="Lý do, vd ghi nợ cũ, xóa nợ"><button class="btn" data-act="pdAdj" data-k="${kind}" data-id="${esc(id)}">Điều chỉnh</button></div></details>` : ""}
    </div>
    <div class="row">${isC && p.debt > 0 ? `<button class="btn sm" data-act="remind" data-id="${esc(id)}">Sao chép tin nhắn nhắc nợ</button>${st.acct ? `<button class="btn sm" data-act="debtQR" data-id="${esc(id)}">Mã QR trả nợ</button>` : ""}` : ""}
      ${!isC && can("stock") ? `<button class="btn sm" data-act="suppReturn" data-id="${esc(id)}">Trả hàng nhà cung cấp</button>` : ""}
      <button class="btn sm" data-act="pdEdit" data-k="${kind}" data-id="${esc(id)}">Sửa thông tin</button><span class="grow"></span>
      ${isOwner() && !p.debt ? `<button class="btn sm danger" data-act="pdDel" data-k="${kind}" data-id="${esc(id)}" data-confirm="Bấm lần nữa để xóa">Xóa</button>` : ""}</div>
    <div><div class="lbl" style="margin-bottom:6px">Lịch sử</div><div id="pdHist"><div class="muted">Đang tải…</div></div></div>`);
  const h = await run(() => api("GET", `/api/${isC ? "customers" : "suppliers"}/${id}/history`)); if (!h || !$("#pdHist")) return;
  $("#pdHist").innerHTML = h.length ? `<div class="tbl"><table><tbody>${h.map(x => {
    let what, eff;
    if (x.type === "invoice") { what = `Hóa đơn ${esc(x.code)} · ${money(x.total)}<div class="ps">${esc(x.lines.map(l => l.name + " ×" + fq(l.qty)).join(", ")).slice(0, 120)}</div>`; eff = x.debt; }
    else if (x.type === "return") { what = `Khách trả hàng ${esc(x.code)} · ${money(x.total)}${x.refund > 0 ? " · " + ({ cash: "hoàn tiền mặt", bank: "hoàn chuyển khoản", credit: "giữ làm số dư" }[x.refundMethod] || "") + " " + money(x.refund) : ""}${x.restock ? "" : " · không nhập kho"}<div class="ps">${esc(x.lines.map(l => l.name + " ×" + fq(l.qty)).join(", ")).slice(0, 120)}</div>`; eff = x.debtEffect; }
    else if (x.type === "pts") { what = `Điều chỉnh điểm ${x.delta > 0 ? "+" : ""}${fmt(x.delta)} · còn ${fmt(x.bal)}${x.note ? " · " + esc(x.note) : ""}<div class="ps">${esc(x.staff || "")}</div>`; eff = 0; }
    else if (x.type === "sreturn") { what = `Trả hàng NCC ${esc(x.code)} · ${money(x.total)}${x.refund > 0 ? " · " + ({ cash: "NCC hoàn tiền mặt", bank: "NCC hoàn chuyển khoản", credit: "giữ làm số dư" }[x.refundMethod] || "") + " " + money(x.refund) : ""}<div class="ps">${esc(x.lines.map(l => l.name + " ×" + fq(l.qty)).join(", ")).slice(0, 120)}</div>`; eff = x.debtEffect; }
    else if (x.type === "receipt") { what = `Phiếu nhập ${esc(x.code)} · ${money(x.total)}`; eff = x.debt; }
    else { what = `${esc(x.cat || (x.kind === "thu" ? "Phiếu thu" : x.kind === "chi" ? "Phiếu chi" : "Điều chỉnh"))} ${esc(x.code)}${x.method === "bank" && x.kind !== "adj" ? " · chuyển khoản" : ""}${x.note ? `<div class="ps">${esc(x.note)}</div>` : ""}`; eff = x.debtEffect; }
    return `<tr class="${x.void ? "void" : ""}"><td class="num ps" style="white-space:nowrap">${dt(x.time)}</td><td>${what}</td><td class="r num" style="color:${eff > 0 ? "var(--red)" : eff < 0 ? "var(--green-text)" : "inherit"}"><b>${eff ? (eff > 0 ? "+" : "") + fmt(eff) : ""}</b></td>
      <td class="r">${x.type === "sreturn" && !x.void && can("manage") ? `<button class="btn sm ghost" data-act="sretVoid" data-id="${esc(x.id)}" data-back="supp:${esc(id)}" data-confirm="Hủy phiếu trả hàng NCC? Nợ và kho trở lại như trước">Hủy</button>` : ""}${x.type === "return" && !x.void && can("manage") ? `<button class="btn sm ghost" data-act="retVoid" data-id="${esc(x.id)}" data-back="cust:${esc(id)}" data-confirm="Hủy phiếu trả hàng? Nợ và kho trở lại như trước">Hủy</button>` : ""}${x.type === "cash" && !x.void && isOwner() ? `<button class="btn sm ghost" data-act="cashVoid" data-id="${esc(x.id)}" data-back="${kind}:${esc(id)}" data-confirm="Hủy phiếu?">Hủy</button>` : ""}</td></tr>`;
  }).join("")}</tbody></table></div>` : `<div class="ps">Chưa có giao dịch.</div>`;
}

/* ---------- thu nợ khách: số tiền, trừ nợ, tiền mặt / chuyển khoản, in phiếu thu ---------- */
let DP = null;
function debtPayHtml() {
  const c = S.customers.get(DP.id), st = S.settings, a = DP.amount, left = c.debt - a, bank = DP.method === "bank" && st.acct;
  const bad = a > c.debt + 0.5;
  return `<div class="sheet-h"><h2>Thu nợ · ${esc(c.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="sumrow"><div><div class="lbl">Khách đang nợ</div><div class="due" style="color:var(--red)">${money(c.debt)}</div></div><div class="ps" style="text-align:right">${esc(c.code)}${c.phone ? "<br>" + esc(c.phone) : ""}</div></div>
    ${dpModeSeg("pay")}
    <div class="stack">
      <div class="field"><label for="dpAmt">Số tiền khách trả lần này</label><div class="row"><input id="dpAmt" class="in num grow" inputmode="numeric" placeholder="vd 500.000" value="${a ? fmt(a) : ""}" style="font-size:20px;font-weight:700;min-width:160px"><button type="button" class="btn" data-act="dpAll">Trả hết nợ</button></div></div>
      <div class="field"><label>Hình thức thanh toán</label><div class="seg" role="group" aria-label="Hình thức">${[["cash", "Tiền mặt"], ["bank", "Chuyển khoản"]].map(([k, t]) => `<button type="button" data-act="dpMethod" data-v="${k}" aria-pressed="${DP.method === k}">${t}</button>`).join("")}</div></div>
      <div class="card" id="dpSum" style="padding:12px;display:grid;gap:4px">${dpSumHtml(c, a, left, bad)}</div>
      ${bank ? `<div id="dpQR" style="display:grid;place-items:center;gap:6px">${a > 0 && !bad ? `<div style="width:230px;background:#fff;padding:8px;border-radius:10px">${qrSvg(vietqr(a, c.code + " tra no"))}</div><div class="ps">${esc(st.bankName || "")} · ${esc(st.acct)} · ${esc(st.holder || "")}<br>Số tiền và nội dung đã điền sẵn trong mã QR. Chỉ bấm Ghi nhận khi tiền đã về tài khoản.</div>` : `<div class="ps">Nhập số tiền để hiện mã QR.</div>`}</div>` : DP.method === "bank" ? `<div class="ps">Chưa khai số tài khoản ở Cài đặt nên chưa có mã QR. Vẫn ghi nhận được.</div>` : ""}
      <div class="field"><label for="dpNote">Ghi chú (không bắt buộc)</label><input id="dpNote" class="in" value="${esc(DP.note)}" placeholder="vd trả đợt 1"></div>
      <div class="row"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button type="button" class="btn pri" data-act="dpSave" id="dpSave" ${a > 0 && !bad ? "" : "disabled"}>Ghi nhận ${a > 0 && !bad ? money(a) : ""}</button></div>
    </div>`;
}
function dpSumHtml(c, a, left, bad) {
  return `<div class="row" style="justify-content:space-between"><span>Nợ hiện tại</span><b class="num">${money(c.debt)}</b></div>
    <div class="row" style="justify-content:space-between"><span>Khách trả (${DP.method === "bank" ? "chuyển khoản" : "tiền mặt"})</span><b class="num" style="color:var(--green-text)">− ${money(a || 0)}</b></div>
    <div class="row" style="justify-content:space-between;border-top:1px solid var(--line);padding-top:6px"><span><b>Nợ còn lại sau khi thu</b></span><b class="num" style="font-size:18px;color:${bad ? "var(--red)" : left > 0 ? "var(--red)" : "var(--green-text)"}">${bad ? "Quá số nợ" : money(left)}</b></div>
    ${bad ? `<div class="ps" style="color:var(--red)">Khách chỉ đang nợ ${money(c.debt)}, không nhập quá số này.</div>` : ""}`;
}
ACT.debtPay = b => { DP = { id: b.dataset.id, amount: 0, method: "cash", note: "" }; openSheet(debtPayHtml(), { focus: "#dpAmt" }); };
ACT.dpAll = () => { DP.amount = Math.max(0, S.customers.get(DP.id).debt); dpRefresh(true); };
ACT.dpMethod = b => { DP.method = b.dataset.v; DP.note = ($("#dpNote") || {}).value || ""; dpRefresh(true); };
function dpRefresh(full) {
  const c = S.customers.get(DP.id), a = DP.amount, left = c.debt - a, bad = a > c.debt + 0.5;
  if (full) { $("#sheetCard").innerHTML = debtPayHtml(); const i = $("#dpAmt"); if (i) { i.focus(); i.setSelectionRange(i.value.length, i.value.length); } return; }
  $("#dpSum").innerHTML = dpSumHtml(c, a, left, bad);
  const sv = $("#dpSave"); sv.disabled = !(a > 0 && !bad); sv.textContent = "Ghi nhận " + (a > 0 && !bad ? money(a) : "");
  const q = $("#dpQR"); if (q) { const st = S.settings; q.innerHTML = a > 0 && !bad ? `<div style="width:230px;background:#fff;padding:8px;border-radius:10px">${qrSvg(vietqr(a, c.code + " tra no"))}</div><div class="ps">${esc(st.bankName || "")} · ${esc(st.acct)} · ${esc(st.holder || "")}<br>Số tiền và nội dung đã điền sẵn trong mã QR. Chỉ bấm Ghi nhận khi tiền đã về tài khoản.</div>` : `<div class="ps">Nhập số tiền để hiện mã QR.</div>`; }
}
ON_INPUT.dpAmt = el => { DP.amount = parseMoney(el.value); dpRefresh(false); };
ACT.dpSave = async b => {
  DP.note = ($("#dpNote") || {}).value || ""; const c = S.customers.get(DP.id), a = DP.amount, method = DP.method, note = DP.note.trim();
  if (!(a > 0)) return toast("Nhập số tiền khách trả", true);
  b.disabled = true;
  const r = await run(() => api("POST", "/api/cash", { kind: "thu", cat: "Thu nợ khách", partyType: "cust", partyId: DP.id, amount: a, method, note, capDebt: true }));
  if (!r) { b.disabled = false; return; }
  await refresh(true);
  DP.done = { e: r, name: c.name, code: c.code, phone: c.phone, before: r.debtAfter + a };
  openSheet(`<div class="sheet-h"><h2>Đã thu nợ</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div style="display:grid;gap:6px;text-align:center;padding:8px 0"><div style="font-size:42px;color:var(--green-text)">✓</div><b style="font-size:22px">${money(a)}</b><div class="ps">${esc(c.name)} · ${method === "bank" ? "chuyển khoản" : "tiền mặt"} · phiếu ${esc(r.code)}</div>
      <div>Nợ trước ${money(r.debtAfter + a)} → <b style="color:${r.debtAfter > 0 ? "var(--red)" : "var(--green-text)"}">còn ${money(r.debtAfter)}</b></div></div>
    <div class="row"><span class="grow"></span><button class="btn" data-act="dpPrint">${icon("print")} In phiếu thu</button><button class="btn pri" data-act="close">Xong</button></div>`);
};
ACT.dpPrint = () => { if (DP && DP.done) printHtml(debtReceiptHtml(DP.done)); };
function debtReceiptHtml(d) {
  const st = S.settings, e = d.e, row = (k, v, bold) => `<tr><td>${k}</td><td class="r">${bold ? "<b>" : ""}${v}${bold ? "</b>" : ""}</td></tr>`;
  return `<div class="rc ${st.paper === "58" ? "p58" : st.paper === "a5" ? "a5" : ""}"><div class="c shop">${esc(st.shopName || "")}</div>${st.addr ? `<div class="c">${esc(st.addr)}</div>` : ""}${st.phone ? `<div class="c">ĐT ${esc(st.phone)}</div>` : ""}<hr>
    <div class="c big">PHIẾU THU NỢ</div><div class="c">${esc(e.code)} · ${dt(e.time)}</div><hr>
    <div>Khách hàng: <b>${esc(d.name)}</b></div>${d.phone ? `<div>Điện thoại: ${esc(d.phone)}</div>` : ""}
    <table>${row("Nợ trước khi thu", money(d.before))}${row("Số tiền thu", money(e.amount), true)}${row("Hình thức", e.method === "bank" ? "Chuyển khoản" : "Tiền mặt")}${row("Nợ còn lại", money(e.debtAfter), true)}</table>
    ${e.note ? `<div>Ghi chú: ${esc(e.note)}</div>` : ""}<div>Thu ngân: ${esc(e.staff || "")}</div><hr><div class="c">${esc(st.footer || "Cảm ơn quý khách")}</div></div>`;
}

/* ---------- khách trả hàng: trừ vào nợ, hoàn tiền phần dư, nhập lại kho ---------- */
let DR = null;
function dpModeSeg(cur) {
  const c = S.customers.get((DP || DR).id);
  return `<div class="seg" role="group" aria-label="Cách thanh toán" style="margin:8px 0">${[["pay", "Khách trả tiền", c.debt > 0], ["ret", "Khách trả hàng", true]].filter(x => x[2]).map(([k, t]) => `<button type="button" data-act="dpMode" data-v="${k}" aria-pressed="${cur === k}">${t}</button>`).join("")}</div>`;
}
ACT.dpMode = b => { if (b.dataset.v === "pay") { DP = { id: (DP || DR).id, amount: 0, method: "cash", note: "" }; $("#sheetCard").innerHTML = debtPayHtml(); } else openReturn((DP || DR).id); };
ACT.custReturn = b => openReturn(b.dataset.id);
async function openReturn(id) {
  DR = { id, items: null, restock: true, toDebt: true, refundMethod: "cash", note: "" };
  const items = await run(() => api("GET", `/api/customers/${id}/returnable`)); if (!items) return;
  DR.items = items.filter(x => x.left > 0).map(x => ({ ...x, qty: 0, priceIn: x.price }));
  drRender();
}
function drTotals() {
  const c = S.customers.get(DR.id);
  const total = DR.items.reduce((t, x) => t + (x.qty > 0 ? x.qty * x.priceIn : 0), 0);
  const apply = DR.toDebt && c.debt > 0 ? Math.min(total, c.debt) : 0, refund = total - apply;
  const bad = DR.items.some(x => x.qty > x.left + 0.0005 || (x.qty > 0 && x.priceIn > x.maxPrice));
  return { c, total: Math.round(total), apply: Math.round(apply), refund: Math.round(refund), bad };
}
function drSumHtml() {
  const { c, total, apply, refund, bad } = drTotals(), credit = DR.refundMethod === "credit";
  const after = c.debt - apply - (credit ? refund : 0);
  return `<div class="row" style="justify-content:space-between"><span>Giá trị hàng khách trả</span><b class="num">${money(total)}</b></div>
    ${c.debt > 0 ? `<div class="row" style="justify-content:space-between"><span>Trừ vào công nợ</span><b class="num" style="color:var(--green-text)">− ${money(apply)}</b></div>` : ""}
    ${refund > 0 ? `<div class="row" style="justify-content:space-between"><span>${credit ? "Giữ làm số dư cho khách" : "Cửa hàng hoàn lại khách"}</span><b class="num">${money(refund)}</b></div>` : ""}
    <div class="row" style="justify-content:space-between;border-top:1px solid var(--line);padding-top:6px"><span><b>Công nợ sau khi trả hàng</b></span><b class="num" style="font-size:18px;color:${after > 0 ? "var(--red)" : "var(--green-text)"}">${after < 0 ? "dư " + money(-after) : money(after)}</b></div>
    ${bad ? `<div class="ps" style="color:var(--red)">Số lượng hoặc giá vượt mức khách đã mua, kiểm tra các dòng bị đỏ.</div>` : ""}`;
}
function drRefundHtml() {
  const { c, refund } = drTotals();
  return refund > 0 ? `<div class="field"><label for="rtMethod">Phần còn dư ${money(refund)}: xử lý thế nào?</label><select id="rtMethod" class="in" data-ch="rtMethod"><option value="cash" ${DR.refundMethod === "cash" ? "selected" : ""}>Hoàn tiền mặt cho khách</option><option value="bank" ${DR.refundMethod === "bank" ? "selected" : ""}>Hoàn chuyển khoản cho khách</option><option value="credit" ${DR.refundMethod === "credit" ? "selected" : ""}>Giữ làm số dư của khách (trừ vào lần mua sau)</option></select></div>` : "";
}
function drRender() {
  const c = S.customers.get(DR.id);
  openSheet(`<div class="sheet-h"><h2>Khách trả hàng · ${esc(c.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="sumrow"><div><div class="lbl">${c.debt > 0 ? "Khách đang nợ" : c.debt < 0 ? "Số dư của khách" : "Công nợ"}</div><div class="due" style="color:${c.debt > 0 ? "var(--red)" : "var(--green-text)"}">${c.debt < 0 ? money(-c.debt) : money(c.debt)}</div></div><div class="ps" style="text-align:right">${esc(c.code)}${c.phone ? "<br>" + esc(c.phone) : ""}</div></div>
    ${c.debt > 0 ? dpModeSeg("ret") : ""}
    ${DR.items.length ? `<div class="stack"><div class="ps">Chọn hàng khách trả (chỉ hàng khách đã mua, giá mặc định là giá khách đã trả sau giảm giá).</div>
      <div class="tbl"><table><thead><tr><th>Mặt hàng</th><th class="r">Còn trả được</th><th class="r" style="width:96px">Số lượng trả</th><th class="r" style="width:110px">Giá / đơn vị</th></tr></thead><tbody>${DR.items.map((x, i) => `<tr id="drRow${i}"><td class="pn">${esc(x.name)}</td><td class="r ps">${fq(x.left)} ${esc(x.unit)}</td><td class="r"><input class="in num" style="width:84px;text-align:right" inputmode="decimal" data-in="rtQty" data-i="${i}" placeholder="0"></td><td class="r"><input class="in num" style="width:100px;text-align:right" inputmode="numeric" data-in="rtPrice" data-i="${i}" value="${fmt(x.priceIn)}"></td></tr>`).join("")}</tbody></table></div>
      <label class="check"><input type="checkbox" data-ch="rtRestock" ${DR.restock ? "checked" : ""}> <span>Nhập lại kho (hàng còn bán được). Bỏ tick nếu hàng hỏng/hết hạn: kho không tăng, cửa hàng chịu lỗ giá vốn.</span></label>
      ${c.debt > 0 ? `<label class="check"><input type="checkbox" data-ch="rtToDebt" ${DR.toDebt ? "checked" : ""}> <span>Trừ giá trị hàng trả vào công nợ</span></label>` : ""}
      <div class="card" id="drSum" style="padding:12px;display:grid;gap:4px">${drSumHtml()}</div>
      <div id="drRef">${drRefundHtml()}</div>
      <div class="field"><label for="drNote">Ghi chú (không bắt buộc)</label><input id="drNote" class="in" placeholder="vd hàng lỗi, giao nhầm"></div>
      <div class="row"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button type="button" class="btn pri" id="drSave" data-act="drSave" disabled>Ghi nhận trả hàng</button></div></div>`
      : `<div class="empty" style="margin:12px 0">Chưa có hàng nào khách này có thể trả (khách chưa mua hàng qua hóa đơn trong phần mềm, hoặc đã trả hết).</div><div class="row"><span class="grow"></span><button class="btn" data-act="close">Đóng</button></div>`}`);
}
function drRefresh() {
  const t = drTotals();
  $("#drSum").innerHTML = drSumHtml(); $("#drRef").innerHTML = drRefundHtml();
  const sv = $("#drSave"); sv.disabled = !(t.total > 0) || t.bad;
  DR.items.forEach((x, i) => { const r = $("#drRow" + i); if (r) r.style.background = x.qty > x.left + 0.0005 || (x.qty > 0 && x.priceIn > x.maxPrice) ? "var(--red-bg, #fdecea)" : ""; });
}
ON_INPUT.rtQty = el => { DR.items[+el.dataset.i].qty = Math.max(0, parseFloat(String(el.value).replace(",", ".")) || 0); drRefresh(); };
ON_INPUT.rtPrice = el => { DR.items[+el.dataset.i].priceIn = parseMoney(el.value); drRefresh(); };
ON_CHANGE.rtRestock = el => { DR.restock = el.checked; };
ON_CHANGE.rtToDebt = el => { DR.toDebt = el.checked; drRefresh(); };
ON_CHANGE.rtMethod = el => { DR.refundMethod = el.value; drRefresh(); };
ACT.drSave = async b => {
  const t = drTotals(); if (!(t.total > 0) || t.bad) return;
  const lines = DR.items.filter(x => x.qty > 0).map(x => ({ pid: x.pid, qty: x.qty, price: x.priceIn }));
  b.disabled = true;
  const r = await run(() => api("POST", `/api/customers/${DR.id}/return`, { lines, restock: DR.restock, toDebt: DR.toDebt, refundMethod: DR.refundMethod, note: ($("#drNote") || {}).value || "" }));
  if (!r) { b.disabled = false; return; }
  await refresh(true);
  DR.done = r;
  const method = { cash: "hoàn tiền mặt", bank: "hoàn chuyển khoản", credit: "giữ làm số dư" }[r.refundMethod] || "";
  openSheet(`<div class="sheet-h"><h2>Đã nhận trả hàng</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div style="display:grid;gap:6px;text-align:center;padding:8px 0"><div style="font-size:42px;color:var(--green-text)">✓</div><b style="font-size:22px">${money(r.total)}</b><div class="ps">${esc(r.custName)} · phiếu ${esc(r.code)}${r.restock ? " · đã nhập lại kho" : ""}</div>
      ${r.applyDebt > 0 ? `<div>Trừ vào nợ <b>${money(r.applyDebt)}</b></div>` : ""}${r.refund > 0 ? `<div>${method[0].toUpperCase() + method.slice(1)} <b>${money(r.refund)}</b></div>` : ""}
      <div>Công nợ sau khi trả hàng: <b style="color:${r.debtAfter > 0 ? "var(--red)" : "var(--green-text)"}">${r.debtAfter < 0 ? "dư " + money(-r.debtAfter) : money(r.debtAfter)}</b></div></div>
    <div class="row"><span class="grow"></span><button class="btn" data-act="drPrint">${icon("print")} In phiếu trả hàng</button><button class="btn pri" data-act="close">Xong</button></div>`);
};
ACT.drPrint = () => { if (DR && DR.done) printHtml(returnReceiptHtml(DR.done)); };
function returnReceiptHtml(r) {
  const st = S.settings, row = (k, v, bold) => `<tr><td>${k}</td><td class="r">${bold ? "<b>" : ""}${v}${bold ? "</b>" : ""}</td></tr>`;
  return `<div class="rc ${st.paper === "58" ? "p58" : st.paper === "a5" ? "a5" : ""}"><div class="c shop">${esc(st.shopName || "")}</div>${st.addr ? `<div class="c">${esc(st.addr)}</div>` : ""}${st.phone ? `<div class="c">ĐT ${esc(st.phone)}</div>` : ""}<hr>
    <div class="c big">PHIẾU TRẢ HÀNG</div><div class="c">${esc(r.code)} · ${dt(r.time)}</div><hr><div>Khách hàng: <b>${esc(r.custName)}</b></div>
    <table>${r.lines.map(l => `<tr><td>${esc(l.name)}<br>${fq(l.qty)} ${esc(l.unit)} × ${fmt(l.price)}</td><td class="r">${fmt(l.qty * l.price)}</td></tr>`).join("")}</table><hr>
    <table>${row("Giá trị hàng trả", money(r.total), true)}${r.applyDebt > 0 ? row("Trừ vào công nợ", money(r.applyDebt)) : ""}${r.refund > 0 ? row(r.refundMethod === "credit" ? "Giữ làm số dư" : r.refundMethod === "bank" ? "Hoàn chuyển khoản" : "Hoàn tiền mặt", money(r.refund)) : ""}${row("Công nợ còn lại", r.debtAfter < 0 ? "dư " + money(-r.debtAfter) : money(r.debtAfter), true)}</table>
    ${r.note ? `<div>Ghi chú: ${esc(r.note)}</div>` : ""}<div>Nhân viên: ${esc(r.staff || "")}</div><hr><div class="c">${esc(st.footer || "Cảm ơn quý khách")}</div></div>`;
}
ACT.retVoid = async b => {
  const r = await run(() => api("POST", `/api/returns/${b.dataset.id}/void`), "Đã hủy phiếu trả hàng");
  if (!r) return; await refresh(true);
  const [k, id] = b.dataset.back.split(":"); partyDetail(k, id);
};
/* ---------- trả hàng nhà cung cấp ---------- */
let SR = null;
ACT.suppReturn = async b => {
  const id = b.dataset.id;
  SR = { id, items: null, toDebt: true, refundMethod: "cash" };
  const items = await run(() => api("GET", `/api/suppliers/${id}/returnable`)); if (!items) return;
  SR.items = items.filter(x => x.left > 0).map(x => ({ ...x, qty: 0, priceIn: x.price }));
  srRender();
};
function srTotals() {
  const s = S.suppliers.get(SR.id);
  const total = SR.items.reduce((t, x) => t + (x.qty > 0 ? x.qty * x.priceIn : 0), 0);
  const apply = SR.toDebt && s.debt > 0 ? Math.min(total, s.debt) : 0, refund = total - apply;
  const bad = SR.items.some(x => x.qty > x.left + 0.0005 || (x.qty > 0 && x.priceIn > x.maxPrice));
  return { s, total: Math.round(total), apply: Math.round(apply), refund: Math.round(refund), bad };
}
function srSumHtml() {
  const { s, total, apply, refund, bad } = srTotals(), credit = SR.refundMethod === "credit";
  const after = s.debt - apply - (credit ? refund : 0);
  return `<div class="row" style="justify-content:space-between"><span>Giá trị hàng trả NCC</span><b class="num">${money(total)}</b></div>
    ${s.debt > 0 ? `<div class="row" style="justify-content:space-between"><span>Trừ vào nợ phải trả</span><b class="num" style="color:var(--green-text)">− ${money(apply)}</b></div>` : ""}
    ${refund > 0 ? `<div class="row" style="justify-content:space-between"><span>${credit ? "Giữ làm số dư (trừ vào lần nhập sau)" : "NCC hoàn lại cho mình"}</span><b class="num">${money(refund)}</b></div>` : ""}
    <div class="row" style="justify-content:space-between;border-top:1px solid var(--line);padding-top:6px"><span><b>Nợ NCC sau khi trả hàng</b></span><b class="num" style="font-size:18px;color:${after > 0 ? "var(--red)" : "var(--green-text)"}">${after < 0 ? "NCC nợ lại " + money(-after) : money(after)}</b></div>
    ${bad ? `<div class="ps" style="color:var(--red)">Số lượng vượt mức trả được hoặc giá cao hơn giá đã nhập, kiểm tra các dòng bị đỏ.</div>` : ""}`;
}
function srRefundHtml() {
  const { refund } = srTotals();
  return refund > 0 ? `<div class="field"><label for="srMethod">Phần dư ${money(refund)}: xử lý thế nào?</label><select id="srMethod" class="in" data-ch="srMethod"><option value="cash" ${SR.refundMethod === "cash" ? "selected" : ""}>NCC hoàn tiền mặt</option><option value="bank" ${SR.refundMethod === "bank" ? "selected" : ""}>NCC hoàn chuyển khoản</option><option value="credit" ${SR.refundMethod === "credit" ? "selected" : ""}>Giữ làm số dư, trừ vào lần nhập sau</option></select></div>` : "";
}
function srRender() {
  const s = S.suppliers.get(SR.id);
  openSheet(`<div class="sheet-h"><h2>Trả hàng nhà cung cấp · ${esc(s.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="sumrow"><div><div class="lbl">${s.debt < 0 ? "NCC đang nợ mình" : "Mình đang nợ"}</div><div class="due" style="color:${s.debt > 0 ? "var(--red)" : "var(--green-text)"}">${money(Math.abs(s.debt))}</div></div></div>
    ${SR.items.length ? `<div class="stack"><div class="ps">Chọn hàng trả lại (chỉ hàng đã nhập từ NCC này và còn tồn kho). Giá mặc định là giá nhập bình quân.</div>
      <div class="tbl"><table><thead><tr><th>Mặt hàng</th><th class="r">Trả được</th><th class="r" style="width:96px">Số lượng trả</th><th class="r" style="width:110px">Giá / đơn vị</th></tr></thead><tbody>${SR.items.map((x, i) => `<tr id="srRow${i}"><td class="pn">${esc(x.name)}</td><td class="r ps">${fq(x.left)} ${esc(x.unit)}</td><td class="r"><input class="in num" style="width:84px;text-align:right" inputmode="decimal" data-in="srQty" data-i="${i}" placeholder="0"></td><td class="r"><input class="in num" style="width:100px;text-align:right" inputmode="numeric" data-in="srPrice" data-i="${i}" value="${fmt(x.priceIn)}"></td></tr>`).join("")}</tbody></table></div>
      ${s.debt > 0 ? `<label class="check"><input type="checkbox" data-ch="srToDebt" ${SR.toDebt ? "checked" : ""}> <span>Trừ giá trị hàng trả vào nợ phải trả NCC</span></label>` : ""}
      <div class="card" id="srSum" style="padding:12px;display:grid;gap:4px">${srSumHtml()}</div>
      <div id="srRef">${srRefundHtml()}</div>
      <div class="field"><label for="srNote">Ghi chú (không bắt buộc)</label><input id="srNote" class="in" placeholder="vd hàng lỗi, hết hạn, giao nhầm"></div>
      <div class="row"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button type="button" class="btn pri" id="srSave" data-act="srSave" disabled>Ghi nhận trả hàng</button></div></div>`
      : `<div class="empty" style="margin:12px 0">Chưa có hàng nào trả được (chưa nhập hàng từ NCC này qua phiếu nhập, hoặc đã trả hết, hoặc kho đã hết).</div><div class="row"><span class="grow"></span><button class="btn" data-act="close">Đóng</button></div>`}`);
}
function srRefresh() {
  const t = srTotals();
  $("#srSum").innerHTML = srSumHtml(); $("#srRef").innerHTML = srRefundHtml();
  $("#srSave").disabled = !(t.total > 0) || t.bad;
  SR.items.forEach((x, i) => { const r = $("#srRow" + i); if (r) r.style.background = x.qty > x.left + 0.0005 || (x.qty > 0 && x.priceIn > x.maxPrice) ? "var(--red-bg, #fdecea)" : ""; });
}
ON_INPUT.srQty = el => { SR.items[+el.dataset.i].qty = Math.max(0, parseFloat(String(el.value).replace(",", ".")) || 0); srRefresh(); };
ON_INPUT.srPrice = el => { SR.items[+el.dataset.i].priceIn = parseMoney(el.value); srRefresh(); };
ON_CHANGE.srToDebt = el => { SR.toDebt = el.checked; srRefresh(); };
ON_CHANGE.srMethod = el => { SR.refundMethod = el.value; srRefresh(); };
ACT.srSave = async b => {
  const t = srTotals(); if (!(t.total > 0) || t.bad) return;
  const lines = SR.items.filter(x => x.qty > 0).map(x => ({ pid: x.pid, qty: x.qty, price: x.priceIn }));
  b.disabled = true;
  const r = await run(() => api("POST", `/api/suppliers/${SR.id}/return`, { lines, toDebt: SR.toDebt, refundMethod: SR.refundMethod, note: ($("#srNote") || {}).value || "" }));
  if (!r) { b.disabled = false; return; }
  await refresh(true);
  SR.done = r;
  const method = { cash: "NCC hoàn tiền mặt", bank: "NCC hoàn chuyển khoản", credit: "Giữ làm số dư" }[r.refundMethod] || "";
  openSheet(`<div class="sheet-h"><h2>Đã trả hàng cho NCC</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div style="display:grid;gap:6px;text-align:center;padding:8px 0"><div style="font-size:42px;color:var(--green-text)">✓</div><b style="font-size:22px">${money(r.total)}</b><div class="ps">${esc(r.suppName)} · phiếu ${esc(r.code)} · đã trừ kho</div>
      ${r.applyDebt > 0 ? `<div>Trừ vào nợ <b>${money(r.applyDebt)}</b></div>` : ""}${r.refund > 0 ? `<div>${method} <b>${money(r.refund)}</b></div>` : ""}
      <div>Nợ NCC sau khi trả hàng: <b style="color:${r.debtAfter > 0 ? "var(--red)" : "var(--green-text)"}">${r.debtAfter < 0 ? "NCC nợ lại " + money(-r.debtAfter) : money(r.debtAfter)}</b></div></div>
    <div class="row"><span class="grow"></span><button class="btn" data-act="srPrint">${icon("print")} In phiếu</button><button class="btn pri" data-act="close">Xong</button></div>`);
};
ACT.srPrint = () => { if (SR && SR.done) printHtml(suppReturnReceiptHtml(SR.done)); };
function suppReturnReceiptHtml(r) {
  const st = S.settings, row = (k, v, bold) => `<tr><td>${k}</td><td class="r">${bold ? "<b>" : ""}${v}${bold ? "</b>" : ""}</td></tr>`;
  return `<div class="rc ${st.paper === "58" ? "p58" : st.paper === "a5" ? "a5" : ""}"><div class="c shop">${esc(st.shopName || "")}</div>${st.addr ? `<div class="c">${esc(st.addr)}</div>` : ""}${st.phone ? `<div class="c">ĐT ${esc(st.phone)}</div>` : ""}<hr>
    <div class="c big">PHIẾU TRẢ HÀNG NHÀ CUNG CẤP</div><div class="c">${esc(r.code)} · ${dt(r.time)}</div><hr><div>Nhà cung cấp: <b>${esc(r.suppName)}</b></div>
    <table>${r.lines.map(l => `<tr><td>${esc(l.name)}<br>${fq(l.qty)} ${esc(l.unit)} × ${fmt(l.price)}</td><td class="r">${fmt(l.qty * l.price)}</td></tr>`).join("")}</table><hr>
    <table>${row("Giá trị hàng trả", money(r.total), true)}${r.applyDebt > 0 ? row("Trừ vào nợ phải trả", money(r.applyDebt)) : ""}${r.refund > 0 ? row(r.refundMethod === "credit" ? "Giữ làm số dư" : r.refundMethod === "bank" ? "NCC hoàn chuyển khoản" : "NCC hoàn tiền mặt", money(r.refund)) : ""}${row("Nợ NCC còn lại", r.debtAfter < 0 ? "-" + money(-r.debtAfter) : money(r.debtAfter), true)}</table>
    ${r.note ? `<div>Ghi chú: ${esc(r.note)}</div>` : ""}<div>Người lập: ${esc(r.staff || "")}</div><hr><div class="c">Ký nhận: ______________</div></div>`;
}
ACT.sretVoid = async b => {
  const r = await run(() => api("POST", `/api/sreturns/${b.dataset.id}/void`), "Đã hủy phiếu trả hàng NCC");
  if (!r) return; await refresh(true);
  const [k, id] = b.dataset.back.split(":"); partyDetail(k, id);
};
ACT.pdAll = b => { $("#pdAmt").value = fmt(+b.dataset.v); };
ACT.pdPay = async b => {
  const k = b.dataset.k, id = b.dataset.id, a = parseMoney($("#pdAmt").value);
  if (!a) return toast("Nhập số tiền", true);
  const body = k === "cust" ? { kind: "thu", cat: "Thu nợ khách", partyType: "cust" } : { kind: "chi", cat: "Trả nợ nhà cung cấp", partyType: "supp" };
  const r = await run(() => api("POST", "/api/cash", Object.assign(body, { partyId: id, amount: a, method: $("#pdMethod").value })), `Đã ghi nhận ${money(a)}`);
  if (r) { await refresh(true); partyDetail(k, id); }
};
ACT.pdAdj = async b => {
  const k = b.dataset.k, id = b.dataset.id, raw = $("#pdAdj").value.trim();
  const a = parseMoney(raw.replace(/^[+-]/, "")) * (raw.startsWith("-") || raw.startsWith("−") ? -1 : 1);
  if (!a) return toast("Nhập số tiền điều chỉnh, vd 50000 hoặc -50000", true);
  const r = await run(() => api("POST", "/api/cash", { kind: "adj", partyType: k, partyId: id, amount: a, cat: "Điều chỉnh công nợ", note: $("#pdAdjNote").value }), "Đã điều chỉnh công nợ");
  if (r) { await refresh(true); partyDetail(k, id); }
};
ACT.pdEdit = b => partyForm(b.dataset.k, (b.dataset.k === "cust" ? S.customers : S.suppliers).get(b.dataset.id));
ACT.pdDel = async b => { const r = await run(() => api("DELETE", `/api/${b.dataset.k === "cust" ? "customers" : "suppliers"}/${b.dataset.id}`), "Đã xóa"); if (r) { closeSheet(); refresh(true); } };
ACT.cashVoid = async b => {
  const r = await run(() => api("POST", `/api/cash/${b.dataset.id}/void`), "Đã hủy phiếu");
  if (!r) return; await refresh(true);
  if (b.dataset.back) { const [k, id] = b.dataset.back.split(":"); partyDetail(k, id); } else if (S.view === "cash") VIEWS.cash();
};
ACT.remind = b => {
  const c = S.customers.get(b.dataset.id), st = S.settings;
  copyText(`Chào ${c.name}, ${st.shopName || "cửa hàng"} xin phép nhắc khoản mua chịu hiện còn ${money(c.debt)}.${st.acct ? ` Có thể chuyển khoản: ${st.bankName || ""} ${st.acct} (${st.holder || ""}), nội dung "${c.code} tra no".` : ""} Cảm ơn ${c.name} nhiều!`, "Đã sao chép, dán vào Zalo để gửi");
};
ACT.debtQR = b => {
  const c = S.customers.get(b.dataset.id);
  openSheet(`<div class="sheet-h"><h2>Mã QR trả nợ · ${esc(c.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div style="display:grid;place-items:center;gap:8px"><div style="width:260px;background:#fff;padding:8px;border-radius:10px">${qrSvg(vietqr(c.debt, c.code + " tra no"))}</div>
    <b class="num" style="font-size:22px">${money(c.debt)}</b><div class="ps">${esc(S.settings.bankName || "")} · ${esc(S.settings.acct)} · ${esc(S.settings.holder || "")}</div>
    <div class="ps">Khách quét bằng app ngân hàng, số tiền đã điền sẵn. Khi nhận được tiền, nhớ bấm Ghi nhận trả nợ.</div></div>`);
};

/* ================= NHÀ CUNG CẤP ================= */
VIEWS.suppliers = function () {
  const all = [...S.suppliers.values()].sort((a, b) => (b.debt || 0) - (a.debt || 0) || a.name.localeCompare(b.name, "vi"));
  const total = all.reduce((a, c) => a + Math.max(0, c.debt || 0), 0);
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Nhà cung cấp</h1>${impBtn("suppliers")}<a class="btn sm" href="/api/export/suppliers" download>Xuất Excel</a><button class="btn pri sm" data-act="suppNew">+ Nhà cung cấp</button></div>
    <div class="stats" style="margin-bottom:12px"><div class="stat"><div class="k">Nhà cung cấp</div><div class="v">${all.length}</div></div><div class="stat bad"><div class="k">Mình đang nợ</div><div class="v">${money(total)}</div></div></div>
    ${all.length ? `<div class="tbl"><table><thead><tr><th>Mã</th><th>Tên</th><th>Điện thoại</th><th>Địa chỉ</th><th class="r">Đang nợ</th></tr></thead><tbody>${all.map(s => `<tr class="click" data-act="suppOpenD" data-id="${esc(s.id)}"><td class="num ps">${esc(s.code)}</td><td class="pn">${esc(s.name)}</td><td class="num">${esc(s.phone || "")}</td><td class="ps">${esc(s.addr || "")}</td><td class="r" style="color:${s.debt > 0 ? "var(--red)" : "inherit"}"><b>${fmt(s.debt)}</b></td></tr>`).join("")}</tbody></table></div>`
      : `<div class="empty"><b>Chưa có nhà cung cấp</b>Thêm đại lý, tổng kho bạn hay lấy hàng để theo dõi công nợ khi nhập hàng.</div>`}</div>`;
};

/* ================= SỔ QUỸ ================= */
S.cbPeriod = "7d";
const CATS_IN = ["Thu khác", "Tiền thừa cuối ngày", "Góp vốn"], CATS_OUT = ["Tiền điện", "Tiền nước", "Thuê mặt bằng", "Lương nhân viên", "Internet / điện thoại", "Vận chuyển", "Sửa chữa", "Rút tiền chi tiêu", "Chi khác"];
VIEWS.cash = async function () {
  const [a, b] = periodRange(S.cbPeriod);
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Sổ quỹ</h1><button class="btn sm" data-act="cbNew" data-k="thu">+ Phiếu thu</button><button class="btn sm pri" data-act="cbNew" data-k="chi">+ Phiếu chi</button></div>
    <div class="chips" style="margin-bottom:12px">${periodChips(S.cbPeriod, "cbPeriod")}</div><div id="cbBody"><div class="muted">Đang tải…</div></div></div>`;
  const rows = await run(() => api("GET", `/api/cashbook?from=${a}&to=${b}`)); if (!rows || S.view !== "cash") return;
  const sum = { thu_cash: 0, thu_bank: 0, chi_cash: 0, chi_bank: 0 };
  rows.filter(r => !r.void).forEach(r => { sum[r.kind + "_" + r.method] += r.amount; });
  $("#cbBody").innerHTML = `<div class="flows" style="margin-bottom:12px">
      <div><div class="k">Thu tiền mặt</div><div class="v" style="color:var(--green-text)">${money(sum.thu_cash)}</div></div><div><div class="k">Chi tiền mặt</div><div class="v" style="color:var(--red)">${money(sum.chi_cash)}</div></div>
      <div><div class="k">Chênh lệch tiền mặt</div><div class="v">${money(sum.thu_cash - sum.chi_cash)}</div></div>
      <div><div class="k">Thu chuyển khoản</div><div class="v" style="color:var(--green-text)">${money(sum.thu_bank)}</div></div><div><div class="k">Chi chuyển khoản</div><div class="v" style="color:var(--red)">${money(sum.chi_bank)}</div></div>
      <div><div class="k">Chênh lệch tài khoản</div><div class="v">${money(sum.thu_bank - sum.chi_bank)}</div></div></div>
    ${rows.length ? `<div class="tbl"><table><thead><tr><th>Thời gian</th><th>Mã</th><th>Loại</th><th>Nội dung</th><th>Hình thức</th><th class="r">Số tiền</th><th></th></tr></thead><tbody>${rows.map(r => `<tr class="${r.void ? "void" : ""}"><td class="num ps">${dt(r.time)}</td><td class="num ps">${esc(r.code || "")}</td><td>${esc(r.cat)}</td><td>${esc(r.party ? r.party + " · " : "")}${esc(r.note || "")}${r.auto ? ' <span class="pill mute">tự động</span>' : ""}</td><td class="ps">${r.method === "bank" ? "Chuyển khoản" : "Tiền mặt"}</td>
      <td class="r" style="color:${r.kind === "thu" ? "var(--green-text)" : "var(--red)"}"><b>${r.kind === "thu" ? "+" : "−"}${fmt(r.amount)}</b></td><td class="r">${r.id && !r.void && isOwner() ? `<button class="btn sm ghost" data-act="cashVoid" data-id="${esc(r.id)}" data-confirm="Hủy phiếu?">Hủy</button>` : ""}</td></tr>`).join("")}</tbody></table></div>` : `<div class="empty">Chưa có thu chi trong khoảng này.</div>`}`;
};
ACT.cbPeriod = b => { S.cbPeriod = b.dataset.v; VIEWS.cash(); };
ACT.cbNew = b => {
  const k = b.dataset.k, cats = k === "thu" ? CATS_IN : CATS_OUT;
  openSheet(`<div class="sheet-h"><h2>${k === "thu" ? "Phiếu thu" : "Phiếu chi"}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <form id="cbForm" class="grid2"><div class="field"><label for="cb-amt">Số tiền</label><input id="cb-amt" class="in num" inputmode="numeric" required></div>
    <div class="field"><label for="cb-cat">Khoản</label><select id="cb-cat" class="in">${cats.map(c => `<option>${c}</option>`).join("")}</select></div>
    <div class="field"><label for="cb-m">Hình thức</label><select id="cb-m" class="in"><option value="cash">Tiền mặt</option><option value="bank">Chuyển khoản</option></select></div>
    <div class="field"><label for="cb-note">Ghi chú</label><input id="cb-note" class="in"></div>
    <div class="row full"><span class="ps grow">Thu nợ khách / trả nợ nhà cung cấp: mở người đó trong Khách hàng hoặc Nhà cung cấp.</span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri">Lưu phiếu</button></div></form>`, { focus: "#cb-amt" });
  $("#cbForm").addEventListener("submit", async e => {
    e.preventDefault();
    const r = await run(() => api("POST", "/api/cash", { kind: k, amount: parseMoney($("#cb-amt").value), cat: $("#cb-cat").value, method: $("#cb-m").value, note: $("#cb-note").value }), "Đã lưu phiếu");
    if (r) { closeSheet(); VIEWS.cash(); }
  });
};

/* ================= BÁO CÁO ================= */
S.rpPeriod = "today"; S.rpCustom = null; S.rpOpen = null; S.rpTab = "sales";
VIEWS.report = async function () {
  const [a, b] = periodRange(S.rpPeriod, S.rpCustom);
  const tabs = [["ov", "Tổng quan", "report"], ["pnl", "Kết quả kinh doanh", "report"], ["close", "Chốt ca cuối ngày", "money"]].filter(x => can(x[2]) && (x[0] !== "pnl" || seeCost()));
  if (!tabs.some(x => x[0] === S.rpTab)) S.rpTab = "ov";
  const tabBar = `<div class="seg" role="group" aria-label="Loại báo cáo" style="margin-bottom:12px;width:fit-content">${tabs.map(([k, t]) => `<button type="button" data-act="rpTab" data-v="${k}" aria-pressed="${S.rpTab === k}">${t}</button>`).join("")}</div>`;
  if (S.rpTab === "close") return renderClosing(tabBar);
  if (S.rpTab === "pnl") return renderPnl(tabBar, a, b);
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Báo cáo</h1><a class="btn sm" href="/api/export/invoices?from=${a}&to=${b}" download>Xuất hóa đơn ra Excel</a>${S.rpPeriod === "today" || S.rpPeriod === "yday" ? `<button class="btn sm" data-act="eodPrint">${icon("print")} In báo cáo cuối ngày</button>` : ""}</div>
    ${tabs.length > 1 ? tabBar : ""}${periodBarHtml(a, b)}
    <div id="rpBody"><div class="muted">Đang tổng hợp…</div></div></div>`;
  const [rp, invs] = await Promise.all([api("GET", `/api/report?from=${a}&to=${b}`).catch(e => (toast(e.message, true), null)), api("GET", `/api/invoices?from=${a}&to=${b}`).catch(() => [])]);
  if (!rp || S.view !== "report") return;
  S.rpData = { rp, invs };
  const owner = seeCost();
  const max = Math.max(1, ...rp.series.map(s => s.rev));
  const nice = (() => { const p = Math.pow(10, Math.floor(Math.log10(max))); const m = max / p; return (m <= 1 ? 1 : m <= 2 ? 2 : m <= 5 ? 5 : 10) * p; })();
  const step = Math.max(1, Math.ceil(rp.series.length / 16));
  const F = rp.flow || {};
  const topMax = Math.max(1, ...rp.top.map(t => t.rev));
  $("#rpBody").innerHTML = `
    <div class="stats"><div class="stat main"><div class="k">Doanh thu</div><div class="v">${money(rp.rev)}</div></div>
      ${owner ? `<div class="stat good"><div class="k">Lãi gộp</div><div class="v">${money(rp.profit)}</div></div><div class="stat"><div class="k">Biên lãi</div><div class="v">${rp.rev ? (rp.profit / rp.rev * 100).toFixed(1) : 0}%</div></div>` : ""}
      <div class="stat"><div class="k">Số hóa đơn</div><div class="v">${fmt(rp.n)}</div></div><div class="stat"><div class="k">Trung bình / hóa đơn</div><div class="v">${money(rp.n ? rp.rev / rp.n : 0)}</div></div>
      ${rp.voids ? `<div class="stat bad"><div class="k">Hóa đơn đã hủy</div><div class="v">${rp.voids}</div></div>` : ""}${rp.returns ? `<div class="stat bad"><div class="k">Khách trả hàng (${rp.retN} phiếu)</div><div class="v">− ${money(rp.returns)}</div></div>` : ""}</div>${rp.returns ? `<div class="ps" style="margin-top:6px">Doanh thu và lãi ở trên đã trừ hàng khách trả.</div>` : ""}
    ${rp.imported ? `<div class="banner" style="margin-top:10px">Có ${rp.imported} hóa đơn nhập từ phần mềm cũ trong khoảng này (tính vào doanh thu, không tính vào tiền quỹ).</div>` : ""}
    <div class="chart" style="margin-top:14px"><div class="row" style="justify-content:space-between;margin-bottom:8px"><h3>${rp.hourly ? "Doanh thu theo giờ" : "Doanh thu theo ngày"}</h3><div class="legend"><span><i style="background:var(--green)"></i>Doanh thu</span>${owner ? '<span><i style="background:var(--tag)"></i>Lãi gộp</span>' : ""}<span class="num">Vạch trên cùng: ${money(nice)}</span></div></div>
      <div class="bars">${rp.series.map((s, i) => `<div class="bar ${i === rp.series.length - 1 ? "hl" : ""}" title="${esc(rp.hourly ? s.k : vnDate(s.k))}: ${money(s.rev)}${owner ? " · lãi " + money(s.profit) : ""} · ${s.n} HĐ"><i style="height:${(s.rev / nice * 100).toFixed(1)}%"></i>${owner ? `<i class="p" style="height:${(Math.max(0, s.profit) / nice * 100).toFixed(1)}%"></i>` : ""}</div>`).join("")}</div>
      <div class="xl">${rp.series.map((s, i) => `<span>${i % step === 0 ? (rp.hourly ? s.k : dm(s.k)) : ""}</span>`).join("")}</div></div>
    <div class="flows" style="margin-top:14px">
      <div><div class="k">Bán thu tiền mặt</div><div class="v">${money(rp.cash)}</div></div><div><div class="k">Bán chuyển khoản</div><div class="v">${money(rp.bank)}</div></div>
      <div><div class="k">Bán ghi nợ</div><div class="v" style="color:var(--red)">${money(rp.debt)}</div></div><div><div class="k">Khách trả nợ</div><div class="v">${money(F.debtCollected || 0)}</div></div>
      <div><div class="k">Tiền nhập hàng</div><div class="v">${money(F.buyTotal || 0)}</div></div><div><div class="k">Chi phí khác</div><div class="v">${money((F.chi_cash || 0) + (F.chi_bank || 0) - (F.suppPaid || 0))}</div></div>
      <div><div class="k">Tiền mặt ròng trong kỳ</div><div class="v">${money(F.netCash || 0)}</div></div><div><div class="k">Tài khoản ròng trong kỳ</div><div class="v">${money(F.netBank || 0)}</div></div></div>
    <div class="grid2" style="margin-top:18px;align-items:start">
      ${custRankHtml(rp, owner)}
      <div class="section"><h2>Hàng bán chạy</h2>${rp.top.length ? `<div class="tbl"><table><thead><tr><th>#</th><th>Mặt hàng</th><th class="r">Đã bán</th><th class="r">Doanh thu</th>${owner ? '<th class="r">Lãi</th>' : ""}</tr></thead><tbody>${rp.top.slice(0, 15).map((t, i) => `<tr><td class="ps num">${i + 1}</td><td><div class="pn">${esc(t.name)}</div><div class="hbar"><i style="width:${(t.rev / topMax * 100).toFixed(1)}%"></i></div></td><td class="r">${fq(t.qty)} ${esc(t.unit)}</td><td class="r"><b>${fmt(t.rev)}</b></td>${owner ? `<td class="r">${fmt(t.profit)}</td>` : ""}</tr>`).join("")}</tbody></table></div>` : `<div class="empty">Chưa có hàng bán ra.</div>`}</div>
      <div class="section"><h2>Theo nhóm hàng</h2>${rp.groups.length ? `<div class="tbl"><table><thead><tr><th>Nhóm</th><th class="r">Doanh thu</th>${owner ? '<th class="r">Lãi</th><th class="r">Biên</th>' : ""}</tr></thead><tbody>${rp.groups.map(g => `<tr><td class="pn">${esc(g.name)}</td><td class="r">${fmt(g.rev)}</td>${owner ? `<td class="r">${fmt(g.profit)}</td><td class="r ps">${g.rev ? (g.profit / g.rev * 100).toFixed(0) : 0}%</td>` : ""}</tr>`).join("")}</tbody></table></div>` : ""}
        ${(rp.staff || []).length > 1 ? `<h2 style="margin-top:12px">Theo thu ngân</h2><div class="tbl"><table><tbody>${rp.staff.map(s => `<tr><td>${esc(s.name)}</td><td class="r">${s.n} HĐ</td><td class="r"><b>${fmt(s.rev)}</b></td></tr>`).join("")}</tbody></table></div>` : ""}</div></div>
    <div class="section" style="margin-top:18px"><h2>Hóa đơn (${invs.length})</h2><div id="rpInv">${invListHtml(invs)}</div></div>`;
};
function invListHtml(invs) {
  if (!invs.length) return `<div class="empty">Chưa có hóa đơn.</div>`;
  return `<div class="tbl"><table><thead><tr><th>Mã</th><th>Thời gian</th><th>Khách</th><th>Mặt hàng</th><th class="r">Tổng</th><th>Thanh toán</th></tr></thead><tbody>${invs.slice(0, 300).map(v => {
    const open = S.rpOpen === v.id;
    const pay = v.void ? '<span class="pill mute">Đã hủy</span>' : [v.cash ? "TM" : "", v.bank ? '<span class="pill green">CK</span>' : "", v.debt ? '<span class="pill red">Nợ ' + fmt(v.debt) + "</span>" : ""].filter(Boolean).join(" ");
    return `<tr class="click ${v.void ? "void" : ""}" data-act="rpOpen" data-id="${esc(v.id)}"><td class="num">${esc(v.code)}</td><td class="num ps">${dt(v.time)}</td><td>${esc(v.custName || "Khách lẻ")}</td><td class="ps">${esc(v.lines.map(l => l.name + " ×" + fq(l.qty)).join(", ")).slice(0, 90)}</td><td class="r"><b>${fmt(v.total)}</b></td><td>${pay}${v.src === "import" ? ' <span class="pill mute">nhập</span>' : ""}</td></tr>
      ${open ? `<tr><td colspan="6" style="background:var(--sunk)"><div class="stack" style="gap:6px">${v.lines.map(l => `<div class="sumrow"><span>${esc(l.name)} · ${fq(l.qty)} ${esc(l.unit)} × ${fmt(l.price)}</span><span class="num">${fmt(l.qty * l.price)}</span></div>`).join("")}
        ${v.disc ? `<div class="sumrow ps"><span>Giảm giá</span><span>−${fmt(v.disc)}</span></div>` : ""}<div class="sumrow"><b>Tổng</b><b class="num">${money(v.total)}</b></div>
        <div class="ps">Thu ngân ${esc(v.staff)}${seeCost() && v.cost ? " · lãi gộp " + money(v.total - v.cost) : ""}${v.note ? " · " + esc(v.note) : ""}</div>
        <div class="row"><button class="btn sm" data-act="rpPrint" data-id="${esc(v.id)}">${icon("print")} In lại</button><button class="btn sm" data-act="rpCopy" data-id="${esc(v.id)}">Sao chép</button>${!v.void && isOwner() ? `<button class="btn sm danger" data-act="rpVoid" data-id="${esc(v.id)}" data-confirm="Bấm lần nữa: hủy hóa đơn, trả hàng về kho">Hủy hóa đơn</button>` : ""}</div></div></td></tr>` : ""}`;
  }).join("")}</tbody></table></div>`;
}
const findInv = id => (S.rpData?.invs || []).find(v => v.id === id);

/* ---------- xếp hạng khách hàng: lợi nhuận / doanh thu ---------- */
S.rpCustSort = null; S.rpCustAll = false;
function custRankHtml(rp, owner) {
  const all = (rp.custs || []).slice();
  if (!all.length) return `<div class="section"><h2>Khách hàng mang lại lợi nhuận</h2><div class="empty">Chưa có hóa đơn bán cho khách có tên trong kỳ này.</div></div>`;
  const by = owner ? (S.rpCustSort || "profit") : "rev";
  all.sort((a, b) => (b[by] - a[by]) || (b.rev - a.rev));
  const rows = S.rpCustAll ? all : all.slice(0, 20);
  const tot = all.reduce((t, c) => ({ rev: t.rev + c.rev, profit: t.profit + c.profit }), { rev: 0, profit: 0 });
  const head = owner
    ? `<div class="chips" style="margin:-4px 0 10px"><button class="chip" data-act="rpCustSort" data-v="profit" aria-pressed="${by === "profit"}">Theo lợi nhuận</button><button class="chip" data-act="rpCustSort" data-v="rev" aria-pressed="${by === "rev"}">Theo doanh thu</button></div>`
    : "";
  return `<div class="section"><h2>${owner ? "Khách hàng mang lại lợi nhuận" : "Xếp hạng khách hàng theo doanh số"}</h2>${head}
    <div class="tbl"><table><thead><tr><th>#</th><th>Khách hàng</th><th>Hạng</th><th class="r">Số đơn</th><th class="r">Doanh thu</th>${owner ? '<th class="r">Giá vốn</th><th class="r">Lợi nhuận</th><th class="r">Biên</th>' : '<th class="r">Tổng đã mua</th>'}</tr></thead><tbody>${rows.map((c, i) => {
      const m = c.rev ? c.profit / c.rev * 100 : 0;
      return `<tr><td class="ps num">${i + 1}</td><td class="pn">${esc(c.name)}</td><td>${c.tier ? `<span class="pill amber">${esc(c.tier)}</span>` : '<span class="ps">—</span>'}</td><td class="r">${fmt(c.n)}</td><td class="r">${owner ? fmt(c.rev) : `<b>${fmt(c.rev)}</b>`}</td>${owner ? `<td class="r ps">${fmt(c.cost)}</td><td class="r"><b style="${c.profit < 0 ? "color:var(--red)" : ""}">${fmt(c.profit)}</b></td><td class="r ps" style="${m < 8 ? "color:var(--red)" : ""}">${m.toFixed(1)}%</td>` : `<td class="r">${fmt(c.spend)}</td>`}</tr>`;
    }).join("")}</tbody>${owner ? `<tfoot><tr><td></td><td class="pn">Cộng ${all.length} khách</td><td></td><td></td><td class="r">${fmt(tot.rev)}</td><td></td><td class="r"><b>${fmt(tot.profit)}</b></td><td class="r ps">${tot.rev ? (tot.profit / tot.rev * 100).toFixed(1) : 0}%</td></tr></tfoot>` : ""}</table></div>
    ${all.length > 20 ? `<div class="row" style="margin-top:8px"><button class="btn sm" data-act="rpCustAll">${S.rpCustAll ? "Chỉ hiện 20 khách đầu" : "Hiện tất cả " + all.length + " khách"}</button></div>` : ""}
    ${owner ? `<p class="ps" style="margin:8px 0 0">Lợi nhuận = tiền khách trả (sau giảm giá) trừ giá vốn hàng đã bán. Biên đỏ là dưới 8%: bán nhiều nhưng lãi mỏng.</p>` : ""}</div>`;
}
function periodBarHtml(a, b) {
  return `<div class="row" style="margin-bottom:12px"><div class="chips grow" style="flex-wrap:wrap;overflow:visible">${periodChips(S.rpPeriod, "rpPeriod")}</div>
      <input type="date" id="rpFrom" class="in sm" style="width:auto" value="${a}"><span class="ps">đến</span><input type="date" id="rpTo" class="in sm" style="width:auto" value="${b}"></div>`;
}
ACT.rpTab = b => { S.rpTab = b.dataset.v; VIEWS.report(); };
/* ---------- kết quả kinh doanh (lãi/lỗ) ---------- */
async function renderPnl(tabBar, a, b) {
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Báo cáo</h1></div>${tabBar}${periodBarHtml(a, b)}<div id="rpBody"><div class="muted">Đang tổng hợp…</div></div></div>`;
  const p = await api("GET", `/api/pnl?from=${a}&to=${b}`).catch(e => (toast(e.message, true), null));
  if (!p || S.view !== "report" || S.rpTab !== "pnl") return;
  const line = (k, v, o = {}) => `<tr${o.bold ? ' style="font-weight:700;border-top:1px solid var(--line)"' : ""}><td${o.ind ? ' style="padding-left:22px"' : ""}>${k}</td><td class="r num" style="${o.color ? "color:" + o.color : ""}">${o.neg ? "− " : ""}${money(Math.abs(v))}</td></tr>`;
  const col = v => v >= 0 ? "var(--green-text)" : "var(--red)";
  $("#rpBody").innerHTML = `<div class="stats"><div class="stat main"><div class="k">Doanh thu thuần</div><div class="v">${money(p.revenue)}</div></div>
      <div class="stat good"><div class="k">Lãi gộp</div><div class="v">${money(p.gross)}</div></div>
      <div class="stat"><div class="k">Tổng chi phí</div><div class="v">${money(p.expenseT)}</div></div>
      <div class="stat ${p.net >= 0 ? "good" : "bad"}"><div class="k">Lợi nhuận ròng</div><div class="v">${p.net < 0 ? "− " : ""}${money(Math.abs(p.net))}</div></div></div>
    <div class="grid2" style="margin-top:14px;align-items:start"><div class="section"><h2>Báo cáo lãi lỗ</h2><div class="tbl"><table><tbody>
      ${line("Doanh thu bán hàng", p.revenue + p.returns)}${p.returns ? line("Khách trả hàng", p.returns, { neg: true, ind: true }) : ""}${line("Doanh thu thuần", p.revenue, { bold: true })}
      ${line("Giá vốn hàng bán", p.cogs, { neg: true })}${line("LÃI GỘP", p.gross, { bold: true, color: col(p.gross), neg: p.gross < 0 })}
      ${p.loss ? line("Hàng hủy, hết hạn, hao hụt kho", p.loss, { neg: true }) : ""}${p.diff ? line(p.diff > 0 ? "Tiền két thừa khi chốt ca" : "Tiền két thiếu khi chốt ca", p.diff, { neg: p.diff < 0 }) : ""}
      ${p.otherIn.map(x => line(esc(x.cat), x.amount, { ind: true })).join("")}${p.otherInT ? line("Thu nhập khác", p.otherInT, { bold: true }) : ""}
      ${p.expenses.map(x => line(esc(x.cat), x.amount, { neg: true, ind: true })).join("")}${p.expenseT ? line("Tổng chi phí", p.expenseT, { neg: true, bold: true }) : `<tr><td class="ps" colspan="2">Chưa có khoản chi nào trong sổ quỹ. Ghi tiền điện, thuê mặt bằng, lương… ở mục Sổ quỹ để tính lãi ròng đúng.</td></tr>`}
      ${line("LỢI NHUẬN RÒNG", p.net, { bold: true, color: col(p.net), neg: p.net < 0 })}</tbody></table></div>
      <p class="ps" style="margin-top:8px">Không tính vào lãi lỗ: tiền nhập hàng (hàng chưa bán nằm trong kho), thu nợ khách, trả nợ NCC, hoàn tiền trả hàng${p.draw ? `, chủ rút tiền chi tiêu ${money(p.draw)}` : ""}${p.capital ? `, góp vốn ${money(p.capital)}` : ""}.</p></div>
      <div class="section"><h2>Theo tháng</h2>${p.months.length ? `<div class="tbl"><table><thead><tr><th>Tháng</th><th class="r">Doanh thu</th><th class="r">Lãi gộp</th><th class="r">Lãi ròng</th></tr></thead><tbody>${p.months.map(m => `<tr><td>${esc(m.month.slice(5) + "/" + m.month.slice(0, 4))}</td><td class="r num">${fmt(m.rev)}</td><td class="r num">${fmt(m.gross)}</td><td class="r num" style="color:${col(m.net)}"><b>${fmt(m.net)}</b></td></tr>`).join("")}</tbody></table></div><div class="ps" style="margin-top:6px">Lãi ròng theo tháng = lãi gộp − chi phí + thu nhập khác (chưa gồm hao hụt kho và chênh lệch két).</div>` : `<div class="ps">Chưa có dữ liệu.</div>`}</div></div>`;
}
/* ---------- chốt ca cuối ngày ---------- */
S.clDate = "";
async function renderClosing(tabBar) {
  if (!S.clDate) S.clDate = dkey();
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Báo cáo</h1></div>${tabBar}<div id="rpBody"><div class="muted">Đang tải…</div></div></div>`;
  const d = await api("GET", `/api/closing?date=${S.clDate}`).catch(e => (toast(e.message, true), null));
  if (!d || S.view !== "report" || S.rpTab !== "close") return;
  const c = d.preview, ex = c.existing;
  S.clData = d;
  const row = (k, v, o = {}) => `<div class="row" style="justify-content:space-between;${o.bold ? "border-top:1px solid var(--line);padding-top:6px;" : ""}"><span>${k}</span><b class="num" style="${o.color ? "color:" + o.color : ""}">${money(v)}</b></div>`;
  $("#rpBody").innerHTML = `<div class="grid2" style="align-items:start"><div class="card stack" style="padding:14px">
      <div class="row"><label for="clDate" class="ps">Ngày chốt</label><input type="date" id="clDate" class="in sm" style="width:auto" value="${c.date}" data-ch="clDate"><span class="grow"></span>${ex ? `<span class="chip" style="cursor:default">Đã chốt lúc ${dt(ex.time)}</span>` : ""}</div>
      <div class="ps">${c.n} hóa đơn · doanh thu ${money(c.rev)} · chuyển khoản ${money(c.bank)} · ghi nợ ${money(c.debt)}</div>
      ${row("Tiền để lại két từ lần chốt trước" + (c.openFrom ? " (" + vnDate(c.openFrom) + ")" : " (chưa có)"), c.opening)}
      ${row("+ Bán hàng thu tiền mặt", c.sales)}${row("+ Thu khác, thu nợ khách", c.thu)}${row("− Chi (sổ quỹ, hoàn khách trả hàng)", c.chi)}${row("− Tiền mặt nhập hàng", c.buy)}
      ${row("Tiền mặt lẽ ra có trong két", c.expected, { bold: true })}
      <div class="field"><label for="clCount">Tiền mặt đếm thực tế</label><input id="clCount" class="in num r" inputmode="numeric" placeholder="${fmt(c.expected)}" value="${ex ? fmt(ex.counted) : ""}"></div>
      <div id="clDiff"></div>
      <div class="field"><label for="clKeep">Để lại két cho ngày mai (tiền lẻ)</label><input id="clKeep" class="in num r" inputmode="numeric" value="${ex ? fmt(ex.kept) : ""}"><div class="ps">Phần còn lại là tiền mang đi cất / nộp ngân hàng.</div></div>
      <div class="field"><label for="clNote">Ghi chú</label><input id="clNote" class="in" value="${esc(ex ? ex.note : "")}" placeholder="vd thiếu 5k tiền lẻ"></div>
      <div class="row"><span class="grow"></span><button class="btn" data-act="clPrint">${icon("print")} In phiếu chốt ca</button><button class="btn pri" data-act="clSave">${ex ? "Chốt lại" : "Chốt ca"}</button></div></div>
    <div class="section"><h2>Các lần chốt gần đây</h2>${d.list.length ? `<div class="tbl"><table><thead><tr><th>Ngày</th><th class="r">Đếm</th><th class="r">Chênh</th><th class="r">Mang đi</th></tr></thead><tbody>${d.list.map(x => `<tr><td>${esc(vnDate(x.date))}<div class="ps">${esc(x.staff || "")}${x.note ? " · " + esc(x.note) : ""}</div></td><td class="r num">${fmt(x.counted)}</td><td class="r num" style="color:${x.diff < 0 ? "var(--red)" : x.diff > 0 ? "var(--green-text)" : "inherit"}">${x.diff ? (x.diff > 0 ? "+" : "") + fmt(x.diff) : "0"}</td><td class="r num">${fmt(x.take)}</td></tr>`).join("")}</tbody></table></div>` : `<div class="ps">Chưa có lần chốt nào.</div>`}</div></div>`;
  clDiff();
}
function clDiff() {
  const c = S.clData.preview, el = $("#clCount"), box = $("#clDiff"); if (!el || !box) return;
  if (!el.value.trim()) { box.innerHTML = ""; return; }
  const diff = parseMoney(el.value) - c.expected;
  box.innerHTML = `<div class="row" style="justify-content:space-between"><span>Chênh lệch</span><b class="num" style="font-size:18px;color:${diff === 0 ? "var(--green-text)" : "var(--red)"}">${diff === 0 ? "Khớp" : (diff > 0 ? "Thừa " : "Thiếu ") + money(Math.abs(diff))}</b></div>`;
}
ON_INPUT.clCount = () => clDiff();
ON_CHANGE.clDate = el => { S.clDate = el.value || dkey(); VIEWS.report(); };
function clForm() { return { date: S.clData.preview.date, counted: parseMoney($("#clCount").value), kept: parseMoney($("#clKeep").value), note: $("#clNote").value }; }
ACT.clSave = async () => {
  if (!$("#clCount").value.trim()) return toast("Nhập số tiền mặt đã đếm", true);
  const f = clForm(), exp = S.clData.preview.expected, diff = f.counted - exp;
  const r = await run(() => api("POST", "/api/closing", f), diff === 0 ? "Đã chốt ca, tiền khớp" : `Đã chốt ca, ${diff > 0 ? "thừa" : "thiếu"} ${money(Math.abs(diff))}`);
  if (r) { S.clLast = r; VIEWS.report(); }
};
ACT.clPrint = () => {
  const c = S.clData.preview, f = clForm(), st = S.settings, diff = f.counted - c.expected;
  const rows = (k, v, bold) => `<tr><td>${k}</td><td class="r">${bold ? "<b>" : ""}${fmt(v)}${bold ? "</b>" : ""}</td></tr>`;
  printHtml(`<div class="rc ${st.paper === "58" ? "p58" : st.paper === "a5" ? "a5" : ""}"><div class="c shop">${esc(st.shopName || "")}</div><hr><div class="c big">CHỐT CA CUỐI NGÀY</div><div class="c">${vnDate(c.date)}</div><hr>
    <table>${rows("Số hóa đơn", c.n)}${rows("Doanh thu", c.rev, true)}${rows("Chuyển khoản", c.bank)}${rows("Ghi nợ", c.debt)}</table><hr>
    <table>${rows("Tiền đầu ca (để lại hôm trước)", c.opening)}${rows("+ Bán tiền mặt", c.sales)}${rows("+ Thu khác", c.thu)}${rows("− Chi", c.chi)}${rows("− Nhập hàng tiền mặt", c.buy)}${rows("TIỀN LẼ RA CÓ", c.expected, true)}${f.counted ? rows("Tiền đếm thực tế", f.counted, true) + `<tr><td>Chênh lệch</td><td class="r"><b>${diff === 0 ? "Khớp" : (diff > 0 ? "+" : "") + fmt(diff)}</b></td></tr>` + rows("Để lại két", f.kept) + rows("Mang đi", f.counted - f.kept) : ""}</table>
    ${f.note ? `<div>Ghi chú: ${esc(f.note)}</div>` : ""}<hr><div>Người chốt: ${esc((S.me || {}).name || "")}</div><div>Ký tên: ______________</div><div class="c">In lúc ${dt(Date.now())}</div></div>`);
};
ACT.rpCustSort = b => { S.rpCustSort = b.dataset.v; VIEWS.report(); };
ACT.rpCustAll = () => { S.rpCustAll = !S.rpCustAll; VIEWS.report(); };
ACT.rpPeriod = b => { S.rpPeriod = b.dataset.v; S.rpOpen = null; VIEWS.report(); };
ON_CHANGE.rpFrom = ON_CHANGE.rpTo = () => { const a = $("#rpFrom").value, b = $("#rpTo").value; if (a && b) { S.rpPeriod = "custom"; S.rpCustom = [a, b]; VIEWS.report(); } };
ACT.rpOpen = b => { S.rpOpen = S.rpOpen === b.dataset.id ? null : b.dataset.id; $("#rpInv").innerHTML = invListHtml(S.rpData.invs); };
ACT.rpPrint = b => { const v = findInv(b.dataset.id); if (v) printHtml(receiptHtml(v)); };
ACT.rpCopy = b => { const v = findInv(b.dataset.id); if (v) copyText(receiptText(v), "Đã sao chép hóa đơn"); };
ACT.rpVoid = async b => { if (await run(() => api("POST", `/api/invoices/${b.dataset.id}/void`), "Đã hủy hóa đơn, hàng đã trả về kho")) { await refresh(true); VIEWS.report(); } };
ACT.eodPrint = () => {
  const d = S.rpData; if (!d) return;
  const rp = d.rp, st = S.settings, F = rp.flow || {};
  const rows = (k, v, bold) => `<tr><td>${k}</td><td class="r">${bold ? "<b>" : ""}${fmt(v)}${bold ? "</b>" : ""}</td></tr>`;
  printHtml(`<div class="rc ${st.paper === "58" ? "p58" : st.paper === "a5" ? "a5" : ""}"><div class="c shop">${esc(st.shopName || "")}</div><hr><div class="c big">BÁO CÁO CUỐI NGÀY</div><div class="c">${vnDate(rp.from)}${rp.to !== rp.from ? " – " + vnDate(rp.to) : ""}</div><hr>
    <table>${rows("Số hóa đơn", rp.n)}${rows("Doanh thu", rp.rev, true)}${rows("Giảm giá", rp.disc)}${rows("Tiền mặt", rp.cash)}${rows("Chuyển khoản", rp.bank)}${rows("Ghi nợ", rp.debt)}${rows("Thu nợ khách", F.debtCollected || 0)}${rows("Chi nhập hàng (tiền mặt)", F.buyCash || 0)}${rows("Chi khác (tiền mặt)", F.chi_cash || 0)}${rows("TIỀN MẶT RÒNG", F.netCash || 0, true)}${seeCost() ? rows("Lãi gộp", rp.profit, true) : ""}</table>
    <hr><div><b>Bán chạy</b></div><table>${rp.top.slice(0, 10).map(t => `<tr><td>${esc(t.name)}</td><td class="r">${fq(t.qty)}</td></tr>`).join("")}</table><hr><div class="c">In lúc ${dt(Date.now())}</div></div>`);
};

/* ================= COMBO ================= */
S.cbDays = 90; S.cbData = null;
VIEWS.combos = async function () {
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Combo & khuyến mãi</h1>
      <label class="ps" for="cbDays">Dựa trên đơn bán</label><select id="cbDays" class="in" style="width:auto">${[30, 60, 90, 180, 365].map(d => `<option value="${d}" ${d === S.cbDays ? "selected" : ""}>${d} ngày gần nhất</option>`).join("")}</select>
      <button class="btn sm pri" data-act="newCombo">+ Tạo combo</button></div>
    <p class="muted" style="margin:-6px 0 14px;max-width:760px">Gợi ý được tính từ chính hóa đơn của tiệm: món nào hay được mua cùng nhau, hàng nào sắp hết hạn cần xả, hàng nào lâu không bán được. Giá đề xuất luôn giữ lãi tối thiểu ${S.settings.minMargin || 10}% trên giá vốn (đổi trong Cài đặt).</p>
    <div id="promoBox"></div>
    <div id="cbBody"><div class="muted">Đang phân tích…</div></div></div>`;
  promoRender();
  const d = await run(() => api("GET", "/api/combos?days=" + S.cbDays)); if (!d || S.view !== "combos") return;
  S.cbData = d;
  const card = (c, i, sec) => `<div class="combo-card">
    <div class="row"><span class="pill ${sec === "expiry" ? "red" : sec === "slow" ? "amber" : "green"}">${sec === "expiry" ? "Xả hàng cận date" : sec === "slow" ? "Đẩy hàng bán chậm" : "Hay mua cùng nhau"}</span></div>
    <div class="items">${c.items.map((it, j) => `<span>${fq(it.qty)} × ${esc(c.names[j])}</span>`).join("")}</div>
    <div class="why">${esc(c.reason)}</div>
    <div class="price"><span class="tag">${fmt(c.price)}</span><span class="old num">${fmt(c.retail)}</span><span class="ps">khách tiết kiệm ${money(c.save)}</span></div>
    ${c.cost ? `<div class="ps">Giá vốn ${money(c.cost)} · lãi ${money(c.price - c.cost)} / combo (${c.margin}%)</div>` : ""}
    <div class="row"><button class="btn sm pri" data-act="cbCreate" data-s="${sec}" data-i="${i}">Tạo combo này</button><button class="btn sm" data-act="cbAd" data-s="${sec}" data-i="${i}">Câu quảng cáo Zalo</button></div></div>`;
  const sect = (t, list, key, empty) => `<div class="section"><h2>${t}</h2>${list.length ? `<div class="combos">${list.map((c, i) => card(c, i, key)).join("")}</div>` : `<div class="empty">${empty}</div>`}</div>`;
  $("#cbBody").innerHTML = (d.baskets < 30 ? `<div class="banner" style="margin-bottom:14px">Mới có ${d.baskets} hóa đơn trong ${d.days} ngày, gợi ý “mua cùng nhau” sẽ chính xác hơn khi có nhiều đơn. Có thể nhập lịch sử hóa đơn từ KiotViet/MISA ở mục Nhập từ Excel.</div>` : "") +
    sect("Xả hàng sắp hết hạn", d.expiry || [], "expiry", "Không có lô hàng nào sắp hết hạn. Nhập hạn sử dụng khi tạo phiếu nhập để phần mềm theo dõi.") +
    sect("Hay được mua cùng nhau", d.pairs || [], "pairs", "Chưa đủ dữ liệu để tìm cặp hàng hay mua chung.") +
    sect("Đẩy hàng bán chậm", d.slow || [], "slow", "Không có hàng tồn lâu không bán.") +
    `<div class="section"><h2>Combo đang bán</h2>${(d.current || []).length ? `<div class="tbl"><table><thead><tr><th>Combo</th><th class="r">Giá</th><th class="r">Giá vốn</th><th class="r">Đã bán (${d.days} ngày)</th><th class="r">Có thể bán</th><th></th></tr></thead><tbody>${d.current.map(c => `<tr class="click" data-act="editProduct" data-id="${esc(c.id)}"><td class="pn">${esc(c.name)}${c.hidden ? ' <span class="pill mute">ngừng</span>' : ""}</td><td class="r">${fmt(c.price)}</td><td class="r">${fmt(c.cost)}</td><td class="r">${fq(c.sold)}</td><td class="r">${fq(c.stock)}</td><td class="r ps">Sửa</td></tr>`).join("")}</tbody></table></div>` : `<div class="empty">Chưa có combo nào.</div>`}</div>`;
};
ON_CHANGE.cbDays = el => { S.cbDays = +el.value; VIEWS.combos(); };
const cbGet = b => (S.cbData[b.dataset.s] || [])[+b.dataset.i];
function comboName(c) {
  const first = w => w.split(" ").slice(0, 3).join(" ");
  return c.kind === "expiry" ? "Combo xả " + first(c.names[0]) : "Combo " + c.names.map(first).join(" + ");
}
ACT.cbCreate = b => {
  const c = cbGet(b); if (!c) return;
  openProductForm({ id: "", name: comboName(c), unit: "combo", group: "Combo", price: c.price, isCombo: true, items: c.items.map(i => ({ pid: i.pid, qty: i.qty })), units: [], lots: [], _new: true });
};
ACT.cbAd = b => {
  const c = cbGet(b); if (!c) return;
  const st = S.settings;
  const lines = c.items.map((it, j) => `• ${fq(it.qty)} ${c.names[j]}`).join("\n");
  const hook = c.kind === "expiry" ? "XẢ HÀNG GIÁ TỐT, số lượng có hạn" : c.kind === "slow" ? "ƯU ĐÃI TRONG TUẦN" : "COMBO TIẾT KIỆM";
  copyText(`${hook} tại ${st.shopName || "tiệm"}!\n${lines}\nChỉ ${money(c.price)} (giá lẻ ${money(c.retail)}, tiết kiệm ${money(c.save)}).\n${st.addr ? "Địa chỉ: " + st.addr + "\n" : ""}${st.phone ? "Đặt hàng: " + st.phone + " hoặc nhắn ngay trong nhóm." : "Nhắn ngay trong nhóm để đặt."}`, "Đã sao chép câu quảng cáo, dán vào nhóm Zalo");
};

/* ================= NHẬP DỮ LIỆU ================= */
S.imp = null; S.impKind = "auto"; S.impRes = null;
ACT.impGo = b => { S.impKind = b.dataset.k || "auto"; S.imp = null; S.impRes = null; go("import"); };
VIEWS.import = function () {
  const P = S.imp;
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Nhập từ Excel: KiotViet, MISA</h1><a class="btn sm" href="/api/export/template" download>Tải file Excel mẫu</a></div>
    ${S.impRes ? impResultHtml() : ""}
    ${!P ? `<div class="stack">
      <label class="drop" id="drop" for="impFile"><b style="font-size:16px">Kéo thả file vào đây, hoặc bấm để chọn file</b><span class="muted">Nhận file Excel .xlsx và .xls (cả đời cũ 97–2003), .csv, file xuất từ KiotViet, MISA SME/AMIS, MISA eShop</span>
        <input type="file" id="impFile" accept=".xlsx,.xls,.csv,.htm,.html,.txt" hidden></label>
      <div class="row"><label for="impKind" class="lbl">Loại dữ liệu</label><select id="impKind" class="in" style="width:auto">${[["auto", "Tự nhận diện"], ["products", "Hàng hóa"], ["customers", "Khách hàng"], ["suppliers", "Nhà cung cấp"], ["invoices", "Lịch sử hóa đơn bán (chi tiết)"]].map(([k, t]) => `<option value="${k}" ${k === S.impKind ? "selected" : ""}>${t}</option>`).join("")}</select></div>
      <div class="steps">
        <div class="card"><h3>Từ KiotViet</h3><ol><li><b>Hàng hóa</b> → Danh sách hàng hóa → nút <b>Xuất file</b> (lấy cả đơn vị quy đổi, mã vạch, vị trí)</li><li><b>Khách hàng</b> / <b>Nhà cung cấp</b> → Xuất file (lấy cả nợ hiện tại)</li><li><b>Hóa đơn</b> → Xuất file <b>chi tiết</b> để có lịch sử bán cho báo cáo và gợi ý combo</li></ol></div>
        <div class="card"><h3>Từ MISA SME / AMIS kế toán</h3><ol><li>Danh mục → <b>Vật tư hàng hóa</b> → Tiện ích → <b>Xuất khẩu</b> ra Excel, tích chọn Mã, Tên, Nhóm VTHH, Đơn vị tính chính, Số lượng tồn, Giá trị tồn, Đơn giá bán</li><li>Danh mục Khách hàng / Nhà cung cấp → Xuất khẩu</li><li>File có dòng tiêu đề “DANH SÁCH…” ở trên vẫn đọc được</li></ol></div>
        <div class="card"><h3>Từ MISA eShop, Sổ Tạp Hóa V1.0, file tự làm</h3><ol><li>MISA eShop: Hàng hóa → Xuất khẩu</li><li>Sổ Tạp Hóa V1.0: Kho hàng → Xuất CSV</li><li>File tự làm: cần một dòng tiêu đề như “Tên hàng, ĐVT, Giá bán, Giá vốn, Tồn kho, Mã vạch, HSD”</li></ol></div></div>
      <p class="ps" style="margin:0">Trước khi nhập, phần mềm tự sao lưu dữ liệu hiện tại. Nhập lại file nhiều lần không bị trùng: hàng trùng mã sẽ được cập nhật.</p></div>` : impPreviewHtml()}
  </div>`;
  const drop = $("#drop");
  if (drop) {
    drop.addEventListener("dragover", e => { e.preventDefault(); drop.classList.add("over"); });
    drop.addEventListener("dragleave", () => drop.classList.remove("over"));
    drop.addEventListener("drop", e => { e.preventDefault(); drop.classList.remove("over"); if (e.dataTransfer.files[0]) impUpload(e.dataTransfer.files[0]); });
  }
};
ON_CHANGE.impKind = el => { S.impKind = el.value; };
ON_CHANGE.impFile = el => { if (el.files[0]) impUpload(el.files[0]); };
async function impUpload(file) {
  const fd = new FormData(); fd.append("file", file); fd.append("kind", S.impKind);
  toast("Đang đọc file " + file.name + "…");
  const p = await run(() => api("POST", "/api/import/preview", fd));
  if (p) { S.imp = p; S.imp.mode = "upsert"; S.impRes = null; VIEWS.import(); }
}
function impPreviewHtml() {
  const P = S.imp, used = new Set(Object.values(P.mapping));
  const colOpts = sel => `<option value="-1">— bỏ qua —</option>` + P.columns.map((c, i) => `<option value="${i}" ${i === sel ? "selected" : ""}>${esc(c || "(cột " + (i + 1) + ")")}</option>`).join("");
  return `<div class="stack">
    <div class="card row"><div class="grow"><div class="lbl">File</div><b>${esc(P.fileName)}</b> · ${fmt(P.rows)} dòng dữ liệu</div>
      <div><div class="lbl">Nhận ra từ</div><span class="pill green">${esc(P.source)}</span></div>
      <div class="field"><label for="impK2">Loại</label><select id="impK2" class="in sm">${[["products", "Hàng hóa"], ["customers", "Khách hàng"], ["suppliers", "Nhà cung cấp"], ["invoices", "Lịch sử hóa đơn"]].map(([k, t]) => `<option value="${k}" ${k === P.kind ? "selected" : ""}>${t}</option>`).join("")}</select></div>
      ${P.sheets.length > 1 ? `<div class="field"><label for="impSheet">Trang tính</label><select id="impSheet" class="in sm">${P.sheets.map((s, i) => `<option value="${i}" ${i === P.sheet ? "selected" : ""}>${esc(s)}</option>`).join("")}</select></div>` : ""}
      <div class="field"><label for="impHeader">Dòng tiêu đề</label><input id="impHeader" class="in sm num" style="width:70px" type="number" min="1" value="${P.header + 1}"></div>
      <button class="btn sm ghost" data-act="impCancel">Chọn file khác</button></div>
    ${(P.notes || []).map(n => `<div class="banner">${esc(n)}</div>`).join("")}
    <div class="grid2" style="align-items:start">
      <div class="card"><h3 style="margin-bottom:8px">Ghép cột</h3><div class="tbl maptbl" style="border:0"><table><tbody>${P.fields.map(f => `<tr><td>${esc(f.label)}${f.need ? ' <span style="color:var(--red)">*</span>' : ""}</td><td><select class="in sm" data-ch="impMap" data-f="${f.key}">${colOpts(P.mapping[f.key] ?? -1)}</select></td></tr>`).join("")}</tbody></table></div></div>
      <div class="card stack"><h3>Cách nhập</h3>
        <label class="check"><input type="radio" name="impMode" value="upsert" data-ch="impMode" ${P.mode === "upsert" ? "checked" : ""}> Thêm mới, và cập nhật nếu đã có (trùng mã / mã vạch / tên)</label>
        <label class="check"><input type="radio" name="impMode" value="skip" data-ch="impMode" ${P.mode === "skip" ? "checked" : ""}> Chỉ thêm mới, bỏ qua dòng đã có</label>
        ${P.kind === "products" ? `<label class="check"><input type="radio" name="impMode" value="keepStock" data-ch="impMode" ${P.mode === "keepStock" ? "checked" : ""}> Cập nhật tên, giá… nhưng giữ nguyên tồn kho hiện tại</label>` : ""}
        <button class="btn pri lg" data-act="impCommit">Nhập ${fmt(P.rows)} dòng</button></div></div>
    <div class="section"><h3>Xem trước ${P.sample.length} dòng đầu</h3><div class="tbl"><table><thead><tr>${P.columns.map((c, i) => `<th class="${used.has(i) ? "colhit" : ""}">${esc(c)}</th>`).join("")}</tr></thead><tbody>${P.sample.map(r => `<tr>${r.map((v, i) => `<td class="${used.has(i) ? "" : "ps"}">${esc(v)}</td>`).join("")}</tr>`).join("")}</tbody></table></div></div></div>`;
}
ON_CHANGE.impMap = el => { const v = +el.value; if (v < 0) delete S.imp.mapping[el.dataset.f]; else S.imp.mapping[el.dataset.f] = v; VIEWS.import(); };
ON_CHANGE.impMode = el => { S.imp.mode = el.value; };
async function impReconf(o) { const p = await run(() => api("POST", "/api/import/reconfigure", Object.assign({ token: S.imp.token, sheet: S.imp.sheet, header: S.imp.header, kind: "" }, o))); if (p) { p.fileName = S.imp.fileName; p.mode = S.imp.mode; S.imp = p; VIEWS.import(); } }
ON_CHANGE.impK2 = el => impReconf({ kind: el.value });
ON_CHANGE.impSheet = el => impReconf({ sheet: +el.value, header: 0 });
ON_CHANGE.impHeader = el => impReconf({ header: Math.max(0, (+el.value || 1) - 1) });
ACT.impCancel = () => { S.imp = null; VIEWS.import(); };
ACT.impCommit = async b => {
  b.disabled = true;
  const r = await run(() => api("POST", "/api/import/commit", { token: S.imp.token, mapping: S.imp.mapping, mode: S.imp.mode }));
  b.disabled = false;
  if (r) { S.impRes = Object.assign(r, { kind: S.imp.kind, file: S.imp.fileName }); S.imp = null; await refresh(true); VIEWS.import(); }
};
function impResultHtml() {
  const r = S.impRes, go = { products: "products", customers: "customers", suppliers: "suppliers", invoices: "report" }[r.kind];
  return `<div class="banner green" style="margin-bottom:14px"><span class="grow">Đã nhập <b>${esc(r.file)}</b>: thêm mới <b>${r.added}</b>, cập nhật <b>${r.updated}</b>${r.units ? `, gộp <b>${r.units}</b> đơn vị quy đổi` : ""}${r.skipped ? `, bỏ qua ${r.skipped} dòng` : ""}.</span><button class="btn sm" data-act="go" data-v="${go}">Xem kết quả</button></div>
    ${(r.errors || []).length ? `<div class="banner red" style="margin-bottom:14px;display:block">${r.errors.map(esc).join("<br>")}</div>` : ""}`;
}

/* ================= CÀI ĐẶT ================= */
VIEWS.settings = function () {
  const st = S.settings, own = isAdmin();
  const lan = S.lan[0] || "";
  $("#main").innerHTML = `<div class="page" style="max-width:980px"><div class="head"><h1>Cài đặt</h1><span class="ps">Digital Shop ${esc(S.appVersion || "")}</span></div>
  ${own ? `<div class="card stack" style="margin-bottom:14px"><h2>Bản quyền</h2><div id="licBox" class="stack"><div class="ps">Đang tải…</div></div></div>
  <form id="setForm" class="stack">
    <div class="card stack"><h2>Cửa hàng</h2><div class="grid2">
      <div class="field full"><label for="st-name">Tên cửa hàng (in trên hóa đơn)</label><input id="st-name" class="in" value="${esc(st.shopName || "")}"></div>
      <div class="field"><label for="st-phone">Điện thoại</label><input id="st-phone" class="in num" value="${esc(st.phone || "")}"></div>
      <div class="field"><label for="st-addr">Địa chỉ</label><input id="st-addr" class="in" value="${esc(st.addr || "")}"></div></div></div>
    <div class="card stack"><h2>Nhận chuyển khoản (VietQR)</h2><div class="grid2" style="align-items:start"><div class="stack">
      <div class="field"><label for="st-bank">Ngân hàng</label><select id="st-bank" class="in"><option value="">— chọn ngân hàng —</option>${BANKS.map(([bin, n]) => `<option value="${bin}" ${bin === st.bankBin ? "selected" : ""}>${n}</option>`).join("")}</select></div>
      <div class="field"><label for="st-acct">Số tài khoản</label><input id="st-acct" class="in num" value="${esc(st.acct || "")}"></div>
      <div class="field"><label for="st-holder">Chủ tài khoản</label><input id="st-holder" class="in" value="${esc(st.holder || "")}" placeholder="VIẾT HOA KHÔNG DẤU"></div></div>
      <div id="qrPrev" style="display:grid;place-items:center;gap:6px">${st.acct ? `<div style="width:180px;background:#fff;padding:6px;border-radius:8px">${qrSvg(vietqr(10000, "THU QR"))}</div><span class="ps">Mã thử 10.000đ: quét bằng app ngân hàng để kiểm tra đúng tên</span>` : '<span class="ps">Nhập ngân hàng và số tài khoản để hiện mã QR khi khách chọn chuyển khoản.</span>'}</div></div></div>
    <div class="card stack"><h2>In hóa đơn</h2><div class="grid2">
      <div class="field"><label for="st-paper">Khổ giấy</label><select id="st-paper" class="in"><option value="80" ${st.paper === "80" ? "selected" : ""}>K80 (máy in nhiệt 80mm)</option><option value="58" ${st.paper === "58" ? "selected" : ""}>K58 (máy in nhiệt 58mm)</option><option value="a5" ${st.paper === "a5" ? "selected" : ""}>A5 (máy in thường)</option></select></div>
      <div class="field"><label for="st-feed">Số dòng trống cuối phiếu (để máy cắt đúng chỗ)</label><input id="st-feed" class="in num" type="number" min="1" max="30" value="${st.printFeed || 8}"></div>
      <div class="field"><label for="st-footer">Lời chào cuối hóa đơn</label><input id="st-footer" class="in" value="${esc(st.footer || "")}"></div>
      <label class="check full"><input type="checkbox" id="st-auto" ${st.autoPrint ? "checked" : ""}> Tự in hóa đơn sau mỗi lần thanh toán</label></div>
      <div><button type="button" class="btn sm" data-act="testPrint">${icon("print")} In thử</button> <span class="ps">Lần đầu in, chọn đúng máy in nhiệt, lề “Không”, tắt “Đầu trang và chân trang”.</span></div></div>
    <div class="card stack"><h2>Bán hàng & kho</h2><div class="grid3">
      <div class="field"><label for="st-warn">Báo cận date trước (ngày)</label><input id="st-warn" class="in num" type="number" min="1" value="${st.expWarn || 30}"></div>
      <div class="field"><label for="st-danger">Báo đỏ trước (ngày)</label><input id="st-danger" class="in num" type="number" min="1" value="${st.expDanger || 7}"></div>
      <div class="field"><label for="st-margin">Lãi tối thiểu khi gợi ý combo (% trên vốn)</label><input id="st-margin" class="in num" type="number" min="0" value="${st.minMargin ?? 10}"></div></div>
      <label class="check"><input type="checkbox" id="st-block" ${st.blockExpired ? "checked" : ""}> Không cho bán hàng đã hết hạn (chỉ bán từ lô còn hạn)</label>
      <label class="check"><input type="checkbox" id="st-neg" ${st.sellNegative ? "checked" : ""}> Cho phép bán khi tồn kho trên máy đã hết (bán âm kho)</label>
      <label class="check"><input type="checkbox" id="st-sprice" ${st.staffEditPrice ? "checked" : ""}> Nhân viên được sửa giá bán trên hóa đơn</label>
      <label class="check"><input type="checkbox" id="st-scost" ${st.staffSeeCost ? "checked" : ""}> Nhân viên được xem giá vốn và lãi</label></div>
    <div class="card stack"><h2>Tách đơn Zalo bằng AI (không bắt buộc)</h2>
      <p class="ps" style="margin:0">Tách đơn thông thường chạy ngay trên máy, không cần internet. Nếu khách hay viết tắt khó hiểu, có thể dùng AI Claude: cần khóa API từ console.anthropic.com, mỗi lần tách tốn khoảng vài chục đến vài trăm đồng.</p>
      <div class="field"><label for="st-key">Khóa API Claude</label><input id="st-key" class="in" type="password" autocomplete="off" placeholder="${S.hasKey ? "Đã lưu khóa · để trống để giữ nguyên" : "sk-ant-…"}"></div>
      ${S.hasKey ? `<label class="check"><input type="checkbox" id="st-delkey"> Xóa khóa đã lưu</label>` : ""}</div>
    <div class="row"><span class="grow"></span><button class="btn pri lg">Lưu cài đặt</button></div></form>
    <div class="card stack" style="margin-top:14px"><h2>Tự xác nhận khi tiền chuyển khoản về</h2>
      <p class="ps" style="margin:0">Khi khách quét mã QR, phần mềm tự kiểm tra tài khoản và báo "Đã nhận tiền" (kèm tiếng ting), có thể tự hoàn tất hóa đơn. Cần máy tính tiền có internet và một dịch vụ trung gian đọc giao dịch ngân hàng.</p>
      <div class="seg" role="group" aria-label="Dịch vụ">${[["", "Tắt"], ["sepay", "SePay"], ["payos", "payOS"]].map(([k, t]) => `<button type="button" data-act="payProv" data-v="${k}" aria-pressed="${(S.payProvSel ?? st.payProvider ?? "") === k}">${t}</button>`).join("")}</div>
      <div id="payCfg">${payCfgHtml()}</div></div>` : ""}
    <div class="card stack" style="margin-top:14px"><h2>Màn hình cho khách</h2>
      <p class="ps" style="margin:0">Màn hình thứ hai cắm vào máy tính tiền, hoặc máy tính bảng / điện thoại cũ đặt ở quầy, hiện tổng tiền và mã QR cho khách quét.</p>
      <div class="row"><button class="btn" data-act="dispMenu">Cài màn hình khách</button>${isAdmin() ? `<button class="btn ghost sm" data-act="dispKey" data-confirm="Bấm lần nữa: màn hình cũ sẽ phải mở lại">Đổi mã màn hình</button>` : ""}</div></div>
    <div class="card stack" style="margin-top:14px"><h2>Dùng trên điện thoại / máy khác</h2>
      ${lan ? `<div class="grid2" style="align-items:center"><div class="stack"><p style="margin:0">Điện thoại hoặc máy tính khác <b>bắt cùng wifi</b> với máy này, mở trình duyệt và vào:</p>${S.lan.map(u => `<div class="acct num" style="font-size:20px;font-weight:800;user-select:all">${esc(u)}</div>`).join("")}<p class="ps" style="margin:0">Lần đầu mở phần mềm, Windows có thể hỏi về tường lửa: chọn <b>Cho phép (Allow)</b>. Máy tính chính phải luôn bật Digital Shop.</p></div>
        <div style="display:grid;place-items:center;gap:6px"><div style="width:170px;background:#fff;padding:6px;border-radius:8px">${qrSvg(lan)}</div><span class="ps">Quét bằng camera điện thoại để mở</span></div></div>` : `<p class="ps" style="margin:0">Máy này chưa kết nối mạng nội bộ (wifi/LAN).</p>`}</div>
    <div class="card stack" style="margin-top:14px"><h2>Nhân viên & mã PIN</h2><div id="usersBox">${usersHtml()}</div></div>
    ${own ? `<div class="card stack" style="margin-top:14px"><h2>Sao lưu dữ liệu</h2>
      <p class="ps" style="margin:0">Phần mềm tự sao lưu mỗi ngày (giữ 40 bản gần nhất). Nên chép thư mục dữ liệu sang USB hoặc Google Drive mỗi tuần.</p>
      <div class="ps">Thư mục dữ liệu: <b class="num" style="user-select:all">${esc(S.dataDir)}</b></div>
      <div class="row"><button class="btn sm" data-act="bkNow">Sao lưu ngay</button><a class="btn sm" href="/api/backups/download" download>Tải file sao lưu về</a><button class="btn sm" data-act="bkFolder">Mở thư mục dữ liệu</button><label class="btn sm" for="bkUpload">Khôi phục từ file (USB…)</label><input type="file" id="bkUpload" accept=".json,.gz" hidden></div>
      <div class="card stack" style="background:var(--sunk);border:0"><h3>Tự chép sao lưu ra USB hoặc Google Drive / OneDrive</h3>
        <div id="extBox" class="ps">Đang dò ổ USB…</div></div>
      <div id="bkList"><div class="ps">Đang tải danh sách…</div></div></div>
    <div class="card row" style="margin-top:14px"><div class="grow"><h2>Tắt phần mềm</h2><span class="ps">Đóng cửa sổ này thì phần mềm vẫn chạy nền (biểu tượng cạnh đồng hồ) cho điện thoại và màn hình khách. Nút này lưu dữ liệu rồi tắt hẳn.</span></div><button class="btn danger" data-act="shutdown" data-confirm="Bấm lần nữa để tắt">Tắt phần mềm</button></div>` : ""}
    ${own ? `<div class="card stack" style="margin-top:14px"><h2>Cập nhật phần mềm</h2>
      <div class="ps">Đang dùng bản <b>${esc(S.appVersion || "")}</b>. Cần có internet. Dữ liệu bán hàng được giữ nguyên khi cập nhật.</div>
      <div id="updBox" class="ps"></div>
      <div class="row" style="flex-wrap:wrap;gap:8px"><button class="btn" data-act="updCheck">Kiểm tra bản mới</button>
      <label class="check"><input type="checkbox" id="upd-auto" ${st.updateOff ? "" : "checked"}> Tự kiểm tra mỗi ngày</label></div></div>` : ""}
    <div class="card stack" style="margin-top:14px"><h2>Giới thiệu & giấy phép</h2>
      <p class="ps" style="margin:0">Digital Shop ${esc(S.appVersion || "")} · Điều khoản sử dụng phiên bản ${esc(st.termsVersion || "1.0")}${st.termsAt ? ` · ${esc(st.termsBy || "chủ cửa hàng")} đồng ý ngày ${new Date(st.termsAt).toLocaleDateString("vi-VN")}` : ""}.</p>
      <div class="row"><button class="btn sm" data-act="legal" data-f="DieuKhoanSuDung">Xem điều khoản sử dụng</button><button class="btn sm" data-act="legal" data-f="GiayPhepBenThuBa">Giấy phép bên thứ ba</button></div></div>
    <div class="row" style="margin-top:14px"><span class="grow"></span><button class="btn" data-act="logout">Đăng xuất (${esc(S.me.name)})</button></div>
  </div>`;
  const f = $("#setForm"); if (f) f.addEventListener("submit", saveSettings);
  if (own) { loadLic(); loadBackups(); loadExt(); }
};
function usersHtml() {
  const own = isAdmin();
  return `<div class="tbl"><table><tbody>${S.users.map(u => `<tr><td class="pn">${esc(u.name)}${u.id === S.me.id ? ' <span class="pill green">bạn</span>' : ""}</td><td>${ROLE_NAME[u.role] || "Thu ngân"}${u.custom ? ' <span class="pill amber">quyền riêng</span>' : ""}${own && u.lanOnly ? ' <span class="pill mute">chỉ mạng nội bộ</span>' : ""}</td>
    <td class="r">${own || u.id === S.me.id ? `<button class="btn sm" data-act="userEdit" data-id="${esc(u.id)}">Đổi PIN / sửa</button>` : ""} ${own && u.id !== S.me.id ? `<button class="btn sm danger" data-act="userDel" data-id="${esc(u.id)}" data-confirm="Xóa?">Xóa</button>` : ""}</td></tr>`).join("")}</tbody></table></div>
    ${own ? `<div><button class="btn sm" data-act="userEdit" data-id="">+ Thêm nhân viên</button></div><p class="ps" style="margin:0">Mỗi nhân viên có vai trò (bộ quyền chọn sẵn) và có thể tick thêm hoặc bớt từng quyền. Chỉ chủ cửa hàng được vào Cài đặt, người dùng, bản quyền và sao lưu.</p>` : ""}`;
}
ACT.userEdit = b => {
  const u = S.users.find(x => x.id === b.dataset.id);
  if (!isAdmin() && (!u || u.id !== S.me.id)) return;
  const adm = isAdmin(), role0 = u ? (ROLE_NAME[u.role] ? u.role : "cashier") : "cashier";
  const have = new Set(u && u.perms ? u.perms : PERM_PRESET[role0]);
  openSheet(`<div class="sheet-h"><h2>${u ? "Sửa " + esc(u.name) : "Thêm nhân viên"}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <form id="uForm" class="grid2" autocomplete="off"><div class="field"><label for="u-name">Tên</label><input id="u-name" class="in" required value="${esc(u?.name || "")}"></div>
    <div class="field"><label for="u-role">Vai trò (chọn sẵn bộ quyền)</label><select id="u-role" class="in" ${adm ? "" : "disabled"}>${["cashier", "warehouse", "accountant", "manager", "owner"].map(k => `<option value="${k}" ${role0 === k ? "selected" : ""}>${ROLE_NAME[k]}</option>`).join("")}</select></div>
    ${adm ? `<div class="field full"><label>Quyền của nhân viên này</label><div id="u-perms" class="stack" style="gap:6px">${PERM_LABEL.map(([k, t]) => `<label class="check"><input type="checkbox" data-p="${k}" ${have.has(k) ? "checked" : ""}> <span>${t}</span></label>`).join("")}</div>
      <div class="ps" id="u-pnote" style="margin-top:4px"></div></div>
    <div class="field full"><label>Nơi được đăng nhập</label>
      <label class="check"><input type="radio" name="u-net" value="lan" ${(u ? u.lanOnly : true) ? "checked" : ""}> <span><b>Chỉ trong mạng nội bộ</b> (wifi/LAN của cửa hàng)</span></label>
      <label class="check"><input type="radio" name="u-net" value="any" ${(u ? u.lanOnly : true) ? "" : "checked"}> <span><b>Cho xem từ xa</b> (qua internet, VPN, tunnel đến máy chủ)</span></label>
      <div class="ps" style="margin-top:4px">Phần mềm phân biệt theo địa chỉ IP của thiết bị. Muốn dùng từ xa, máy chủ phải truy cập được từ ngoài (VPN, Cloudflare Tunnel…). Nên để thu ngân ở chế độ chỉ mạng nội bộ.</div></div>` : ""}
    <div class="field full"><label for="u-pin">${u ? "Mã PIN mới (để trống nếu giữ nguyên)" : "Mã PIN (4–8 số)"}</label><input id="u-pin" class="in num" inputmode="numeric" type="password" ${u ? "" : "required"} minlength="4" maxlength="8"></div>
    <div class="row full"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri">Lưu</button></div></form>`, { focus: "#u-name" });
  const boxes = () => [...document.querySelectorAll("#u-perms input[data-p]")];
  const note = () => {
    const el = $("#u-pnote"); if (!el) return;
    const r = $("#u-role").value, own = r === "owner";
    boxes().forEach(x => { x.disabled = own; if (own) x.checked = true; });
    const cur = boxes().filter(x => x.checked).map(x => x.dataset.p).sort().join(), def = PERM_PRESET[r].slice().sort().join();
    el.textContent = own ? "Chủ cửa hàng luôn có đủ quyền." : cur === def ? "Đúng bộ quyền của vai trò này." : "Đã tùy chỉnh: khác với bộ quyền mặc định của vai trò.";
  };
  if (adm) {
    $("#u-role").addEventListener("change", () => { const p = new Set(PERM_PRESET[$("#u-role").value]); boxes().forEach(x => { x.checked = p.has(x.dataset.p); }); note(); });
    $("#u-perms").addEventListener("change", note); note();
  }
  $("#uForm").addEventListener("submit", async e => {
    e.preventDefault();
    if (!adm) return toast("Nhờ chủ cửa hàng đổi PIN giúp", true);
    const perms = boxes().filter(x => x.checked).map(x => x.dataset.p), lanOnly = (document.querySelector('input[name="u-net"]:checked') || {}).value === "lan";
    const r = await run(() => api("POST", "/api/users", { id: u?.id || "", name: $("#u-name").value, role: $("#u-role").value, perms, lanOnly, pin: $("#u-pin").value }), "Đã lưu");
    if (r) { closeSheet(); await refresh(true); }
  });
};
ACT.userDel = async b => { if (await run(() => api("DELETE", "/api/users/" + b.dataset.id), "Đã xóa nhân viên")) refresh(true); };
async function saveSettings(e) {
  e.preventDefault();
  const v = id => $(id).value, c = id => $(id).checked;
  const bin = v("#st-bank"), key = v("#st-key").trim(), del = $("#st-delkey") && c("#st-delkey");
  const body = { shopName: v("#st-name"), phone: v("#st-phone"), addr: v("#st-addr"), bankBin: bin, bankName: (BANKS.find(b => b[0] === bin) || [])[1] || "", acct: v("#st-acct").replace(/\s/g, ""), holder: v("#st-holder").toUpperCase(),
    paper: v("#st-paper"), footer: v("#st-footer"), printFeed: Math.min(30, Math.max(1, +v("#st-feed") || 8)), autoPrint: c("#st-auto"), expWarn: +v("#st-warn") || 30, expDanger: +v("#st-danger") || 7, minMargin: +v("#st-margin") || 0,
    blockExpired: c("#st-block"), sellNegative: c("#st-neg"), staffEditPrice: c("#st-sprice"), staffSeeCost: c("#st-scost"), apiKey: del ? "" : key, keepKey: !del && !key };
  if (await run(() => api("POST", "/api/settings", body), "Đã lưu cài đặt")) await refresh(true);
}
ACT.testPrint = () => printHtml(receiptHtml({ code: "HD-THU", time: Date.now(), staff: S.me.name, lines: [{ name: "Mì Hảo Hảo tôm chua cay", unit: "gói", qty: 2, price: 4500 }, { name: "Bia Sài Gòn lon", unit: "thùng", qty: 1, price: 270000 }], sub: 279000, disc: 0, total: 279000, cash: 279000, given: 300000 }));
async function loadBackups() {
  const l = await api("GET", "/api/backups").catch(() => null); const box = $("#bkList"); if (!l || !box) return;
  box.innerHTML = l.length ? `<div class="tbl"><table><tbody>${l.slice(0, 12).map(b => `<tr><td class="num">${esc(b.name)}</td><td class="r ps">${(b.size / 1024).toFixed(0)} KB</td><td class="r"><button class="btn sm" data-act="bkRestore" data-n="${esc(b.name)}" data-confirm="Bấm lần nữa: khôi phục về bản này">Khôi phục</button></td></tr>`).join("")}</tbody></table></div>` : "Chưa có bản sao lưu.";
}
ACT.bkNow = async () => { if (await run(() => api("POST", "/api/backups"), r => "Đã sao lưu " + r.name)) loadBackups(); };
ACT.bkFolder = () => run(() => api("POST", "/api/open-folder"), "Đã mở thư mục trên máy chính");
ACT.bkRestore = async b => { if (await run(() => api("POST", "/api/backups/restore", { name: b.dataset.n }), "Đã khôi phục dữ liệu")) await refresh(); };
ON_CHANGE.bkUpload = async el => { const f = el.files[0]; if (!f) return; const fd = new FormData(); fd.append("file", f); if (await run(() => api("POST", "/api/backups/upload", fd), "Đã khôi phục từ file")) await refresh(); };
ACT.shutdown = async () => { await api("POST", "/api/shutdown").catch(() => {}); clearInterval(pollT); $("#app").innerHTML = `<div class="login"><div class="box"><div class="sign"><small>Digital Shop</small><b>Đã tắt</b></div><p class="muted" style="margin:0;text-align:center">Dữ liệu đã được lưu. Mở lại bằng biểu tượng Digital Shop trên màn hình. Có thể đóng cửa sổ này.</p></div></div>`; };

/* ---------- tự xác nhận chuyển khoản ---------- */
S.payProvSel = undefined;
function payCfgHtml() {
  const st = S.settings, k = S.payProvSel ?? st.payProvider ?? "", has = S.hasPayKeys || {};
  const auto = `<label class="check"><input type="checkbox" id="pc-auto" ${st.autoBankDone ? "checked" : ""}> Tiền về là tự hoàn tất hóa đơn (không cần bấm Thanh toán)</label>`;
  const save = `<div class="row"><span class="grow"></span><button type="button" class="btn pri" data-act="paySave">Lưu và kiểm tra kết nối</button></div>`;
  if (k === "sepay") return `<div class="stack"><ol class="small" style="margin:0;padding-left:18px">
      <li>Đăng ký tài khoản tại my.sepay.vn, liên kết tài khoản ngân hàng nhận tiền (đúng số tài khoản ở mục VietQR phía trên).</li>
      <li>Vào Cấu hình công ty → API Access → Thêm API, sao chép API token dán vào đây.</li>
      <li>Mỗi hóa đơn có mã nội dung riêng (vd TXK3P9QA) nằm sẵn trong mã QR; khách không được xóa nội dung khi chuyển.</li></ol>
    <div class="field"><label for="pc-sepay">API token SePay</label><input id="pc-sepay" class="in" type="password" autocomplete="off" placeholder="${has.sepay ? "Đã lưu · để trống để giữ nguyên" : "dán API token"}"></div>${auto}
    <label class="check"><input type="checkbox" id="pc-voice" ${st.payVoice ? "checked" : ""}> Đọc to "Đã nhận … đồng" khi tiền về (máy nối loa qua cổng aux hoặc Bluetooth)</label>
    <div class="row"><label class="check grow"><input type="checkbox" id="pc-voice-dev" ${voiceOn() ? "checked" : ""} data-ch="voiceDev"> Phát loa trên thiết bị này</label><button type="button" class="btn sm" data-act="voiceTest">Phát thử</button></div>${save}
    <p class="ps" style="margin:0">Phí dịch vụ do SePay quy định, xem trên trang của SePay.</p></div>`;
  if (k === "payos") return `<div class="stack"><ol class="small" style="margin:0;padding-left:18px">
      <li>Đăng ký tại payos.vn, liên kết tài khoản ngân hàng, tạo một kênh thanh toán.</li>
      <li>Sao chép Client ID, API Key và Checksum Key của kênh dán vào đây.</li>
      <li>Mã QR mỗi hóa đơn do payOS cấp; tiền vẫn về tài khoản ngân hàng của tiệm.</li></ol>
    <div class="grid3"><div class="field"><label for="pc-id">Client ID</label><input id="pc-id" class="in" autocomplete="off" placeholder="${has.payos ? "Đã lưu" : ""}"></div>
      <div class="field"><label for="pc-key">API Key</label><input id="pc-key" class="in" type="password" autocomplete="off" placeholder="${has.payos ? "Đã lưu" : ""}"></div>
      <div class="field"><label for="pc-sum">Checksum Key</label><input id="pc-sum" class="in" type="password" autocomplete="off" placeholder="${has.payos ? "Đã lưu" : ""}"></div></div>${auto}${save}
    <p class="ps" style="margin:0">Phí dịch vụ do payOS quy định, xem trên trang của payOS.</p></div>`;
  return `<div class="row"><span class="ps grow">Đang tắt: nhân viên tự kiểm tra tài khoản rồi bấm Thanh toán.</span>${st.payProvider ? `<button type="button" class="btn" data-act="paySave">Lưu (tắt)</button>` : ""}</div>`;
}
ACT.payProv = b => { S.payProvSel = b.dataset.v; $$('[data-act="payProv"]').forEach(x => x.setAttribute("aria-pressed", x.dataset.v === b.dataset.v)); $("#payCfg").innerHTML = payCfgHtml(); };
ACT.paySave = async b => {
  const k = S.payProvSel ?? S.settings.payProvider ?? "", v = id => ($(id) ? $(id).value.trim() : "");
  b.disabled = true;
  const r = await run(() => api("POST", "/api/pay/config", { provider: k, sepayToken: v("#pc-sepay"), payosClientId: v("#pc-id"), payosApiKey: v("#pc-key"), payosChecksum: v("#pc-sum"), autoBankDone: !!($("#pc-auto") && $("#pc-auto").checked), payVoice: !!($("#pc-voice") && $("#pc-voice").checked) }),
    k ? "Kết nối được, đã bật tự xác nhận chuyển khoản" : "Đã tắt tự xác nhận chuyển khoản");
  b.disabled = false;
  if (r) { S.payProvSel = undefined; await refresh(true); }
};
ACT.voiceTest = () => { if (!speakVN("Đã nhận " + vnNum(150000) + " đồng")) toast("Thiết bị này chưa có giọng đọc tiếng Việt. Cài giọng ở Cài đặt hệ thống → Văn bản thành giọng nói", true); };
ON_CHANGE.voiceDev = el => { try { localStorage.setItem("dsVoice", el.checked ? "1" : "0"); } catch (e) {} };
ACT.dispKey = async () => { if (await run(() => api("POST", "/api/display/key"), "Đã đổi mã. Mở lại màn hình khách từ nút Cài màn hình khách")) refresh(true); };

/* ---------- sao lưu ra USB ---------- */
async function loadExt() {
  const box = $("#extBox"); if (!box) return;
  const d = await api("GET", "/api/backups/external").catch(() => null); if (!d || !$("#extBox")) return;
  const st = d.status || {}, cur = S.settings.backupDir2 || "";
  const kindName = { usb: "USB", disk: "Ổ cứng", cloud: "Đám mây" };
  box.innerHTML = `<p style="margin:0">Cắm USB (hoặc chọn thư mục Google Drive/OneDrive trên máy), bấm <b>Chọn ổ này</b>. Từ đó mỗi khi USB đang cắm, phần mềm tự chép bản sao lưu mới vào thư mục <b>TapHoaX_SaoLuu</b> trên USB (giữ 60 bản gần nhất).</p>
    ${cur ? `<div class="banner ${st.ok ? "green" : "red"}" style="margin:0"><span class="grow">Đang chép ra <b class="num">${esc(cur)}</b>${st.ok ? ` · lần gần nhất ${st.lastAt ? dt(st.lastAt) : "—"} · ${st.files} bản trên ổ` : ` · ${esc(st.msg || "chưa chép được")}`}</span><button class="btn sm" data-act="extNow">Chép ngay</button><button class="btn sm ghost" data-act="extSet" data-dir="" data-confirm="Bấm lần nữa để ngừng">Ngừng chép</button></div>` : ""}
    ${(d.drives || []).length ? `<div class="tbl"><table><tbody>${d.drives.map(x => `<tr><td><b class="num">${esc(x.path)}</b> ${x.label ? esc(x.label) : ""}</td><td><span class="pill ${x.kind === "usb" ? "green" : x.kind === "cloud" ? "blue" : "mute"}">${kindName[x.kind] || x.kind}</span></td><td class="r ps">${x.totalGB ? `trống ${x.freeGB.toFixed(1)}/${x.totalGB.toFixed(0)} GB` : ""}</td><td class="r">${x.path === cur ? '<span class="pill green">đang dùng</span>' : `<button class="btn sm" data-act="extSet" data-dir="${esc(x.path)}">Chọn ổ này</button>`}</td></tr>`).join("")}</tbody></table></div>`
      : `<p class="ps" style="margin:0">Chưa thấy USB nào. Cắm USB rồi bấm Dò lại.</p>`}
    <div class="row"><input id="extPath" class="in sm grow" placeholder="Hoặc gõ đường dẫn, vd E:\\ hoặc D:\\SaoLuu" style="min-width:200px"><button class="btn sm" data-act="extSetPath">Dùng đường dẫn này</button><button class="btn sm ghost" data-act="extReload">Dò lại</button></div>
    <p class="ps" style="margin:0">Khôi phục khi máy hỏng: cài Digital Shop trên máy khác, thiết lập tạm, rồi vào đây bấm <b>Khôi phục từ file (USB…)</b> và chọn file mới nhất trong TapHoaX_SaoLuu.</p>`;
}
ACT.extReload = () => { $("#extBox").innerHTML = "Đang dò…"; loadExt(); };
async function extSet(dir) {
  const r = await run(() => api("POST", "/api/backups/external", { dir }), dir ? "Đã chép sao lưu ra " + dir : "Đã ngừng chép ra ổ ngoài");
  if (r) { await refresh(true); }
}
ACT.extSet = b => extSet(b.dataset.dir);
ACT.extSetPath = () => { const v = $("#extPath").value.trim(); if (!v) return toast("Gõ đường dẫn ổ hoặc thư mục", true); extSet(v); };
ACT.extNow = async () => { if (await run(() => api("POST", "/api/backups/external/now"), r => `Đã chép, trên ổ có ${r.files} bản`)) loadExt(); };

/* ---------- khởi động ---------- */
boot();

/* ---------- bản quyền ---------- */
async function loadLic(st) {
  const box = $("#licBox"); if (!box) return;
  try { st = st || await api("GET", "/api/license"); } catch (e) { box.innerHTML = `<div class="ps">${esc(e.message)}</div>`; return; }
  S.lic = st; licBar();
  const l = st.license, vn = d => d ? d.split("-").reverse().join("/") : "";
  let head;
  if (st.state === "active" && st.revokeOn) head = `<span class="pill red">Sắp bị khóa</span> Bản quyền ${esc(l.id)} đã bị thu hồi. Từ <b>${esc(fmtRev(st.revokeOn))}</b> phần mềm chuyển sang chế độ chỉ xem và xuất dữ liệu. Liên hệ Nam Linh Group (0968.646.853).`;
  else if (st.state === "active") head = `<span class="pill green">Đã kích hoạt</span> Gói <b>Đầy đủ</b> · ${l.exp ? "hết hạn " + vn(l.exp) + " (còn " + st.daysLeft + " ngày)" : "vĩnh viễn"}<div class="ps">Mã ${esc(l.id)} · ${esc(l.shop)}</div>`;
  else if (st.state === "trial") head = `<span class="pill">Dùng thử</span> Còn <b>${st.daysLeft}</b> ngày, đầy đủ tính năng`;
  else if (st.state === "revoked") head = `<span class="pill red">Đã thu hồi</span> Bản quyền ${esc(l.id)} đã bị thu hồi. Dữ liệu vẫn xem và xuất được; liên hệ Nam Linh Group (0968.646.853).`;
  else if (st.state === "expired") head = `<span class="pill red">Hết hạn</span> Bản quyền hết hạn ngày ${vn(l.exp)}. Dữ liệu vẫn xem và xuất được.`;
  else head = `<span class="pill red">Hết dùng thử</span> Dữ liệu vẫn xem và xuất được; muốn bán hàng tiếp hãy mua bản quyền.`;
  box.innerHTML = `<div>${head}</div>${st.note ? `<div class="ps" style="color:#b3261e">${esc(st.note)}</div>` : ""}
    <div class="row" style="flex-wrap:wrap;gap:8px">${st.state === "active" && l && !l.exp ? "" : `<button class="btn pri" data-act="licBuy">${st.state === "active" ? "Gia hạn bản quyền" : "Mua bản quyền"}</button>`}
      <button class="btn" data-act="licClaim">Nhận mã đã mua</button>${l ? `<button class="btn" data-act="licCheck">Kiểm tra trạng thái ngay</button>` : ""}</div>
    <div class="ps">Mua xong và chuyển khoản, phần mềm tự nhận mã và kích hoạt (chờ khoảng 1 phút). Nếu chưa thấy, bấm “Nhận mã đã mua”.</div>
    <div class="field"><label for="lic-key">Hoặc dán mã kích hoạt (bắt đầu bằng DS1.)</label><div class="row"><input id="lic-key" class="in num grow" autocomplete="off" placeholder="DS1.…"><button class="btn" data-act="licKey">Kích hoạt</button></div></div>
    <div class="ps">Mã máy này: <b class="num" style="user-select:all">${esc(st.machine)}</b></div>`;
}
let licWatchT = null;
function licWatch() {
  clearInterval(licWatchT);
  const id0 = S.lic && S.lic.license ? S.lic.license.id : "", t0 = Date.now();
  licWatchT = setInterval(async () => {
    if (Date.now() - t0 > 30 * 60 * 1000) return clearInterval(licWatchT);
    try {
      const r = await api("POST", "/api/license/claim");
      if (r.new && r.status.license && r.status.license.id !== id0) { clearInterval(licWatchT); toast("Đã nhận và kích hoạt bản quyền"); loadLic(r.status); }
    } catch (e) {}
  }, 5000);
}
let buyPoll = null;
function buyStop() { clearInterval(buyPoll); buyPoll = null; }
ACT.licBuy = async () => {
  const L = S.lic && S.lic.state === "active" && S.lic.license;
  if (L && !L.exp) return toast("Bản quyền của bạn là vĩnh viễn, không cần gia hạn.");
  let info;
  try { info = await api("GET", "/api/license/buy/info"); } catch (e) { return toast(e.message, true); }
  const cfg = info.config || {}, prices = (cfg.prices || {})[info.product] || {}, plans = cfg.plans || {};
  const keys = Object.keys(plans).filter(k => prices[k]);
  if (!keys.length) return toast("Chưa có bảng giá, liên hệ Nam Linh Group", true);
  const def = keys.includes("1y") ? "1y" : keys[0];
  openSheet(`<div class="sheet-h"><h2>Mua bản quyền Digital Shop</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div id="buyBody"><form id="buyForm" class="form">
      <div class="row full"><label class="ps">Chọn thời hạn</label><div id="buyPlans" style="display:grid;gap:6px">${keys.map(k => `<label class="check" style="justify-content:space-between"><span><input type="radio" name="bplan" value="${esc(k)}" ${k === def ? "checked" : ""}> ${esc(plans[k])}</span><b>${money(prices[k])}</b></label>`).join("")}</div></div>
      <div class="row"><input id="bshop" class="in grow" placeholder="Tên cửa hàng" value="${esc(info.shop || "")}" required><input id="bphone" class="in" style="width:150px" placeholder="Số điện thoại" inputmode="tel" value="${esc(info.phone || "")}" required></div>
      <div class="row"><input id="bemail" class="in grow" placeholder="Email nhận hóa đơn (không bắt buộc)"><input id="bmst" class="in" style="width:170px" placeholder="Mã số thuế (nếu có)"></div>
      <div class="row"><input id="bpart" class="in grow" placeholder="Mã đối tác (nếu có)" autocomplete="off"><button type="button" class="btn" id="bpartOk">Áp dụng</button></div>
      <div id="bpartMsg" class="ps" style="min-height:18px"></div>
      <div id="bErr" class="ps" style="color:var(--red);min-height:18px"></div>
      <label class="check"><input type="checkbox" id="bagree" required> <span>Tôi đồng ý Điều khoản sử dụng của Digital Shop (xem ở Cài đặt → Bản quyền)</span></label>
      <div class="row full"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri" type="submit">Tiếp tục thanh toán</button></div>
    </form></div>`, { wide: false, onClose: buyStop });
  const plan = () => document.querySelector('input[name="bplan"]:checked').value;
  const err = m => { $("#bErr").textContent = m || ""; };
  let applied = "";
  $("#bpartOk").onclick = async () => {
    const code = $("#bpart").value.trim(); err("");
    if (!code) { applied = ""; $("#bpartMsg").textContent = ""; return; }
    try {
      const r = await api("GET", "/api/license/buy/partner?plan=" + encodeURIComponent(plan()) + "&code=" + encodeURIComponent(code));
      applied = code;
      $("#bpartMsg").style.color = "var(--green-text)";
      $("#bpartMsg").textContent = r.percent >= 100 ? "Giảm 100%: miễn phí, mã kích hoạt được cấp ngay" : "Giảm " + r.percent + "%: " + money(r.list) + " còn " + money(r.amount);
    } catch (e) { applied = ""; $("#bpartMsg").style.color = "var(--red)"; $("#bpartMsg").textContent = e.message; }
  };
  $("#buyForm").onsubmit = async ev => {
    ev.preventDefault(); err("");
    const btn = ev.target.querySelector('button[type="submit"]'); btn.disabled = true;
    try {
      const o = await api("POST", "/api/license/buy/order", { plan: plan(), shop: $("#bshop").value.trim(), phone: $("#bphone").value.trim(), email: $("#bemail").value.trim(), mst: $("#bmst").value.trim(), partner: $("#bpart").value.trim(), agree: $("#bagree").checked });
      if (o.status === "paid") return buyDone(o);
      buyPay(o);
    } catch (e) { err(e.message); btn.disabled = false; }
  };
};
function buyPay(o) {
  const b = $("#buyBody"); if (!b) return;
  b.innerHTML = `<p class="ps" style="margin:0 0 8px">Quét mã QR bằng app ngân hàng. Số tiền và nội dung đã điền sẵn, bạn không cần sửa. Thanh toán xong, phần mềm tự kích hoạt trong ít giây.</p>
    <div style="display:grid;place-items:center;gap:8px"><div style="width:240px;min-height:240px;background:#fff;padding:8px;border-radius:10px"><img id="bqr" alt="QR thanh toán" style="width:100%;display:block"></div>
    <div id="bqrErr" class="ps" hidden>Không tải được mã QR, hãy chuyển khoản theo thông tin bên dưới.</div></div>
    <table class="kv" style="margin-top:10px"><tbody>
      <tr><td class="ps">Ngân hàng</td><td><b>${esc(o.bank.bank)}</b></td></tr>
      <tr><td class="ps">Số tài khoản</td><td><b class="num">${esc(o.bank.acc)}</b></td></tr>
      <tr><td class="ps">Chủ tài khoản</td><td><b>${esc(o.bank.name)}</b></td></tr>
      <tr><td class="ps">Số tiền</td><td><b class="num">${money(o.amount)}</b></td></tr>
      <tr><td class="ps">Nội dung</td><td><b class="num">${esc(o.code)}</b></td></tr></tbody></table>
    <div id="bwait" class="ps" style="margin-top:8px">Đang chờ thanh toán…</div>
    <div class="row" style="margin-top:8px"><button class="btn pri" id="bcheck">Tôi đã thanh toán, kiểm tra ngay</button><span class="grow"></span><button class="btn" data-act="close">Đóng, thanh toán sau</button></div>`;
  const img = $("#bqr"); img.onerror = () => { img.style.display = "none"; $("#bqrErr").hidden = false; }; img.src = o.qr;
  buyStop(); const t0 = Date.now();
  const id0 = S.lic && S.lic.license ? S.lic.license.id : "";
  let busy = false;
  // Kiểm tra: hỏi đơn hàng; đồng thời hỏi theo mã máy (phòng khi đơn đã trả tiền nhưng tra theo mã đơn không ra).
  const check = async (manual) => {
    if (busy) return; busy = true;
    const w = $("#bwait");
    try {
      const r = await api("GET", "/api/license/buy/order?code=" + encodeURIComponent(o.code) + "&t=" + encodeURIComponent(o.token) + "&_=" + Date.now());
      if (r.status === "paid") return buyDone(r);
      const c = await api("POST", "/api/license/claim");
      if (c.found && c.status && c.status.license && c.status.license.id !== id0) { loadLic(c.status); return buyDone({ product: "Digital Shop", plan: "", exp: c.status.license.exp || "", claimed: true }); }
      if (manual && w) w.textContent = "Chưa thấy khoản thanh toán. Nếu vừa chuyển khoản, đợi vài giây rồi bấm lại.";
    } catch (e) { if (w) w.textContent = "Chưa kiểm tra được: " + e.message; }
    finally { busy = false; }
  };
  const btn = $("#bcheck"); if (btn) btn.onclick = () => check(true);
  buyPoll = setInterval(() => {
    if (Date.now() - t0 > 30 * 60 * 1000) { buyStop(); const w = $("#bwait"); if (w) w.textContent = "Hết thời gian chờ. Nếu đã chuyển khoản, bấm Kiểm tra thanh toán."; return; }
    check(false);
  }, 4000);
}
async function buyDone(o) {
  buyStop();
  const b = $("#buyBody"); if (!b) return;
  let ok = false;
  try { const r = await api("POST", "/api/license/claim"); ok = r.found; loadLic(r.status); } catch (e) {}
  if (o.claimed) ok = true;
  b.innerHTML = `<div style="text-align:center;padding:14px 0"><div style="font-size:42px">✓</div><h3 style="margin:6px 0">${ok ? "Đã kích hoạt bản quyền" : "Thanh toán thành công"}</h3>
    <p class="ps" style="margin:0">${esc(o.product || "")} · ${esc(o.plan || "")}${o.exp ? " · dùng đến hết ngày " + esc(o.exp.split("-").reverse().join("/")) : " · vĩnh viễn"}</p>
    ${ok ? "" : '<p class="ps">Chưa kích hoạt tự động được. Đóng hộp này rồi bấm "Nhận mã đã mua".</p>'}</div>
    <div class="row"><span class="grow"></span><button class="btn pri" data-act="close">Xong</button></div>`;
}
ACT.licClaim = async () => {
  const r = await run(() => api("POST", "/api/license/claim"));
  if (!r) return;
  toast(r.new ? "Đã nhận và kích hoạt bản quyền" : r.found ? "Bản quyền đã được kích hoạt" : "Chưa thấy khoản thanh toán nào cho máy này", !r.found);
  loadLic(r.status);
};
ACT.licCheck = async () => {
  const st = await run(() => api("POST", "/api/license/check"));
  if (!st) return;
  toast(st.state === "revoked" ? "Bản quyền đã bị thu hồi" : st.revokeOn ? "Bản quyền đã bị thu hồi, sắp chuyển sang chỉ xem" : "Bản quyền đang hoạt động bình thường");
  loadLic(st);
};
ACT.licKey = async () => {
  const k = $("#lic-key").value.trim(); if (!k) return toast("Hãy dán mã kích hoạt", true);
  const st = await run(() => api("POST", "/api/license/activate", { key: k }), "Đã kích hoạt bản quyền");
  if (st) loadLic(st);
};

/* ---------- cập nhật phần mềm ---------- */
ACT.updCheck = async () => {
  const box = $("#updBox"); if (box) box.textContent = "Đang kiểm tra…";
  const r = await run(() => api("POST", "/api/update/check"));
  if (!r) { if (box) box.textContent = ""; return; }
  showUpd(r);
};
function showUpd(r) {
  const box = $("#updBox"); if (!box) return;
  if (!r.newer) { box.innerHTML = "Bạn đang dùng bản mới nhất."; return; }
  box.innerHTML = `<div style="color:var(--text)"><b>Có bản mới ${esc(r.latest)}</b>${r.size ? " (" + (r.size / 1048576).toFixed(1) + " MB)" : ""}${r.notes ? "<br>" + esc(r.notes) : ""}</div>
    ${r.can ? `<div style="margin-top:6px"><button class="btn pri" data-act="updApply">Cập nhật ngay</button> <span class="ps">Phần mềm sẽ tự mở lại sau vài giây.</span></div>` : `<div class="ps">Máy này chưa hỗ trợ tự cập nhật, hãy cài bản mới bằng tay.</div>`}`;
}
ACT.updApply = async () => {
  const box = $("#updBox"); if (box) box.textContent = "Đang tải và cài bản mới, đừng tắt máy…";
  const r = await run(() => api("POST", "/api/update/apply"));
  if (!r) { if (box) box.textContent = ""; return; }
  $("#app").innerHTML = `<div class="login"><div class="box"><div class="sign"><small>Digital Shop</small><b>Đang cập nhật…</b></div><p class="muted" style="margin:0;text-align:center">Phần mềm đang mở lại bản ${esc(r.version)}. Chờ khoảng 10 giây rồi tải lại trang này (hoặc mở lại biểu tượng Digital Shop).</p></div></div>`;
  clearInterval(pollT);
  setTimeout(function again() { fetch("/api/ping").then(x => x.text()).then(t => { if (t === "taphoax") location.reload(); else setTimeout(again, 2000); }).catch(() => setTimeout(again, 2000)); }, 8000);
};
ON_CHANGE["upd-auto"] = async el => { await run(() => api("POST", "/api/update/config", { auto: el.checked })); };


/* ================= KHUYẾN MÃI TỰ ĐỘNG & HẠNG KHÁCH ================= */
const WDN = ["CN", "T2", "T3", "T4", "T5", "T6", "T7"];
function tiersNow() { return (S.settings.tiers && S.settings.tiers.length ? S.settings.tiers : [{ name: "Bạc", min: 2000000 }, { name: "Vàng", min: 10000000 }, { name: "Kim cương", min: 30000000 }]).slice().sort((a, b) => a.min - b.min); }
function promoWhen(p) {
  const w = [];
  if (p.days && p.days.length && p.days.length < 7) w.push(p.days.slice().sort().map(d => WDN[d]).join(", "));
  if (p.tStart && p.tEnd) w.push(p.tStart + "–" + p.tEnd);
  if (p.from || p.to) w.push((p.from ? vnDate(p.from) : "…") + " → " + (p.to ? vnDate(p.to) : "…"));
  return w.join(" · ") || "Mọi lúc";
}
function promoWhat(p) {
  const t = p.scope === "product" ? (S.products.get(p.target) || {}).name || "(mặt hàng đã xóa)" : p.scope === "group" ? "Nhóm " + p.target : "Toàn bộ hàng";
  return t + " · giảm " + (p.kind === "amt" ? money(p.value) + "/đơn vị" : p.value + "%");
}
function loyaltyCard() {
  const l = S.settings.loyalty || {}, on = !!l.on;
  return `<div class="card stack" style="margin-bottom:12px"><div class="row"><h3 class="grow" style="margin:0">Tích điểm khách hàng</h3><button class="btn sm pri" data-act="loySave">Lưu</button></div>
    <label class="check"><input type="checkbox" id="loy-on" ${on ? "checked" : ""}> Bật tích điểm (chỉ tính cho khách đã chọn trên hóa đơn)</label>
    <div class="grid2">
      <div class="field"><label for="loy-earn">Mua bao nhiêu đồng được 1 điểm</label><input id="loy-earn" class="in r num" inputmode="numeric" value="${fmt(l.earnPer || 10000)}"></div>
      <div class="field"><label for="loy-val">1 điểm đổi được bao nhiêu đồng</label><input id="loy-val" class="in r num" inputmode="numeric" value="${fmt(l.pointValue || 100)}"></div>
      <div class="field"><label for="loy-min">Dùng tối thiểu (điểm)</label><input id="loy-min" class="in r num" inputmode="numeric" value="${fmt(l.minRedeem || 0)}"></div>
      <div class="field"><label for="loy-pct">Tối đa % hóa đơn trả bằng điểm</label><input id="loy-pct" class="in r num" inputmode="numeric" value="${fmt(l.maxPct || 100)}"></div></div>
    <p class="ps" style="margin:0">Ví dụ: 10.000đ = 1 điểm, 1 điểm = 100đ nghĩa là mua 1 triệu được 100 điểm, đổi ra 10.000đ (hoàn 1%). Điểm tính trên số tiền khách thực trả (sau khuyến mãi và điểm đã dùng). Hủy hóa đơn hoặc khách trả hàng sẽ thu hồi điểm tương ứng. Khi mất mạng (Android) vẫn tích điểm nhưng chưa dùng điểm được.</p></div>`;
}
ACT.loySave = async () => {
  const body = { on: $("#loy-on").checked, earnPer: parseMoney($("#loy-earn").value), pointValue: parseMoney($("#loy-val").value), minRedeem: parseMoney($("#loy-min").value), maxPct: parseMoney($("#loy-pct").value) };
  const r = await run(() => api("POST", "/api/loyalty", body), "Đã lưu cài đặt tích điểm");
  if (r) { await refresh(true); promoRender(); }
};
ACT.ptsAdj = b => {
  const id = b.dataset.id, c = S.customers.get(id); if (!c) return;
  openSheet(`<div class="sheet-h"><h2>Điểm của ${esc(c.name)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <p class="ps">Hiện có <b>${fmt(c.points || 0)}</b> điểm. Nhập số dương để cộng, số âm để trừ (ví dụ tặng điểm sinh nhật, sửa sai).</p>
    <div class="grid2"><div class="field"><label for="pa-n">Số điểm (+/−)</label><input id="pa-n" class="in r num" inputmode="numeric" placeholder="+50"></div>
    <div class="field"><label for="pa-note">Lý do</label><input id="pa-note" class="in"></div></div>
    <div class="row"><span class="grow"></span><button class="btn pri" data-act="ptsAdjGo" data-id="${esc(id)}">Ghi nhận</button></div>`, { focus: "#pa-n" });
};
ACT.ptsAdjGo = async b => {
  const delta = parseInt(String($("#pa-n").value).replace(/[^\d+-]/g, ""), 10);
  if (!delta) return toast("Nhập số điểm cần cộng hoặc trừ", true);
  const r = await run(() => api("POST", `/api/customers/${b.dataset.id}/points`, { delta, note: $("#pa-note").value }), "Đã cập nhật điểm");
  if (r) { await refresh(true); closeSheet(); }
};
function promoRender() {
  const box = $("#promoBox"); if (!box) return;
  const list = S.settings.promos || [], tiers = tiersNow();
  box.innerHTML = loyaltyCard() + `<div class="section"><div class="row"><h2 class="grow" style="margin:0">Khuyến mãi tự động</h2><button class="btn sm pri" data-act="promoEdit" data-i="-1">+ Khuyến mãi</button></div>
    <p class="muted" style="margin:6px 0 10px;max-width:760px">Khuyến mãi tự áp dụng khi bán hàng đúng khung giờ, đúng ngày, đúng hạng khách. Hết giờ tự về giá thường. Mỗi mặt hàng chỉ lấy một khuyến mãi có lợi nhất cho khách (không cộng dồn).</p>
    ${list.length ? `<div class="tbl"><table><thead><tr><th>Chương trình</th><th>Áp dụng cho</th><th>Khi nào</th><th>Khách</th><th></th></tr></thead><tbody>${list.map((p, i) => `<tr class="click" data-act="promoEdit" data-i="${i}"><td class="pn">${esc(p.name)} ${p.active ? '<span class="pill green">đang bật</span>' : '<span class="pill mute">tắt</span>'}</td><td>${esc(promoWhat(p))}</td><td class="ps">${esc(promoWhen(p))}</td><td class="ps">${p.minTier ? "Hạng " + esc((tiers[p.minTier - 1] || {}).name || "") + " trở lên" : "Mọi khách"}</td><td class="r ps">Sửa</td></tr>`).join("")}</tbody></table></div>` : `<div class="empty">Chưa có khuyến mãi nào. Ví dụ: giảm 20% đồ uống từ 18h đến 22h, hoặc khách hạng Vàng giảm 5% mọi mặt hàng.</div>`}
    <div class="card stack" style="margin-top:12px"><div class="row"><h3 class="grow" style="margin:0">Hạng khách hàng</h3><button class="btn sm" data-act="tierSave">Lưu hạng</button></div>
      <p class="ps" style="margin:0">Khách đạt hạng khi tổng tiền đã mua (không tính hóa đơn hủy) đạt mức bên dưới. Xếp từ thấp đến cao.</p>
      <div id="tierRows" class="stack">${tiers.map(t => tierRow(t)).join("")}</div>
      <div><button class="btn sm" data-act="tierAdd">+ Thêm hạng</button></div></div></div>`;
}
const tierRow = t => `<div class="row tier"><input class="in grow t-name" placeholder="Tên hạng" value="${esc(t.name)}"><span class="ps">từ</span><input class="in r num t-min" style="width:150px" inputmode="numeric" value="${fmt(t.min)}"><span class="ps">đ</span><button class="btn sm ghost" data-act="tierDel" aria-label="Xóa hạng">×</button></div>`;
ACT.tierAdd = () => { const d = document.createElement("div"); d.innerHTML = tierRow({ name: "", min: 0 }); $("#tierRows").appendChild(d.firstChild); };
ACT.tierDel = b => b.closest(".tier").remove();
async function promoPost(promos, tiers, okMsg) {
  const r = await run(() => api("POST", "/api/promos", { promos, tiers }), okMsg);
  if (r) { await refresh(true); promoRender(); }
  return r;
}
ACT.tierSave = () => {
  const tiers = [...document.querySelectorAll("#tierRows .tier")].map(r => ({ name: r.querySelector(".t-name").value.trim(), min: parseMoney(r.querySelector(".t-min").value) })).filter(t => t.name);
  promoPost(S.settings.promos || [], tiers, "Đã lưu hạng khách hàng");
};
ACT.promoEdit = b => {
  const i = +b.dataset.i, p = i >= 0 ? (S.settings.promos || [])[i] : { name: "", active: true, kind: "pct", value: 10, scope: "all", target: "", days: [], from: "", to: "", tStart: "", tEnd: "", minTier: 0 };
  if (!p) return;
  const tiers = tiersNow();
  const groups = [...new Set([...S.products.values()].map(x => x.group).filter(Boolean))].sort((a, c) => a.localeCompare(c, "vi"));
  const prodLabel = x => (x.code ? x.code + " · " : "") + x.name;
  const curProd = p.scope === "product" ? S.products.get(p.target) : null;
  openSheet(`<div class="sheet-h"><h2>${i >= 0 ? "Sửa khuyến mãi" : "Thêm khuyến mãi"}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <form id="promoForm" class="form">
      <div class="field full"><label for="pm-name">Tên chương trình</label><input id="pm-name" class="in" value="${esc(p.name)}" placeholder="vd Giờ vàng đồ uống" required></div>
      <div class="field"><label for="pm-kind">Cách giảm</label><select id="pm-kind" class="in"><option value="pct" ${p.kind !== "amt" ? "selected" : ""}>Giảm theo %</option><option value="amt" ${p.kind === "amt" ? "selected" : ""}>Giảm số tiền trên mỗi đơn vị bán</option></select></div>
      <div class="field"><label for="pm-val">Mức giảm</label><input id="pm-val" class="in num" inputmode="decimal" value="${p.value}" required></div>
      <div class="field"><label for="pm-scope">Áp dụng cho</label><select id="pm-scope" class="in"><option value="all" ${p.scope === "all" ? "selected" : ""}>Toàn bộ hàng</option><option value="group" ${p.scope === "group" ? "selected" : ""}>Một nhóm hàng</option><option value="product" ${p.scope === "product" ? "selected" : ""}>Một mặt hàng</option></select></div>
      <div class="field"><label for="pm-target">Nhóm / mặt hàng</label><input id="pm-target" class="in" list="pm-list" autocomplete="off" value="${esc(p.scope === "group" ? p.target : curProd ? prodLabel(curProd) : "")}" placeholder="Chọn khi áp dụng cho nhóm hoặc mặt hàng"><datalist id="pm-list">${groups.map(g => `<option value="${esc(g)}">`).join("")}${[...S.products.values()].filter(x => !x.hidden).slice(0, 3000).map(x => `<option value="${esc(prodLabel(x))}">`).join("")}</datalist></div>
      <div class="field full"><label>Ngày trong tuần (bỏ trống = mọi ngày)</label><div class="row" style="flex-wrap:wrap;gap:10px">${WDN.map((n, d) => `<label class="check"><input type="checkbox" class="pm-day" value="${d}" ${(p.days || []).includes(d) ? "checked" : ""}> ${n}</label>`).join("")}</div></div>
      <div class="field"><label for="pm-ts">Từ giờ</label><input id="pm-ts" class="in" type="time" value="${esc(p.tStart || "")}"></div>
      <div class="field"><label for="pm-te">Đến giờ (có thể qua nửa đêm)</label><input id="pm-te" class="in" type="time" value="${esc(p.tEnd || "")}"></div>
      <div class="field"><label for="pm-from">Từ ngày</label><input id="pm-from" class="in" type="date" value="${esc(p.from || "")}"></div>
      <div class="field"><label for="pm-to">Đến ngày</label><input id="pm-to" class="in" type="date" value="${esc(p.to || "")}"></div>
      <div class="field full"><label for="pm-tier">Khách được hưởng</label><select id="pm-tier" class="in"><option value="0">Mọi khách (kể cả khách lẻ)</option>${tiers.map((t, k) => `<option value="${k + 1}" ${p.minTier === k + 1 ? "selected" : ""}>Khách hạng ${esc(t.name)} trở lên</option>`).join("")}</select></div>
      <div class="field full"><label class="check"><input type="checkbox" id="pm-on" ${p.active ? "checked" : ""}> Đang bật</label></div>
      <div class="row full">${i >= 0 ? `<button type="button" class="btn danger" data-act="promoDel" data-i="${i}" data-confirm="Bấm lần nữa để xóa">Xóa</button>` : ""}<span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri" type="submit">Lưu</button></div></form>`, { focus: "#pm-name" });
  $("#promoForm").onsubmit = async ev => {
    ev.preventDefault();
    const scope = $("#pm-scope").value, tv = $("#pm-target").value.trim();
    let target = "";
    if (scope === "group") target = tv;
    if (scope === "product") { const f = [...S.products.values()].find(x => prodLabel(x) === tv || x.name === tv); if (!f) return toast("Chọn mặt hàng trong danh sách gợi ý", true); target = f.id; }
    const np = { id: p.id || "", name: $("#pm-name").value.trim(), active: $("#pm-on").checked, kind: $("#pm-kind").value, value: parseFloat(String($("#pm-val").value).replace(",", ".")) || 0, scope, target,
      days: [...document.querySelectorAll(".pm-day:checked")].map(x => +x.value), from: $("#pm-from").value, to: $("#pm-to").value, tStart: $("#pm-ts").value, tEnd: $("#pm-te").value, minTier: +$("#pm-tier").value };
    const all = (S.settings.promos || []).slice();
    if (i >= 0) all[i] = np; else all.push(np);
    if (await promoPost(all, tiersNow(), "Đã lưu khuyến mãi")) closeSheet();
  };
};
ACT.promoDel = async b => {
  const all = (S.settings.promos || []).filter((_, k) => k !== +b.dataset.i);
  if (await promoPost(all, tiersNow(), "Đã xóa khuyến mãi")) closeSheet();
};
