// State: server-derived + local selected preset (persisted in localStorage).
let TARGET_W = 1072;
let TARGET_H = 1448;
let TARGET_ASPECT = TARGET_W / TARGET_H;
const SEL_KEY = "kss.selected";

let state = { device: null, presets: [], active: "", unclaimed: 0 };
let selected = localStorage.getItem(SEL_KEY) || "";

// ---- DOM ------------------------------------------------------------------

const metaEl = document.getElementById("meta");

const claimBanner = document.getElementById("claim-banner");
const claimCount = document.getElementById("claim-count");
const claimName = document.getElementById("claim-name");
const claimForm = document.getElementById("claim-form");

const emptyBanner = document.getElementById("empty-banner");
const firstName = document.getElementById("first-name");
const firstForm = document.getElementById("first-form");

const presetsSection = document.getElementById("presets");
const tabsEl = document.getElementById("tabs");
const activateBtn = document.getElementById("activate-btn");
const renameBtn = document.getElementById("rename-btn");
const deleteBtn = document.getElementById("delete-btn");
const newPresetBtn = document.getElementById("new-preset-btn");

const grid = document.getElementById("grid");
const fileInput = document.getElementById("file");
const picker = document.querySelector(".picker");

const importPath = document.getElementById("import-path");
const importName = document.getElementById("import-name");
const importForm = document.getElementById("import-form");

const modal = document.getElementById("modal");
const modalImg = document.getElementById("modal-img");
const modalClose = document.getElementById("modal-close");

// ---- helpers --------------------------------------------------------------

async function api(method, url, body) {
  const opts = { method };
  if (body !== undefined) {
    opts.headers = { "Content-Type": "application/json" };
    opts.body = JSON.stringify(body);
  }
  const r = await fetch(url, opts);
  if (!r.ok) throw new Error((await r.text()).trim() || `${r.status}`);
  return r;
}

// ---- generic dialog (replaces window prompt/confirm/alert) ----------------

const dlg = document.getElementById("dialog");
const dlgTitle = document.getElementById("dialog-title");
const dlgMessage = document.getElementById("dialog-message");
const dlgInput = document.getElementById("dialog-input");
const dlgForm = document.getElementById("dialog-form");
const dlgConfirm = document.getElementById("dialog-confirm");
const dlgCancel = document.getElementById("dialog-cancel");

let dlgResolve = null;

function openDialog({
  title,
  message = "",
  input = false,
  value = "",
  placeholder = "",
  confirmText = "OK",
  danger = false,
  hideCancel = false,
}) {
  return new Promise((resolve) => {
    dlgResolve = resolve;
    dlgTitle.textContent = title;
    dlgMessage.textContent = message;
    dlgMessage.hidden = !message;
    dlgInput.hidden = !input;
    dlgInput.value = value;
    dlgInput.placeholder = placeholder;
    dlgConfirm.textContent = confirmText;
    dlgConfirm.classList.toggle("danger", danger);
    dlgCancel.hidden = hideCancel;
    dlg.hidden = false;
    document.body.classList.add("locked");
    if (input)
      setTimeout(() => {
        dlgInput.focus();
        dlgInput.select();
      }, 0);
    else setTimeout(() => dlgConfirm.focus(), 0);
  });
}

function closeDialog(value) {
  dlg.hidden = true;
  document.body.classList.remove("locked");
  const r = dlgResolve;
  dlgResolve = null;
  if (r) r(value);
}

dlgForm.addEventListener("submit", (e) => {
  e.preventDefault();
  closeDialog(dlgInput.hidden ? true : dlgInput.value.trim());
});
dlgCancel.addEventListener("click", () => closeDialog(null));
dlg.querySelector(".dialog-backdrop").addEventListener("click", () => closeDialog(null));
window.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && !dlg.hidden) closeDialog(null);
});

async function promptDialog(title, opts = {}) {
  const v = await openDialog({ title, input: true, ...opts });
  return typeof v === "string" ? v : null;
}
async function confirmDialog(title, opts = {}) {
  return (await openDialog({ title, ...opts })) === true;
}
async function alertDialog(message) {
  await openDialog({
    title: "Error",
    message,
    confirmText: "OK",
    hideCancel: true,
  });
}

