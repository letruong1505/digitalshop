"use strict";
/* ================= HÓA ĐƠN ĐẦU VÀO =================
   Lấy hóa đơn điện tử mua vào (file XML/ZIP hoặc cổng hoadondientu.gdt.gov.vn),
   khớp với mặt hàng trong tiệm, duyệt để tạo phiếu nhập kho. */
S.inFilter = "new"; S.inList = null; S.inv = null; S.gdt = null; S.inPickI = -1;
let gdtPollT = null;

const IN_ST = { new: "Chờ duyệt", done: "Đã nhập kho", expense: "Chi phí", ignored: "Bỏ qua" };
function inChip(x) {
  if (x.status === "done") return `<span class="pill green">Đã nhập ${esc(x.receiptCode || "")}</span>`;
  if (x.status === "expense") return `<span class="pill blue">Chi phí</span>`;
  if (x.status === "ignored") return `<span class="pill mute">Bỏ qua</span>`;
  return x.ready ? `<span class="pill green">Sẵn sàng · ${x.ok}/${x.n} dòng</span>` : `<span class="pill amber">Cần xác nhận · ${x.ok}/${x.n} dòng</span>`;
}

VIEWS.inin = async function () {
  $("#main").innerHTML = `<div class="page"><div class="head"><h1>Hóa đơn đầu vào</h1>
      <label class="btn sm" for="inFiles" style="cursor:pointer">${icon("import")} Thêm file XML / ZIP</label><input type="file" id="inFiles" accept=".xml,.zip" multiple hidden>
      <button class="btn sm pri" data-act="gdtOpen">Lấy từ cổng thuế</button></div>
    <div id="gdtBar"></div>
    <div class="chips" style="margin-bottom:12px" id="inChips"></div>
    <div id="inBody"><div class="muted">Đang tải…</div></div></div>`;
  const [list, st] = await Promise.all([run(() => api("GET", "/api/inin")), api("GET", "/api/gdt/status").catch(() => null)]);
  if (S.view !== "inin") return;
  S.inList = list || []; S.gdt = st;
  inRenderList(); gdtBar();
  if (st && st.job && st.job.running) gdtPoll();
};

