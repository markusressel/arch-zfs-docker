let allPackages = [];
let lastBuilds = [];
let lastUpstream = {};
let eventSource = null;
let currentSettings = null;

// Builds that finish faster than this were cache hits / no-ops and say nothing about real build time.
const MIN_REAL_BUILD_SEC = 120;

document.addEventListener("DOMContentLoaded", () => {
  fetchSettings();
  fetchPackages();
  fetchBuilds();
  fetchUpstream();
  fetchKernelVersions();

  document.getElementById("search-input").addEventListener("input", (e) => {
    filterPackages(e.target.value);
  });

  document.getElementById("variant-select").addEventListener("change", fetchKernelVersions);

  document.getElementById("build-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    await triggerBuild(false);
  });

  const settingsForm = document.getElementById("settings-form");
  if (settingsForm) {
    settingsForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      await saveSettings();
    });
  }

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") closeDialog();
  });

  // Polling for builds & upstream status every 8 seconds
  setInterval(() => {
    fetchBuilds();
    fetchUpstream();
  }, 8000);
});

function switchTab(tab) {
  for (const name of ["dashboard", "analytics", "settings"]) {
    document.getElementById(`tab-${name}`).classList.toggle("active", name === tab);
    document.getElementById(`view-${name}`).style.display = name === tab ? "block" : "none";
  }
  if (tab === "settings") {
    fetchSettings();
  } else {
    fetchBuilds();
    if (tab === "dashboard") fetchPackages();
  }
}

// ---- Upstream / kernel status ----

async function fetchUpstream() {
  try {
    const res = await fetch("/api/upstream");
    lastUpstream = await res.json();
    renderKernelStatus();
  } catch (err) {
    console.error("Failed to fetch upstream status:", err);
  }
}

function renderKernelStatus() {
  const container = document.getElementById("kernel-status");
  const checked = document.getElementById("upstream-checked");
  if (!container) return;

  const entries = Object.values(lastUpstream).sort((a, b) => a.kernelPkg.localeCompare(b.kernelPkg));
  if (entries.length === 0) {
    container.innerHTML = "";
    checked.textContent = "Waiting for first upstream check...";
    return;
  }

  const newest = entries.reduce((m, e) => (e.checkedAt > m ? e.checkedAt : m), "");
  checked.textContent = `Last checked ${new Date(newest).toLocaleString()}`;

  container.innerHTML = entries.map(st => {
    const building = lastBuilds.some(b => (b.status === "Running" || b.status === "Pending") && (b.variant || "") === (st.variant || ""));
    let cls, pill;
    if (st.upToDate) {
      cls = "ok";
      pill = `<span class="status-pill status-Succeeded">Up to date</span>`;
    } else if (building) {
      cls = "behind";
      pill = `<span class="status-pill status-Running">Building</span>`;
    } else {
      cls = "behind";
      pill = `<span class="status-pill status-Pending">Build needed</span>`;
    }
    return `
      <div class="kernel-item ${cls}">
        <div class="kernel-item-head"><strong>${escapeHtml(st.kernelPkg)}</strong>${pill}</div>
        <div class="kernel-row"><span>Arch upstream</span><code>${escapeHtml(st.upstreamKernel)}</code></div>
        <div class="kernel-row"><span>Our repository</span><code>${escapeHtml(st.localKernel || "none built")}</code></div>
      </div>`;
  }).join("");
}

async function checkUpstreamNow() {
  const btn = document.getElementById("btn-check-upstream");
  btn.disabled = true;
  btn.textContent = "Checking...";
  try {
    await fetch("/api/upstream/check", { method: "POST" });
    setTimeout(async () => {
      await fetchUpstream();
      await fetchBuilds();
    }, 2000);
  } catch (err) {
    showDialog({ title: "Check failed", body: `<p>${escapeHtml(err.message)}</p>` });
  } finally {
    setTimeout(() => {
      btn.disabled = false;
      btn.textContent = "Check Now";
    }, 2000);
  }
}