// ---- state load + render --------------------------------------------------

async function refresh() {
  try {
    const r = await fetch("api/state");
    state = await r.json();
  } catch (e) {
    metaEl.textContent = `failed to load: ${e.message}`;
    return;
  }

  TARGET_W = state.device.w;
  TARGET_H = state.device.h;
  TARGET_ASPECT = TARGET_W / TARGET_H;
  document.documentElement.style.setProperty("--card-aspect", `${TARGET_W} / ${TARGET_H}`);

  const names = state.presets.map((p) => p.name);
  if (!names.includes(selected)) {
    selected = state.active || names[0] || "";
  }
  if (selected) localStorage.setItem(SEL_KEY, selected);

  renderMeta();
  renderBanners();
  syncTabs();
  renderActions();
  await syncGridFromServer();
}

function renderMeta() {
  const d = state.device;
  const parts = [d.model];
  if (!d.model.includes("×")) parts.push(`${d.w}×${d.h}`);
  parts.push(d.grayscale ? `${d.bpp}-bit gray` : `${d.bpp}-bit color`);
  metaEl.textContent = parts.join(" · ");
}

function renderBanners() {
  const showClaim = state.unclaimed > 0;
  const showEmpty = !showClaim && state.presets.length === 0;
  claimBanner.hidden = !showClaim;
  emptyBanner.hidden = !showEmpty;
  presetsSection.hidden = state.presets.length === 0;
  if (showClaim) claimCount.textContent = state.unclaimed;
}

// Diff-render tabs: reuse existing DOM nodes by preset name so non-changes
// don't flicker. Only attributes/text get updated.
function syncTabs() {
  const have = new Map();
  for (const node of tabsEl.children) have.set(node.dataset.name, node);

  let prev = null;
  for (const p of state.presets) {
    let node = have.get(p.name);
    if (!node) {
      node = document.createElement("button");
      node.className = "tab";
      node.dataset.name = p.name;
      node.innerHTML = `<span class="tab-name"></span><span class="count"></span>`;
      node.addEventListener("click", () => {
        const next = node.dataset.name;
        if (next === selected) return;
        selected = next;
        localStorage.setItem(SEL_KEY, selected);
        syncTabs();
        renderActions();
        syncGridFromServer();
      });
    } else {
      have.delete(p.name);
    }
    node.querySelector(".tab-name").textContent = p.name;
    node.querySelector(".count").textContent = p.files;
    node.classList.toggle("selected", p.name === selected);
    node.classList.toggle("applied", p.name === state.active);
    const target = prev ? prev.nextSibling : tabsEl.firstChild;
    if (node !== target) tabsEl.insertBefore(node, target);
    prev = node;
  }
  for (const node of have.values()) node.remove();
}

function renderActions() {
  const present = state.presets.some((p) => p.name === selected);
  const isActive = present && selected === state.active;
  activateBtn.disabled = !present || isActive;
  activateBtn.textContent = isActive ? "Applied" : "Apply";
  activateBtn.classList.toggle("applied", isActive);
  activateBtn.classList.toggle("primary", present && !isActive);
  renameBtn.disabled = !present;
  deleteBtn.disabled = !present || isActive;
}

async function syncGridFromServer() {
  if (!selected) {
    grid.innerHTML = "";
    grid.dataset.preset = "";
    return;
  }
  let items;
  try {
    const r = await fetch(`api/presets/${encodeURIComponent(selected)}/files`);
    items = (await r.json()) || [];
  } catch (e) {
    alertDialog(`Couldn't list files: ${e.message}`);
    return;
  }
  items.sort((a, b) => a.name.localeCompare(b.name));
  syncGrid(items);
}