function inRenderList() {
  const L = S.inList || [], F = S.inFilter;
  const cnt = k => L.filter(x => x.status === k).length;
  $("#inChips").innerHTML = [["new", "Chờ duyệt"], ["done", "Đã nhập kho"], ["expense", "Chi phí"], ["ignored", "Bỏ qua"], ["all", "Tất cả"]]
    .map(([k, t]) => `<button class="chip" data-act="inFilter" data-v="${k}" aria-pressed="${F === k}">${t}${k !== "all" && cnt(k) ? ` (${cnt(k)})` : ""}</button>`).join("");
  const rows = L.filter(x => F === "all" || x.status === F);
  const readyN = L.filter(x => x.status === "new" && x.ready).length;
  const sum = rows.reduce((a, x) => a + (x.total || 0), 0);
  $("#inBody").innerHTML = !L.length ? `<label class="drop" id="inDrop" for="inFiles"><b style="font-size:16px">Chưa có hóa đơn đầu vào</b>
      <span class="muted">Bấm <b>Lấy từ cổng thuế</b> để tải hóa đơn mua vào từ hoadondientu.gdt.gov.vn,<br>hoặc kéo thả file hóa đơn điện tử (.xml, .zip) nhà cung cấp gửi qua email/Zalo vào đây.</span></label>
      <p class="ps">Mỗi hóa đơn được khớp với mặt hàng trong tiệm. Lần đầu bạn xác nhận, các lần sau cùng nhà cung cấp sẽ tự khớp. Bấm <b>Nhập kho</b> thì tồn kho, giá vốn (đã gồm thuế GTGT) và công nợ nhà cung cấp được cập nhật.</p>`
    : `${F === "new" && readyN ? `<div class="banner green" style="margin-bottom:12px"><span class="grow">${readyN} hóa đơn đã khớp đủ mặt hàng, có thể nhập kho ngay.</span><button class="btn sm pri" data-act="inBulk" data-confirm="Bấm lần nữa: nhập kho ${readyN} hóa đơn (trả tiền mặt)">Nhập kho tất cả (${readyN})</button></div>` : ""}
    ${rows.length ? `<div class="tbl"><table><thead><tr><th>Ngày</th><th>Người bán</th><th>Số hóa đơn</th><th class="r">Tổng thanh toán</th><th>Trạng thái</th><th></th></tr></thead><tbody>${rows.map(x => `<tr>
      <td class="num">${vnDate(x.date)}</td><td><div class="pn">${esc(x.sellerName || "—")}</div><div class="ps num">MST ${esc(x.sellerMst)}${x.src === "portal" ? " · từ cổng thuế" : ""}</div></td>
      <td class="num">${esc(x.form + x.serial)} · ${esc(x.no)}</td><td class="r"><b>${fmt(x.total)}</b></td><td>${inChip(x)}</td>
      <td class="r"><button class="btn sm ${x.status === "new" ? "pri" : ""}" data-act="inOpen" data-id="${esc(x.id)}">${x.status === "new" ? "Xem & duyệt" : "Xem"}</button></td></tr>`).join("")}
      </tbody><tfoot><tr><td colspan="3" class="ps">${rows.length} hóa đơn</td><td class="r"><b>${fmt(sum)}</b></td><td colspan="2"></td></tr></tfoot></table></div>`
      : `<div class="empty">Không có hóa đơn ở mục “${esc(IN_ST[F] || "")}”.</div>`}`;
  const drop = $("#main .page");
  if (drop && !drop._dnd) {
    drop._dnd = true;
    drop.addEventListener("dragover", e => { if (e.dataTransfer && [...e.dataTransfer.types].includes("Files")) { e.preventDefault(); const d = $("#inDrop"); if (d) d.classList.add("over"); } });
    drop.addEventListener("drop", e => { if (!e.dataTransfer || !e.dataTransfer.files.length) return; e.preventDefault(); inUpload(e.dataTransfer.files); });
  }
}
ACT.inFilter = b => { S.inFilter = b.dataset.v; inRenderList(); };
ON_CHANGE.inFiles = el => { if (el.files.length) inUpload(el.files); el.value = ""; };
async function inUpload(files) {
  const fd = new FormData(); for (const f of files) fd.append("files", f);
  toast(`Đang đọc ${files.length} file…`);
  const r = await run(() => api("POST", "/api/inin/upload", fd));
  if (!r) return;
  const msg = `Thêm ${r.added ? r.added.length : 0} hóa đơn${r.dup ? `, ${r.dup} hóa đơn đã có` : ""}${r.errs && r.errs.length ? `, ${r.errs.length} file lỗi` : ""}`;
  toast(msg, !(r.added && r.added.length) && r.errs && r.errs.length);
  if (r.errs && r.errs.length) openSheet(`<div class="sheet-h"><h2>Có file không đọc được</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div><p class="muted">${esc(msg)}.</p><ul>${r.errs.slice(0, 20).map(e => `<li class="ps">${esc(e)}</li>`).join("")}</ul><div class="row"><span class="grow"></span><button class="btn pri" data-act="close">Đóng</button></div>`);
  S.inFilter = "new"; await refresh(true); if (S.view === "inin") VIEWS.inin();
}

