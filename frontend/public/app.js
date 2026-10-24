"use strict";

const API = "/api/v1/diary";

const SEARCH_LIMIT = 20;
const NOTES_LIMIT = 50;

const ICONS = {
  photo: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><rect x="3" y="6" width="18" height="13" rx="3"/><circle cx="12" cy="12.5" r="3.2"/></svg>',
  plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>',
  today: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="7"/><circle cx="12" cy="12" r="1.6" fill="currentColor" stroke="none"/></svg>',
  search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.2-3.2"/></svg>',
};

const state = {
  section: "diario", // "diario" | "notas"
  tab: "hoy", // "hoy" | "buscar"
  date: todayISO(),
  ignoreYear: true,

  entries: [], // día actual
  notes: [], // atemporales
  notesHasMore: false,
  prevDayEmpty: null, // null = sin comprobar / no aplica

  query: "",
  results: [],
  resultsOffset: 0,
  resultsHasMore: false,

  draft: "",
  pendingFiles: [],
  editingId: null,
  busy: false,
};

// ---------------------------------------------------------------- utilidades

function todayISO() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function shiftISO(iso, days) {
  const d = new Date(iso + "T00:00:00");
  d.setDate(d.getDate() + days);
  const p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function prevISO(iso) { return shiftISO(iso, -1); }
function nextISO(iso) { return shiftISO(iso, 1); }

function fmtChip(iso) {
  if (iso === todayISO()) return "Hoy";
  const d = new Date(iso + "T00:00:00");
  return d.toLocaleDateString("es-ES", { day: "numeric", month: "short", year: "numeric" });
}

function fmtLong(iso) {
  const d = new Date(iso + "T00:00:00");
  const s = d.toLocaleDateString("es-ES", { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => (
    { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]
  ));
}

let toastTimer = null;
function toast(msg, isError) {
  const el = document.getElementById("toast");
  el.textContent = msg;
  el.classList.toggle("error", !!isError);
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, 3200);
}

async function api(path, opts) {
  const res = await fetch(API + path, opts);
  if (!res.ok) {
    let msg = `Error ${res.status}`;
    try {
      const body = await res.json();
      if (body && body.error) msg = body.error;
    } catch (_) { /* respuesta sin JSON */ }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

function thumbURL(key) { return `${API}/attachments/${encodeURIComponent(key)}/thumb`; }
function fullURL(key) { return `${API}/attachments/${encodeURIComponent(key)}/full`; }

// ------------------------------------------------------------------- carga

async function loadDay() {
  const qs = new URLSearchParams({ date: state.date, ignore_year: String(state.ignoreYear) });
  const data = await api(`/?${qs}`);
  state.entries = data.entries || [];
  await checkPrevDay();
}

// Solo interesa cuando la entrada que vas a escribir sería la primera del día.
// Es información secundaria: si la petición falla, no se avisa de nada.
async function checkPrevDay() {
  state.prevDayEmpty = null;
  // Con "Otros años" activo, entries trae también las de otros años: solo cuentan las de este día.
  if (state.entries.some((e) => e.date === state.date)) return;

  const qs = new URLSearchParams({ date: prevISO(state.date), ignore_year: "false" });
  try {
    const data = await api(`/?${qs}`);
    state.prevDayEmpty = (data.entries || []).length === 0;
  } catch (_) { /* sin aviso */ }
}

async function loadNotes() {
  const qs = new URLSearchParams({ limit: String(NOTES_LIMIT), offset: "0" });
  const data = await api(`/atemporal?${qs}`);
  state.notes = data.entries || [];
  state.notesHasMore = !!data.has_more;
}

async function runSearch(append) {
  const q = state.query.trim();
  if (!q) {
    state.results = [];
    state.resultsOffset = 0;
    state.resultsHasMore = false;
    renderResults();
    return;
  }
  const offset = append ? state.resultsOffset : 0;
  const qs = new URLSearchParams({ text: q, limit: String(SEARCH_LIMIT), offset: String(offset) });
  try {
    const data = await api(`/search?${qs}`);
    const found = data.entries || [];
    state.results = append ? state.results.concat(found) : found;
    state.resultsOffset = offset + found.length;
    state.resultsHasMore = !!data.has_more;
  } catch (err) {
    toast(err.message, true);
  }
  renderResults();
}

async function refresh() {
  state.busy = true;
  try {
    if (state.section === "notas") await loadNotes();
    else if (state.tab === "hoy") await loadDay();
  } catch (err) {
    toast(err.message, true);
  } finally {
    state.busy = false;
  }
  render();
}

// ---------------------------------------------------------------- mutaciones

async function createEntry() {
  const text = state.draft.trim();
  if (!text && state.pendingFiles.length === 0) return;

  const fd = new FormData();
  fd.append("text", text);
  // Sin campo "date" = entrada atemporal (el backend lo guarda como NULL).
  if (state.section === "diario") fd.append("date", state.date);
  for (const f of state.pendingFiles) fd.append("files", f, f.name);

  try {
    await api("/", { method: "POST", body: fd });
    state.draft = "";
    state.pendingFiles = [];
    await refresh();
    toast("Guardado");
  } catch (err) {
    toast(err.message, true);
  }
}

async function saveEdit(id, text) {
  const fd = new FormData();
  fd.append("text", text);
  try {
    await api(`/${id}`, { method: "PUT", body: fd });
    state.editingId = null;
    await refresh();
    toast("Actualizado");
  } catch (err) {
    toast(err.message, true);
  }
}

async function deleteEntry(id) {
  if (!confirm("¿Borrar esta entrada? No se puede deshacer.")) return;
  try {
    await api(`/${id}`, { method: "DELETE" });
    await refresh();
    toast("Borrado");
  } catch (err) {
    toast(err.message, true);
  }
}

// ------------------------------------------------------------------ plantillas

function entryHTML(e, opts) {
  const showDate = opts && opts.showDate && e.date;
  const editing = state.editingId === e.id;

  const head = showDate ? `<div class="entry-date">${esc(fmtLong(e.date))}</div>` : "";

  const body = editing
    ? `<textarea data-edit="${e.id}">${esc(e.text)}</textarea>`
    : `<div class="entry-text">${esc(e.text)}</div>`;

  const thumbs = (e.keys || []).length
    ? `<div class="thumbs">${e.keys.map((k) =>
        `<img loading="lazy" src="${thumbURL(k)}" data-full="${esc(fullURL(k))}" alt="">`
      ).join("")}</div>`
    : "";

  const actions = editing
    ? `<div class="entry-actions">
         <button class="link-btn" data-act="save" data-id="${e.id}">Guardar</button>
         <button class="link-btn" data-act="cancel">Cancelar</button>
       </div>`
    : `<div class="entry-actions">
         <button class="link-btn" data-act="edit" data-id="${e.id}">Editar</button>
         <button class="link-btn danger" data-act="del" data-id="${e.id}">Borrar</button>
       </div>`;

  return `<article class="entry">${head}${body}${thumbs}${actions}</article>`;
}

function composerHTML(placeholder) {
  const pending = state.pendingFiles.length
    ? `<div class="pending"><span>${state.pendingFiles.length} foto(s) seleccionadas</span>
         <button data-act="clear-files">Quitar</button></div>`
    : "";
  const canAdd = state.draft.trim() || state.pendingFiles.length;
  return `
    <section class="composer">
      <textarea id="draft" rows="2" placeholder="${esc(placeholder)}">${esc(state.draft)}</textarea>
      ${pending}
      <div class="composer-actions">
        <button class="btn-photo" data-act="pick-files">${ICONS.photo}Foto</button>
        <button class="btn-add" id="add-btn" data-act="add" ${canAdd ? "" : "disabled"}>${ICONS.plus}Añadir</button>
      </div>
    </section>
    <input type="file" id="file-input" accept="image/*" multiple hidden>`;
}

function noticeHTML() {
  if (state.prevDayEmpty !== true) return "";
  const prev = prevISO(state.date);
  const label = prev === prevISO(todayISO()) ? "Ayer" : `El ${fmtChip(prev)}`;
  return `
    <div class="notice">
      <span>${esc(label)} no escribiste nada en el diario.</span>
      <button data-act="dismiss-notice" aria-label="Ocultar aviso">&times;</button>
    </div>`;
}

function listHTML(entries, opts) {
  if (!entries.length) {
    return `<div class="empty">${esc(opts.emptyText)}</div>`;
  }
  return entries.map((e) => entryHTML(e, opts)).join("");
}

// --------------------------------------------------------------------- render

function renderTopbar() {
  const bar = document.getElementById("topbar");

  if (state.section === "notas") {
    bar.innerHTML = `
      <div class="topbar-row"><span class="eyebrow">NOTAS</span></div>
      <div class="rule"></div>`;
    return;
  }

  // El chip es un botón que abre el selector con showPicker(). El <input type="date">
  // va debajo, invisible y sin recibir toques, solo para que el selector se ancle al chip.
  const dateControls = state.tab === "hoy"
    ? `<div class="topbar-tools">
         <button class="chip ${state.ignoreYear ? "on" : "ghost"}" data-act="toggle-year"
                 title="Ver este día en todos los años">Otros años</button>
         <span class="date-pick">
           <button class="chip" data-act="pick-date" aria-label="Elegir fecha">${esc(fmtChip(state.date))}</button>
           <input type="date" id="date-input" value="${state.date}" max="${todayISO()}" tabindex="-1" aria-hidden="true">
         </span>
       </div>`
    : "";

  const jump = state.tab === "hoy" && state.date !== todayISO()
    ? `<button class="icon-btn" data-act="go-today" title="Volver a hoy">${ICONS.today}</button>`
    : "";

  bar.innerHTML = `
    <div class="topbar-row">
      <span class="eyebrow">DIARIO</span>
      ${dateControls}
    </div>
    <div class="tabs">
      <div class="tabs-group">
        <button class="tab ${state.tab === "hoy" ? "active" : ""}" data-tab="hoy">Hoy</button>
        <button class="tab ${state.tab === "buscar" ? "active" : ""}" data-tab="buscar">Buscar</button>
      </div>
      ${jump}
    </div>
    <div class="rule"></div>`;
}

function renderResults() {
  const box = document.getElementById("results");
  if (!box) return;
  if (!state.query.trim()) {
    box.innerHTML = `<div class="empty">Escribe algo para buscar en todo el diario.</div>`;
    return;
  }
  if (!state.results.length) {
    box.innerHTML = `<div class="empty">Sin resultados para "${esc(state.query)}".</div>`;
    return;
  }
  const items = state.results.map((r) => {
    const thumb = (r.keys || []).length ? `<img loading="lazy" src="${thumbURL(r.keys[0])}" alt="">` : "";
    const label = r.date ? fmtLong(r.date) : "Sin fecha";
    return `<button class="result" data-act="open-result" data-id="${r.id}" data-date="${esc(r.date || "")}">
        ${thumb}
        <div style="min-width:0">
          <div class="entry-date">${esc(label)}</div>
          <div class="result-text">${esc(r.text)}</div>
        </div>
      </button>`;
  }).join("");

  const more = state.resultsHasMore
    ? `<button class="more" data-act="more-results">Cargar más</button>` : "";

  box.innerHTML = `<div class="section-label">${state.results.length} RESULTADO(S)</div>${items}${more}`;
}

function renderView() {
  const view = document.getElementById("view");

  if (state.section === "notas") {
    view.innerHTML = `
      ${composerHTML("Una nota sin fecha…")}
      <div class="pad">
        <div class="section-label">TODAS LAS NOTAS</div>
        <div id="list">${listHTML(state.notes, { emptyText: "Todavía no hay notas sueltas." })}</div>
      </div>`;
    return;
  }

  if (state.tab === "buscar") {
    view.innerHTML = `
      <div class="pad">
        <div class="searchbar">
          ${ICONS.search}
          <input id="search-input" type="search" placeholder="Buscar en todo el diario…" value="${esc(state.query)}">
        </div>
        <div id="results"></div>
      </div>`;
    renderResults();
    return;
  }

  // tab "hoy"
  const heading = state.ignoreYear
    ? "ESTE DÍA EN OTROS AÑOS"
    : esc(fmtLong(state.date).toUpperCase());

  view.innerHTML = `
    ${noticeHTML()}
    ${composerHTML("Escribe algo sobre este día…")}
    <div class="pad">
      <div class="section-label">${heading}</div>
      <div id="list">${listHTML(state.entries, {
        showDate: state.ignoreYear,
        emptyText: "Aún no has escrito nada este día.",
      })}</div>
    </div>`;
}

function renderNav() {
  document.querySelectorAll(".nav-btn").forEach((b) => {
    b.classList.toggle("active", b.dataset.section === state.section);
  });
}

function render() {
  renderTopbar();
  renderView();
  renderNav();
}

function renderList() {
  const box = document.getElementById("list");
  if (!box) return;
  if (state.section === "notas") {
    box.innerHTML = listHTML(state.notes, { emptyText: "Todavía no hay notas sueltas." });
  } else {
    box.innerHTML = listHTML(state.entries, {
      showDate: state.ignoreYear,
      emptyText: "Aún no has escrito nada este día.",
    });
  }
  const ta = box.querySelector("textarea[data-edit]");
  if (ta) { ta.focus(); ta.setSelectionRange(ta.value.length, ta.value.length); }
}

// ---------------------------------------------------------------- eventos

let searchTimer = null;

document.addEventListener("click", async (ev) => {
  const nav = ev.target.closest(".nav-btn");
  if (nav) {
    state.section = nav.dataset.section;
    state.editingId = null;
    state.draft = "";
    state.pendingFiles = [];
    await refresh();
    return;
  }

  const tab = ev.target.closest(".tab");
  if (tab) {
    state.tab = tab.dataset.tab;
    state.editingId = null;
    if (state.tab === "hoy") await refresh();
    else render();
    return;
  }

  const thumb = ev.target.closest(".thumbs img");
  if (thumb) {
    const box = document.getElementById("lightbox");
    box.querySelector("img").src = thumb.dataset.full;
    box.hidden = false;
    return;
  }

  if (ev.target.closest("#lightbox")) {
    const box = document.getElementById("lightbox");
    box.hidden = true;
    box.querySelector("img").src = "";
    return;
  }

  const btn = ev.target.closest("[data-act]");
  if (!btn) return;
  const act = btn.dataset.act;

  if (act === "add") return createEntry();

  if (act === "pick-files") {
    document.getElementById("file-input").click();
    return;
  }

  if (act === "dismiss-notice") {
    state.prevDayEmpty = false;
    renderView();
    return;
  }

  if (act === "clear-files") {
    state.pendingFiles = [];
    renderView();
    return;
  }

  if (act === "go-today") {
    state.date = todayISO();
    await refresh();
    return;
  }

  if (act === "pick-date") {
    const input = document.getElementById("date-input");
    try {
      input.showPicker();
    } catch (_) {
      // Navegadores sin showPicker()
      input.focus();
      input.click();
    }
    return;
  }

  if (act === "toggle-year") {
    state.ignoreYear = !state.ignoreYear;
    await refresh();
    return;
  }

  if (act === "edit") {
    state.editingId = Number(btn.dataset.id);
    renderList();
    return;
  }

  if (act === "cancel") {
    state.editingId = null;
    renderList();
    return;
  }

  if (act === "save") {
    const ta = document.querySelector(`textarea[data-edit="${btn.dataset.id}"]`);
    if (!ta) return;
    const text = ta.value.trim();
    if (!text) { toast("El texto no puede quedar vacío", true); return; }
    return saveEdit(Number(btn.dataset.id), text);
  }

  if (act === "del") return deleteEntry(Number(btn.dataset.id));

  if (act === "more-results") return runSearch(true);

  if (act === "open-result") {
    const date = btn.dataset.date;
    if (date) {
      state.section = "diario";
      state.tab = "hoy";
      state.date = date;
      await refresh();
    } else {
      state.section = "notas";
      await refresh();
    }
    return;
  }
});

document.addEventListener("input", (ev) => {
  if (ev.target.id === "draft") {
    state.draft = ev.target.value;
    const add = document.getElementById("add-btn");
    if (add) add.disabled = !(state.draft.trim() || state.pendingFiles.length);
    return;
  }

  if (ev.target.id === "search-input") {
    state.query = ev.target.value;
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => runSearch(false), 300);
  }
});

document.addEventListener("change", async (ev) => {
  if (ev.target.id === "file-input") {
    state.pendingFiles = Array.from(ev.target.files || []);
    renderView();
    return;
  }

  if (ev.target.id === "date-input" && ev.target.value) {
    state.date = ev.target.value;
    await refresh();
  }
});

// Deslizar en horizontal sobre el día: izquierda = día siguiente, derecha = anterior.
const SWIPE_MIN_PX = 60;
let touchStart = null;

document.getElementById("view").addEventListener("touchstart", (ev) => {
  // Sin swipe con varios dedos (zoom) ni al tocar campos de texto (mover el cursor).
  if (ev.touches.length !== 1 || ev.target.closest("textarea, input")) {
    touchStart = null;
    return;
  }
  touchStart = { x: ev.touches[0].clientX, y: ev.touches[0].clientY };
}, { passive: true });

document.getElementById("view").addEventListener("touchend", async (ev) => {
  if (!touchStart) return;
  const dx = ev.changedTouches[0].clientX - touchStart.x;
  const dy = ev.changedTouches[0].clientY - touchStart.y;
  touchStart = null;

  if (state.section !== "diario" || state.tab !== "hoy" || state.busy) return;
  // Tiene que ser claramente horizontal para no confundirlo con el scroll.
  if (Math.abs(dx) < SWIPE_MIN_PX || Math.abs(dx) < Math.abs(dy) * 1.5) return;

  const target = dx < 0 ? nextISO(state.date) : prevISO(state.date);
  if (target > todayISO()) return;

  state.date = target;
  state.editingId = null;
  await refresh();
});

refresh();
