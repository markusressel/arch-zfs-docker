let allPackages = [];
let eventSource = null;
let currentSettings = null;

document.addEventListener("DOMContentLoaded", () => {
  fetchSettings();
  fetchPackages();
  fetchBuilds();
  fetchUpstream();

  // Search filter
  document.getElementById("search-input").addEventListener("input", (e) => {
    filterPackages(e.target.value);
  });

  // Form submit
  document.getElementById("build-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    await triggerBuild();
  });

  // Settings form submit
  const settingsForm = document.getElementById("settings-form");
  if (settingsForm) {
    settingsForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      await saveSettings();
    });
  }

  // Polling for builds & upstream status every 8 seconds
  setInterval(() => {
    fetchBuilds();
    fetchUpstream();
  }, 8000);
});

function switchTab(tab) {
  const dashTab = document.getElementById("tab-dashboard");
  const settTab = document.getElementById("tab-settings");
  const dashView = document.getElementById("view-dashboard");
  const settView = document.getElementById("view-settings");

  if (tab === "settings") {
    dashTab.classList.remove("active");
    settTab.classList.add("active");
    dashView.style.display = "none";
    settView.style.display = "block";
    fetchSettings();
  } else {
    settTab.classList.remove("active");
    dashTab.classList.add("active");
    settView.style.display = "none";
    dashView.style.display = "block";
    fetchPackages();
    fetchBuilds();
  }
}

async function fetchUpstream() {
  try {
    const res = await fetch("/api/upstream");
    const data = await res.json();
    const upElem = document.getElementById("stat-upstream");
    if (!upElem) return;

    if (Object.keys(data).length === 0) {
      upElem.textContent = "Checking...";
      return;
    }

    let allOk = true;
    let parts = [];
    for (const [pkg, status] of Object.entries(data)) {
      if (!status.upToDate) allOk = false;
      const statusIcon = status.upToDate ? "✓" : "⚡";
      parts.push(`${statusIcon} ${pkg}: ${status.upstreamKernel}`);
    }

    upElem.style.color = allOk ? "var(--accent-green)" : "var(--accent-orange)";
    upElem.textContent = parts.join(" | ");
  } catch (err) {
    console.error("Failed to fetch upstream status:", err);
  }
}

async function checkUpstreamNow() {
  const upElem = document.getElementById("stat-upstream");
  const formBtn = document.getElementById("btn-check-upstream");
  if (upElem) upElem.textContent = "Checking upstream...";
  if (formBtn) {
    formBtn.disabled = true;
    formBtn.textContent = "Checking...";
  }

  try {
    await fetch("/api/upstream/check", { method: "POST" });
    setTimeout(async () => {
      await fetchUpstream();
      await fetchBuilds();
    }, 2000);
  } catch (err) {
    alert("Failed to trigger check: " + err.message);
  } finally {
    if (formBtn) {
      setTimeout(() => {
        formBtn.disabled = false;
        formBtn.innerHTML = `
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"></polyline>
            <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
          </svg>
          Check Now
        `;
      }, 1000);
    }
  }
}

async function fetchPackages() {
  try {
    const res = await fetch("/api/packages");
    const data = await res.json();
    allPackages = data.packages || [];

    document.getElementById("stat-count").textContent = data.packageCount || 0;
    document.getElementById("stat-size").textContent = data.totalSizeHuman || "0 B";

    if (data.dbLastModified) {
      const dt = new Date(data.dbLastModified);
      document.getElementById("stat-updated").textContent = dt.toLocaleString();
    } else {
      document.getElementById("stat-updated").textContent = "Not created yet";
    }

    renderPackages(allPackages);
  } catch (err) {
    console.error("Failed to fetch packages:", err);
  }
}

function renderPackages(packages) {
  const tbody = document.getElementById("packages-table");
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
    const builds = await res.json();
    renderBuilds(builds);
  } catch (err) {
    console.error("Failed to fetch builds:", err);
  }
}

function formatDuration(seconds) {
  if (isNaN(seconds) || seconds < 0) return "-";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) {
    return `${s}s`;
  }
  const h = Math.floor(m / 60);
  const remM = m % 60;
  if (h === 0) {
    return `${m}m ${s < 10 ? '0' : ''}${s}s`;
  }
  return `${h}h ${remM < 10 ? '0' : ''}${remM}m ${s < 10 ? '0' : ''}${s}s`;
}

function updateLiveDurations() {
  document.querySelectorAll("[data-job-running='true']").forEach(elem => {
    const startIso = elem.getAttribute("data-start-time");
    if (!startIso) return;
    const startMs = new Date(startIso).getTime();
    if (isNaN(startMs)) return;
    const nowMs = Date.now();
    const elapsedSec = Math.max(0, Math.floor((nowMs - startMs) / 1000));
    elem.textContent = formatDuration(elapsedSec);
  });
}

// Update running job counters every second
setInterval(updateLiveDurations, 1000);

function renderBuilds(builds) {
  const tbody = document.getElementById("builds-table");
  if (!builds || builds.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" style="text-align: center; color: var(--text-muted);">No recent build jobs</td></tr>`;
    return;
  }

  tbody.innerHTML = builds.map(b => {
    const statusClass = `status-${b.status}`;
    const startTime = b.startTime ? new Date(b.startTime).toLocaleTimeString() : "-";
    const startIso = b.startTime ? b.startTime : "";
    const isRunning = b.status === "Running";

    let durationDisplay;
    if (isRunning && startIso) {
      const elapsed = Math.max(0, Math.floor((Date.now() - new Date(startIso).getTime()) / 1000));
      durationDisplay = `<span data-job-running="true" data-start-time="${escapeHtml(startIso)}">${formatDuration(elapsed)}</span>`;
    } else if (b.durationSec > 0) {
      durationDisplay = formatDuration(b.durationSec);
    } else {
      durationDisplay = "-";
    }

    return `
      <tr>
        <td><code>${escapeHtml(b.name)}</code></td>
        <td><span class="status-pill ${statusClass}">${escapeHtml(b.status)}</span></td>
        <td style="color: var(--text-secondary);">${startTime}</td>
        <td>${durationDisplay}</td>
        <td>
          <button class="btn btn-secondary btn-sm" onclick="openLogs('${escapeHtml(b.name)}')">
            Live Logs
          </button>
        </td>
      </tr>
    `;
  }).join("");
}

async function triggerBuild() {
  const btn = document.getElementById("btn-submit-build");
  btn.disabled = true;
  btn.textContent = "Starting job...";

  const kernel = document.getElementById("kernel-input").value.trim();
  const variant = document.getElementById("variant-select").value;
  const force = document.getElementById("force-build").checked;

  try {
    const res = await fetch("/api/builds", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        kernelVersion: kernel,
        variant: variant,
        forceBuild: force,
      }),
    });

    if (!res.ok) {
      const errText = await res.text();
      alert(`Failed to trigger build: ${errText}`);
    } else {
      const newJob = await res.json();
      await fetchBuilds();
      openLogs(newJob.name);
    }
  } catch (err) {
    alert(`Error: ${err.message}`);
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
      setupSnippet.textContent = `[${data.repoName || "zfslocal"}] Server = ${window.location.origin}/$repo/$arch`;
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

function copySetup() {
  const repoName = currentSettings?.repoName || "zfslocal";
  const snippet = `[${repoName}]\nSigLevel = Optional TrustAll\nServer = ${window.location.origin}/$repo/$arch`;
  navigator.clipboard.writeText(snippet).then(() => {
    alert("Copied pacman.conf configuration to clipboard!");
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