// ---- Kernel version autocomplete ----

async function fetchKernelVersions() {
  const list = document.getElementById("kernel-versions");
  const variant = document.getElementById("variant-select").value;
  try {
    const res = await fetch(`/api/kernels?variant=${encodeURIComponent(variant)}`);
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    list.innerHTML = (data.versions || []).map(v => `<option value="${escapeHtml(v)}"></option>`).join("");
  } catch (err) {
    // Free text input still works without suggestions.
    list.innerHTML = "";
    console.warn("Failed to fetch kernel versions:", err);
  }
}

// ---- Dialog ----

function showDialog({ title, body, actions }) {
  document.getElementById("dialog-title").textContent = title;
  document.getElementById("dialog-body").innerHTML = body;
  const bar = document.getElementById("dialog-actions");
  bar.innerHTML = "";
  for (const a of actions || [{ label: "Close" }]) {
    const btn = document.createElement("button");
    btn.className = `btn ${a.cls || "btn-secondary"}`;
    btn.textContent = a.label;
    btn.onclick = () => {
      closeDialog();
      if (a.onClick) a.onClick();
    };
    bar.appendChild(btn);
  }
  document.getElementById("dialog-modal").classList.add("active");
}

function closeDialog() {
  document.getElementById("dialog-modal").classList.remove("active");
}

async function fetchPackages() {
  try {
    const res = await fetch("/api/packages");
    const data = await res.json();
    allPackages = data.packages || [];

    filterPackages(document.getElementById("search-input").value);
  } catch (err) {
    console.error("Failed to fetch packages:", err);
  }
}

function renderPackages(packages) {
  const tbody = document.getElementById("packages-table");
  const summary = document.getElementById("catalog-summary");
  const bytes = (packages || []).reduce((n, p) => n + p.sizeBytes, 0);
  const filtered = packages.length !== allPackages.length;
  summary.textContent = `${packages.length}${filtered ? ` of ${allPackages.length}` : ""} packages \u00b7 ${formatBytes(bytes)}`;
  if (!packages || packages.length === 0) {
    tbody.innerHTML = `<tr><td colspan="6" style="text-align: center; color: var(--text-muted);">No packages found</td></tr>`;
    return;
  }

  tbody.innerHTML = packages.map(p => {
    const dt = new Date(p.modTime).toLocaleDateString();
    return `
      <tr>
        <td><strong>${escapeHtml(p.packageName)}</strong></td>
        <td><code>${escapeHtml(p.version)}</code></td>
        <td><span style="color: var(--accent-blue);">${escapeHtml(p.kernelVersion || "-")}</span></td>
        <td>${escapeHtml(p.sizeHuman)}</td>
        <td style="color: var(--text-secondary);">${dt}</td>
        <td>
          <a href="${escapeHtml(p.downloadUrl)}" class="btn btn-secondary btn-sm" download>
            Download
          </a>
        </td>
      </tr>
    `;
  }).join("");
}