// Diff-render grid by filename. A different preset means full reset; otherwise
// only added/removed cards touch the DOM, so existing <img> elements (and their
// caches) stay put — no flash, no reload.
function syncGrid(items) {
  if (grid.dataset.preset !== selected) {
    grid.innerHTML = "";
    grid.dataset.preset = selected;
  }
  const have = new Map();
  for (const node of grid.children) {
    if (node.dataset.name) have.set(node.dataset.name, node);
  }
  const emptyNode = grid.querySelector(".grid-empty");
  if (items.length === 0) {
    for (const n of have.values()) n.remove();
    if (!emptyNode) {
      const empty = document.createElement("div");
      empty.className = "grid-empty";
      empty.textContent = "No wallpapers in this preset yet.";
      grid.appendChild(empty);
    }
    return;
  }
  if (emptyNode) emptyNode.remove();

  const preset = selected;
  let prev = null;
  for (const it of items) {
    let node = have.get(it.name);
    if (!node) {
      const url = `api/presets/${encodeURIComponent(preset)}/files/${encodeURIComponent(it.name)}`;
      node = document.createElement("div");
      node.className = "card";
      node.dataset.name = it.name;
      node.innerHTML = `
        <img src="${url}" loading="lazy" alt="${it.name}">
        <div class="name">
          <span>${it.name}</span>
          <button class="del" title="delete">×</button>
        </div>`;
      const fname = it.name;
      node.addEventListener("click", (e) => {
        if (e.target.classList.contains("del")) return;
        openPreview(url);
      });
      node.querySelector(".del").addEventListener("click", (e) => {
        e.stopPropagation();
        deleteFile(fname);
      });
    } else {
      have.delete(it.name);
    }
    const target = prev ? prev.nextSibling : grid.firstChild;
    if (node !== target) grid.insertBefore(node, target);
    prev = node;
  }
  for (const node of have.values()) node.remove();
}

// ---- preview modal --------------------------------------------------------

function openPreview(url) {
  modalImg.src = url;
  modal.hidden = false;
}
function closePreview() {
  modal.hidden = true;
  modalImg.src = "";
}
modal.addEventListener("click", (e) => {
  if (e.target === modal || e.target.classList.contains("modal-backdrop")) closePreview();
});
modalClose.addEventListener("click", closePreview);
window.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && !modal.hidden) closePreview();
});

// ---- preset actions -------------------------------------------------------

claimForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = claimName.value.trim();
  if (!name) return;
  try {
    await api("POST", "api/claim", { name });
    selected = name;
    await refresh();
  } catch (err) {
    alertDialog(err.message);
  }
});

firstForm.addEventListener("submit", (e) => {
  e.preventDefault();
  createPreset(firstName.value.trim());
});

newPresetBtn.addEventListener("click", async () => {
  const name = await promptDialog("New preset", {
    placeholder: "preset name",
    confirmText: "Create",
  });
  if (name) createPreset(name);
});

async function createPreset(name) {
  if (!name) return;
  try {
    await api("POST", "api/presets", { name });
    selected = name;
    firstName.value = "";
    await refresh();
  } catch (err) {
    alertDialog(err.message);
  }
}

activateBtn.addEventListener("click", async () => {
  if (!selected || selected === state.active) return;
  try {
    await api("POST", `api/presets/${encodeURIComponent(selected)}/activate`);
    state.active = selected;
    syncTabs();
    renderActions();
    renderMeta();
  } catch (err) {
    alertDialog(err.message);
  }
});

renameBtn.addEventListener("click", async () => {
  if (!selected) return;
  const to = await promptDialog(`Rename "${selected}"`, {
    value: selected,
    confirmText: "Rename",
  });
  if (!to || to === selected) return;
  try {
    await api("POST", `api/presets/${encodeURIComponent(selected)}/rename`, {
      to,
    });
    const wasActive = selected === state.active;
    selected = to;
    if (wasActive) state.active = to;
    await refresh();
  } catch (err) {
    alertDialog(err.message);
  }
});

deleteBtn.addEventListener("click", async () => {
  if (!selected) return;
  const ok = await confirmDialog(`Delete preset "${selected}"?`, {
    message: "All wallpapers in this preset will be removed.",
    confirmText: "Delete",
    danger: true,
  });
  if (!ok) return;
  try {
    await api("DELETE", `api/presets/${encodeURIComponent(selected)}`);
    selected = "";
    await refresh();
  } catch (err) {
    alertDialog(err.message);
  }
});

