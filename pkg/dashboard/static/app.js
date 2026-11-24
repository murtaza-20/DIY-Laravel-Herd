async function api(path, opts = {}) {
  const res = await fetch(path, {
    headers: { Accept: "application/json", ...(opts.headers || {}) },
    ...opts,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

function pill(ok, label) {
  return `<span class="pill ${ok ? "ok" : "bad"}">${ok ? "●" : "○"} ${label}</span>`;
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c])
  );
}

function renderSites(sites) {
  const el = document.getElementById("sites");
  if (!sites.length) {
    el.innerHTML = `<div class="empty">No sites yet. Run: <code>herd serve myapp ~/path</code></div>`;
    return;
  }
  el.innerHTML = sites
    .map(
      (s) => `<div class="row" data-name="${esc(s.name)}">
      <div>
        <h3><a href="${esc(s.url)}" target="_blank" rel="noopener">${esc(s.name)}</a></h3>
        <div class="meta">${esc(s.path)}</div>
      </div>
      <div class="meta">PHP ${esc(s.php_version)}</div>
      <div class="meta">${esc(s.project_type)}</div>
      <div class="row-actions">
        <a href="${esc(s.url)}" target="_blank" rel="noopener"><button type="button" class="ghost">Open</button></a>
        <button type="button" class="danger btn-forget" data-name="${esc(s.name)}">Forget</button>
      </div>
    </div>`
    )
    .join("");

  el.querySelectorAll(".btn-forget").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const name = btn.dataset.name;
      if (!confirm(`Forget site "${name}"?`)) return;
      try {
        await api("/api/sites/" + encodeURIComponent(name), { method: "DELETE" });
        await refresh();
      } catch (e) {
        alert(e.message);
      }
    });
  });
}

function renderPHP(list) {
  const el = document.getElementById("php");
  if (!list.length) {
    el.innerHTML = `<div class="empty">No PHP versions detected.</div>`;
    return;
  }
  el.innerHTML = list
    .map(
      (p) => `<div class="card">
      <h3>PHP ${esc(p.version)}</h3>
      <div style="margin:.5rem 0">${pill(p.active, p.active ? "fpm active" : "fpm inactive")}</div>
      <div class="meta">${esc(p.socket)}</div>
      <div style="margin-top:.5rem">${pill(p.socket_ok, p.socket_ok ? "socket ok" : "socket missing")}</div>
    </div>`
    )
    .join("");
}

function renderServices(list) {
  const el = document.getElementById("services");
  el.innerHTML = list
    .map(
      (s) => `<div class="card">
      <h3>${esc(s.name)}</h3>
      <div class="meta">${esc(s.detail)}</div>
      <div style="margin:.55rem 0">${pill(s.active, s.active ? "active" : "inactive")}</div>
      <button type="button" class="ghost btn-restart" data-name="${esc(s.name)}">Restart</button>
    </div>`
    )
    .join("");

  el.querySelectorAll(".btn-restart").forEach((btn) => {
    btn.addEventListener("click", async () => {
      try {
        await api("/api/services/" + encodeURIComponent(btn.dataset.name) + "/restart", {
          method: "POST",
        });
        await refresh();
      } catch (e) {
        alert(e.message + "\nIf sudo -n failed, restart manually via systemctl.");
      }
    });
  });
}

function renderDatabases(list) {
  renderDockerGrid("databases", list, "No database info.");
}

function renderExtras(list) {
  renderDockerGrid("extras", list, "No extras info.");
}

function renderDockerGrid(id, list, emptyMsg) {
  const el = document.getElementById(id);
  if (!el) return;
  if (!list || !list.length) {
    el.innerHTML = `<div class="empty">${emptyMsg}</div>`;
    return;
  }
  el.innerHTML = list
    .map(
      (d) => `<div class="card">
      <h3>${esc(d.kind)}</h3>
      <div class="meta">${esc(d.name)}</div>
      <div style="margin:.55rem 0">${pill(d.running, d.running ? "running" : d.detail || "stopped")}</div>
      <div class="meta">${esc(d.dsn)}</div>
    </div>`
    )
    .join("");
}

function renderQueues(list) {
  const el = document.getElementById("queues");
  if (!el) return;
  if (!list || !list.length) {
    el.innerHTML = `<div class="empty">No workers. Run: <code>herd queue work myapp</code></div>`;
    return;
  }
  el.innerHTML = list
    .map(
      (q) => `<div class="row">
      <div>
        <h3>${esc(q.site)} <span class="meta">/ ${esc(q.queue)}</span></h3>
        <div class="meta">${esc(q.log_file)}</div>
      </div>
      <div class="meta">PHP ${esc(q.php_version)}</div>
      <div class="meta">pid ${esc(q.pid)}</div>
      <div class="meta">${esc(q.started_at ? new Date(q.started_at).toLocaleString() : "")}</div>
    </div>`
    )
    .join("");
}

async function refresh() {
  const data = await api("/api/status");
  const activePHP = (data.php || []).filter((p) => p.active).length;
  const activeSvc = (data.services || []).filter((s) => s.active).length;
  const activeDB = (data.databases || []).filter((d) => d.running).length;
  const workers = (data.queues || []).length;
  document.getElementById("summary").textContent =
    `${(data.sites || []).length} sites · ${activePHP} PHP · ${activeDB} DBs · ${workers} workers · ${activeSvc} services`;
  document.getElementById("herd-dir").textContent = data.herd_dir || "";
  document.getElementById("foot-meta").textContent =
    `Herd ${data.version || ""} · ${new Date(data.time).toLocaleString()}`;
  renderSites(data.sites || []);
  renderPHP(data.php || []);
  renderDatabases(data.databases || []);
  renderExtras(data.extras || []);
  renderQueues(data.queues || []);
  renderServices(data.services || []);
}

async function loadLogs() {
  const name = document.getElementById("log-service").value.trim() || "nginx";
  const view = document.getElementById("log-view");
  view.textContent = "Loading…";
  try {
    const data = await api("/api/logs/" + encodeURIComponent(name));
    view.textContent = data.log || "(empty)";
  } catch (e) {
    view.textContent = e.message;
  }
}

document.getElementById("btn-refresh").addEventListener("click", () =>
  refresh().catch((e) => alert(e.message))
);
document.getElementById("btn-nginx").addEventListener("click", async () => {
  try {
    await api("/api/services/nginx/reload", { method: "POST" });
    alert("Nginx reload requested");
  } catch (e) {
    alert(e.message);
  }
});
document.getElementById("btn-logs").addEventListener("click", () => loadLogs());

refresh().catch((e) => {
  document.getElementById("summary").textContent = "Failed to load: " + e.message;
});
setInterval(() => refresh().catch(() => {}), 15000);