function formatBytes(n) {
  if (n < 1024) return `${n} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let i = -1;
  do {
    n /= 1024;
    i++;
  } while (n >= 1024 && i < units.length - 1);
  return `${n.toFixed(2)} ${units[i]}`;
}

function filterPackages(term) {
  const q = term.toLowerCase().trim();
  if (!q) {
    renderPackages(allPackages);
    return;
  }
  const filtered = allPackages.filter(p =>
    p.packageName.toLowerCase().includes(q) ||
    (p.kernelVersion && p.kernelVersion.toLowerCase().includes(q)) ||
    p.version.toLowerCase().includes(q)
  );
  renderPackages(filtered);
}

async function fetchBuilds() {
  try {
    const res = await fetch("/api/builds");
    lastBuilds = await res.json();
    renderBuilds(lastBuilds);
    renderKernelStatus();
    if (document.getElementById("view-analytics").style.display !== "none") renderAnalytics();
  } catch (err) {
    console.error("Failed to fetch builds:", err);
  }
}

// Fixed-width MM:SS (or H:MM:SS) so the column does not jitter while ticking.
function formatDuration(seconds) {
  if (isNaN(seconds) || seconds < 0) return "-";
  seconds = Math.floor(seconds);
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const pad = n => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;
}

function median(values) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 ? sorted[mid] : (sorted[mid - 1] + sorted[mid]) / 2;
}

// Estimate the duration of a build from recent real (non cache-hit) successful builds,
// preferring jobs of the same kernel variant.
function estimateDuration(variant) {
  const real = lastBuilds.filter(b => b.status === "Succeeded" && b.durationSec >= MIN_REAL_BUILD_SEC);
  const same = real.filter(b => (b.variant || "") === (variant || ""));
  const pool = (same.length > 0 ? same : real).slice(0, 5); // list is newest first
  return pool.length > 0 ? median(pool.map(b => b.durationSec)) : 0;
}

function elapsedSince(iso) {
  return Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000));
}

function etaText(elapsed, estimate) {
  if (!estimate) return "";
  const remaining = estimate - elapsed;
  return remaining > 0 ? `~${formatDuration(remaining)} left` : "taking longer than usual";
}

function updateLiveDurations() {
  document.querySelectorAll("[data-job-running='true']").forEach(elem => {
    const startIso = elem.getAttribute("data-start-time");
    if (!startIso || isNaN(new Date(startIso).getTime())) return;
    const elapsed = elapsedSince(startIso);
    const estimate = Number(elem.getAttribute("data-estimate")) || 0;
    elem.querySelector(".duration").textContent = formatDuration(elapsed);
    const eta = elem.querySelector(".eta");
    if (eta) eta.textContent = etaText(elapsed, estimate);
    const bar = elem.querySelector(".progress > div");
    if (bar && estimate) bar.style.width = `${Math.min(99, (elapsed / estimate) * 100)}%`;
  });
}

// Update running job counters every second
setInterval(updateLiveDurations, 1000);

function renderBuilds(builds) {
  const tbody = document.getElementById("builds-table");
  const clearBtn = document.getElementById("btn-clear-finished");
  if (clearBtn) clearBtn.disabled = !(builds || []).some(isFinished);

  if (!builds || builds.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" style="text-align: center; color: var(--text-muted);">No recent build jobs</td></tr>`;
    return;
  }

  tbody.innerHTML = builds.map(b => {
    const startTime = b.startTime ? new Date(b.startTime).toLocaleTimeString() : "-";
    const isRunning = b.status === "Running" && b.startTime;

    let durationDisplay;
    if (isRunning) {
      const elapsed = elapsedSince(b.startTime);
      const estimate = estimateDuration(b.variant);
      const pct = estimate ? Math.min(99, (elapsed / estimate) * 100) : 0;
      durationDisplay = `
        <div data-job-running="true" data-start-time="${escapeHtml(b.startTime)}" data-estimate="${estimate}">
          <span class="duration">${formatDuration(elapsed)}</span>
          ${estimate ? `<span class="eta">${etaText(elapsed, estimate)}</span><div class="progress"><div style="width: ${pct}%"></div></div>` : ""}
        </div>`;
    } else if (b.durationSec > 0) {
      durationDisplay = `<span class="duration">${formatDuration(b.durationSec)}</span>`;
    } else {
      durationDisplay = `<span class="duration">-</span>`;
    }

    const kernelInfo = b.kernelVersion || b.variant
      ? `<div class="muted small">${escapeHtml(b.variant ? "lts" : "linux")}${b.kernelVersion ? " " + escapeHtml(b.kernelVersion) : ""}</div>`
      : "";

    return `
      <tr>
        <td><code>${escapeHtml(b.name)}</code>${kernelInfo}</td>
        <td><span class="status-pill status-${escapeHtml(b.status)}">${escapeHtml(b.status)}</span></td>
        <td style="color: var(--text-secondary);">${startTime}</td>
        <td>${durationDisplay}</td>
        <td>
          <div class="row-actions">
            <button class="btn btn-secondary btn-sm" onclick="openLogs('${escapeHtml(b.name)}')">Logs</button>
            <button class="btn btn-secondary btn-sm" onclick="showKubectl('${escapeHtml(b.name)}')">kubectl</button>
            ${isFinished(b) ? `<button class="btn btn-secondary btn-sm" title="Delete job" onclick="deleteBuild('${escapeHtml(b.name)}')">Delete</button>` : ""}
          </div>
        </td>
      </tr>
    `;
  }).join("");
}