async function deleteFile(name) {
  const ok = await confirmDialog(`Delete ${name}?`, {
    confirmText: "Delete",
    danger: true,
  });
  if (!ok) return;
  try {
    await api(
      "DELETE",
      `api/presets/${encodeURIComponent(selected)}/files/${encodeURIComponent(name)}`,
    );
    // Remove just this card; no full refresh, no flicker.
    const card = grid.querySelector(`.card[data-name="${CSS.escape(name)}"]`);
    if (card) card.remove();
    const preset = state.presets.find((p) => p.name === selected);
    if (preset) preset.files = Math.max(0, preset.files - 1);
    syncTabs();
    if (!grid.querySelector(".card")) syncGrid([]);
  } catch (err) {
    alertDialog(err.message);
  }
}

// ---- import ---------------------------------------------------------------

importForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const path = importPath.value.trim();
  const name = importName.value.trim();
  if (!path || !name) return;
  try {
    await api("POST", "api/import", { path, name });
    importPath.value = "";
    importName.value = "";
    selected = name;
    await refresh();
  } catch (err) {
    alertDialog(err.message);
  }
});

// ---- upload ---------------------------------------------------------------

picker.addEventListener("click", () => fileInput.click());
picker.addEventListener("dragover", (e) => e.preventDefault());
picker.addEventListener("drop", (e) => {
  e.preventDefault();
  if (e.dataTransfer?.files) processFiles(e.dataTransfer.files);
});
fileInput.addEventListener("change", (e) => {
  processFiles(e.target.files);
  fileInput.value = "";
});

async function processFiles(files) {
  if (!files || files.length === 0) return;
  if (!selected) {
    alertDialog("Select or create a preset first.");
    return;
  }
  let existing;
  try {
    existing = await fetchExistingNames(selected);
  } catch (err) {
    alertDialog(err.message);
    return;
  }
  let next = nextFreeNumber(existing);
  let done = 0;
  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    try {
      const crop = await cropInteractive(file, i + 1, files.length);
      if (crop === "cancel") break;
      if (crop === null) continue;
      const png = await processToKindlePNG(file, crop);
      const name = `bg_ss${String(next).padStart(2, "0")}.png`;
      next++;
      const fd = new FormData();
      fd.append("file", new File([png], name, { type: "image/png" }));
      const up = await fetch(`api/presets/${encodeURIComponent(selected)}/files`, {
        method: "POST",
        body: fd,
      });
      if (!up.ok) throw new Error((await up.text()).trim() || `upload failed (${up.status})`);
      done++;
    } catch (err) {
      alertDialog(`${file.name}: ${err.message}`);
      break;
    }
  }
  if (done > 0) await refresh();
}

async function fetchExistingNames(preset) {
  const r = await fetch(`api/presets/${encodeURIComponent(preset)}/files`);
  const items = (await r.json()) || [];
  return new Set(items.map((i) => i.name));
}

function nextFreeNumber(existing) {
  for (let n = 0; n < 999; n++) {
    if (!existing.has(`bg_ss${String(n).padStart(2, "0")}.png`)) return n;
  }
  throw new Error("no free slot");
}

async function processToKindlePNG(file, crop) {
  const img = await fileToImage(file);
  const canvas = document.createElement("canvas");
  canvas.width = TARGET_W;
  canvas.height = TARGET_H;
  const ctx = canvas.getContext("2d");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(img, crop.x, crop.y, crop.w, crop.h, 0, 0, TARGET_W, TARGET_H);

  const data = ctx.getImageData(0, 0, TARGET_W, TARGET_H).data;
  if (state.device?.grayscale === false) {
    const rgb = new Uint8Array(TARGET_W * TARGET_H * 3);
    for (let i = 0, j = 0; j < data.length; i += 3, j += 4) {
      rgb[i] = data[j];
      rgb[i + 1] = data[j + 1];
      rgb[i + 2] = data[j + 2];
    }
    return await encodePNG(rgb, TARGET_W, TARGET_H, 2, 3);
  }
  const gray = new Uint8Array(TARGET_W * TARGET_H);
  for (let i = 0, j = 0; i < gray.length; i++, j += 4) {
    gray[i] = (0.2126 * data[j] + 0.7152 * data[j + 1] + 0.0722 * data[j + 2]) | 0;
  }
  return await encodePNG(gray, TARGET_W, TARGET_H, 0, 1);
}

// ---- crop UI --------------------------------------------------------------