/* ---------- chi tiết & khớp hàng ---------- */
function guessRatioJS(name) {
  const n = norm(name);
  const res = [/\b(?:thung|th|t|ket|k|loc|l|hop|h|bich|goi|x|bo|vi|kien|bao)\s*[x/*]?\s*(\d{1,3})\b/, /\b(\d{1,3})\s*(?:lon|chai|goi|hop|cai|vi|tui|bich|hu|lo|cuon|qua|banh|thanh|ly|coc|tuyp|chiec)\b/];
  for (const re of res) { const m = n.match(re); if (m && +m[1] >= 2 && +m[1] <= 200) return +m[1]; }
  return 0;
}
function unitForJS(p, invUnit, name) {
  const k = norm(invUnit).replace(/[^a-z0-9]/g, "");
  const kk = s => norm(s).replace(/[^a-z0-9]/g, "");
  if (!k || k === kk(p.unit)) return { punit: p.unit, ratio: 1, sure: true };
  for (const u of p.units || []) if (kk(u.name) === k) return { punit: u.name, ratio: u.ratio, sure: true };
  const g = guessRatioJS(name);
  if (g) return { punit: invUnit.trim().toLowerCase(), ratio: g, sure: false };
  return { punit: p.unit, ratio: 1, sure: false };
}
ACT.inOpen = async b => {
  const r = await run(() => api("GET", "/api/inin/" + b.dataset.id)); if (!r) return;
  const inv = r.inv;
  S.inv = { inv, pay: "cash", sel: inv.lines.map(l => ({ pid: l.pid || "", punit: l.punit || "", ratio: l.ratio || 1, skip: !!l.skip, new: !!l.new, auto: l.auto || "" })) };
  inRenderDetail();
};
function inLineHtml(l, i) {
  const s = S.inv.sel[i], ro = S.inv.inv.status !== "new";
  const head = `<div class="pn">${esc(l.name)}</div><div class="ps num">${fq(l.qty)} ${esc(l.unit)} × ${fmt(l.price)}${l.rateTx ? ` · thuế ${esc(l.rateTx)}` : ""}${l.kind === 2 ? ' · <span class="pill blue">khuyến mại</span>' : ""}</div>`;
  const cost = `<td class="r num">${l.kind === 1 ? fmt(l.cost) : "0"}<div class="ps">/${esc(l.unit || "đv")}</div></td>`;
  let mid;
  if (s.skip) mid = `<span class="pill mute">Không nhập kho</span>${ro ? "" : ` <button class="btn sm ghost" data-act="inUndo" data-i="${i}">Hoàn tác</button>`}`;
  else if (s.new) mid = `<span class="pill blue">Tạo mặt hàng mới</span> <span class="ps">“${esc(l.name)}”, đơn vị ${esc((l.unit || "cái").toLowerCase())}</span>${ro ? "" : ` <button class="btn sm ghost" data-act="inUndo" data-i="${i}">Hoàn tác</button>`}`;
  else if (s.pid && S.products.get(s.pid)) {
    const p = S.products.get(s.pid);
    const tag = { map: '<span class="pill green">Đã nhớ</span>', code: '<span class="pill green">Trùng mã vạch</span>', user: '<span class="pill green">Đã chọn</span>', guess: '<span class="pill amber">Gợi ý, kiểm tra lại</span>' }[s.auto] || "";
    const conv = s.ratio > 1 || norm(s.punit) !== norm(p.unit)
      ? `<div class="row" style="gap:6px;margin-top:6px;flex-wrap:nowrap"><span class="ps">1 ${esc(l.unit || s.punit)} =</span>${ro ? `<b>${fq(s.ratio)}</b>` : `<input class="in sm r num" style="width:64px" data-in="inRatio" data-i="${i}" value="${fq(s.ratio)}" inputmode="decimal">`}<span class="ps">${esc(p.unit)} → nhập ${fq(l.qty * s.ratio)} ${esc(p.unit)}</span></div>`
      : `<div class="ps" style="margin-top:4px">nhập ${fq(l.qty)} ${esc(p.unit)}</div>`;
    mid = `<div class="row" style="gap:6px"><b>${esc(p.name)}</b> ${tag}${ro ? "" : ` <button class="btn sm ghost" data-act="inChange" data-i="${i}">Đổi</button>${s.auto === "guess" ? ` <button class="btn sm" data-act="inOk" data-i="${i}">Đúng</button>` : ""}`}</div>${conv}`;
  } else {
    const cands = (l.cands || []).filter(c => S.products.get(c.pid)).slice(0, 3);
    mid = ro ? `<span class="ps">—</span>` : `<div class="row" style="gap:6px">${cands.map(c => `<button class="btn sm" data-act="inCand" data-i="${i}" data-id="${esc(c.pid)}">${esc(c.name)}</button>`).join("")}</div>
      <div class="search" style="position:relative;margin-top:6px">${icon("search")}<input class="in sm" data-in="inQ" data-i="${i}" autocomplete="off" placeholder="Tìm mặt hàng trong tiệm"><div class="dd" id="inDD-${i}" hidden style="left:0;right:0;top:100%"></div></div>
      <div class="row" style="gap:6px;margin-top:6px"><button class="btn sm" data-act="inNew" data-i="${i}">+ Tạo mặt hàng mới</button><button class="btn sm ghost" data-act="inSkip" data-i="${i}">Không nhập kho</button></div>`;
  }
  return `<tr><td style="min-width:220px">${head}</td>${cost}<td style="min-width:300px">${mid}</td></tr>`;
}
function inRenderDetail() {
  const X = S.inv, v = X.inv, ro = v.status !== "new";
  const goods = v.lines.map((l, i) => [l, i]).filter(([l]) => l.kind === 1 || l.kind === 2);
  const other = v.lines.filter(l => l.kind === 3 || l.kind === 4);
  const html = `<div class="sheet-h"><h2>Hóa đơn ${esc(v.form + v.serial)} số ${esc(v.no)}</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
    <div class="grid2" style="margin-bottom:10px"><div><div class="lbl">Người bán</div><b>${esc(v.sellerName)}</b><div class="ps">MST ${esc(v.sellerMst)} · ${esc(v.sellerAddr || "")}</div></div>
      <div><div class="lbl">Ngày lập · tổng thanh toán</div><b class="num">${vnDate(v.date)} · ${money(v.total)}</b><div class="ps">${esc(v.title || "")}${v.vat ? ` · tiền thuế ${money(v.vat)}` : ""}</div></div></div>
    <details style="margin-bottom:10px"><summary><b>Nội dung hóa đơn gốc</b> <span class="ps">(người mua, từng dòng, thuế, tổng tiền)</span></summary>
      <div class="ps" style="margin:6px 0">Người mua: <b>${esc(v.buyerName || "—")}</b>${v.buyerMst ? " · MST " + esc(v.buyerMst) : ""}${v.pay ? " · Hình thức thanh toán: " + esc(v.pay) : ""}</div>
      <div class="tbl"><table><thead><tr><th>STT</th><th>Tên hàng hóa, dịch vụ</th><th>ĐVT</th><th class="r">Số lượng</th><th class="r">Đơn giá</th><th class="r">Thành tiền</th><th class="r">Thuế suất</th></tr></thead><tbody>${v.lines.map((l, i) => `<tr><td class="num">${i + 1}</td><td>${esc(l.name)}${l.kind === 2 ? ' <span class="pill blue">khuyến mại</span>' : l.kind === 3 ? ' <span class="pill mute">chiết khấu</span>' : l.kind === 4 ? ' <span class="pill mute">ghi chú</span>' : ""}</td><td>${esc(l.unit || "")}</td><td class="r num">${l.kind === 1 || l.kind === 2 ? fq(l.qty) : ""}</td><td class="r num">${l.kind === 1 || l.kind === 2 ? fmt(l.price) : ""}</td><td class="r num">${l.amount ? fmt(l.amount) : ""}</td><td class="r">${esc(l.rateTx || "")}</td></tr>`).join("")}</tbody>
        <tfoot><tr><td colspan="5" class="r">Cộng tiền hàng</td><td class="r num">${fmt(v.pre)}</td><td></td></tr><tr><td colspan="5" class="r">Tiền thuế GTGT</td><td class="r num">${fmt(v.vat)}</td><td></td></tr><tr><td colspan="5" class="r"><b>Tổng thanh toán</b></td><td class="r num"><b>${fmt(v.total)}</b></td><td></td></tr></tfoot></table></div></details>
    ${goods.length ? `<div class="tbl"><table><thead><tr><th>Trên hóa đơn</th><th class="r">Giá vốn</th><th>Mặt hàng trong tiệm</th></tr></thead><tbody>${goods.map(([l, i]) => inLineHtml(l, i)).join("")}</tbody></table></div>`
      : `<div class="empty">Hóa đơn không có dòng hàng hóa để nhập kho (dịch vụ, chi phí).</div>`}
    ${other.length ? `<p class="ps">${other.map(l => `${l.kind === 3 ? "Chiết khấu" : "Ghi chú"}: ${esc(l.name)}${l.amount ? " " + money(l.amount) : ""}`).join(" · ")}. Giá vốn đã trừ chiết khấu và cộng thuế GTGT.</p>` : `<p class="ps">Giá vốn đã cộng thuế GTGT (hộ kinh doanh không được khấu trừ thuế đầu vào).</p>`}
    ${ro ? `<div class="row">${v.status === "done" ? `<span class="pill green">Đã nhập kho phiếu ${esc(v.receiptCode)}</span><span class="ps">Muốn sửa: vào Nhập hàng, hủy phiếu này rồi duyệt lại.</span>` : `<span class="pill ${v.status === "expense" ? "blue" : "mute"}">${IN_ST[v.status]}</span>`}<span class="grow"></span>${v.status !== "done" ? `<button class="btn" data-act="inStatus" data-v="new">Đưa về chờ duyệt</button>` : ""}<button class="btn pri" data-act="close">Đóng</button></div>`
    : `<div class="row" style="align-items:end"><div class="field" style="margin:0"><label>Thanh toán cho nhà cung cấp</label><div class="seg">${[["cash", "Đã trả tiền mặt"], ["bank", "Đã chuyển khoản"], ["debt", "Ghi nợ"]].map(([k, t]) => `<button data-act="inPay" data-v="${k}" aria-pressed="${X.pay === k}">${t}</button>`).join("")}</div></div>
      <span class="grow"></span><button class="btn ghost" data-act="inStatus" data-v="ignored">Bỏ qua hóa đơn</button><button class="btn" data-act="inStatus" data-v="expense">Ghi là chi phí</button><button class="btn pri" data-act="inApprove">Nhập kho</button></div>`}`;
  const sc = $("#sheetCard"); const top = sheetOpen() && sc ? sc.scrollTop : 0;
  openSheet(html, { wide: true });
  if (top) $("#sheetCard").scrollTop = top;
}
const inSel = b => S.inv.sel[+b.dataset.i];
function inChoose(i, pid, auto) {
  const p = S.products.get(pid), l = S.inv.inv.lines[i], s = S.inv.sel[i];
  const u = unitForJS(p, l.unit, l.name);
  Object.assign(s, { pid, punit: u.punit, ratio: u.ratio, skip: false, new: false, auto: auto || "user" });
  inRenderDetail();
}
ACT.inCand = b => inChoose(+b.dataset.i, b.dataset.id);
ACT.inOk = b => { inSel(b).auto = "user"; inRenderDetail(); };
ACT.inChange = b => { Object.assign(inSel(b), { pid: "", auto: "" }); inRenderDetail(); const q = $(`[data-in="inQ"][data-i="${b.dataset.i}"]`); if (q) q.focus(); };
ACT.inNew = b => { Object.assign(inSel(b), { pid: "", new: true, skip: false }); inRenderDetail(); };
ACT.inSkip = b => { Object.assign(inSel(b), { pid: "", skip: true, new: false }); inRenderDetail(); };
ACT.inUndo = b => { Object.assign(inSel(b), { skip: false, new: false, pid: "" }); inRenderDetail(); };
ON_INPUT.inQ = el => { S.inPickI = +el.dataset.i; const d = $("#inDD-" + el.dataset.i); d.hidden = !el.value.trim(); d.innerHTML = pickList("product", el.value, "inPick", 8); };
ACT.inPick = b => { const p = S.products.get(b.dataset.id); if (!p || p.isCombo) return toast("Không nhập kho cho combo", true); inChoose(S.inPickI, p.id); };
ON_INPUT.inRatio = el => { const s = S.inv.sel[+el.dataset.i]; s.ratio = parseQty(el.value) || 1; if (s.auto === "guess") s.auto = "user"; };
ON_CHANGE.inRatio = () => inRenderDetail();
ACT.inPay = b => { S.inv.pay = b.dataset.v; $$('[data-act="inPay"]').forEach(x => x.setAttribute("aria-pressed", x.dataset.v === b.dataset.v)); };
function inSelPayload() {
  return S.inv.sel.map((s, i) => ({ i, pid: s.pid, punit: s.punit, ratio: s.ratio, skip: s.skip, new: s.new })).filter((s, i) => { const k = S.inv.inv.lines[i].kind; return k === 1 || k === 2; });
}
ACT.inApprove = async () => {
  const X = S.inv;
  const miss = X.inv.lines.findIndex((l, i) => (l.kind === 1 || l.kind === 2) && !X.sel[i].pid && !X.sel[i].skip && !X.sel[i].new);
  if (miss >= 0) return toast(`Dòng “${X.inv.lines[miss].name}” chưa chọn mặt hàng`, true);
  const r = await run(() => api("POST", `/api/inin/${X.inv.id}/approve`, { lines: inSelPayload(), pay: X.pay }), x => `Đã nhập kho ${x.code} · ${money(x.total)}`);
  if (r) { S.inv = null; closeSheet(); await refresh(true); if (S.view === "inin") VIEWS.inin(); }
};
ACT.inStatus = async b => {
  if (await run(() => api("POST", `/api/inin/${S.inv.inv.id}/status`, { status: b.dataset.v }), b.dataset.v === "new" ? "Đã đưa về chờ duyệt" : b.dataset.v === "expense" ? "Đã ghi là chi phí" : "Đã bỏ qua hóa đơn")) {
    S.inv = null; closeSheet(); await refresh(true); if (S.view === "inin") VIEWS.inin();
  }
};
ACT.inBulk = async () => {
  const ids = (S.inList || []).filter(x => x.status === "new" && x.ready).map(x => x.id);
  let ok = 0, fail = 0;
  for (const id of ids) {
    try {
      const d = await api("GET", "/api/inin/" + id);
      if (!d.ready) { fail++; continue; }
      await api("POST", `/api/inin/${id}/approve`, { lines: d.inv.lines.map((l, i) => ({ i, pid: l.pid || "", punit: l.punit || "", ratio: l.ratio || 1, skip: !!l.skip, new: !!l.new })).filter((x, i) => [1, 2].includes(d.inv.lines[i].kind)), pay: "cash" });
      ok++;
    } catch (e) { fail++; }
  }
  toast(`Đã nhập kho ${ok} hóa đơn${fail ? `, ${fail} hóa đơn cần xem lại` : ""}`, fail > 0 && !ok);
  await refresh(true); if (S.view === "inin") VIEWS.inin();
};

/* ---------- cổng hoadondientu.gdt.gov.vn ---------- */
function gdtBar() {
  const el = $("#gdtBar"); if (!el) return;
  const st = S.gdt, j = st && st.job;
  if (!j || !j.started) { el.innerHTML = ""; return; }
  const errs = (j.errs || []).length ? `<details style="margin-top:6px"><summary class="ps">${j.errs.length} lỗi</summary><ul>${j.errs.map(e => `<li class="ps">${esc(e)}</li>`).join("")}</ul><p class="ps">Chi tiết kỹ thuật ghi trong file TapHoaX.log ở thư mục dữ liệu (Cài đặt → Mở thư mục dữ liệu).</p></details>` : "";
  el.innerHTML = `<div class="banner ${j.running ? "" : j.errs && j.errs.length && !j.added ? "red" : "green"}" style="margin-bottom:12px;display:block">
    <div class="row"><b class="grow">${j.running ? esc(j.phase) : j.expired ? "Phiên đăng nhập cổng thuế đã hết hạn" : "Đã lấy xong hóa đơn từ cổng thuế"}</b><span class="ps num">${vnDate(j.from)} – ${vnDate(j.to)}</span></div>
    <div class="ps">Tìm thấy ${j.found} hóa đơn · tải chi tiết ${j.done}/${j.todo || 0} · thêm mới ${j.added} · đã có từ trước ${j.dup}${j.fail ? ` · lỗi ${j.fail}` : ""}</div>${errs}</div>`;
}
function gdtPoll() {
  clearInterval(gdtPollT);
  gdtPollT = setInterval(async () => {
    if (S.view !== "inin") { clearInterval(gdtPollT); return; }
    try { S.gdt = await api("GET", "/api/gdt/status"); } catch (e) { return; }
    gdtBar();
    if (!S.gdt.job.running) { clearInterval(gdtPollT); await refresh(true); const l = await api("GET", "/api/inin").catch(() => null); if (l && S.view === "inin") { S.inList = l; inRenderList(); } }
  }, 1500);
}
function gdtDefaultRange() {
  const d = new Date(); const from = new Date(d.getFullYear(), d.getMonth() - 1, 1);
  return [dkey(from), dkey(d)];
}
ACT.gdtOpen = async () => {
  const st = await run(() => api("GET", "/api/gdt/status")); if (!st) return;
  S.gdt = st;
  const [a, b] = S.gdtRange || gdtDefaultRange();
  const range = `<div class="grid2"><div class="field"><label for="gdtFrom">Từ ngày (ngày lập hóa đơn)</label><input type="date" id="gdtFrom" class="in" value="${a}"></div><div class="field"><label for="gdtTo">Đến ngày</label><input type="date" id="gdtTo" class="in" value="${b}"></div></div>`;
  const note = `<p class="ps">Digital Shop chỉ <b>đọc</b> hóa đơn mua vào trên cổng, không thay đổi gì. Mật khẩu gửi thẳng tới cổng thuế, không được lưu lại và không ghi vào nhật ký.</p>`;
  openSheet(st.loggedIn
    ? `<div class="sheet-h"><h2>Lấy hóa đơn từ cổng thuế</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
      <p>Đang đăng nhập <b class="num">${esc(st.user)}</b> trên hoadondientu.gdt.gov.vn.</p>${range}${note}
      <div class="row"><button class="btn ghost" data-act="gdtLogout">Đăng xuất</button><span class="grow"></span><button class="btn" data-act="close">Đóng</button><button class="btn pri" data-act="gdtFetch">Lấy hóa đơn</button></div>`
    : `<div class="sheet-h"><h2>Đăng nhập hoadondientu.gdt.gov.vn</h2><button class="x" data-act="close" aria-label="Đóng">×</button></div>
      <p class="ps" style="margin-top:0">Dùng tài khoản tra cứu hóa đơn của cửa hàng (tên đăng nhập là mã số thuế).</p>
      <form id="gdtForm" autocomplete="off"><div class="grid2"><div class="field"><label for="gdtUser">Mã số thuế / tên đăng nhập</label><input id="gdtUser" class="in num" value="${esc(S.settings.gdtUser || "")}" required></div>
        <div class="field"><label for="gdtPass">Mật khẩu</label><input id="gdtPass" type="password" class="in" required autocomplete="new-password"></div></div>
      <div class="field"><label for="gdtCap">Mã captcha (gõ đúng chữ trong ảnh)</label><div class="row" style="flex-wrap:nowrap;gap:8px"><div id="gdtCapImg" style="background:#fff;border:1px solid var(--line);border-radius:8px;width:200px;height:52px;display:grid;place-items:center;flex:none"><span class="ps">Đang tải…</span></div>
        <button type="button" class="btn sm" data-act="gdtCaptcha" title="Đổi ảnh khác">Đổi ảnh</button><input id="gdtCap" class="in" required autocomplete="off" style="max-width:160px"></div></div>
      ${range}${note}
      <div class="row"><span class="grow"></span><button type="button" class="btn" data-act="close">Đóng</button><button class="btn pri" type="submit">Đăng nhập & lấy hóa đơn</button></div></form>`, { focus: st.loggedIn ? null : (S.settings.gdtUser ? "#gdtPass" : "#gdtUser") });
  if (!st.loggedIn) {
    gdtCaptcha();
    $("#gdtForm").addEventListener("submit", async e => {
      e.preventDefault();
      const btn = e.submitter || $("#gdtForm button[type=submit]"); btn.disabled = true;
      const r = await run(() => api("POST", "/api/gdt/login", { user: $("#gdtUser").value, pass: $("#gdtPass").value, cvalue: $("#gdtCap").value, ckey: S.gdtCkey }));
      btn.disabled = false;
      if (!r) { $("#gdtCap").value = ""; gdtCaptcha(); $("#gdtCap").focus(); return; }
      S.settings.gdtUser = $("#gdtUser").value.trim();
      gdtFetch();
    });
  }
};
async function gdtCaptcha() {
  const box = $("#gdtCapImg"); if (!box) return;
  box.innerHTML = '<span class="ps">Đang tải…</span>';
  try {
    const c = await api("GET", "/api/gdt/captcha");
    S.gdtCkey = c.key;
    box.innerHTML = `<img alt="Mã captcha" style="max-width:100%;max-height:50px" src="data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(c.svg)))}">`;
  } catch (e) { box.innerHTML = `<span class="ps" style="color:var(--red);padding:4px;text-align:center">${esc(e.message)}</span>`; }
}
ACT.gdtCaptcha = () => gdtCaptcha();
ACT.gdtLogout = async () => { await run(() => api("POST", "/api/gdt/logout"), "Đã đăng xuất cổng thuế"); closeSheet(); };
async function gdtFetch() {
  const from = $("#gdtFrom").value, to = $("#gdtTo").value;
  S.gdtRange = [from, to];
  const st = await run(() => api("POST", "/api/gdt/fetch", { from, to }));
  if (!st) return;
  closeSheet(); S.gdt = st; S.inFilter = "new";
  if (S.view !== "inin") go("inin"); else gdtBar();
  gdtPoll();
}
ACT.gdtFetch = () => gdtFetch();