function isFinished(b) {
  return b.status === "Succeeded" || b.status === "Failed";
}

async function triggerBuild(overwrite) {
  const btn = document.getElementById("btn-submit-build");
  btn.disabled = true;
  btn.textContent = "Starting job...";

  const kernel = document.getElementById("kernel-input").value.trim();
  const variant = document.getElementById("variant-select").value;

  try {
    const res = await fetch("/api/builds", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ kernelVersion: kernel, variant: variant, forceBuild: overwrite }),
    });

    if (res.status === 409) {
      const info = await res.json();
      const pkg = info.package || {};
      showDialog({
        title: "Overwrite existing packages?",
        body: `
          <p>A build for <strong>${escapeHtml(variant ? "linux-" + variant : "linux")} ${escapeHtml(info.kernelVersion)}</strong> already exists in the repository:</p>
          <div class="cmd"><code>${escapeHtml(pkg.filename || "")}</code></div>
          <p style="margin-top: 12px;">Starting this job will rebuild and <strong>replace the existing files</strong>.</p>`,
        actions: [
          { label: "Cancel" },
          { label: "Rebuild & overwrite", cls: "btn-danger", onClick: () => triggerBuild(true) },
        ],
      });
    } else if (!res.ok) {
      showDialog({ title: "Failed to trigger build", body: `<p>${escapeHtml(await res.text())}</p>` });
    } else {
      const newJob = await res.json();
      await fetchBuilds();
      openLogs(newJob.name);
    }
  } catch (err) {
    showDialog({ title: "Error", body: `<p>${escapeHtml(err.message)}</p>` });
  } finally {
    btn.disabled = false;
    btn.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="5 3 19 12 5 21 5 3"></polygon>
      </svg>
      Start Build Job
    `;
  }
}

// ---- Job cleanup ----

function deleteBuild(name) {
  showDialog({
    title: "Delete build job?",
    body: `<p>Removes <code>${escapeHtml(name)}</code> and its pod from the cluster. Built packages stay in the repository.</p>`,
    actions: [
      { label: "Cancel" },
      {
        label: "Delete", cls: "btn-danger", onClick: async () => {
          const res = await fetch(`/api/builds/${encodeURIComponent(name)}`, { method: "DELETE" });
          if (!res.ok && res.status !== 404) {
            showDialog({ title: "Delete failed", body: `<p>${escapeHtml(await res.text())}</p>` });
          }
          fetchBuilds();
        },
      },
    ],
  });
}

function clearFinishedBuilds() {
  const count = lastBuilds.filter(isFinished).length;
  if (count === 0) return;
  showDialog({
    title: "Clear finished jobs?",
    body: `<p>Removes ${count} succeeded/failed job${count === 1 ? "" : "s"} from the cluster. Running jobs and built packages are not affected.</p>`,
    actions: [
      { label: "Cancel" },
      {
        label: "Clear", cls: "btn-danger", onClick: async () => {
          const res = await fetch("/api/builds", { method: "DELETE" });
          if (!res.ok) showDialog({ title: "Cleanup failed", body: `<p>${escapeHtml(await res.text())}</p>` });
          fetchBuilds();
        },
      },
    ],
  });
}

// ---- kubectl helpers ----

function showKubectl(name) {
  const job = lastBuilds.find(b => b.name === name) || {};
  const ns = job.namespace || currentSettings?.namespace || "default";
  const cmds = [
    ["Follow logs", `kubectl -n ${ns} logs -f job/${name}`],
    ["Job status", `kubectl -n ${ns} describe job ${name}`],
    ["Pods of this job", `kubectl -n ${ns} get pods -l job-name=${name}`],
    ["Delete job", `kubectl -n ${ns} delete job ${name}`],
  ];
  const dash = currentSettings?.dashboardUrl;
  showDialog({
    title: `kubectl: ${name}`,
    body: cmds.map(([label, cmd], i) => `
        <div class="cmd-label">${label}</div>
        <div class="cmd"><code>${escapeHtml(cmd)}</code>
          <button class="btn btn-secondary btn-sm" onclick="copyText(${i}, this)">Copy</button></div>`).join("")
      + (dash ? `<p style="margin-top: 16px;"><a href="${escapeHtml(dash)}" target="_blank" rel="noopener" style="color: var(--accent-blue);">Open Kubernetes dashboard &rarr;</a></p>` : ""),
  });
  kubectlCommands = cmds.map(c => c[1]);
}

let kubectlCommands = [];

function copyText(i, btn) {
  navigator.clipboard.writeText(kubectlCommands[i]).then(() => {
    btn.textContent = "Copied";
    setTimeout(() => (btn.textContent = "Copy"), 1500);
  });
}

// ---- Analytics ----

function renderAnalytics() {
  const finished = lastBuilds.filter(b => isFinished(b) && b.durationSec > 0).reverse(); // oldest first
  const succeeded = finished.filter(b => b.status === "Succeeded");
  const real = succeeded.filter(b => b.durationSec >= MIN_REAL_BUILD_SEC);
  const rate = finished.length ? Math.round((succeeded.length / finished.length) * 100) : null;
  const last = [...succeeded].reverse()[0];

  const kpis = [
    ["Finished jobs", finished.length],
    ["Success rate", rate === null ? "-" : `${rate}%`],
    ["Median build time", real.length ? formatDuration(median(real.map(b => b.durationSec))) : "-"],
    ["Last success", last && last.endTime ? new Date(last.endTime).toLocaleDateString() : "-"],
  ];
  document.getElementById("analytics-kpis").innerHTML = kpis.map(([t, v]) => `
    <div class="card"><div class="card-title">${t}</div><div class="kpi-value">${escapeHtml(String(v))}</div></div>`).join("");

  // Per-run bars (last 30)
  const runs = finished.slice(-30);
  const max = Math.max(1, ...runs.map(b => b.durationSec));
  document.getElementById("chart-runs").innerHTML = runs.length === 0
    ? `<p class="muted" style="margin-top: 16px;">No finished jobs yet.</p>`
    : `<div class="chart">${runs.map(b => `
        <div class="chart-col" title="${escapeHtml(b.name)} &mdash; ${escapeHtml(b.status)}, ${formatDuration(b.durationSec)}">
          <div class="chart-bar ${b.status === "Failed" ? "failed" : ""}" style="height: ${(b.durationSec / max) * 100}%"></div>
        </div>`).join("")}</div>
      <div class="chart-legend">
        <span><i style="background: var(--accent-blue)"></i>Succeeded</span>
        <span><i style="background: var(--status-error)"></i>Failed</span>
        <span>Tallest bar: ${formatDuration(max)}</span>
      </div>`;

  // Histogram
  const buckets = [
    ["<1m", 0, 60], ["1-5m", 60, 300], ["5-15m", 300, 900],
    ["15-30m", 900, 1800], ["30-60m", 1800, 3600], [">1h", 3600, Infinity],
  ].map(([label, lo, hi]) => ({ label, count: finished.filter(b => b.durationSec >= lo && b.durationSec < hi).length }));
  const maxCount = Math.max(1, ...buckets.map(b => b.count));
  document.getElementById("chart-histogram").innerHTML = finished.length === 0
    ? `<p class="muted" style="margin-top: 16px;">No finished jobs yet.</p>`
    : `<div class="chart" style="margin-bottom: 8px;">${buckets.map(b => `
        <div class="chart-col" style="max-width: 90px;">
          <span class="chart-count">${b.count}</span>
          <div class="chart-bar" style="height: ${(b.count / maxCount) * 85}%"></div>
          <span class="chart-label">${b.label}</span>
        </div>`).join("")}</div>`;
}

let logBuffer = [];
let renderScheduled = false;
const MAX_LOG_LINES = 2000;

function flushLogBuffer() {
  const term = document.getElementById("terminal-content");
  if (!term || logBuffer.length === 0) {
    renderScheduled = false;
    return;
  }

  // Check if user is currently scrolled near the bottom (within 80px)
  const isAtBottom = (term.scrollHeight - term.clientHeight) - term.scrollTop <= 80;

  // Append buffered chunks
  const chunk = logBuffer.join("\n") + "\n";
  logBuffer = [];

  term.textContent += chunk;

  // Trim excess lines if buffer grows large
  if (term.textContent.length > 500000) {
    const lines = term.textContent.split("\n");
    if (lines.length > MAX_LOG_LINES) {
      term.textContent = lines.slice(lines.length - MAX_LOG_LINES).join("\n");
    }
  }

  // Only auto-scroll down if user was already at bottom
  if (isAtBottom) {
    term.scrollTop = term.scrollHeight;
  } else {
    const scrollBtn = document.getElementById("btn-scroll-bottom");
    if (scrollBtn) scrollBtn.style.display = "inline-flex";
  }

  renderScheduled = false;
}

function scrollLogToBottom() {
  const term = document.getElementById("terminal-content");
  if (term) {
    term.scrollTop = term.scrollHeight;
    const scrollBtn = document.getElementById("btn-scroll-bottom");
    if (scrollBtn) scrollBtn.style.display = "none";
  }
}

function openLogs(jobName) {
  const modal = document.getElementById("log-modal");
  const title = document.getElementById("modal-job-title");
  const term = document.getElementById("terminal-content");
  const scrollBtn = document.getElementById("btn-scroll-bottom");

  if (scrollBtn) scrollBtn.style.display = "none";
  title.textContent = `Logs: ${jobName}`;
  term.textContent = "Connecting to log stream...\n";
  logBuffer = [];
  modal.classList.add("active");

  // Track manual scrolling: show/hide Jump to Bottom button
  term.onscroll = () => {
    const isAtBottom = (term.scrollHeight - term.clientHeight) - term.scrollTop <= 80;
    if (scrollBtn) {
      scrollBtn.style.display = isAtBottom ? "none" : "inline-flex";
    }
  };

  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }

  eventSource = new EventSource(`/api/builds/${encodeURIComponent(jobName)}/logs`);

  eventSource.onmessage = (event) => {
    logBuffer.push(event.data);
    if (!renderScheduled) {
      renderScheduled = true;
      requestAnimationFrame(flushLogBuffer);
    }
  };

  eventSource.onerror = () => {
    logBuffer.push("\n[Stream disconnected or job completed]");
    flushLogBuffer();
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
  };
}

function closeLogModal() {
  const modal = document.getElementById("log-modal");
  const term = document.getElementById("terminal-content");
  const scrollBtn = document.getElementById("btn-scroll-bottom");

  modal.classList.remove("active");
  if (scrollBtn) scrollBtn.style.display = "none";
  if (term) term.onscroll = null;

  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  logBuffer = [];
  renderScheduled = false;
  // Delay refreshing builds/packages to prevent blocking modal exit animation
  setTimeout(() => {
    fetchPackages();
    fetchBuilds();
  }, 100);
}

async function fetchSettings() {
  try {
    const res = await fetch("/api/settings");
    if (!res.ok) return;
    const data = await res.json();
    currentSettings = data;

    // Populate settings form
    const intervalSelect = document.getElementById("setting-interval");
    if (intervalSelect) {
      intervalSelect.value = data.autoCheckInterval || "6h";
    }

    const nodeInput = document.getElementById("setting-build-node");
    if (nodeInput) {
      nodeInput.value = data.buildNode || "";
    }

    const nsInput = document.getElementById("setting-namespace");
    if (nsInput) {
      nsInput.value = data.namespace || "";
    }

    const dashInput = document.getElementById("setting-dashboard");
    if (dashInput) {
      dashInput.value = data.dashboardUrl || "";
    }

    const repoInput = document.getElementById("setting-repo-name");
    if (repoInput) {
      repoInput.value = data.repoName || "";
    }

    // Update UI elements dependent on settings
    const repoBadge = document.getElementById("repo-badge");
    if (repoBadge) {
      repoBadge.textContent = `${data.repoName || "zfslocal"} • x86_64`;
    }

    const setupSnippet = document.getElementById("setup-snippet");
    if (setupSnippet) {
      setupSnippet.textContent = pacmanSnippet(data.repoName);
    }

    const intervalDesc = document.getElementById("info-interval-desc");
    if (intervalDesc) {
      const val = data.autoCheckInterval;
      if (!val || val === "0" || val === "0s") {
        intervalDesc.textContent = "manual trigger only (auto-checking disabled)";
      } else if (val === "1h") {
        intervalDesc.textContent = "every 1 hour";
      } else if (val === "3h") {
        intervalDesc.textContent = "every 3 hours";
      } else if (val === "6h") {
        intervalDesc.textContent = "every 6 hours";
      } else if (val === "12h") {
        intervalDesc.textContent = "every 12 hours";
      } else if (val === "24h") {
        intervalDesc.textContent = "every 24 hours";
      } else {
        intervalDesc.textContent = `every ${val}`;
      }
    }
  } catch (err) {
    console.error("Failed to fetch settings:", err);
  }
}

async function saveSettings() {
  const saveBtn = document.getElementById("btn-save-settings");
  const toast = document.getElementById("settings-toast");
  if (saveBtn) {
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving...";
  }

  const interval = document.getElementById("setting-interval").value;
  const buildNode = document.getElementById("setting-build-node").value.trim();

  try {
    const res = await fetch("/api/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        autoCheckInterval: interval,
        buildNode: buildNode,
      }),
    });

    if (!res.ok) {
      const errText = await res.text();
      showToast(toast, `Failed to save settings: ${errText}`, false);
    } else {
      await fetchSettings();
      showToast(toast, "Settings saved successfully!", true);
    }
  } catch (err) {
    showToast(toast, `Error: ${err.message}`, false);
  } finally {
    if (saveBtn) {
      saveBtn.disabled = false;
      saveBtn.textContent = "Save Settings";
    }
  }
}

function showToast(elem, message, isSuccess) {
  if (!elem) return;
  elem.textContent = message;
  elem.className = `toast ${isSuccess ? 'toast-success' : 'toast-error'}`;
  setTimeout(() => {
    elem.className = "toast";
  }, 4000);
}

function pacmanSnippet(repoName) {
  return `[${repoName || "zfslocal"}]\nSigLevel = Optional TrustAll\nServer = ${window.location.origin}/$repo/$arch`;
}

function copySetup() {
  navigator.clipboard.writeText(pacmanSnippet(currentSettings?.repoName)).then(() => {
    const btn = document.getElementById("btn-copy-setup");
    btn.textContent = "Copied";
    setTimeout(() => (btn.textContent = "Copy"), 1500);
  });
}

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/[&<>'"]/g, tag => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    "'": '&#39;',
    '"': '&quot;'
  }[tag] || tag));
}