const cropPanel = document.getElementById("crop");
const cropStage = document.getElementById("crop-stage");
const cropImgEl = document.getElementById("crop-img");
const cropRect = document.getElementById("crop-rect");
const cropZoom = document.getElementById("crop-zoom");
const cropStatusEl = document.getElementById("crop-status");
const cropNext = document.getElementById("crop-next");
const cropSkip = document.getElementById("crop-skip");
const cropCancel = document.getElementById("crop-cancel");

async function cropInteractive(file, idx, total) {
  const img = await fileToImage(file);
  cropImgEl.src = img.src;

  const iw = img.naturalWidth;
  const ih = img.naturalHeight;

  document.body.classList.add("locked");
  cropPanel.hidden = false;
  await new Promise((r) => requestAnimationFrame(r));

  return new Promise((resolve) => {
    const sw = cropStage.clientWidth;
    const sh = cropStage.clientHeight;
    const dispScale = Math.min(sw / iw, sh / ih);
    const dispW = iw * dispScale;
    const dispH = ih * dispScale;
    const offX = (sw - dispW) / 2;
    const offY = (sh - dispH) / 2;
    cropImgEl.style.left = `${offX}px`;
    cropImgEl.style.top = `${offY}px`;
    cropImgEl.style.width = `${dispW}px`;
    cropImgEl.style.height = `${dispH}px`;

    const maxCw = iw / ih > TARGET_ASPECT ? ih * TARGET_ASPECT : iw;
    const minCw = Math.min(maxCw, Math.max(80, maxCw * 0.15));
    const zoomRange = Math.max(1e-6, maxCw - minCw);

    let cw = maxCw;
    let ch = cw / TARGET_ASPECT;
    let cx = (iw - cw) / 2;
    let cy = (ih - ch) / 2;

    function clamp() {
      cw = Math.max(minCw, Math.min(maxCw, cw));
      ch = cw / TARGET_ASPECT;
      cx = Math.max(0, Math.min(iw - cw, cx));
      cy = Math.max(0, Math.min(ih - ch, cy));
    }
    function render() {
      clamp();
      cropRect.style.left = `${offX + cx * dispScale}px`;
      cropRect.style.top = `${offY + cy * dispScale}px`;
      cropRect.style.width = `${cw * dispScale}px`;
      cropRect.style.height = `${ch * dispScale}px`;
      const tag = total > 1 ? `[${idx}/${total}] ` : "";
      cropStatusEl.textContent = `${tag}${Math.round(cw)}×${Math.round(ch)} @ (${Math.round(cx)},${Math.round(cy)})  →  ${TARGET_W}×${TARGET_H}`;
      cropZoom.value = (maxCw - cw) / zoomRange;
    }

    let drag = null;
    const onDown = (e) => {
      cropStage.setPointerCapture(e.pointerId);
      const handle = e.target.classList?.contains("crop-h") ? e.target.dataset.h : null;
      drag = { x: e.clientX, y: e.clientY, cx, cy, cw, ch, handle };
    };
    const onMove = (e) => {
      if (!drag) return;
      const dx = (e.clientX - drag.x) / dispScale;
      const dy = (e.clientY - drag.y) / dispScale;
      if (!drag.handle) {
        cx = drag.cx + dx;
        cy = drag.cy + dy;
      } else {
        const right = drag.handle.includes("r");
        const bottom = drag.handle.includes("b");
        const anchorX = right ? drag.cx : drag.cx + drag.cw;
        const anchorY = bottom ? drag.cy : drag.cy + drag.ch;
        let nw = right ? drag.cw + dx : drag.cw - dx;
        let nh = bottom ? drag.ch + dy : drag.ch - dy;
        if (nw / TARGET_ASPECT > nh) nh = nw / TARGET_ASPECT;
        else nw = nh * TARGET_ASPECT;
        cw = nw;
        ch = nh;
        cx = right ? anchorX : anchorX - cw;
        cy = bottom ? anchorY : anchorY - ch;
      }
      render();
    };
    const onUp = () => {
      drag = null;
    };
    const onZoom = () => {
      const t = parseFloat(cropZoom.value);
      const cxC = cx + cw / 2,
        cyC = cy + ch / 2;
      cw = maxCw - zoomRange * t;
      ch = cw / TARGET_ASPECT;
      cx = cxC - cw / 2;
      cy = cyC - ch / 2;
      render();
    };
    const onWheel = (e) => {
      e.preventDefault();
      const f = e.deltaY > 0 ? 1.06 : 1 / 1.06;
      const cxC = cx + cw / 2,
        cyC = cy + ch / 2;
      cw *= f;
      clamp();
      cx = cxC - cw / 2;
      cy = cyC - ch / 2;
      render();
    };
    const onNext = () => {
      clamp();
      finish({ x: cx, y: cy, w: cw, h: ch });
    };
    const onSkip = () => finish(null);
    const onCancel = () => finish("cancel");

    cropStage.addEventListener("pointerdown", onDown);
    cropStage.addEventListener("pointermove", onMove);
    cropStage.addEventListener("pointerup", onUp);
    cropStage.addEventListener("pointercancel", onUp);
    cropStage.addEventListener("wheel", onWheel, { passive: false });
    cropZoom.addEventListener("input", onZoom);
    cropNext.addEventListener("click", onNext);
    cropSkip.addEventListener("click", onSkip);
    cropCancel.addEventListener("click", onCancel);

    const finish = (val) => {
      cropStage.removeEventListener("pointerdown", onDown);
      cropStage.removeEventListener("pointermove", onMove);
      cropStage.removeEventListener("pointerup", onUp);
      cropStage.removeEventListener("pointercancel", onUp);
      cropStage.removeEventListener("wheel", onWheel);
      cropZoom.removeEventListener("input", onZoom);
      cropNext.removeEventListener("click", onNext);
      cropSkip.removeEventListener("click", onSkip);
      cropCancel.removeEventListener("click", onCancel);
      cropPanel.hidden = true;
      document.body.classList.remove("locked");
      cropImgEl.src = "";
      resolve(val);
    };

    render();
  });
}

