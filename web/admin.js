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
      items.slice(0, 500).map((c, i) => `<button class="debtrow" data-act="custOpenD" data-id="${esc(c.id)}"><span class="idx num">${i + 1}</span><span class="who">${esc(c.name)}</span>
        <span class="amt num ${(c.debt || 0) <= 0 ? "zero" : ""}">${(c.debt || 0) <= 0 ? (c.debt < 0 ? "trả dư " + fmt(-c.debt) : "không nợ") : money(c.debt)}</span>
        <span class="meta">${c.phone ? esc(c.phone) + " · " : ""}${esc(c.code)}${c.last ? " · giao dịch " + new Date(c.last).toLocaleDateString("vi-VN") : ""}${c.note ? " · " + esc(c.note) : ""}</span></button>`).join("")}
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
    <div class="card stack" style="padding:12px">
      <div class="field"><label for="pdAmt">${isC ? "Khách trả nợ" : "Trả nợ nhà cung cấp"}</label><div class="row"><input id="pdAmt" class="in num grow" inputmode="numeric" placeholder="Số tiền" style="min-width:140px">
        <select id="pdMethod" class="in" style="width:auto"><option value="cash">Tiền mặt</option><option value="bank">Chuyển khoản</option></select>
        <button class="btn" data-act="pdAll" data-v="${Math.max(0, p.debt)}">Hết nợ</button><button class="btn pri" data-act="pdPay" data-k="${kind}" data-id="${esc(id)}">Ghi nhận</button></div></div>
      ${isOwner() ? `<details><summary class="ps" style="cursor:pointer">Điều chỉnh công nợ (không qua tiền)</summary><div class="row" style="margin-top:8px"><input id="pdAdj" class="in num" style="width:150px" inputmode="numeric" placeholder="+ hoặc − số tiền"><input id="pdAdjNote" class="in grow" placeholder="Lý do, vd ghi nợ cũ, xóa nợ"><button class="btn" data-act="pdAdj" data-k="${kind}" data-id="${esc(id)}">Điều chỉnh</button></div></details>` : ""}
    </div>
    <div class="row">${isC && p.debt > 0 ? `<button class="btn sm" data-act="remind" data-id="${esc(id)}">Sao chép tin nhắn nhắc nợ</button>${st.acct ? `<button class="btn sm" data-act="debtQR" data-id="${esc(id)}">Mã QR trả nợ</button>` : ""}` : ""}
      <button class="btn sm" data-act="pdEdit" data-k="${kind}" data-id="${esc(id)}">Sửa thông tin</button><span class="grow"></span>
      ${isOwner() && !p.debt ? `<button class="btn sm danger" data-act="pdDel" data-k="${kind}" data-id="${esc(id)}" data-confirm="Bấm lần nữa để xóa">Xóa</button>` : ""}</div>
    <div><div class="lbl" style="margin-bottom:6px">Lịch sử</div><div id="pdHist"><div class="muted">Đang tải…</div></div></div>`);
  const h = await run(() => api("GET", `/api/${isC ? "customers" : "suppliers"}/${id}/history`)); if (!h || !$("#pdHist")) return;
  $("#pdHist").innerHTML = h.length ? `<div class="tbl"><table><tbody>${h.map(x => {
    let what, eff;
    if (x.type === "invoice") { what = `Hóa đơn ${esc(x.code)} · ${money(x.total)}<div class="ps">${esc(x.lines.map(l => l.name + " ×" + fq(l.qty)).join(", ")).slice(0, 120)}</div>`; eff = x.debt; }
    else if (x.type === "receipt") { what = `Phiếu nhập ${esc(x.code)} · ${money(x.total)}`; eff = x.debt; }
    else { what = `${esc(x.cat || (x.kind === "thu" ? "Phiếu thu" : x.kind === "chi" ? "Phiếu chi" : "Điều chỉnh"))} ${esc(x.code)}${x.method === "bank" && x.kind !== "adj" ? " · chuyển khoản" : ""}${x.note ? `<div class="ps">${esc(x.note)}</div>` : ""}`; eff = x.debtEffect; }
    return `<tr class="${x.void ? "void" : ""}"><td class="num ps" style="white-space:nowrap">${dt(x.time)}</td><td>${what}</td><td class="r num" style="color:${eff > 0 ? "var(--red)" : eff < 0 ? "var(--green-text)" : "inherit"}"><b>${eff ? (eff > 0 ? "+" : "") + fmt(eff) : ""}</b></td>
      <td class="r">${x.type === "cash" && !x.void && isOwner() ? `<button class="btn sm ghost" data-act="cashVoid" data-id="${esc(x.id)}" data-back="${kind}:${esc(id)}" data-confirm="Hủy phiếu?">Hủy</button>` : ""}</td></tr>`;
  }).join("")}</tbody></table></div>` : `<div class="ps">Chưa có giao dịch.</div>`;
}
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
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Báo cáo</h1><a class="btn sm" href="/api/export/invoices?from=${a}&to=${b}" download>Xuất hóa đơn ra Excel</a>${S.rpPeriod === "today" || S.rpPeriod === "yday" ? `<button class="btn sm" data-act="eodPrint">${icon("print")} In báo cáo cuối ngày</button>` : ""}</div>
    <div class="row" style="margin-bottom:12px"><div class="chips grow">${periodChips(S.rpPeriod, "rpPeriod")}</div>
      <input type="date" id="rpFrom" class="in sm" style="width:auto" value="${a}"><span class="ps">đến</span><input type="date" id="rpTo" class="in sm" style="width:auto" value="${b}"></div>
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
      ${rp.voids ? `<div class="stat bad"><div class="k">Hóa đơn đã hủy</div><div class="v">${rp.voids}</div></div>` : ""}</div>
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
    <div id="cbBody"><div class="muted">Đang phân tích…</div></div></div>`;
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
  const st = S.settings, own = isOwner();
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
      <div class="row"><button class="btn" data-act="dispMenu">Cài màn hình khách</button>${isOwner() ? `<button class="btn ghost sm" data-act="dispKey" data-confirm="Bấm lần nữa: màn hình cũ sẽ phải mở lại">Đổi mã màn hình</button>` : ""}</div></div>
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
  const own = isOwner();
  return `<div class="tbl"><table><tbody>${S.users.map(u => `<tr><td class="pn">${esc(u.name)}${u.id === S.me.id ? ' <span class="pill green">bạn</span>' : ""}</td><td>${u.role === "owner" ? "Chủ cửa hàng" : "Nhân viên bán hàng"}</td>
    <td class="r">${own || u.id === S.me.id ? `<button class="btn sm" data-act="userEdit" data-id="${esc(u.id)}">Đổi PIN / sửa</button>` : ""} ${own && u.id !== S.me.id ? `<button class="btn sm danger" data-act="userDel" data-id="${esc(u.id)}" data-confirm="Xóa?">Xóa</button>` : ""}</td></tr>`).join("")}</tbody></table></div>
    ${own ? `<div><button class="btn sm" data-act="userEdit" data-id="">+ Thêm nhân viên</button></div><p class="ps" style="margin:0">Nhân viên bán được hàng, nhận đơn đặt, thu nợ, nhập hàng, kiểm kho. Chỉ chủ cửa hàng được sửa giá, xóa hàng, hủy hóa đơn, xem lãi, nhập dữ liệu và cài đặt.</p>` : ""}`;
}
ACT.userEdit = b => {
  const u = S.users.find(x => x.id === b.dataset.id);
  if (!isOwner() && (!u || u.id !== S.me.id)) return;
  openSheet(`<div class="sheet-h"><h2>${u ? "Sửa " + esc(u.name) : "Thêm nhân viên"}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <form id="uForm" class="grid2" autocomplete="off"><div class="field"><label for="u-name">Tên</label><input id="u-name" class="in" required value="${esc(u?.name || "")}"></div>
    <div class="field"><label for="u-role">Vai trò</label><select id="u-role" class="in" ${isOwner() ? "" : "disabled"}><option value="staff">Nhân viên bán hàng</option><option value="owner" ${u?.role === "owner" ? "selected" : ""}>Chủ cửa hàng</option></select></div>
    <div class="field full"><label for="u-pin">${u ? "Mã PIN mới (để trống nếu giữ nguyên)" : "Mã PIN (4–8 số)"}</label><input id="u-pin" class="in num" inputmode="numeric" type="password" ${u ? "" : "required"} minlength="4" maxlength="8"></div>
    <div class="row full"><span class="grow"></span><button type="button" class="btn" data-act="close">Hủy</button><button class="btn pri">Lưu</button></div></form>`, { focus: "#u-name" });
  $("#uForm").addEventListener("submit", async e => {
    e.preventDefault();
    if (!isOwner()) return toast("Nhờ chủ cửa hàng đổi PIN giúp", true);
    const r = await run(() => api("POST", "/api/users", { id: u?.id || "", name: $("#u-name").value, role: $("#u-role").value, pin: $("#u-pin").value }), "Đã lưu");
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
    <div class="field"><label for="pc-sepay">API token SePay</label><input id="pc-sepay" class="in" type="password" autocomplete="off" placeholder="${has.sepay ? "Đã lưu · để trống để giữ nguyên" : "dán API token"}"></div>${auto}${save}
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
  const r = await run(() => api("POST", "/api/pay/config", { provider: k, sepayToken: v("#pc-sepay"), payosClientId: v("#pc-id"), payosApiKey: v("#pc-key"), payosChecksum: v("#pc-sum"), autoBankDone: !!($("#pc-auto") && $("#pc-auto").checked) }),
    k ? "Kết nối được, đã bật tự xác nhận chuyển khoản" : "Đã tắt tự xác nhận chuyển khoản");
  b.disabled = false;
  if (r) { S.payProvSel = undefined; await refresh(true); }
};
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
  if (st.state === "active") head = `<span class="pill green">Đã kích hoạt</span> Gói <b>Đầy đủ</b> · ${l.exp ? "hết hạn " + vn(l.exp) + " (còn " + st.daysLeft + " ngày)" : "vĩnh viễn"}<div class="ps">Mã ${esc(l.id)} · ${esc(l.shop)}</div>`;
  else if (st.state === "trial") head = `<span class="pill">Dùng thử</span> Còn <b>${st.daysLeft}</b> ngày, đầy đủ tính năng`;
  else if (st.state === "revoked") head = `<span class="pill red">Đã thu hồi</span> Bản quyền ${esc(l.id)} đã bị thu hồi. Dữ liệu vẫn xem và xuất được; liên hệ Nam Linh Group (0968.646.853).`;
  else if (st.state === "expired") head = `<span class="pill red">Hết hạn</span> Bản quyền hết hạn ngày ${vn(l.exp)}. Dữ liệu vẫn xem và xuất được.`;
  else head = `<span class="pill red">Hết dùng thử</span> Dữ liệu vẫn xem và xuất được; muốn bán hàng tiếp hãy mua bản quyền.`;
  box.innerHTML = `<div>${head}</div>${st.note ? `<div class="ps" style="color:#b3261e">${esc(st.note)}</div>` : ""}
    <div class="row" style="flex-wrap:wrap;gap:8px"><a class="btn pri" href="${esc(st.buyUrl || "#")}" target="_blank" rel="noopener" data-act="licBuy">${st.state === "active" ? "Gia hạn bản quyền" : "Mua bản quyền"}</a>
      <button class="btn" data-act="licClaim">Nhận mã đã mua</button></div>
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
ACT.licBuy = b => { window.open(b.getAttribute("href"), "_blank", "noopener"); licWatch(); };
ACT.licClaim = async () => {
  const r = await run(() => api("POST", "/api/license/claim"));
  if (!r) return;
  toast(r.new ? "Đã nhận và kích hoạt bản quyền" : r.found ? "Bản quyền đã được kích hoạt" : "Chưa thấy khoản thanh toán nào cho máy này", !r.found);
  loadLic(r.status);
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