function fileToImage(file) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("invalid image"));
    };
    img.src = url;
  });
}

// ---- PNG encoder (8-bit gray) --------------------------------------------

const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();

function crc32(bytes) {
  let c = 0xffffffff;
  for (let i = 0; i < bytes.length; i++) c = CRC_TABLE[(c ^ bytes[i]) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const buf = new Uint8Array(8 + data.length + 4);
  const dv = new DataView(buf.buffer);
  dv.setUint32(0, data.length);
  for (let i = 0; i < 4; i++) buf[4 + i] = type.charCodeAt(i);
  buf.set(data, 8);
  dv.setUint32(8 + data.length, crc32(buf.subarray(4, 8 + data.length)));
  return buf;
}

async function deflate(bytes) {
  const stream = new Blob([bytes]).stream().pipeThrough(new CompressionStream("deflate"));
  return new Uint8Array(await new Response(stream).arrayBuffer());
}

// encodePNG writes an 8-bit PNG. colorType 0 = grayscale (1 channel),
// 2 = truecolor RGB (3 channels). Filter type 0 (None) for every row.
async function encodePNG(pixels, w, h, colorType, channels) {
  const sig = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
  const ihdr = new Uint8Array(13);
  const dv = new DataView(ihdr.buffer);
  dv.setUint32(0, w);
  dv.setUint32(4, h);
  ihdr[8] = 8;
  ihdr[9] = colorType;
  ihdr[10] = 0;
  ihdr[11] = 0;
  ihdr[12] = 0;
  const stride = w * channels;
  const raw = new Uint8Array((stride + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (stride + 1)] = 0;
    raw.set(pixels.subarray(y * stride, (y + 1) * stride), y * (stride + 1) + 1);
  }
  const idat = await deflate(raw);
  const parts = [sig, chunk("IHDR", ihdr), chunk("IDAT", idat), chunk("IEND", new Uint8Array(0))];
  let total = 0;
  for (const p of parts) total += p.length;
  const out = new Uint8Array(total);
  let off = 0;
  for (const p of parts) {
    out.set(p, off);
    off += p.length;
  }
  return out;
}

// ---- boot -----------------------------------------------------------------

refresh().catch((e) => {
  metaEl.textContent = `load failed: ${e.message}`;
});
